package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/nexspence/terraform-provider-nexspence/internal/client"
)

// repoFormats is the full Nexspence format enum (14 formats).
var repoFormats = []string{
	"maven2", "npm", "pypi", "docker", "go", "nuget", "raw",
	"apt", "yum", "helm", "cargo", "conan", "conda", "terraform",
}

// NewRepositoryResource is registered in provider.Resources.
func NewRepositoryResource() resource.Resource { return &repositoryResource{} }

type repositoryResource struct {
	client *client.Client
}

type repoProxyModel struct {
	RemoteURL types.String `tfsdk:"remote_url"`
}

type repoGroupModel struct {
	MemberNames    []types.String `tfsdk:"member_names"`
	WritableMember types.String   `tfsdk:"writable_member"`
}

type repositoryModel struct {
	ID               types.String    `tfsdk:"id"`
	Name             types.String    `tfsdk:"name"`
	Format           types.String    `tfsdk:"format"`
	Type             types.String    `tfsdk:"type"`
	BlobStore        types.String    `tfsdk:"blob_store"`
	Online           types.Bool      `tfsdk:"online"`
	AllowAnonymous   types.Bool      `tfsdk:"allow_anonymous"`
	Description      types.String    `tfsdk:"description"`
	QuotaBytes       types.Int64     `tfsdk:"quota_bytes"`
	CleanupPolicyIDs []types.String  `tfsdk:"cleanup_policy_ids"`
	Proxy            *repoProxyModel `tfsdk:"proxy"`
	Group            *repoGroupModel `tfsdk:"group"`
	URL              types.String    `tfsdk:"url"`
}

func (r *repositoryResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_repository"
}

func (r *repositoryResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A Nexspence repository (hosted, proxy, or group) of any supported format.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"format": schema.StringAttribute{
				Required:      true,
				Validators:    []validator.String{stringvalidator.OneOf(repoFormats...)},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"type": schema.StringAttribute{
				Required:      true,
				Validators:    []validator.String{stringvalidator.OneOf("hosted", "proxy", "group")},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"blob_store": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("default"),
				Description: "Blob store name (resolved to its ID against the API).",
			},
			"online":          schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(true)},
			"allow_anonymous": schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(false)},
			"description":     schema.StringAttribute{Optional: true},
			"quota_bytes": schema.Int64Attribute{
				Optional:   true,
				Validators: []validator.Int64{int64validator.AtLeast(1)},
			},
			"cleanup_policy_ids": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
			},
			"url": schema.StringAttribute{Computed: true},
		},
		Blocks: map[string]schema.Block{
			"proxy": schema.SingleNestedBlock{
				Description: "Proxy settings (type = proxy).",
				Attributes: map[string]schema.Attribute{
					"remote_url": schema.StringAttribute{Optional: true},
				},
			},
			"group": schema.SingleNestedBlock{
				Description: "Group settings (type = group).",
				Attributes: map[string]schema.Attribute{
					"member_names": schema.ListAttribute{
						Optional:    true,
						ElementType: types.StringType,
						Validators:  []validator.List{listvalidator.SizeAtLeast(1)},
					},
					"writable_member": schema.StringAttribute{Optional: true},
				},
			},
		},
	}
}

func (r *repositoryResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data repositoryModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() || data.Type.IsUnknown() || data.Type.IsNull() {
		return
	}
	typ := data.Type.ValueString()
	if typ == "proxy" {
		if data.Proxy == nil || data.Proxy.RemoteURL.IsNull() {
			resp.Diagnostics.AddAttributeError(path.Root("proxy"), "Missing proxy configuration",
				`type = "proxy" requires a proxy block with remote_url`)
		}
	} else if data.Proxy != nil {
		resp.Diagnostics.AddAttributeError(path.Root("proxy"), "Unexpected proxy block",
			`proxy block is only valid when type = "proxy"`)
	}
	if typ == "group" {
		if data.Group == nil || len(data.Group.MemberNames) == 0 {
			resp.Diagnostics.AddAttributeError(path.Root("group"), "Missing group configuration",
				`type = "group" requires a group block with member_names`)
		}
	} else if data.Group != nil {
		resp.Diagnostics.AddAttributeError(path.Root("group"), "Unexpected group block",
			`group block is only valid when type = "group"`)
	}
}

