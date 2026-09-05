package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

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

// repoFormats is the full Nexspence format enum (16 formats).
var repoFormats = []string{
	"maven2", "npm", "pypi", "docker", "oci", "go", "nuget", "raw",
	"apt", "yum", "helm", "cargo", "conan", "conda", "terraform", "rubygems",
}

// NewRepositoryResource is registered in provider.Resources.
func NewRepositoryResource() resource.Resource { return &repositoryResource{} }

type repositoryResource struct {
	client *client.Client
}

type repoProxyModel struct {
	RemoteURL         types.String `tfsdk:"remote_url"`
	RemoteUsername    types.String `tfsdk:"remote_username"`
	RemotePassword    types.String `tfsdk:"remote_password"`
	HTTPProxy         types.String `tfsdk:"http_proxy"`
	HTTPSProxy        types.String `tfsdk:"https_proxy"`
	SOCKS5Proxy       types.String `tfsdk:"socks5_proxy"`
	NoProxy           types.String `tfsdk:"no_proxy"`
	ProxyUsername     types.String `tfsdk:"proxy_username"`
	ProxyPassword     types.String `tfsdk:"proxy_password"`
	MinimumPackageAge types.Int64  `tfsdk:"minimum_package_age"`
	MetadataMaxAge    types.Int64  `tfsdk:"metadata_max_age"`
}

type repoGroupModel struct {
	MemberNames    []types.String `tfsdk:"member_names"`
	WritableMember types.String   `tfsdk:"writable_member"`
}

