package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
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

// NewBlobStoreResource is registered in provider.Resources.
func NewBlobStoreResource() resource.Resource { return &blobStoreResource{} }

type blobStoreResource struct {
	client *client.Client
}

type blobStoreS3Model struct {
	Bucket         types.String `tfsdk:"bucket"`
	Region         types.String `tfsdk:"region"`
	Endpoint       types.String `tfsdk:"endpoint"`
	AccessKey      types.String `tfsdk:"access_key"`
	SecretKey      types.String `tfsdk:"secret_key"`
	ForcePathStyle types.Bool   `tfsdk:"force_path_style"`
}

type blobStoreModel struct {
	ID         types.String      `tfsdk:"id"`
	Name       types.String      `tfsdk:"name"`
	Type       types.String      `tfsdk:"type"`
	Path       types.String      `tfsdk:"path"`
	S3         *blobStoreS3Model `tfsdk:"s3"`
	QuotaBytes types.Int64       `tfsdk:"quota_bytes"`
}

func (r *blobStoreResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_blobstore"
}

func (r *blobStoreResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A Nexspence blob store (local filesystem or S3).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"type": schema.StringAttribute{
				Required:      true,
				Validators:    []validator.String{stringvalidator.OneOf("local", "s3")},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"path": schema.StringAttribute{
				Optional:    true,
				Description: "Filesystem path (type = local).",
			},
			"quota_bytes": schema.Int64Attribute{
				Optional:   true,
				Validators: []validator.Int64{int64validator.AtLeast(1)},
			},
			"s3": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "S3 connection settings (type = s3).",
				Attributes: map[string]schema.Attribute{
					"bucket":           schema.StringAttribute{Required: true},
					"region":           schema.StringAttribute{Optional: true},
					"endpoint":         schema.StringAttribute{Optional: true, Description: "Custom S3 endpoint (MinIO/Ceph)."},
					"access_key":       schema.StringAttribute{Optional: true},
					"secret_key":       schema.StringAttribute{Optional: true, Sensitive: true},
					"force_path_style": schema.BoolAttribute{Optional: true},
				},
			},
		},
	}
}

func (r *blobStoreResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data blobStoreModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() || data.Type.IsUnknown() {
		return
	}
	switch data.Type.ValueString() {
	case "s3":
		if data.S3 == nil {
			resp.Diagnostics.AddAttributeError(path.Root("s3"), "Missing s3 block", `type = "s3" requires an s3 block`)
		}
	case "local":
		if data.S3 != nil {
			resp.Diagnostics.AddAttributeError(path.Root("s3"), "Unexpected s3 block", `s3 block is only valid when type = "s3"`)
		}
	}
}

func (r *blobStoreResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// toAPI builds the API payload from the plan model.
func (m *blobStoreModel) toAPI() *client.BlobStore {
	cfg := map[string]any{}
	switch m.Type.ValueString() {
	case "local":
		if !m.Path.IsNull() {
			cfg["path"] = m.Path.ValueString()
		}
	case "s3":
		cfg["bucket"] = m.S3.Bucket.ValueString()
		if !m.S3.Region.IsNull() {
			cfg["region"] = m.S3.Region.ValueString()
		}
		if !m.S3.Endpoint.IsNull() {
			cfg["endpoint"] = m.S3.Endpoint.ValueString()
		}
		if !m.S3.AccessKey.IsNull() {
			cfg["access_key"] = m.S3.AccessKey.ValueString()
		}
		if !m.S3.SecretKey.IsNull() {
			cfg["secret_key"] = m.S3.SecretKey.ValueString()
		}
		if !m.S3.ForcePathStyle.IsNull() {
			cfg["force_path_style"] = m.S3.ForcePathStyle.ValueBool()
		}
	}
	return &client.BlobStore{
		Name:       m.Name.ValueString(),
		Type:       m.Type.ValueString(),
		Config:     cfg,
		QuotaBytes: m.QuotaBytes.ValueInt64(),
	}
}

// fromAPI refreshes non-secret state from the API object. Secrets (secret_key)
// keep their prior state value — the API never returns them.
func (m *blobStoreModel) fromAPI(bs *client.BlobStore) {
	m.ID = types.StringValue(bs.ID)
	m.Name = types.StringValue(bs.Name)
	m.Type = types.StringValue(bs.Type)
	if bs.QuotaBytes > 0 {
		m.QuotaBytes = types.Int64Value(bs.QuotaBytes)
	} else {
		m.QuotaBytes = types.Int64Null()
	}
	str := func(k string) types.String {
		if v, ok := bs.Config[k].(string); ok && v != "" {
			return types.StringValue(v)
		}
		return types.StringNull()
	}
	switch bs.Type {
	case "local":
		m.Path = str("path")
	case "s3":
		if m.S3 == nil {
			m.S3 = &blobStoreS3Model{SecretKey: types.StringNull()}
		}
		m.S3.Bucket = str("bucket")
		m.S3.Region = str("region")
		m.S3.Endpoint = str("endpoint")
		m.S3.AccessKey = str("access_key")
		if v, ok := bs.Config["force_path_style"].(bool); ok {
			m.S3.ForcePathStyle = types.BoolValue(v)
		} else {
			m.S3.ForcePathStyle = types.BoolNull()
		}
	}
}

func (r *blobStoreResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan blobStoreModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	created, err := r.client.CreateBlobStore(ctx, plan.toAPI())
	if err != nil {
		resp.Diagnostics.AddError("Create blob store failed", err.Error())
		return
	}
	plan.fromAPI(created)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *blobStoreResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state blobStoreModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	bs, err := r.client.GetBlobStore(ctx, state.Name.ValueString())
	if errors.Is(err, client.ErrNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Read blob store failed", err.Error())
		return
	}
	state.fromAPI(bs)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *blobStoreResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan blobStoreModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if _, err := r.client.UpdateBlobStore(ctx, plan.toAPI()); err != nil {
		resp.Diagnostics.AddError("Update blob store failed", err.Error())
		return
	}
	// The PUT endpoint returns an empty id — re-fetch to get the full resource state.
	bs, err := r.client.GetBlobStore(ctx, plan.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Read blob store after update failed", err.Error())
		return
	}
	plan.fromAPI(bs)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *blobStoreResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state blobStoreModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteBlobStore(ctx, state.Name.ValueString()); err != nil && !errors.Is(err, client.ErrNotFound) {
		resp.Diagnostics.AddError("Delete blob store failed", err.Error())
	}
}

// ImportState imports by blob store name.
func (r *blobStoreResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
}