func (r *repositoryResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// toAPI resolves blob_store name -> ID and assembles the API payload.
func (r *repositoryResource) toAPI(ctx context.Context, m *repositoryModel) (*client.Repository, error) {
	bs, err := r.client.GetBlobStore(ctx, m.BlobStore.ValueString())
	if err != nil {
		return nil, fmt.Errorf("resolve blob store %q: %w", m.BlobStore.ValueString(), err)
	}
	online := m.Online.ValueBool()
	repo := &client.Repository{
		Name:           m.Name.ValueString(),
		Format:         m.Format.ValueString(),
		Type:           m.Type.ValueString(),
		BlobStoreID:    bs.ID,
		Online:         &online,
		AllowAnonymous: m.AllowAnonymous.ValueBool(),
		Description:    m.Description.ValueString(),
		QuotaBytes:     m.QuotaBytes.ValueInt64(),
	}
	for _, id := range m.CleanupPolicyIDs {
		repo.CleanupPolicyIDs = append(repo.CleanupPolicyIDs, id.ValueString())
	}
	if m.Proxy != nil {
		repo.ProxyConfig = map[string]any{"remote_url": m.Proxy.RemoteURL.ValueString()}
	}
	if m.Group != nil {
		members := make([]any, 0, len(m.Group.MemberNames))
		for _, n := range m.Group.MemberNames {
			members = append(members, n.ValueString())
		}
		fc := map[string]any{"member_names": members}
		if !m.Group.WritableMember.IsNull() {
			fc["writable_member"] = m.Group.WritableMember.ValueString()
		}
		repo.FormatConfig = fc
	}
	return repo, nil
}

// fromAPI refreshes state from the API object, mapping blobStoreId -> name.
func (r *repositoryResource) fromAPI(ctx context.Context, m *repositoryModel, repo *client.Repository) error {
	m.ID = types.StringValue(repo.ID)
	m.Name = types.StringValue(repo.Name)
	m.Format = types.StringValue(repo.Format)
	m.Type = types.StringValue(repo.Type)
	if repo.Online != nil {
		m.Online = types.BoolValue(*repo.Online)
	} else {
		m.Online = types.BoolValue(true)
	}
	m.AllowAnonymous = types.BoolValue(repo.AllowAnonymous)
	if repo.Description != "" {
		m.Description = types.StringValue(repo.Description)
	} else {
		m.Description = types.StringNull()
	}
	if repo.QuotaBytes > 0 {
		m.QuotaBytes = types.Int64Value(repo.QuotaBytes)
	} else {
		m.QuotaBytes = types.Int64Null()
	}
	m.CleanupPolicyIDs = nil
	for _, id := range repo.CleanupPolicyIDs {
		m.CleanupPolicyIDs = append(m.CleanupPolicyIDs, types.StringValue(id))
	}
	m.URL = types.StringValue(repo.URL)

	// blobStoreId -> name
	m.BlobStore = types.StringNull()
	if repo.BlobStoreID != "" {
		stores, err := r.client.ListBlobStores(ctx)
		if err != nil {
			return fmt.Errorf("list blob stores: %w", err)
		}
		for _, s := range stores {
			if s.ID == repo.BlobStoreID {
				m.BlobStore = types.StringValue(s.Name)
				break
			}
		}
	}

	if repo.Type == "proxy" {
		ru, _ := repo.ProxyConfig["remote_url"].(string)
		m.Proxy = &repoProxyModel{RemoteURL: types.StringValue(ru)}
	} else {
		m.Proxy = nil
	}
	if repo.Type == "group" {
		g := &repoGroupModel{WritableMember: types.StringNull()}
		if raw, ok := repo.FormatConfig["member_names"].([]any); ok {
			for _, v := range raw {
				if s, ok := v.(string); ok {
					g.MemberNames = append(g.MemberNames, types.StringValue(s))
				}
			}
		}
		if wm, ok := repo.FormatConfig["writable_member"].(string); ok && wm != "" {
			g.WritableMember = types.StringValue(wm)
		}
		m.Group = g
	} else {
		m.Group = nil
	}
	return nil
}

func (r *repositoryResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan repositoryModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	payload, err := r.toAPI(ctx, &plan)
	if err != nil {
		resp.Diagnostics.AddError("Create repository failed", err.Error())
		return
	}
	if _, err := r.client.CreateRepository(ctx, payload); err != nil {
		resp.Diagnostics.AddError("Create repository failed", err.Error())
		return
	}
	created, err := r.client.GetRepository(ctx, plan.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Refresh after create failed", err.Error())
		return
	}
	if err := r.fromAPI(ctx, &plan, created); err != nil {
		resp.Diagnostics.AddError("Refresh after create failed", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *repositoryResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state repositoryModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	repo, err := r.client.GetRepository(ctx, state.Name.ValueString())
	if errors.Is(err, client.ErrNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Read repository failed", err.Error())
		return
	}
	if err := r.fromAPI(ctx, &state, repo); err != nil {
		resp.Diagnostics.AddError("Read repository failed", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *repositoryResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan repositoryModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	payload, err := r.toAPI(ctx, &plan)
	if err != nil {
		resp.Diagnostics.AddError("Update repository failed", err.Error())
		return
	}
	if _, err := r.client.UpdateRepository(ctx, payload); err != nil {
		resp.Diagnostics.AddError("Update repository failed", err.Error())
		return
	}
	updated, err := r.client.GetRepository(ctx, plan.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Refresh after update failed", err.Error())
		return
	}
	if err := r.fromAPI(ctx, &plan, updated); err != nil {
		resp.Diagnostics.AddError("Refresh after update failed", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *repositoryResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state repositoryModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteRepository(ctx, state.Name.ValueString()); err != nil && !errors.Is(err, client.ErrNotFound) {
		resp.Diagnostics.AddError("Delete repository failed", err.Error())
	}
}

// ImportState imports by repository name.
func (r *repositoryResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
}