type repoAptModel struct {
	SigningKey           types.String `tfsdk:"signing_key"`
	SigningKeyPassphrase types.String `tfsdk:"signing_key_passphrase"`
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
	RoutingRuleID    types.String    `tfsdk:"routing_rule_id"`
	Proxy            *repoProxyModel `tfsdk:"proxy"`
	Group            *repoGroupModel `tfsdk:"group"`
	Apt              *repoAptModel   `tfsdk:"apt"`
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
			"routing_rule_id": schema.StringAttribute{
				Optional:    true,
				Description: "ID of a routing rule (nexspence_routing_rule.id) attached to this repository. Empty/omitted detaches it.",
			},
			"url": schema.StringAttribute{Computed: true},
		},
		Blocks: map[string]schema.Block{
			"proxy": schema.SingleNestedBlock{
				Description: "Proxy settings (type = proxy).",
				Attributes: map[string]schema.Attribute{
					"remote_url": schema.StringAttribute{Optional: true},
					"remote_username": schema.StringAttribute{
						Optional:    true,
						Description: "HTTP Basic username for the upstream registry itself (private Maven, Mapbox, corporate npm).",
					},
					"remote_password": schema.StringAttribute{
						Optional:    true,
						Sensitive:   true,
						Description: "HTTP Basic password for the upstream registry. Write-only — the API never returns it.",
					},
					"http_proxy": schema.StringAttribute{
						Optional:    true,
						Description: "Outbound HTTP forward-proxy URL used to reach the upstream.",
					},
					"https_proxy": schema.StringAttribute{
						Optional:    true,
						Description: "Outbound HTTPS forward-proxy URL used to reach the upstream.",
					},
					"socks5_proxy": schema.StringAttribute{
						Optional:    true,
						Description: "Outbound SOCKS5 proxy (takes precedence over http(s)_proxy).",
					},
					"no_proxy": schema.StringAttribute{
						Optional:    true,
						Description: "Comma-separated hosts that bypass the outbound proxy.",
					},
					"proxy_username": schema.StringAttribute{
						Optional:    true,
						Description: "Username for the outbound forward proxy (not the upstream registry).",
					},
					"proxy_password": schema.StringAttribute{
						Optional:    true,
						Sensitive:   true,
						Description: "Password for the outbound forward proxy. Write-only — the API never returns it.",
					},
					"minimum_package_age": schema.Int64Attribute{
						Optional:    true,
						Validators:  []validator.Int64{int64validator.AtLeast(1)},
						Description: "Supply-chain gate for npm/PyPI proxies: hide versions younger than this many seconds.",
					},
					"metadata_max_age": schema.Int64Attribute{
						Optional:    true,
						Validators:  []validator.Int64{int64validator.AtLeast(1)},
						Description: "Freshness TTL for proxied metadata (indexes, packuments) in seconds. Default on the server is 600.",
					},
				},
			},
			"apt": schema.SingleNestedBlock{
				Description: "APT hosted signing (format = apt).",
				Attributes: map[string]schema.Attribute{
					"signing_key": schema.StringAttribute{
						Optional:    true,
						Sensitive:   true,
						Description: "ASCII-armored OpenPGP private key used to sign Release/InRelease.",
					},
					"signing_key_passphrase": schema.StringAttribute{
						Optional:    true,
						Sensitive:   true,
						Description: "Passphrase for the signing key, when the key is protected.",
					},
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
	if data.Apt != nil && !data.Format.IsUnknown() && !data.Format.IsNull() && data.Format.ValueString() != "apt" {
		resp.Diagnostics.AddAttributeError(path.Root("apt"), "Unexpected apt block",
			`apt block is only valid when format = "apt"`)
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
// detachRouting sends an empty routingRuleId so an omitted attribute clears
// the attachment on update. Create leaves the pointer nil when unset.
func (r *repositoryResource) toAPI(ctx context.Context, m *repositoryModel, detachRouting bool) (*client.Repository, error) {
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
	if !m.RoutingRuleID.IsNull() && m.RoutingRuleID.ValueString() != "" {
		id := m.RoutingRuleID.ValueString()
		repo.RoutingRuleID = &id
	} else if detachRouting {
		empty := ""
		repo.RoutingRuleID = &empty
	}
	if m.Proxy != nil {
		repo.ProxyConfig = proxyConfigFromModel(m.Proxy)
	}
	fc := map[string]any{}
	if m.Group != nil {
		members := make([]any, 0, len(m.Group.MemberNames))
		for _, n := range m.Group.MemberNames {
			members = append(members, n.ValueString())
		}
		fc["member_names"] = members
		if !m.Group.WritableMember.IsNull() {
			fc["writable_member"] = m.Group.WritableMember.ValueString()
		}
	}
	if m.Apt != nil {
		setCfgSecret(fc, "signing_key", m.Apt.SigningKey)
		setCfgSecret(fc, "signing_key_passphrase", m.Apt.SigningKeyPassphrase)
	}
	if len(fc) > 0 {
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
	if repo.RoutingRuleID != nil && *repo.RoutingRuleID != "" {
		m.RoutingRuleID = types.StringValue(*repo.RoutingRuleID)
	} else {
		m.RoutingRuleID = types.StringNull()
	}

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
		m.Proxy = proxyModelFromConfig(repo.ProxyConfig, m.Proxy)
	} else {
		m.Proxy = nil
	}
	if repo.Type == "group" {
		g := &repoGroupModel{WritableMember: types.StringNull()}
		switch raw := map[string]any(repo.FormatConfig)["member_names"].(type) {
		case []string:
			for _, s := range raw {
				g.MemberNames = append(g.MemberNames, types.StringValue(s))
			}
		case []any:
			for _, v := range raw {
				if s, ok := v.(string); ok {
					g.MemberNames = append(g.MemberNames, types.StringValue(s))
				}
			}
		}
		if wm := cfgString(repo.FormatConfig, "writable_member"); wm != "" {
			g.WritableMember = types.StringValue(wm)
		}
		m.Group = g
	} else {
		m.Group = nil
	}
	if repo.Format == "apt" && (cfgString(repo.FormatConfig, "signing_key") != "" || m.Apt != nil) {
		apt := &repoAptModel{
			SigningKey:           optString(cfgString(repo.FormatConfig, "signing_key")),
			SigningKeyPassphrase: types.StringNull(),
		}
		if m.Apt != nil {
			if apt.SigningKey.IsNull() {
				apt.SigningKey = keepSecret(m.Apt.SigningKey)
			}
			apt.SigningKeyPassphrase = keepSecret(m.Apt.SigningKeyPassphrase)
		}
		m.Apt = apt
	} else if repo.Format != "apt" {
		m.Apt = nil
	}
	return nil
}

func (r *repositoryResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan repositoryModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	payload, err := r.toAPI(ctx, &plan, false)
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
	payload, err := r.toAPI(ctx, &plan, true)
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

func cfgString(cfg map[string]any, key string) string {
	if cfg == nil {
		return ""
	}
	s, _ := cfg[key].(string)
	return s
}

// cfgInt64 reads a whole-number config entry. JSON numbers arrive as float64;
// the server also accepts int and numeric strings (metadata_max_age convention).
func cfgInt64(cfg map[string]any, key string) (int64, bool) {
	if cfg == nil {
		return 0, false
	}
	raw, ok := cfg[key]
	if !ok || raw == nil {
		return 0, false
	}
	switch v := raw.(type) {
	case float64:
		return int64(v), true
	case float32:
		return int64(v), true
	case int:
		return int64(v), true
	case int64:
		return v, true
	case json.Number:
		n, err := v.Int64()
		if err != nil {
			f, ferr := v.Float64()
			if ferr != nil {
				return 0, false
			}
			return int64(f), true
		}
		return n, true
	case string:
		s := strings.TrimSpace(v)
		if s == "" {
			return 0, false
		}
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			f, ferr := strconv.ParseFloat(s, 64)
			if ferr != nil {
				return 0, false
			}
			return int64(f), true
		}
		return n, true
	default:
		return 0, false
	}
}

func setCfgString(cfg map[string]any, key string, v types.String) {
	if v.IsNull() || v.IsUnknown() {
		return
	}
	if s := v.ValueString(); s != "" {
		cfg[key] = s
	}
}

// setCfgSecret writes the value even when empty so an explicit "" clears a
// stored password (the server treats omitted secrets as "unchanged").
func setCfgSecret(cfg map[string]any, key string, v types.String) {
	if v.IsNull() || v.IsUnknown() {
		return
	}
	cfg[key] = v.ValueString()
}

func setCfgInt64(cfg map[string]any, key string, v types.Int64) {
	if v.IsNull() || v.IsUnknown() {
		return
	}
	cfg[key] = v.ValueInt64()
}

func optInt64(cfg map[string]any, key string) types.Int64 {
	if n, ok := cfgInt64(cfg, key); ok && n > 0 {
		return types.Int64Value(n)
	}
	return types.Int64Null()
}

func optString(s string) types.String {
	if s == "" {
		return types.StringNull()
	}
	return types.StringValue(s)
}

// keepSecret prefers the prior Terraform value: the API redacts passwords and
// returns only a *_set marker.
func keepSecret(prev types.String) types.String {
	if prev.IsNull() || prev.IsUnknown() {
		return types.StringNull()
	}
	return prev
}

func proxyConfigFromModel(p *repoProxyModel) map[string]any {
	if p == nil {
		return nil
	}
	pc := map[string]any{"remote_url": p.RemoteURL.ValueString()}
	setCfgString(pc, "remote_username", p.RemoteUsername)
	setCfgSecret(pc, "remote_password", p.RemotePassword)
	setCfgString(pc, "http_proxy", p.HTTPProxy)
	setCfgString(pc, "https_proxy", p.HTTPSProxy)
	setCfgString(pc, "socks5_proxy", p.SOCKS5Proxy)
	setCfgString(pc, "no_proxy", p.NoProxy)
	setCfgString(pc, "proxy_username", p.ProxyUsername)
	setCfgSecret(pc, "proxy_password", p.ProxyPassword)
	setCfgInt64(pc, "minimum_package_age", p.MinimumPackageAge)
	setCfgInt64(pc, "metadata_max_age", p.MetadataMaxAge)
	return pc
}

func proxyModelFromConfig(pc map[string]any, prev *repoProxyModel) *repoProxyModel {
	out := &repoProxyModel{
		RemoteURL:         types.StringValue(cfgString(pc, "remote_url")),
		RemoteUsername:    optString(cfgString(pc, "remote_username")),
		HTTPProxy:         optString(cfgString(pc, "http_proxy")),
		HTTPSProxy:        optString(cfgString(pc, "https_proxy")),
		SOCKS5Proxy:       optString(cfgString(pc, "socks5_proxy")),
		NoProxy:           optString(cfgString(pc, "no_proxy")),
		ProxyUsername:     optString(cfgString(pc, "proxy_username")),
		MinimumPackageAge: optInt64(pc, "minimum_package_age"),
		MetadataMaxAge:    optInt64(pc, "metadata_max_age"),
		RemotePassword:    types.StringNull(),
		ProxyPassword:     types.StringNull(),
	}
	if prev != nil {
		out.RemotePassword = keepSecret(prev.RemotePassword)
		out.ProxyPassword = keepSecret(prev.ProxyPassword)
	}
	return out
}
