package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	schemavalidator "github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	natsjwt "github.com/nats-io/jwt/v2"
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
		},
	}
}

func (d *UserDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data UserDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	userKP, err := keypairFromSeed(data.Seed.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid User Seed", fmt.Sprintf("Failed to parse user seed: %s", err))
		return
	}

	userPub, err := userKP.PublicKey()
	if err != nil {
		resp.Diagnostics.AddError("Public Key Error", fmt.Sprintf("Failed to get user public key: %s", err))
		return
	}

	accountKP, err := keypairFromSeed(data.AccountSeed.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Account Seed", fmt.Sprintf("Failed to parse account seed: %s", err))
		return
	}

	claims := natsjwt.NewUserClaims(userPub)
	claims.Name = data.Name.ValueString()

	if !data.IssuerAccount.IsNull() {
		claims.IssuerAccount = data.IssuerAccount.ValueString()
	}

	// Permissions
	if !data.Permissions.IsNull() {
		var perms UserPermissionsModel
		resp.Diagnostics.Append(data.Permissions.As(ctx, &perms, objectAsOptions)...)
		if resp.Diagnostics.HasError() {
			return
		}

		var pubAllow, pubDeny, subAllow, subDeny []string
		if !perms.PubAllow.IsNull() {
			resp.Diagnostics.Append(perms.PubAllow.ElementsAs(ctx, &pubAllow, false)...)
		}
		if !perms.PubDeny.IsNull() {
			resp.Diagnostics.Append(perms.PubDeny.ElementsAs(ctx, &pubDeny, false)...)
		}
		if !perms.SubAllow.IsNull() {
			resp.Diagnostics.Append(perms.SubAllow.ElementsAs(ctx, &subAllow, false)...)
		}
		if !perms.SubDeny.IsNull() {
			resp.Diagnostics.Append(perms.SubDeny.ElementsAs(ctx, &subDeny, false)...)
		}
		if resp.Diagnostics.HasError() {
			return
		}

		claims.Pub = buildPermission(pubAllow, pubDeny)
		claims.Sub = buildPermission(subAllow, subDeny)

		if !perms.RespMaxMsgs.IsNull() || !perms.RespTTL.IsNull() {
			claims.Resp = &natsjwt.ResponsePermission{}
			if !perms.RespMaxMsgs.IsNull() {
				claims.Resp.MaxMsgs = int(perms.RespMaxMsgs.ValueInt64())
			}
			if !perms.RespTTL.IsNull() {
				ttl, err := time.ParseDuration(perms.RespTTL.ValueString())
				if err != nil {
					resp.Diagnostics.AddError("Invalid Duration", fmt.Sprintf("Failed to parse resp_ttl: %s", err))
					return
				}
				claims.Resp.Expires = ttl
			}
		}
	}

	// Limits
	if !data.Limits.IsNull() {
		var limits UserLimitsModel
		resp.Diagnostics.Append(data.Limits.As(ctx, &limits, objectAsOptions)...)
		if resp.Diagnostics.HasError() {
			return
		}
		if !limits.Subs.IsNull() {
			claims.Subs = limits.Subs.ValueInt64()
		} else {
			claims.Subs = -1
		}
		if !limits.Data.IsNull() {
			claims.Limits.Data = limits.Data.ValueInt64()
		} else {
			claims.Limits.Data = -1
		}
		if !limits.Payload.IsNull() {
			claims.Limits.NatsLimits.Payload = limits.Payload.ValueInt64()
		} else {
			claims.Limits.NatsLimits.Payload = -1
		}
	}

	// Bearer token
	if !data.BearerToken.IsNull() {
		claims.BearerToken = data.BearerToken.ValueBool()
	}

	// Allowed connection types
	if !data.AllowedConnectionTypes.IsNull() {
		var connTypes []string
		resp.Diagnostics.Append(data.AllowedConnectionTypes.ElementsAs(ctx, &connTypes, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		claims.AllowedConnectionTypes = connTypes
	}

	// Source networks
	if !data.SourceNetworks.IsNull() {
		var networks []string
		resp.Diagnostics.Append(data.SourceNetworks.ElementsAs(ctx, &networks, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		claims.Src = networks
	}

	// Time restrictions
	if !data.TimeRestrictions.IsNull() {
		var timeRanges []TimeRangeModel
		resp.Diagnostics.Append(data.TimeRestrictions.ElementsAs(ctx, &timeRanges, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		for _, tr := range timeRanges {
			claims.Times = append(claims.Times, natsjwt.TimeRange{
				Start: tr.Start.ValueString(),
				End:   tr.End.ValueString(),
			})
		}
	}

	// Locale
	if !data.Locale.IsNull() {
		claims.Locale = data.Locale.ValueString()
	}

	// Tags
	if !data.Tags.IsNull() {
		var tags []string
		resp.Diagnostics.Append(data.Tags.ElementsAs(ctx, &tags, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		claims.Tags = tags
	}

	jwtString, err := encodeDeterministic(claims, accountKP)
	if err != nil {
		resp.Diagnostics.AddError("JWT Encoding Error", fmt.Sprintf("Failed to encode user JWT: %s", err))
		return
	}

	data.PublicKey = types.StringValue(userPub)
	data.JWT = types.StringValue(jwtString)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
