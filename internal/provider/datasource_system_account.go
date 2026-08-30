package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
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
