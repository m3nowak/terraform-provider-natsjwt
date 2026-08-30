package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	schemavalidator "github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/m3nowak/terraform-provider-natsjwt/internal/issuance"
	"github.com/nats-io/nkeys"
)

var _ datasource.DataSource = &UserDataSource{}

type UserDataSource struct{}

type UserPermissionsModel struct {
	PubAllow    types.List   `tfsdk:"pub_allow"`
	PubDeny     types.List   `tfsdk:"pub_deny"`
	SubAllow    types.List   `tfsdk:"sub_allow"`
	SubDeny     types.List   `tfsdk:"sub_deny"`
	RespMaxMsgs types.Int64  `tfsdk:"resp_max_msgs"`
	RespTTL     types.String `tfsdk:"resp_ttl"`
}

type UserLimitsModel struct {
	Subs    types.Int64 `tfsdk:"subs"`
	Data    types.Int64 `tfsdk:"data"`
	Payload types.Int64 `tfsdk:"payload"`
}

type TimeRangeModel struct {
	Start types.String `tfsdk:"start"`
	End   types.String `tfsdk:"end"`
}

type UserDataSourceModel struct {
	Name                   types.String `tfsdk:"name"`
	Seed                   types.String `tfsdk:"seed"`
	AccountSeed            types.String `tfsdk:"account_seed"`
	IssuerAccount          types.String `tfsdk:"issuer_account"`
	IssuedAt               types.Int64  `tfsdk:"issued_at"`
	Expires                types.Int64  `tfsdk:"expires"`
	NotBefore              types.Int64  `tfsdk:"not_before"`
	Permissions            types.Object `tfsdk:"permissions"`
	Limits                 types.Object `tfsdk:"limits"`
	BearerToken            types.Bool   `tfsdk:"bearer_token"`
	AllowedConnectionTypes types.List   `tfsdk:"allowed_connection_types"`
	SourceNetworks         types.List   `tfsdk:"source_networks"`
	TimeRestrictions       types.List   `tfsdk:"time_restrictions"`
	Locale                 types.String `tfsdk:"locale"`
	Tags                   types.List   `tfsdk:"tags"`
	PublicKey              types.String `tfsdk:"public_key"`
	JWT                    types.String `tfsdk:"jwt"`
	Creds                  types.String `tfsdk:"creds"`
}

func NewUserDataSource() datasource.DataSource {
	return &UserDataSource{}
}

func (d *UserDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

func (d *UserDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Generates a signed NATS user JWT from the given seeds and configuration.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required:    true,
				Description: "User name.",
			},
			"seed": schema.StringAttribute{
				Required:    true,
				Sensitive:   true,
				Description: "User NKey seed (starts with SU).",
				Validators:  []schemavalidator.String{SeedTypeValidator(nkeys.PrefixByteUser)},
			},
			"account_seed": schema.StringAttribute{
				Required:    true,
				Sensitive:   true,
				Description: "Account or signing key seed used to sign the user JWT (starts with SA).",
				Validators:  []schemavalidator.String{SeedTypeValidator(nkeys.PrefixByteAccount)},
			},
			"issuer_account": schema.StringAttribute{
				Optional:    true,
				Description: "Account public key. Set this when using a signing key instead of the account key directly.",
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
			"permissions": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "User permissions for publish and subscribe.",
				Attributes: map[string]schema.Attribute{
					"pub_allow": schema.ListAttribute{
						ElementType: types.StringType,
						Optional:    true,
						Description: "Subjects allowed for publishing.",
					},
					"pub_deny": schema.ListAttribute{
						ElementType: types.StringType,
						Optional:    true,
						Description: "Subjects denied for publishing.",
					},
					"sub_allow": schema.ListAttribute{
						ElementType: types.StringType,
						Optional:    true,
						Description: "Subjects allowed for subscribing.",
					},
					"sub_deny": schema.ListAttribute{
						ElementType: types.StringType,
						Optional:    true,
						Description: "Subjects denied for subscribing.",
					},
					"resp_max_msgs": schema.Int64Attribute{
						Optional:    true,
						Description: "Maximum number of response messages.",
					},
					"resp_ttl": schema.StringAttribute{
						Optional:    true,
						Description: "Response permission TTL (Go duration string, e.g., '1m', '5s').",
					},
				},
			},
			"limits": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Connection limits for the user.",
				Attributes: map[string]schema.Attribute{
					"subs": schema.Int64Attribute{
						Optional:    true,
						Description: "Maximum subscriptions. -1 for unlimited.",
					},
					"data": schema.Int64Attribute{
						Optional:    true,
						Description: "Maximum data in bytes. -1 for unlimited.",
					},
					"payload": schema.Int64Attribute{
						Optional:    true,
						Description: "Maximum payload size in bytes. -1 for unlimited.",
					},
				},
			},
			"bearer_token": schema.BoolAttribute{
				Optional:    true,
				Description: "Allow bearer token authentication. Default false.",
			},
			"allowed_connection_types": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Description: "Allowed connection types: STANDARD, WEBSOCKET, LEAFNODE, MQTT.",
			},
			"source_networks": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Description: "Allowed source networks (CIDR notation).",
			},
			"time_restrictions": schema.ListNestedAttribute{
				Optional:    true,
				Description: "Time-based access restrictions.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"start": schema.StringAttribute{
							Required:    true,
							Description: "Start time in HH:MM:SS format.",
						},
						"end": schema.StringAttribute{
							Required:    true,
							Description: "End time in HH:MM:SS format.",
						},
					},
				},
			},
			"locale": schema.StringAttribute{
				Optional:    true,
				Description: "Timezone for time restrictions (e.g., 'America/New_York').",
			},
			"tags": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Description: "Tags for the user.",
			},
			"public_key": schema.StringAttribute{
				Computed:    true,
				Description: "The user's public key.",
			},
			"jwt": schema.StringAttribute{
				Computed:    true,
				Description: "The signed user JWT.",
			},
			"creds": schema.StringAttribute{
				Computed:    true,
				Sensitive:   true,
				Description: "NATS user credentials file content (decorated JWT + decorated seed).",
			},
		},
	}
}

