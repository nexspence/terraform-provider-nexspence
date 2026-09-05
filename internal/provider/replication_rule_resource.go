package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/nexspence/terraform-provider-nexspence/internal/client"
)

// NewReplicationRuleResource is registered in provider.Resources.
func NewReplicationRuleResource() resource.Resource { return &replicationRuleResource{} }

type replicationRuleResource struct {
	client *client.Client
}

type replicationRuleModel struct {
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	SourceRepo     types.String `tfsdk:"source_repo"`
	TargetURL      types.String `tfsdk:"target_url"`
	TargetRepo     types.String `tfsdk:"target_repo"`
	TargetUsername types.String `tfsdk:"target_username"`
	TargetPassword types.String `tfsdk:"target_password"`
	CronExpr       types.String `tfsdk:"cron_expr"`
	Enabled        types.Bool   `tfsdk:"enabled"`
	LastRunAt      types.String `tfsdk:"last_run_at"`
	LastRunStatus  types.String `tfsdk:"last_run_status"`
}

func (r *replicationRuleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_replication_rule"
}

func (r *replicationRuleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A push-replication rule that copies artifacts from a local repository to a remote Nexspence instance on a cron schedule.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name":        schema.StringAttribute{Required: true},
			"source_repo": schema.StringAttribute{Required: true, Description: "Local repository name to push from."},
			"target_url":  schema.StringAttribute{Required: true, Description: "Base URL of the remote Nexspence instance."},
			"target_repo": schema.StringAttribute{Required: true, Description: "Repository name on the remote instance."},
			"target_username": schema.StringAttribute{
				Optional:    true,
				Description: "Username used to authenticate to the remote instance.",
			},
			"target_password": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "Password for the remote instance. Write-only — the API never returns it.",
			},
			"cron_expr": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("0 2 * * *"),
				Description: "Standard cron schedule. Empty means manual-only; the server default is nightly 02:00.",
			},
			"enabled": schema.BoolAttribute{
				Optional: true, Computed: true, Default: booldefault.StaticBool(true),
			},
			"last_run_at":     schema.StringAttribute{Computed: true, Description: "Timestamp of the last run, if any."},
			"last_run_status": schema.StringAttribute{Computed: true, Description: "ok, error, running, or empty."},
		},
	}
}

func (r *replicationRuleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data", fmt.Sprintf("expected *client.Client, got %T", req.ProviderData))
		return
	}
	r.client = c
}

func (m *replicationRuleModel) fromAPI(rr *client.ReplicationRule) {
	m.ID = types.StringValue(rr.ID)
	m.Name = types.StringValue(rr.Name)
	m.SourceRepo = types.StringValue(rr.SourceRepo)
	m.TargetURL = types.StringValue(rr.TargetURL)
	m.TargetRepo = types.StringValue(rr.TargetRepo)
	m.TargetUsername = optString(rr.TargetUsername)
	m.CronExpr = types.StringValue(rr.CronExpr)
	if rr.CronExpr == "" {
		m.CronExpr = types.StringValue("0 2 * * *")
	}
	m.Enabled = types.BoolValue(rr.Enabled)
	m.LastRunAt = optString(rr.LastRunAt)
	m.LastRunStatus = optString(rr.LastRunStatus)
}

func (m *replicationRuleModel) toAPI() *client.ReplicationRule {
	return &client.ReplicationRule{
		ID:             m.ID.ValueString(),
		Name:           m.Name.ValueString(),
		SourceRepo:     m.SourceRepo.ValueString(),
		TargetURL:      m.TargetURL.ValueString(),
		TargetRepo:     m.TargetRepo.ValueString(),
		TargetUsername: m.TargetUsername.ValueString(),
		TargetPassword: m.TargetPassword.ValueString(),
		CronExpr:       m.CronExpr.ValueString(),
		Enabled:        m.Enabled.ValueBool(),
	}
}

func (r *replicationRuleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan replicationRuleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	created, err := r.client.CreateReplicationRule(ctx, plan.toAPI())
	if err != nil {
		resp.Diagnostics.AddError("Create replication rule failed", err.Error())
		return
	}
	plan.fromAPI(created)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *replicationRuleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state replicationRuleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	rr, err := r.client.GetReplicationRule(ctx, state.ID.ValueString())
	if errors.Is(err, client.ErrNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Read replication rule failed", err.Error())
		return
	}
	state.fromAPI(rr) // target_password preserved
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *replicationRuleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state replicationRuleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.ID = state.ID
	updated, err := r.client.UpdateReplicationRule(ctx, plan.toAPI())
	if err != nil {
		resp.Diagnostics.AddError("Update replication rule failed", err.Error())
		return
	}
	plan.fromAPI(updated) // keeps plan.TargetPassword
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *replicationRuleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state replicationRuleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteReplicationRule(ctx, state.ID.ValueString()); err != nil && !errors.Is(err, client.ErrNotFound) {
		resp.Diagnostics.AddError("Delete replication rule failed", err.Error())
	}
}

// ImportState accepts the rule NAME, resolves it to the API ID via list.
// The target password cannot be imported (write-only) and must be re-supplied.
func (r *replicationRuleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	all, err := r.client.ListReplicationRules(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Import failed", err.Error())
		return
	}
	for _, rr := range all {
		if rr.Name == req.ID {
			resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), rr.ID)...)
			return
		}
	}
	resp.Diagnostics.AddError("Import failed", fmt.Sprintf("replication rule %q not found", req.ID))
}
