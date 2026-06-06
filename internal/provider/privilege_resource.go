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

const privilegeType = "repository-content-selector"

// NewPrivilegeResource is registered in provider.Resources.
func NewPrivilegeResource() resource.Resource { return &privilegeResource{} }

type privilegeResource struct {
	client *client.Client
}

type privilegeModel struct {
	ID              types.String `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	Description     types.String `tfsdk:"description"`
	ContentSelector types.String `tfsdk:"content_selector"`
}

func (r *privilegeResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_privilege"
}

func (r *privilegeResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A content-selector-scoped privilege (type repository-content-selector).",
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
			"content_selector": schema.StringAttribute{
				Required:    true,
				Description: "Name of the content selector this privilege grants access through.",
			},
		},
	}
}

func (r *privilegeResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// selectorIDByName resolves a content selector name to its ID.
func (r *privilegeResource) selectorIDByName(ctx context.Context, name string) (string, error) {
	all, err := r.client.ListContentSelectors(ctx)
	if err != nil {
		return "", fmt.Errorf("list content selectors: %w", err)
	}
	for _, cs := range all {
		if cs.Name == name {
			return cs.ID, nil
		}
	}
	return "", fmt.Errorf("content selector %q not found", name)
}

// fromAPI refreshes state, mapping contentSelectorId -> name.
func (r *privilegeResource) fromAPI(ctx context.Context, m *privilegeModel, p *client.Privilege) error {
	m.ID = types.StringValue(p.ID)
	m.Name = types.StringValue(p.Name)
	if p.Description != "" {
		m.Description = types.StringValue(p.Description)
	} else {
		m.Description = types.StringNull()
	}
	m.ContentSelector = types.StringNull()
	if p.ContentSelectorID != "" {
		cs, err := r.client.GetContentSelector(ctx, p.ContentSelectorID)
		if err != nil && !errors.Is(err, client.ErrNotFound) {
			return fmt.Errorf("resolve content selector: %w", err)
		}
		if cs != nil {
			m.ContentSelector = types.StringValue(cs.Name)
		}
	}
	return nil
}

func (r *privilegeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan privilegeModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	csID, err := r.selectorIDByName(ctx, plan.ContentSelector.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Create privilege failed", err.Error())
		return
	}
	created, err := r.client.CreatePrivilege(ctx, &client.Privilege{
		Name:              plan.Name.ValueString(),
		Description:       plan.Description.ValueString(),
		Type:              privilegeType,
		ContentSelectorID: csID,
	})
	if err != nil {
		resp.Diagnostics.AddError("Create privilege failed", err.Error())
		return
	}
	if err := r.fromAPI(ctx, &plan, created); err != nil {
		resp.Diagnostics.AddError("Refresh after create failed", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *privilegeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state privilegeModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	p, err := r.client.GetPrivilege(ctx, state.ID.ValueString())
	if errors.Is(err, client.ErrNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Read privilege failed", err.Error())
		return
	}
	if err := r.fromAPI(ctx, &state, p); err != nil {
		resp.Diagnostics.AddError("Read privilege failed", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *privilegeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state privilegeModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.ID = state.ID
	csID, err := r.selectorIDByName(ctx, plan.ContentSelector.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Update privilege failed", err.Error())
		return
	}
	updated, err := r.client.UpdatePrivilege(ctx, &client.Privilege{
		ID:                plan.ID.ValueString(),
		Name:              plan.Name.ValueString(),
		Description:       plan.Description.ValueString(),
		Type:              privilegeType,
		ContentSelectorID: csID,
	})
	if err != nil {
		resp.Diagnostics.AddError("Update privilege failed", err.Error())
		return
	}
	if err := r.fromAPI(ctx, &plan, updated); err != nil {
		resp.Diagnostics.AddError("Refresh after update failed", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *privilegeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state privilegeModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeletePrivilege(ctx, state.ID.ValueString()); err != nil && !errors.Is(err, client.ErrNotFound) {
		resp.Diagnostics.AddError("Delete privilege failed", err.Error())
	}
}

// ImportState accepts the privilege NAME, resolves to ID via list.
func (r *privilegeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	all, err := r.client.ListPrivileges(ctx)
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
	resp.Diagnostics.AddError("Import failed", fmt.Sprintf("privilege %q not found", req.ID))
}
