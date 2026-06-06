package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
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

func (p *nexspenceProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "nexspence"
	resp.Version = p.version
}

func (p *nexspenceProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manage Nexspence repositories, blob stores, and RBAC objects.",
		Attributes:  map[string]schema.Attribute{},
	}
}

func (p *nexspenceProvider) Configure(_ context.Context, _ provider.ConfigureRequest, _ *provider.ConfigureResponse) {
}

func (p *nexspenceProvider) Resources(_ context.Context) []func() resource.Resource {
	return nil
}

func (p *nexspenceProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return nil
}
