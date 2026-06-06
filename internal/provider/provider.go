package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/nexspence/terraform-provider-nexspence/internal/client"
)

// New returns the provider factory used by main and by acceptance tests.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &nexspenceProvider{version: version}
	}
}

type nexspenceProvider struct {
	version string
}

type providerModel struct {
	URL      types.String `tfsdk:"url"`
	Token    types.String `tfsdk:"token"`
	Username types.String `tfsdk:"username"`
	Password types.String `tfsdk:"password"`
}

func (p *nexspenceProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "nexspence"
	resp.Version = p.version
}

func (p *nexspenceProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manage Nexspence repositories, blob stores, and RBAC objects.",
		Attributes: map[string]schema.Attribute{
			"url": schema.StringAttribute{
				Optional:    true,
				Description: "Base URL of the Nexspence server. Falls back to NEXSPENCE_URL.",
			},
			"token": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "nxs_* API token (preferred). Falls back to NEXSPENCE_TOKEN. Mutually exclusive with username/password.",
			},
			"username": schema.StringAttribute{
				Optional:    true,
				Description: "Username for basic auth. Falls back to NEXSPENCE_USERNAME.",
			},
			"password": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "Password for basic auth. Falls back to NEXSPENCE_PASSWORD.",
			},
		},
	}
}

func (p *nexspenceProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var cfg providerModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	get := func(v types.String, env string) string {
		if !v.IsNull() && v.ValueString() != "" {
			return v.ValueString()
		}
		return os.Getenv(env)
	}
	c, err := client.New(client.Config{
		URL:      get(cfg.URL, "NEXSPENCE_URL"),
		Token:    get(cfg.Token, "NEXSPENCE_TOKEN"),
		Username: get(cfg.Username, "NEXSPENCE_USERNAME"),
		Password: get(cfg.Password, "NEXSPENCE_PASSWORD"),
	})
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("url"), "Invalid provider configuration", err.Error())
		return
	}
	resp.ResourceData = c
	resp.DataSourceData = c
}

func (p *nexspenceProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{NewBlobStoreResource, NewRepositoryResource, NewContentSelectorResource}
}

func (p *nexspenceProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return nil
}
