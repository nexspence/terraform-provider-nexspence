package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/nexspence/terraform-provider-nexspence/internal/client"
)

// NewCleanupPolicyResource is registered in provider.Resources.
func NewCleanupPolicyResource() resource.Resource { return &cleanupPolicyResource{} }

type cleanupPolicyResource struct {
	client *client.Client
}

type cleanupPolicyModel struct {
	ID                 types.String `tfsdk:"id"`
	Name               types.String `tfsdk:"name"`
	Description        types.String `tfsdk:"description"`
	Format             types.String `tfsdk:"format"`
	ScheduleCron       types.String `tfsdk:"schedule_cron"`
	Enabled            types.Bool   `tfsdk:"enabled"`
	DryRun             types.Bool   `tfsdk:"dry_run"`
	RetainNVersions    types.Int64  `tfsdk:"retain_n_versions"`
	LastDownloadedDays types.Int64  `tfsdk:"last_downloaded_days"`
	ArtifactAgeDays    types.Int64  `tfsdk:"artifact_age_days"`
	CriteriaPathPrefix types.String `tfsdk:"criteria_path_prefix"`
	CriteriaNameGlob   types.String `tfsdk:"criteria_name_glob"`
	ScopeRepository    types.String `tfsdk:"scope_repository"`
	ScopePathPrefix    types.String `tfsdk:"scope_path_prefix"`
}

func (r *cleanupPolicyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cleanup_policy"
}

func (r *cleanupPolicyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A cleanup policy that deletes stale assets on a schedule. Attach it to repositories via their cleanup_policy_ids.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name":        schema.StringAttribute{Required: true},
			"description": schema.StringAttribute{Optional: true},
			"format": schema.StringAttribute{
				Optional: true, Computed: true, Default: stringdefault.StaticString("*"),
				Description: "Repository format this policy applies to (\"*\" = all).",
			},
			"schedule_cron": schema.StringAttribute{
				Optional: true, Computed: true, Default: stringdefault.StaticString("0 2 * * *"),
				Description: "Cron expression for the background run.",
			},
			"enabled": schema.BoolAttribute{
				Optional: true, Computed: true, Default: booldefault.StaticBool(true),
			},
			"dry_run": schema.BoolAttribute{
				Optional: true, Computed: true, Default: booldefault.StaticBool(false),
				Description: "Log what would be deleted without deleting.",
			},
			"retain_n_versions": schema.Int64Attribute{
				Optional: true, Computed: true, Default: int64default.StaticInt64(0),
				Description: "Keep the N newest versions (0 = unlimited).",
			},
			"last_downloaded_days": schema.Int64Attribute{
				Optional:    true,
				Description: "Delete assets not downloaded in this many days (criteria).",
			},
			"artifact_age_days": schema.Int64Attribute{
				Optional:    true,
				Description: "Delete assets older than this many days (criteria).",
			},
			"criteria_path_prefix": schema.StringAttribute{
				Optional:    true,
				Description: "Only consider assets whose path starts with this prefix (criteria).",
			},
			"criteria_name_glob": schema.StringAttribute{
				Optional:    true,
				Description: "Only consider assets whose name matches this glob (criteria).",
			},
			"scope_repository": schema.StringAttribute{
				Optional:    true,
				Description: "Limit the policy to a single repository by name.",
			},
			"scope_path_prefix": schema.StringAttribute{
				Optional:    true,
				Description: "Limit the policy to assets under this path prefix.",
			},
		},
	}
}

func (r *cleanupPolicyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func optStr(s types.String) (string, bool) {
	if s.IsNull() || s.IsUnknown() {
		return "", false
	}
	return s.ValueString(), true
}

func critInt(m map[string]any, key string) types.Int64 {
	if v, ok := m[key]; ok {
		if f, ok := v.(float64); ok {
			return types.Int64Value(int64(f))
		}
	}
	return types.Int64Null()
}

func critStr(m map[string]any, key string) types.String {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok && s != "" {
			return types.StringValue(s)
		}
	}
	return types.StringNull()
}

