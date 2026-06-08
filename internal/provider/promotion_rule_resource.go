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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/nexspence/terraform-provider-nexspence/internal/client"
)

// NewPromotionRuleResource is registered in provider.Resources.
func NewPromotionRuleResource() resource.Resource { return &promotionRuleResource{} }

type promotionRuleResource struct {
	client *client.Client
}

type promotionRuleModel struct {
	ID                    types.String `tfsdk:"id"`
	Name                  types.String `tfsdk:"name"`
	FromRepo              types.String `tfsdk:"from_repo"`
	ToRepo                types.String `tfsdk:"to_repo"`
	PathFilter            types.String `tfsdk:"path_filter"`
	RequireScanPass       types.Bool   `tfsdk:"require_scan_pass"`
	RequireManualApproval types.Bool   `tfsdk:"require_manual_approval"`
}

func (r *promotionRuleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_promotion_rule"
}

func (r *promotionRuleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A build-promotion rule copying components from one repository to another, optionally gated on a CEL path filter, a passing security scan, and/or manual approval.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name":      schema.StringAttribute{Required: true},
			"from_repo": schema.StringAttribute{Required: true, Description: "Source repository name."},
			"to_repo":   schema.StringAttribute{Required: true, Description: "Target repository name (must differ from from_repo)."},
			"path_filter": schema.StringAttribute{
				Optional:    true,
				Description: "CEL expression over format, path, repository. Empty matches all paths.",
			},
			"require_scan_pass": schema.BoolAttribute{
				Optional: true, Computed: true, Default: booldefault.StaticBool(false),
				Description: "Block promotion when the component has HIGH/CRITICAL scan findings.",
			},
			"require_manual_approval": schema.BoolAttribute{
				Optional: true, Computed: true, Default: booldefault.StaticBool(false),
				Description: "Hold promotions as pending until an admin approves.",
			},
		},
	}
}

func (r *promotionRuleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (m *promotionRuleModel) fromAPI(pr *client.PromotionRule) {
	m.ID = types.StringValue(pr.ID)
	m.Name = types.StringValue(pr.Name)
	m.FromRepo = types.StringValue(pr.FromRepo)
	m.ToRepo = types.StringValue(pr.ToRepo)
	if pr.PathFilter != "" {
		m.PathFilter = types.StringValue(pr.PathFilter)
	} else {
		m.PathFilter = types.StringNull()
	}
	m.RequireScanPass = types.BoolValue(pr.RequireScanPass)
	m.RequireManualApproval = types.BoolValue(pr.RequireManualApproval)
}

func (m *promotionRuleModel) toAPI() *client.PromotionRule {
	return &client.PromotionRule{
		ID:                    m.ID.ValueString(),
		Name:                  m.Name.ValueString(),
		FromRepo:              m.FromRepo.ValueString(),
		ToRepo:                m.ToRepo.ValueString(),
		PathFilter:            m.PathFilter.ValueString(),
		RequireScanPass:       m.RequireScanPass.ValueBool(),
		RequireManualApproval: m.RequireManualApproval.ValueBool(),
	}
}

func (r *promotionRuleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan promotionRuleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	created, err := r.client.CreatePromotionRule(ctx, plan.toAPI())
	if err != nil {
		resp.Diagnostics.AddError("Create promotion rule failed", err.Error())
		return
	}
	plan.fromAPI(created)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *promotionRuleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state promotionRuleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	pr, err := r.client.GetPromotionRule(ctx, state.ID.ValueString())
	if errors.Is(err, client.ErrNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Read promotion rule failed", err.Error())
		return
	}
	state.fromAPI(pr)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *promotionRuleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state promotionRuleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.ID = state.ID
	updated, err := r.client.UpdatePromotionRule(ctx, plan.toAPI())
	if err != nil {
		resp.Diagnostics.AddError("Update promotion rule failed", err.Error())
		return
	}
	plan.fromAPI(updated)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *promotionRuleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state promotionRuleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeletePromotionRule(ctx, state.ID.ValueString()); err != nil && !errors.Is(err, client.ErrNotFound) {
		resp.Diagnostics.AddError("Delete promotion rule failed", err.Error())
	}
}

// ImportState accepts the rule NAME, resolves it to the API ID via list.
func (r *promotionRuleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	all, err := r.client.ListPromotionRules(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Import failed", err.Error())
		return
	}
	for _, pr := range all {
		if pr.Name == req.ID {
			resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), pr.ID)...)
			return
		}
	}
	resp.Diagnostics.AddError("Import failed", fmt.Sprintf("promotion rule %q not found", req.ID))
}
