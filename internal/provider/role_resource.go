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

// NewRoleResource is registered in provider.Resources.
func NewRoleResource() resource.Resource { return &roleResource{} }

type roleResource struct {
	client *client.Client
}

type roleModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Privileges  types.Set    `tfsdk:"privileges"`
}

func (r *roleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role"
}

func (r *roleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A Nexspence role grouping privileges.",
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
			"privileges": schema.SetAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Privilege names attached to this role.",
			},
		},
	}
}

func (r *roleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// privilegeNamesToIDs resolves privilege names -> IDs.
func (r *roleResource) privilegeNamesToIDs(ctx context.Context, names []string) ([]string, error) {
	all, err := r.client.ListPrivileges(ctx)
	if err != nil {
		return nil, fmt.Errorf("list privileges: %w", err)
	}
	byName := make(map[string]string, len(all))
	for _, p := range all {
		byName[p.Name] = p.ID
	}
	ids := make([]string, 0, len(names))
	for _, n := range names {
		id, ok := byName[n]
		if !ok {
			return nil, fmt.Errorf("privilege %q not found", n)
		}
		ids = append(ids, id)
	}
	return ids, nil
}

// fromAPI refreshes state, mapping privilege IDs -> names.
func (r *roleResource) fromAPI(ctx context.Context, m *roleModel, ro *client.Role) error {
	m.ID = types.StringValue(ro.ID)
	m.Name = types.StringValue(ro.Name)
	if ro.Description != "" {
		m.Description = types.StringValue(ro.Description)
	} else {
		m.Description = types.StringNull()
	}
	all, err := r.client.ListPrivileges(ctx)
	if err != nil {
		return fmt.Errorf("list privileges: %w", err)
	}
	byID := make(map[string]string, len(all))
	for _, p := range all {
		byID[p.ID] = p.Name
	}
	names := make([]string, 0, len(ro.Privileges))
	for _, id := range ro.Privileges {
		if n, ok := byID[id]; ok {
			names = append(names, n)
		}
	}
	if len(names) == 0 {
		m.Privileges = types.SetNull(types.StringType)
	} else {
		set, diags := types.SetValueFrom(ctx, types.StringType, names)
		if diags.HasError() {
			return errors.New("build privileges set")
		}
		m.Privileges = set
	}
	return nil
}

// planPrivilegeNames extracts the configured privilege names from the plan.
func (m *roleModel) planPrivilegeNames(ctx context.Context) ([]string, error) {
	if m.Privileges.IsNull() || m.Privileges.IsUnknown() {
		return nil, nil
	}
	var names []string
	if diags := m.Privileges.ElementsAs(ctx, &names, false); diags.HasError() {
		return nil, errors.New("decode privileges set")
	}
	return names, nil
}

func (r *roleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan roleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	names, err := plan.planPrivilegeNames(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Create role failed", err.Error())
		return
	}
	ids, err := r.privilegeNamesToIDs(ctx, names)
	if err != nil {
		resp.Diagnostics.AddError("Create role failed", err.Error())
		return
	}
	created, err := r.client.CreateRole(ctx, &client.Role{
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
		Privileges:  ids,
	})
	if err != nil {
		resp.Diagnostics.AddError("Create role failed", err.Error())
		return
	}
	// Re-fetch via list in case the echo body is incomplete (some Nexspence PUTs/POSTs
	// return partial responses — privileges may be missing from the create echo).
	fetched, fetchErr := r.readByID(ctx, created.ID)
	if fetchErr == nil {
		created = fetched
	}
	if err := r.fromAPI(ctx, &plan, created); err != nil {
		resp.Diagnostics.AddError("Refresh after create failed", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// readByID finds a role via list (the API has no GET-by-id for roles).
func (r *roleResource) readByID(ctx context.Context, id string) (*client.Role, error) {
	all, err := r.client.ListRoles(ctx)
	if err != nil {
		return nil, err
	}
	for i := range all {
		if all[i].ID == id {
			return &all[i], nil
		}
	}
	return nil, client.ErrNotFound
}

func (r *roleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state roleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	ro, err := r.readByID(ctx, state.ID.ValueString())
	if errors.Is(err, client.ErrNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Read role failed", err.Error())
		return
	}
	if err := r.fromAPI(ctx, &state, ro); err != nil {
		resp.Diagnostics.AddError("Read role failed", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *roleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state roleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.ID = state.ID
	names, err := plan.planPrivilegeNames(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Update role failed", err.Error())
		return
	}
	ids, err := r.privilegeNamesToIDs(ctx, names)
	if err != nil {
		resp.Diagnostics.AddError("Update role failed", err.Error())
		return
	}
	updated, err := r.client.UpdateRole(ctx, &client.Role{
		ID:          plan.ID.ValueString(),
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
		Privileges:  ids,
	})
	if err != nil {
		resp.Diagnostics.AddError("Update role failed", err.Error())
		return
	}
	// Re-fetch in case update echo is partial.
	fetched, fetchErr := r.readByID(ctx, updated.ID)
	if fetchErr == nil {
		updated = fetched
	}
	if err := r.fromAPI(ctx, &plan, updated); err != nil {
		resp.Diagnostics.AddError("Refresh after update failed", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *roleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state roleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteRole(ctx, state.ID.ValueString()); err != nil && !errors.Is(err, client.ErrNotFound) {
		resp.Diagnostics.AddError("Delete role failed", err.Error())
	}
}

// ImportState accepts the role NAME, resolves to ID via list.
func (r *roleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	all, err := r.client.ListRoles(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Import failed", err.Error())
		return
	}
	for _, ro := range all {
		if ro.Name == req.ID {
			resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), ro.ID)...)
			return
		}
	}
	resp.Diagnostics.AddError("Import failed", fmt.Sprintf("role %q not found", req.ID))
}