func (d *UserDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data UserDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	generateUser(ctx, &data, &resp.Diagnostics)
	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	}
}

func generateUser(ctx context.Context, data *UserDataSourceModel, diagnostics *diag.Diagnostics) {
	input := issuance.UserInput{
		Name:                   data.Name.ValueString(),
		Seed:                   data.Seed.ValueString(),
		AccountSeed:            data.AccountSeed.ValueString(),
		IssuerAccount:          optionalString(data.IssuerAccount),
		Temporal:               temporalInput(data.IssuedAt, data.Expires, data.NotBefore),
		BearerToken:            optionalBool(data.BearerToken),
		AllowedConnectionTypes: decodeStringList(ctx, data.AllowedConnectionTypes, diagnostics),
		SourceNetworks:         decodeStringList(ctx, data.SourceNetworks, diagnostics),
		Locale:                 optionalString(data.Locale),
		Tags:                   decodeStringList(ctx, data.Tags, diagnostics),
	}
	if !data.Permissions.IsNull() {
		var perms UserPermissionsModel
		diagnostics.Append(data.Permissions.As(ctx, &perms, objectAsOptions)...)
		input.Permissions = &issuance.UserPermissions{
			Permissions: issuance.Permissions{
				Publish: issuance.Permission{
					Allow: decodeStringList(ctx, perms.PubAllow, diagnostics),
					Deny:  decodeStringList(ctx, perms.PubDeny, diagnostics),
				},
				Subscribe: issuance.Permission{
					Allow: decodeStringList(ctx, perms.SubAllow, diagnostics),
					Deny:  decodeStringList(ctx, perms.SubDeny, diagnostics),
				},
			},
			ResponseMaxMessages: optionalInt64(perms.RespMaxMsgs),
			ResponseTTL:         optionalString(perms.RespTTL),
		}
	}
	if !data.Limits.IsNull() {
		var limits UserLimitsModel
		diagnostics.Append(data.Limits.As(ctx, &limits, objectAsOptions)...)
		input.Limits = &issuance.UserLimits{
			Subscriptions: optionalInt64(limits.Subs),
			Data:          optionalInt64(limits.Data),
			Payload:       optionalInt64(limits.Payload),
		}
	}
	if !data.TimeRestrictions.IsNull() {
		var timeRanges []TimeRangeModel
		diagnostics.Append(data.TimeRestrictions.ElementsAs(ctx, &timeRanges, false)...)
		for _, tr := range timeRanges {
			input.TimeRestrictions = append(input.TimeRestrictions, issuance.TimeRange{
				Start: tr.Start.ValueString(),
				End:   tr.End.ValueString(),
			})
		}
	}
	if diagnostics.HasError() {
		return
	}
	artifacts, err := issuance.IssueUser(input)
	if err != nil {
		diagnostics.AddError("JWT Issuance Error", err.Error())
		return
	}
	data.PublicKey = types.StringValue(artifacts.PublicKey)
	data.JWT = types.StringValue(artifacts.JWT)
	data.Creds = types.StringValue(artifacts.Creds)
}
