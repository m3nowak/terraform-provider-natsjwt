package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/m3nowak/terraform-provider-natsjwt/internal/issuance"
	"github.com/nats-io/nkeys"
)

var _ datasource.DataSource = &OperatorDataSource{}

type OperatorDataSource struct{}

type OperatorDataSourceModel struct {
	Name                  types.String `tfsdk:"name"`
	Seed                  types.String `tfsdk:"seed"`
	SigningKeys           types.List   `tfsdk:"signing_keys"`
	AccountServerURL      types.String `tfsdk:"account_server_url"`
	OperatorServiceURLs   types.List   `tfsdk:"operator_service_urls"`
	SystemAccount         types.String `tfsdk:"system_account"`
	StrictSigningKeyUsage types.Bool   `tfsdk:"strict_signing_key_usage"`
	IssuedAt              types.Int64  `tfsdk:"issued_at"`
	Expires               types.Int64  `tfsdk:"expires"`
	NotBefore             types.Int64  `tfsdk:"not_before"`
	Tags                  types.List   `tfsdk:"tags"`
	PublicKey             types.String `tfsdk:"public_key"`
	JWT                   types.String `tfsdk:"jwt"`
}

func NewOperatorDataSource() datasource.DataSource {
	return &OperatorDataSource{}
}

func (d *OperatorDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_operator"
}

func (d *OperatorDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Generates a signed NATS operator JWT from the given seed and configuration.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Operator name.",
			},
			"seed": schema.StringAttribute{
				Required:    true,
				Sensitive:   true,
				Description: "Operator NKey seed (starts with SO).",
				Validators:  []validator.String{SeedTypeValidator(nkeys.PrefixByteOperator)},
			},
			"signing_keys": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Description: "Additional signing key public keys.",
			},
			"account_server_url": schema.StringAttribute{
				Optional:    true,
				Description: "Account server URL.",
			},
			"operator_service_urls": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Description: "Operator service URLs.",
			},
			"system_account": schema.StringAttribute{
				Optional:    true,
				Description: "Public key of the system account.",
			},
			"strict_signing_key_usage": schema.BoolAttribute{
				Optional:    true,
				Description: "Require signing keys for all operations.",
			},
			"issued_at": schema.Int64Attribute{
				Optional:    true,
				Description: "JWT issued-at timestamp as Unix seconds. Defaults to 0 (Unix epoch).",
			},
			"expires": schema.Int64Attribute{
				Optional:    true,
				Description: "JWT expiration timestamp as Unix seconds. Defaults to no expiration.",
			},
			"not_before": schema.Int64Attribute{
				Optional:    true,
				Description: "JWT not-before timestamp as Unix seconds. Defaults to issued_at.",
			},
			"tags": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Description: "Tags for the operator.",
			},
			"public_key": schema.StringAttribute{
				Computed:    true,
				Description: "The operator's public key.",
			},
			"jwt": schema.StringAttribute{
				Computed:    true,
				Description: "The signed operator JWT.",
			},
		},
	}
}

func (d *OperatorDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data OperatorDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	generateOperator(ctx, &data, &resp.Diagnostics)
	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	}
}

func generateOperator(ctx context.Context, data *OperatorDataSourceModel, diagnostics *diag.Diagnostics) {
	input := issuance.OperatorInput{
		Name:                  data.Name.ValueString(),
		Seed:                  data.Seed.ValueString(),
		SigningKeys:           decodeStringList(ctx, data.SigningKeys, diagnostics),
		AccountServerURL:      optionalString(data.AccountServerURL),
		OperatorServiceURLs:   decodeStringList(ctx, data.OperatorServiceURLs, diagnostics),
		SystemAccount:         optionalString(data.SystemAccount),
		StrictSigningKeyUsage: optionalBool(data.StrictSigningKeyUsage),
		Temporal:              temporalInput(data.IssuedAt, data.Expires, data.NotBefore),
		Tags:                  decodeStringList(ctx, data.Tags, diagnostics),
	}
	if diagnostics.HasError() {
		return
	}
	artifacts, err := issuance.IssueOperator(input)
	if err != nil {
		diagnostics.AddError("JWT Issuance Error", err.Error())
		return
	}
	data.PublicKey = types.StringValue(artifacts.PublicKey)
	data.JWT = types.StringValue(artifacts.JWT)
}
