package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/nexspence/terraform-provider-nexspence/internal/client"
)

// NewRoutingRuleResource is registered in provider.Resources.
func NewRoutingRuleResource() resource.Resource { return &routingRuleResource{} }

type routingRuleResource struct {
	client *client.Client
}

type routingRuleModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Mode        types.String `tfsdk:"mode"`
	Matchers    types.List   `tfsdk:"matchers"`
}

func (r *routingRuleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_routing_rule"
}

func (r *routingRuleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A routing rule restricting which paths a repository may request. Attach it to a repository via routing_rule_id.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name":        schema.StringAttribute{Required: true},
			"description": schema.StringAttribute{Optional: true},
			"mode": schema.StringAttribute{
				Required:    true,
				Description: "ALLOW (only matching paths pass) or BLOCK (matching paths are denied).",
				Validators:  []validator.String{stringvalidator.OneOf("ALLOW", "BLOCK")},
			},
			"matchers": schema.ListAttribute{
				Required:    true,
				ElementType: types.StringType,
				Description: "Go regular expressions matched against the request path.",
			},
		},
	}
}

func (r *routingRuleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (m *routingRuleModel) fromAPI(ctx context.Context, rr *client.RoutingRule) {
	m.ID = types.StringValue(rr.ID)
	m.Name = types.StringValue(rr.Name)
	if rr.Description != "" {
		m.Description = types.StringValue(rr.Description)
	} else {
		m.Description = types.StringNull()
	}
	m.Mode = types.StringValue(rr.Mode)
	matchers := rr.Matchers
	if matchers == nil {
		matchers = []string{}
	}
	m.Matchers, _ = types.ListValueFrom(ctx, types.StringType, matchers)
}

func (m *routingRuleModel) toAPI(ctx context.Context) *client.RoutingRule {
	var matchers []string
	_ = m.Matchers.ElementsAs(ctx, &matchers, false)
	return &client.RoutingRule{
		ID:          m.ID.ValueString(),
		Name:        m.Name.ValueString(),
		Description: m.Description.ValueString(),
		Mode:        m.Mode.ValueString(),
		Matchers:    matchers,
	}
}

func (r *routingRuleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan routingRuleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	created, err := r.client.CreateRoutingRule(ctx, plan.toAPI(ctx))
	if err != nil {
		resp.Diagnostics.AddError("Create routing rule failed", err.Error())
		return
	}
	plan.fromAPI(ctx, created)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *routingRuleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state routingRuleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	rr, err := r.client.GetRoutingRule(ctx, state.ID.ValueString())
	if errors.Is(err, client.ErrNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Read routing rule failed", err.Error())
		return
	}
	state.fromAPI(ctx, rr)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *routingRuleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state routingRuleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.ID = state.ID
	updated, err := r.client.UpdateRoutingRule(ctx, plan.toAPI(ctx))
	if err != nil {
		resp.Diagnostics.AddError("Update routing rule failed", err.Error())
		return
	}
	plan.fromAPI(ctx, updated)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *routingRuleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state routingRuleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteRoutingRule(ctx, state.ID.ValueString()); err != nil && !errors.Is(err, client.ErrNotFound) {
		resp.Diagnostics.AddError("Delete routing rule failed", err.Error())
	}
}

// ImportState accepts the rule NAME, resolves it to the API ID via list.
func (r *routingRuleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	all, err := r.client.ListRoutingRules(ctx)
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
	resp.Diagnostics.AddError("Import failed", fmt.Sprintf("routing rule %q not found", req.ID))
}
