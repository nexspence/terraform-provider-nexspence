package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/nexspence/terraform-provider-nexspence/internal/client"
)

// NewRepositoryDataSource is registered in provider.DataSources.
func NewRepositoryDataSource() datasource.DataSource { return &repositoryDataSource{} }

type repositoryDataSource struct {
	client *client.Client
}

type repositoryDataSourceModel struct {
	Name   types.String `tfsdk:"name"`
	Format types.String `tfsdk:"format"`
	Type   types.String `tfsdk:"type"`
	Online types.Bool   `tfsdk:"online"`
	URL    types.String `tfsdk:"url"`
}

func (d *repositoryDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_repository"
}

func (d *repositoryDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Look up one repository by name.",
		Attributes: map[string]schema.Attribute{
			"name":   schema.StringAttribute{Required: true},
			"format": schema.StringAttribute{Computed: true},
			"type":   schema.StringAttribute{Computed: true},
			"online": schema.BoolAttribute{Computed: true},
			"url":    schema.StringAttribute{Computed: true},
		},
	}
}

func (d *repositoryDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data", fmt.Sprintf("expected *client.Client, got %T", req.ProviderData))
		return
	}
	d.client = c
}

func (d *repositoryDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data repositoryDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	repo, err := d.client.GetRepository(ctx, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Read repository failed", err.Error())
		return
	}
	data.Format = types.StringValue(repo.Format)
	data.Type = types.StringValue(repo.Type)
	online := true
	if repo.Online != nil {
		online = *repo.Online
	}
	data.Online = types.BoolValue(online)
	data.URL = types.StringValue(repo.URL)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
