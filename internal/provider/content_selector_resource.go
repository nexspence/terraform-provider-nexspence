package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/nexspence/terraform-provider-nexspence/internal/client"
)

// NewContentSelectorResource is registered in provider.Resources.
func NewContentSelectorResource() resource.Resource { return &contentSelectorResource{} }

type contentSelectorResource struct {
	client *client.Client
}

type contentSelectorModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Expression  types.String `tfsdk:"expression"`
}

func (r *contentSelectorResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_content_selector"
}

func (r *contentSelectorResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A CEL content selector (variables: format, path, repository).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"description": schema.StringAttribute{Optional: true},
			"expression":  schema.StringAttribute{Required: true},
		},
	}
}

func (r *contentSelectorResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (m *contentSelectorModel) fromAPI(cs *client.ContentSelector) {
	m.ID = types.StringValue(cs.ID)
	m.Name = types.StringValue(cs.Name)
	if cs.Description != "" {
		m.Description = types.StringValue(cs.Description)
	} else {
		m.Description = types.StringNull()
	}
	m.Expression = types.StringValue(cs.Expression)
}

func (r *contentSelectorResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan contentSelectorModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	created, err := r.client.CreateContentSelector(ctx, &client.ContentSelector{
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
		Expression:  plan.Expression.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Create content selector failed", err.Error())
		return
	}
	plan.fromAPI(created)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *contentSelectorResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state contentSelectorModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	cs, err := r.client.GetContentSelector(ctx, state.ID.ValueString())
	if errors.Is(err, client.ErrNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Read content selector failed", err.Error())
		return
	}
	state.fromAPI(cs)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *contentSelectorResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan contentSelectorModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Read state to get the ID (name is RequiresReplace so ID is stable across updates).
	var state contentSelectorModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.ID = state.ID

	updated, err := r.client.UpdateContentSelector(ctx, &client.ContentSelector{
		ID:          plan.ID.ValueString(),
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
		Expression:  plan.Expression.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Update content selector failed", err.Error())
		return
	}
	plan.fromAPI(updated)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *contentSelectorResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state contentSelectorModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteContentSelector(ctx, state.ID.ValueString()); err != nil && !errors.Is(err, client.ErrNotFound) {
		resp.Diagnostics.AddError("Delete content selector failed", err.Error())
	}
}

// ImportState accepts the selector NAME, resolves it to the API ID via list.
func (r *contentSelectorResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	all, err := r.client.ListContentSelectors(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Import failed", err.Error())
		return
	}
	for _, cs := range all {
		if cs.Name == req.ID {
			resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), cs.ID)...)
			return
		}
	}
	resp.Diagnostics.AddError("Import failed", fmt.Sprintf("content selector %q not found", req.ID))
}
