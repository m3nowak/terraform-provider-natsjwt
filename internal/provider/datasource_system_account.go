package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	natsjwt "github.com/nats-io/jwt/v2"
)

var _ datasource.DataSource = &SystemAccountDataSource{}

type SystemAccountDataSource struct{}

func NewSystemAccountDataSource() datasource.DataSource {
	return &SystemAccountDataSource{}
}

func (d *SystemAccountDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_system_account"
}

func (d *SystemAccountDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = accountSchema("Generates a signed NATS system account JWT with system-appropriate defaults (includes $SYS.> public service export).")
}

func (d *SystemAccountDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data AccountDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	generateAccount(ctx, &data, true, &resp.Diagnostics)
	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	}
}

func applySystemAccountDefaults(claims *natsjwt.AccountClaims) {
	// Add the $SYS.> public service export, matching nsc behavior
	hasSysExport := false
	for _, exp := range claims.Exports {
		if exp.Subject == "$SYS.>" {
			hasSysExport = true
			break
		}
	}
	if !hasSysExport {
		claims.Exports = append(claims.Exports, &natsjwt.Export{
			Name:                 "account-monitoring-services",
			Subject:              "$SYS.REQ.ACCOUNT.*.*",
			Type:                 natsjwt.Service,
			ResponseType:         natsjwt.ResponseTypeStream,
			AccountTokenPosition: 4,
			Info: natsjwt.Info{
				Description: "Request account specific monitoring services for: SUBSZ, CONNZ, LEAFZ, JSZ and INFO",
				InfoURL:     "https://docs.nats.io/nats-server/configuration/sys_accounts",
			},
		})
		claims.Exports = append(claims.Exports, &natsjwt.Export{
			Name:                 "account-monitoring-streams",
			Subject:              "$SYS.ACCOUNT.*.>",
			Type:                 natsjwt.Stream,
			AccountTokenPosition: 3,
			Info: natsjwt.Info{
				Description: "Account specific monitoring stream",
				InfoURL:     "https://docs.nats.io/nats-server/configuration/sys_accounts",
			},
		})
	}
}
