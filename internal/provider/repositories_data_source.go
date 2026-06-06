package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/nexspence/terraform-provider-nexspence/internal/client"
)

// NewRepositoriesDataSource is registered in provider.DataSources.
func NewRepositoriesDataSource() datasource.DataSource { return &repositoriesDataSource{} }

type repositoriesDataSource struct {
	client *client.Client
}

type repositoriesItemModel struct {
	Name   types.String `tfsdk:"name"`
	Format types.String `tfsdk:"format"`
	Type   types.String `tfsdk:"type"`
}

type repositoriesDataSourceModel struct {
	Repositories []repositoriesItemModel `tfsdk:"repositories"`
}

func (d *repositoriesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_repositories"
}

func (d *repositoriesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "List all repositories.",
		Attributes: map[string]schema.Attribute{
			"repositories": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name":   schema.StringAttribute{Computed: true},
						"format": schema.StringAttribute{Computed: true},
						"type":   schema.StringAttribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *repositoriesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *repositoriesDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	repos, err := d.client.ListRepositories(ctx)
	if err != nil {
		resp.Diagnostics.AddError("List repositories failed", err.Error())
		return
	}
	var data repositoriesDataSourceModel
	for _, r := range repos {
		data.Repositories = append(data.Repositories, repositoriesItemModel{
			Name:   types.StringValue(r.Name),
			Format: types.StringValue(r.Format),
			Type:   types.StringValue(r.Type),
		})
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
