package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

var _ provider.Provider = &NatsjwtProvider{}
var _ provider.ProviderWithFunctions = &NatsjwtProvider{}

type NatsjwtProvider struct {
	version string
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &NatsjwtProvider{
			version: version,
		}
	}
}

func (p *NatsjwtProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "natsjwt"
	resp.Version = p.version
}

func (p *NatsjwtProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manage NATS JWT credentials offline without a running NATS server.",
	}
}

func (p *NatsjwtProvider) Configure(_ context.Context, _ provider.ConfigureRequest, _ *provider.ConfigureResponse) {
}

func (p *NatsjwtProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewNkeyResource,
	}
}

func (p *NatsjwtProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewOperatorDataSource,
		NewAccountDataSource,
		NewSystemAccountDataSource,
		NewUserDataSource,
		NewConfigHelperDataSource,
	}
}

func (p *NatsjwtProvider) Functions(_ context.Context) []func() function.Function {
	return []func() function.Function{
		NewSeedPublicKeyFunction,
	}
}
