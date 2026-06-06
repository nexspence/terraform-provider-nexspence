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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/nexspence/terraform-provider-nexspence/internal/client"
)

// NewUserResource is registered in provider.Resources.
func NewUserResource() resource.Resource { return &userResource{} }

type userResource struct {
	client *client.Client
}

type userModel struct {
	Username  types.String `tfsdk:"username"`
	Password  types.String `tfsdk:"password"`
	Email     types.String `tfsdk:"email"`
	FirstName types.String `tfsdk:"first_name"`
	LastName  types.String `tfsdk:"last_name"`
	Status    types.String `tfsdk:"status"`
	Roles     types.Set    `tfsdk:"roles"`
}

func (r *userResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

func (r *userResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A local Nexspence user.",
		Attributes: map[string]schema.Attribute{
			"username": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"password": schema.StringAttribute{
				Required:    true,
				Sensitive:   true,
				Description: "Write-only: the API never returns it. Changing it triggers a password reset.",
			},
			"email":      schema.StringAttribute{Required: true},
			"first_name": schema.StringAttribute{Optional: true},
			"last_name":  schema.StringAttribute{Optional: true},
			"status": schema.StringAttribute{
				Optional:   true,
				Computed:   true,
				Default:    stringdefault.StaticString("active"),
				Validators: []validator.String{stringvalidator.OneOf("active", "disabled")},
			},
			"roles": schema.SetAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Role names assigned to the user.",
			},
		},
	}
}

func (r *userResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// roleNamesToIDs resolves role names -> role IDs via the roles list.
func (r *userResource) roleNamesToIDs(ctx context.Context, names []string) ([]string, error) {
	all, err := r.client.ListRoles(ctx)
	if err != nil {
		return nil, fmt.Errorf("list roles: %w", err)
	}
	byName := make(map[string]string, len(all))
	for _, ro := range all {
		byName[ro.Name] = ro.ID
	}
	ids := make([]string, 0, len(names))
	for _, n := range names {
		id, ok := byName[n]
		if !ok {
			return nil, fmt.Errorf("role %q not found", n)
		}
		ids = append(ids, id)
	}
	return ids, nil
}

// planRoleNames extracts role names from the plan model.
// Returns nil ONLY when the attribute is omitted (null/unknown); an explicit
// `roles = []` yields a non-nil empty slice so SetUserRoles still fires.
func (m *userModel) planRoleNames(ctx context.Context) ([]string, error) {
	if m.Roles.IsNull() || m.Roles.IsUnknown() {
		return nil, nil
	}
	names := []string{}
	if diags := m.Roles.ElementsAs(ctx, &names, false); diags.HasError() {
		return nil, errors.New("decode roles set")
	}
	return names, nil
}

// fromAPI refreshes non-secret state. GET returns role NAMES directly.
// Preserves null-vs-empty distinction for roles based on the model's prior value.
func (m *userModel) fromAPI(ctx context.Context, u *client.User) error {
	m.Username = types.StringValue(u.Username)
	m.Email = types.StringValue(u.Email)
	str := func(v string) types.String {
		if v == "" {
			return types.StringNull()
		}
		return types.StringValue(v)
	}
	m.FirstName = str(u.FirstName)
	m.LastName = str(u.LastName)
	m.Status = types.StringValue(u.Status)
	if len(u.Roles) == 0 && m.Roles.IsNull() {
		m.Roles = types.SetNull(types.StringType)
		return nil
	}
	roles := u.Roles
	if roles == nil {
		roles = []string{}
	}
	set, diags := types.SetValueFrom(ctx, types.StringType, roles)
	if diags.HasError() {
		return errors.New("build roles set")
	}
	m.Roles = set
	return nil
}

func (r *userResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan userModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	_, err := r.client.CreateUser(ctx, &client.User{
		Username:  plan.Username.ValueString(),
		Email:     plan.Email.ValueString(),
		FirstName: plan.FirstName.ValueString(),
		LastName:  plan.LastName.ValueString(),
		Status:    plan.Status.ValueString(),
	}, plan.Password.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Create user failed", err.Error())
		return
	}
	names, err := plan.planRoleNames(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Create user failed", err.Error())
		return
	}
	if names != nil {
		ids, err := r.roleNamesToIDs(ctx, names)
		if err != nil {
			resp.Diagnostics.AddError("Assign roles failed", err.Error())
			return
		}
		if err := r.client.SetUserRoles(ctx, plan.Username.ValueString(), ids); err != nil {
			resp.Diagnostics.AddError("Assign roles failed", err.Error())
			return
		}
	}
	u, err := r.client.GetUser(ctx, plan.Username.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Refresh after create failed", err.Error())
		return
	}
	if err := plan.fromAPI(ctx, u); err != nil {
		resp.Diagnostics.AddError("Refresh after create failed", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *userResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state userModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	u, err := r.client.GetUser(ctx, state.Username.ValueString())
	if errors.Is(err, client.ErrNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Read user failed", err.Error())
		return
	}
	if err := state.fromAPI(ctx, u); err != nil {
		resp.Diagnostics.AddError("Read user failed", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *userResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state userModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	username := plan.Username.ValueString()

	if _, err := r.client.UpdateUser(ctx, &client.User{
		Username:  username,
		Email:     plan.Email.ValueString(),
		FirstName: plan.FirstName.ValueString(),
		LastName:  plan.LastName.ValueString(),
		Status:    plan.Status.ValueString(),
	}); err != nil {
		resp.Diagnostics.AddError("Update user failed", err.Error())
		return
	}

	// Password change requested in config?
	if !plan.Password.Equal(state.Password) {
		if err := r.client.ChangePassword(ctx, username, plan.Password.ValueString()); err != nil {
			resp.Diagnostics.AddError("Change password failed", err.Error())
			return
		}
	}

	names, err := plan.planRoleNames(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Update user failed", err.Error())
		return
	}
	if names != nil {
		ids, err := r.roleNamesToIDs(ctx, names)
		if err != nil {
			resp.Diagnostics.AddError("Assign roles failed", err.Error())
			return
		}
		if err := r.client.SetUserRoles(ctx, username, ids); err != nil {
			resp.Diagnostics.AddError("Assign roles failed", err.Error())
			return
		}
	}

	u, err := r.client.GetUser(ctx, username)
	if err != nil {
		resp.Diagnostics.AddError("Refresh after update failed", err.Error())
		return
	}
	if err := plan.fromAPI(ctx, u); err != nil {
		resp.Diagnostics.AddError("Refresh after update failed", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *userResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state userModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteUser(ctx, state.Username.ValueString()); err != nil && !errors.Is(err, client.ErrNotFound) {
		resp.Diagnostics.AddError("Delete user failed", err.Error())
	}
}

// ImportState imports by username. Password lands as null in state — the next
// plan will show a password change; documented in the resource docs.
func (r *userResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("username"), req, resp)
}