func (m *cleanupPolicyModel) toAPI() *client.CleanupPolicy {
	crit := map[string]any{}
	if !m.LastDownloadedDays.IsNull() && !m.LastDownloadedDays.IsUnknown() {
		crit["lastDownloadedDays"] = m.LastDownloadedDays.ValueInt64()
	}
	if !m.ArtifactAgeDays.IsNull() && !m.ArtifactAgeDays.IsUnknown() {
		crit["artifactAgeDays"] = m.ArtifactAgeDays.ValueInt64()
	}
	if v, ok := optStr(m.CriteriaPathPrefix); ok {
		crit["pathPrefix"] = v
	}
	if v, ok := optStr(m.CriteriaNameGlob); ok {
		crit["nameGlob"] = v
	}

	var scope *client.CleanupScope
	repo, hasRepo := optStr(m.ScopeRepository)
	pathPrefix, hasPath := optStr(m.ScopePathPrefix)
	if hasRepo || hasPath {
		scope = &client.CleanupScope{RepositoryName: repo, PathPrefix: pathPrefix}
	}

	return &client.CleanupPolicy{
		ID:              m.ID.ValueString(),
		Name:            m.Name.ValueString(),
		Description:     m.Description.ValueString(),
		Format:          m.Format.ValueString(),
		Criteria:        crit,
		ScheduleCron:    m.ScheduleCron.ValueString(),
		Enabled:         m.Enabled.ValueBool(),
		DryRun:          m.DryRun.ValueBool(),
		RetainNVersions: m.RetainNVersions.ValueInt64(),
		Scope:           scope,
	}
}

func (m *cleanupPolicyModel) fromAPI(p *client.CleanupPolicy) {
	m.ID = types.StringValue(p.ID)
	m.Name = types.StringValue(p.Name)
	if p.Description != "" {
		m.Description = types.StringValue(p.Description)
	} else {
		m.Description = types.StringNull()
	}
	m.Format = types.StringValue(p.Format)
	m.ScheduleCron = types.StringValue(p.ScheduleCron)
	m.Enabled = types.BoolValue(p.Enabled)
	m.DryRun = types.BoolValue(p.DryRun)
	m.RetainNVersions = types.Int64Value(p.RetainNVersions)

	crit := p.Criteria
	if crit == nil {
		crit = map[string]any{}
	}
	m.LastDownloadedDays = critInt(crit, "lastDownloadedDays")
	m.ArtifactAgeDays = critInt(crit, "artifactAgeDays")
	m.CriteriaPathPrefix = critStr(crit, "pathPrefix")
	m.CriteriaNameGlob = critStr(crit, "nameGlob")

	if p.Scope != nil && p.Scope.RepositoryName != "" {
		m.ScopeRepository = types.StringValue(p.Scope.RepositoryName)
	} else {
		m.ScopeRepository = types.StringNull()
	}
	if p.Scope != nil && p.Scope.PathPrefix != "" {
		m.ScopePathPrefix = types.StringValue(p.Scope.PathPrefix)
	} else {
		m.ScopePathPrefix = types.StringNull()
	}
}

func (r *cleanupPolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan cleanupPolicyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	created, err := r.client.CreateCleanupPolicy(ctx, plan.toAPI())
	if err != nil {
		resp.Diagnostics.AddError("Create cleanup policy failed", err.Error())
		return
	}
	plan.fromAPI(created)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *cleanupPolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state cleanupPolicyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	p, err := r.client.GetCleanupPolicy(ctx, state.ID.ValueString())
	if errors.Is(err, client.ErrNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Read cleanup policy failed", err.Error())
		return
	}
	state.fromAPI(p)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *cleanupPolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state cleanupPolicyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.ID = state.ID
	updated, err := r.client.UpdateCleanupPolicy(ctx, plan.toAPI())
	if err != nil {
		resp.Diagnostics.AddError("Update cleanup policy failed", err.Error())
		return
	}
	plan.fromAPI(updated)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *cleanupPolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state cleanupPolicyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteCleanupPolicy(ctx, state.ID.ValueString()); err != nil && !errors.Is(err, client.ErrNotFound) {
		resp.Diagnostics.AddError("Delete cleanup policy failed", err.Error())
	}
}

// ImportState accepts the policy NAME, resolves it to the API ID via list.
func (r *cleanupPolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	all, err := r.client.ListCleanupPolicies(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Import failed", err.Error())
		return
	}
	for _, p := range all {
		if p.Name == req.ID {
			resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), p.ID)...)
			return
		}
	}
	resp.Diagnostics.AddError("Import failed", fmt.Sprintf("cleanup policy %q not found", req.ID))
}
