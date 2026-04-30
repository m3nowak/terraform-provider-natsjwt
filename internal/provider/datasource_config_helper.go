package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	natsjwt "github.com/nats-io/jwt/v2"
)

var _ datasource.DataSource = &ConfigHelperDataSource{}

type ConfigHelperDataSource struct{}

type ConfigHelperDataSourceModel struct {
	OperatorJWT      types.String `tfsdk:"operator_jwt"`
	AccountJWTs      types.List   `tfsdk:"account_jwts"`
	SystemAccountJWT types.String `tfsdk:"system_account_jwt"`
	ServerConfig     types.String `tfsdk:"server_config"`
	Operator         types.String `tfsdk:"operator"`
	SystemAccount    types.String `tfsdk:"system_account"`
	ResolverPreload  types.Map    `tfsdk:"resolver_preload"`
}

func NewConfigHelperDataSource() datasource.DataSource {
	return &ConfigHelperDataSource{}
}

func (d *ConfigHelperDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_config_helper"
}

func (d *ConfigHelperDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Generates NATS server configuration snippet from operator and account JWTs.",
		Attributes: map[string]schema.Attribute{
			"operator_jwt": schema.StringAttribute{
				Required:    true,
				Description: "The operator JWT.",
			},
			"account_jwts": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Description: "List of account JWTs to include in the resolver preload.",
			},
			"system_account_jwt": schema.StringAttribute{
				Optional:    true,
				Description: "The system account JWT.",
			},
			"server_config": schema.StringAttribute{
				Computed:    true,
				Description: "Complete NATS server configuration snippet containing operator, system_account, and resolver_preload.",
			},
			"operator": schema.StringAttribute{
				Computed:    true,
				Description: "The operator JWT value for the config.",
			},
			"system_account": schema.StringAttribute{
				Computed:    true,
				Description: "The system account public key.",
			},
			"resolver_preload": schema.MapAttribute{
				ElementType: types.StringType,
				Computed:    true,
				Description: "Map of account public keys to their JWTs for preloading in the resolver.",
			},
		},
	}
}

func (d *ConfigHelperDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ConfigHelperDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	operatorJWT := data.OperatorJWT.ValueString()

	preload := make(map[string]string)

	var systemAccountPub string
	if !data.SystemAccountJWT.IsNull() {
		sysJWT := data.SystemAccountJWT.ValueString()
		sysClaims, err := natsjwt.DecodeAccountClaims(sysJWT)
		if err != nil {
			resp.Diagnostics.AddError("Invalid System Account JWT",
				fmt.Sprintf("Failed to decode system account JWT: %s", err))
			return
		}
		systemAccountPub = sysClaims.Subject
		preload[systemAccountPub] = sysJWT
	}

	if !data.AccountJWTs.IsNull() {
		var accountJWTs []string
		resp.Diagnostics.Append(data.AccountJWTs.ElementsAs(ctx, &accountJWTs, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		for _, jwt := range accountJWTs {
			acctClaims, err := natsjwt.DecodeAccountClaims(jwt)
			if err != nil {
				resp.Diagnostics.AddError("Invalid Account JWT",
					fmt.Sprintf("Failed to decode account JWT: %s", err))
				return
			}
			preload[acctClaims.Subject] = jwt
		}
	}

	preloadMap := make(map[string]string)
	for k, v := range preload {
		preloadMap[k] = v
	}

	preloadTF, diags := types.MapValueFrom(ctx, types.StringType, preloadMap)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("operator: %s\n", operatorJWT))
	if systemAccountPub != "" {
		sb.WriteString(fmt.Sprintf("system_account: %s\n", systemAccountPub))
	}
	if len(preload) > 0 {
		sb.WriteString("resolver_preload: {\n")
		for pub, jwt := range preload {
			sb.WriteString(fmt.Sprintf("  %s: %s\n", pub, jwt))
		}
		sb.WriteString("}\n")
	}

	data.ServerConfig = types.StringValue(sb.String())
	data.Operator = types.StringValue(operatorJWT)
	data.SystemAccount = types.StringValue(systemAccountPub)
	data.ResolverPreload = preloadTF

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
