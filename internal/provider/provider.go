package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ provider.Provider = &NatsjwtProvider{}
var _ provider.ProviderWithFunctions = &NatsjwtProvider{}

type NatsjwtProvider struct {
	version string
}

type NatsjwtProviderData struct {
	NatsUrl types.String `tfsdk:"nats_url"`
	Creds   types.String `tfsdk:"creds"`
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
		Attributes: map[string]schema.Attribute{
			"nats_url": schema.StringAttribute{
				Optional:    true,
				Description: "NATS server URL for resolver interactions (e.g. nats://localhost:4222). Required when using natsjwt_resolver_account resources.",
			},
			"creds": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "Contents of a NATS credentials file (.creds) for authentication.",
			},
		},
	}
}

func (p *NatsjwtProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config NatsjwtProviderData
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.ResourceData = &config
}

func (p *NatsjwtProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewNkeyResource,
		NewResolverAccountResource,
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
