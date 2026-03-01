package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	schemavalidator "github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	natsjwt "github.com/nats-io/jwt/v2"
	"github.com/nats-io/nkeys"
)

var _ datasource.DataSource = &AccountDataSource{}

type AccountDataSource struct{}

// Shared model types used by both account and system_account data sources.

type NatsLimitsModel struct {
	Subs    types.Int64 `tfsdk:"subs"`
	Data    types.Int64 `tfsdk:"data"`
	Payload types.Int64 `tfsdk:"payload"`
}

type AccountLimitsModel struct {
	Imports          types.Int64 `tfsdk:"imports"`
	Exports          types.Int64 `tfsdk:"exports"`
	WildcardExports  types.Bool  `tfsdk:"wildcard_exports"`
	DisallowBearer   types.Bool  `tfsdk:"disallow_bearer"`
	Conn             types.Int64 `tfsdk:"conn"`
	LeafNodeConn     types.Int64 `tfsdk:"leaf_node_conn"`
}

type JetStreamLimitsModel struct {
	Tier               types.String `tfsdk:"tier"`
	MemStorage         types.Int64  `tfsdk:"mem_storage"`
	DiskStorage        types.Int64  `tfsdk:"disk_storage"`
	Streams            types.Int64  `tfsdk:"streams"`
	Consumer           types.Int64  `tfsdk:"consumer"`
	MaxAckPending      types.Int64  `tfsdk:"max_ack_pending"`
	MemMaxStreamBytes  types.Int64  `tfsdk:"mem_max_stream_bytes"`
	DiskMaxStreamBytes types.Int64  `tfsdk:"disk_max_stream_bytes"`
	MaxBytesRequired   types.Bool   `tfsdk:"max_bytes_required"`
}

type DefaultPermissionsModel struct {
	PubAllow types.List `tfsdk:"pub_allow"`
	PubDeny  types.List `tfsdk:"pub_deny"`
	SubAllow types.List `tfsdk:"sub_allow"`
	SubDeny  types.List `tfsdk:"sub_deny"`
}

type TraceModel struct {
	Destination types.String `tfsdk:"destination"`
	Sampling    types.Int64  `tfsdk:"sampling"`
}

type AccountDataSourceModel struct {
	Name               types.String `tfsdk:"name"`
	Seed               types.String `tfsdk:"seed"`
	OperatorSeed       types.String `tfsdk:"operator_seed"`
	SigningKeys        types.List   `tfsdk:"signing_keys"`
	IssuedAt           types.Int64  `tfsdk:"issued_at"`
	Expires            types.Int64  `tfsdk:"expires"`
	NotBefore          types.Int64  `tfsdk:"not_before"`
	Description        types.String `tfsdk:"description"`
	InfoURL            types.String `tfsdk:"info_url"`
	Tags               types.List   `tfsdk:"tags"`
	NatsLimits         types.Object `tfsdk:"nats_limits"`
	AccountLimits      types.Object `tfsdk:"account_limits"`
	JetStreamLimits    types.List   `tfsdk:"jetstream_limits"`
	DefaultPermissions types.Object `tfsdk:"default_permissions"`
	Trace              types.Object `tfsdk:"trace"`
	PublicKey          types.String `tfsdk:"public_key"`
	JWT                types.String `tfsdk:"jwt"`
}

func NewAccountDataSource() datasource.DataSource {
	return &AccountDataSource{}
}

func (d *AccountDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_account"
}

func (d *AccountDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = accountSchema("Generates a signed NATS account JWT from the given seeds and configuration.")
}

func accountSchema(description string) schema.Schema {
	return schema.Schema{
		Description: description,
		Attributes:  accountSchemaAttributes(),
	}
}

func accountSchemaAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"name": schema.StringAttribute{
			Required:    true,
			Description: "Account name.",
		},
		"seed": schema.StringAttribute{
			Required:    true,
			Sensitive:   true,
			Description: "Account NKey seed (starts with SA).",
			Validators:  []schemavalidator.String{SeedTypeValidator(nkeys.PrefixByteAccount)},
		},
		"operator_seed": schema.StringAttribute{
			Required:    true,
			Sensitive:   true,
			Description: "Operator or signing key seed used to sign the account JWT (starts with SO).",
			Validators:  []schemavalidator.String{SeedTypeValidator(nkeys.PrefixByteOperator)},
		},
		"signing_keys": schema.ListAttribute{
			ElementType: types.StringType,
			Optional:    true,
			Description: "Additional signing key public keys for this account.",
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
		"description": schema.StringAttribute{
			Optional:    true,
			Description: "Account description.",
		},
		"info_url": schema.StringAttribute{
			Optional:    true,
			Description: "Link to external information about this account.",
		},
		"tags": schema.ListAttribute{
			ElementType: types.StringType,
			Optional:    true,
			Description: "Tags for the account.",
		},
		"nats_limits": schema.SingleNestedAttribute{
			Optional:    true,
			Description: "NATS connection limits.",
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
		"account_limits": schema.SingleNestedAttribute{
			Optional:    true,
			Description: "Account-level limits.",
			Attributes: map[string]schema.Attribute{
				"imports": schema.Int64Attribute{
					Optional:    true,
					Description: "Maximum imports. -1 for unlimited.",
				},
				"exports": schema.Int64Attribute{
					Optional:    true,
					Description: "Maximum exports. -1 for unlimited.",
				},
				"wildcard_exports": schema.BoolAttribute{
					Optional:    true,
					Description: "Allow wildcard exports. Default true.",
				},
				"disallow_bearer": schema.BoolAttribute{
					Optional:    true,
					Description: "Disallow bearer tokens. Default false.",
				},
				"conn": schema.Int64Attribute{
					Optional:    true,
					Description: "Maximum connections. -1 for unlimited.",
				},
				"leaf_node_conn": schema.Int64Attribute{
					Optional:    true,
					Description: "Maximum leaf node connections. -1 for unlimited.",
				},
			},
		},
		"jetstream_limits": schema.ListNestedAttribute{
			Optional:    true,
			Description: "JetStream limits. Entries without a tier apply globally; entries with a tier (e.g., R1, R3) apply to that replication tier.",
			NestedObject: schema.NestedAttributeObject{
				Attributes: map[string]schema.Attribute{
					"tier": schema.StringAttribute{
						Optional:    true,
						Description: "Replication tier (e.g., R1, R3). Empty for global limits.",
					},
					"mem_storage": schema.Int64Attribute{
						Optional:    true,
						Description: "Memory storage limit in bytes. 0 = disabled.",
					},
					"disk_storage": schema.Int64Attribute{
						Optional:    true,
						Description: "Disk storage limit in bytes. 0 = disabled.",
					},
					"streams": schema.Int64Attribute{
						Optional:    true,
						Description: "Maximum streams. -1 for unlimited.",
					},
					"consumer": schema.Int64Attribute{
						Optional:    true,
						Description: "Maximum consumers. -1 for unlimited.",
					},
					"max_ack_pending": schema.Int64Attribute{
						Optional:    true,
						Description: "Maximum pending acks. -1 for unlimited.",
					},
					"mem_max_stream_bytes": schema.Int64Attribute{
						Optional:    true,
						Description: "Maximum bytes per memory stream. 0 = unlimited.",
					},
					"disk_max_stream_bytes": schema.Int64Attribute{
						Optional:    true,
						Description: "Maximum bytes per disk stream. 0 = unlimited.",
					},
					"max_bytes_required": schema.BoolAttribute{
						Optional:    true,
						Description: "Require max_bytes to be set on streams. Default false.",
					},
				},
			},
		},
		"default_permissions": schema.SingleNestedAttribute{
			Optional:    true,
			Description: "Default permissions applied to users of this account.",
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
			},
		},
		"trace": schema.SingleNestedAttribute{
			Optional:    true,
			Description: "Message trace configuration.",
			Attributes: map[string]schema.Attribute{
				"destination": schema.StringAttribute{
					Optional:    true,
					Description: "Trace destination subject.",
				},
				"sampling": schema.Int64Attribute{
					Optional:    true,
					Description: "Sampling percentage (0-100).",
				},
			},
		},
		"public_key": schema.StringAttribute{
			Computed:    true,
			Description: "The account's public key.",
		},
		"jwt": schema.StringAttribute{
			Computed:    true,
			Description: "The signed account JWT.",
		},
	}
}

func (d *AccountDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data AccountDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	claims, pub, err := buildAccountClaims(ctx, data, resp)
	if err != nil || resp.Diagnostics.HasError() {
		return
	}

	operatorKP, err := keypairFromSeed(data.OperatorSeed.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Operator Seed", fmt.Sprintf("Failed to parse operator seed: %s", err))
		return
	}

	jwtString, err := encodeDeterministic(claims, operatorKP)
	if err != nil {
		resp.Diagnostics.AddError("JWT Encoding Error", fmt.Sprintf("Failed to encode account JWT: %s", err))
		return
	}

	data.PublicKey = types.StringValue(pub)
	data.JWT = types.StringValue(jwtString)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// buildAccountClaims constructs account claims from the data model. Shared by account and system_account.
func buildAccountClaims(ctx context.Context, data AccountDataSourceModel, resp *datasource.ReadResponse) (*natsjwt.AccountClaims, string, error) {
	accountKP, err := keypairFromSeed(data.Seed.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Account Seed", fmt.Sprintf("Failed to parse account seed: %s", err))
		return nil, "", err
	}

	pub, err := accountKP.PublicKey()
	if err != nil {
		resp.Diagnostics.AddError("Public Key Error", fmt.Sprintf("Failed to get public key: %s", err))
		return nil, "", err
	}

	claims := natsjwt.NewAccountClaims(pub)
	claims.Name = data.Name.ValueString()
	if !data.IssuedAt.IsNull() {
		claims.IssuedAt = data.IssuedAt.ValueInt64()
	} else {
		claims.IssuedAt = 0
	}
	if !data.Expires.IsNull() {
		claims.Expires = data.Expires.ValueInt64()
	}
	if !data.NotBefore.IsNull() {
		claims.NotBefore = data.NotBefore.ValueInt64()
	} else {
		claims.NotBefore = claims.IssuedAt
	}

	if !data.SigningKeys.IsNull() {
		var signingKeys []string
		resp.Diagnostics.Append(data.SigningKeys.ElementsAs(ctx, &signingKeys, false)...)
		if resp.Diagnostics.HasError() {
			return nil, "", fmt.Errorf("failed to read signing keys")
		}
		for _, sk := range signingKeys {
			claims.SigningKeys.Add(sk)
		}
	}

	if !data.Description.IsNull() {
		claims.Description = data.Description.ValueString()
	}

	if !data.InfoURL.IsNull() {
		claims.InfoURL = data.InfoURL.ValueString()
	}

	if !data.Tags.IsNull() {
		var tags []string
		resp.Diagnostics.Append(data.Tags.ElementsAs(ctx, &tags, false)...)
		if resp.Diagnostics.HasError() {
			return nil, "", fmt.Errorf("failed to read tags")
		}
		claims.Tags = tags
	}

	// NATS limits
	if !data.NatsLimits.IsNull() {
		var nl NatsLimitsModel
		resp.Diagnostics.Append(data.NatsLimits.As(ctx, &nl, objectAsOptions)...)
		if resp.Diagnostics.HasError() {
			return nil, "", fmt.Errorf("failed to read nats limits")
		}
		if !nl.Subs.IsNull() {
			claims.Limits.Subs = nl.Subs.ValueInt64()
		} else {
			claims.Limits.Subs = -1
		}
		if !nl.Data.IsNull() {
			claims.Limits.Data = nl.Data.ValueInt64()
		} else {
			claims.Limits.Data = -1
		}
		if !nl.Payload.IsNull() {
			claims.Limits.Payload = nl.Payload.ValueInt64()
		} else {
			claims.Limits.Payload = -1
		}
	}

	// Account limits
	if !data.AccountLimits.IsNull() {
		var al AccountLimitsModel
		resp.Diagnostics.Append(data.AccountLimits.As(ctx, &al, objectAsOptions)...)
		if resp.Diagnostics.HasError() {
			return nil, "", fmt.Errorf("failed to read account limits")
		}
		if !al.Imports.IsNull() {
			claims.Limits.Imports = al.Imports.ValueInt64()
		} else {
			claims.Limits.Imports = -1
		}
		if !al.Exports.IsNull() {
			claims.Limits.Exports = al.Exports.ValueInt64()
		} else {
			claims.Limits.Exports = -1
		}
		if !al.WildcardExports.IsNull() {
			claims.Limits.WildcardExports = al.WildcardExports.ValueBool()
		} else {
			claims.Limits.WildcardExports = true
		}
		if !al.DisallowBearer.IsNull() {
			claims.Limits.DisallowBearer = al.DisallowBearer.ValueBool()
		}
		if !al.Conn.IsNull() {
			claims.Limits.Conn = al.Conn.ValueInt64()
		} else {
			claims.Limits.Conn = -1
		}
		if !al.LeafNodeConn.IsNull() {
			claims.Limits.LeafNodeConn = al.LeafNodeConn.ValueInt64()
		} else {
			claims.Limits.LeafNodeConn = -1
		}
	}

	// JetStream limits
	if !data.JetStreamLimits.IsNull() {
		var jsLimits []JetStreamLimitsModel
		resp.Diagnostics.Append(data.JetStreamLimits.ElementsAs(ctx, &jsLimits, false)...)
		if resp.Diagnostics.HasError() {
			return nil, "", fmt.Errorf("failed to read jetstream limits")
		}

		for _, jsl := range jsLimits {
			limit := natsjwt.JetStreamLimits{}
			if !jsl.MemStorage.IsNull() {
				limit.MemoryStorage = jsl.MemStorage.ValueInt64()
			}
			if !jsl.DiskStorage.IsNull() {
				limit.DiskStorage = jsl.DiskStorage.ValueInt64()
			}
			if !jsl.Streams.IsNull() {
				limit.Streams = jsl.Streams.ValueInt64()
			} else {
				limit.Streams = -1
			}
			if !jsl.Consumer.IsNull() {
				limit.Consumer = jsl.Consumer.ValueInt64()
			} else {
				limit.Consumer = -1
			}
			if !jsl.MaxAckPending.IsNull() {
				limit.MaxAckPending = jsl.MaxAckPending.ValueInt64()
			} else {
				limit.MaxAckPending = -1
			}
			if !jsl.MemMaxStreamBytes.IsNull() {
				limit.MemoryMaxStreamBytes = jsl.MemMaxStreamBytes.ValueInt64()
			}
			if !jsl.DiskMaxStreamBytes.IsNull() {
				limit.DiskMaxStreamBytes = jsl.DiskMaxStreamBytes.ValueInt64()
			}
			if !jsl.MaxBytesRequired.IsNull() {
				limit.MaxBytesRequired = jsl.MaxBytesRequired.ValueBool()
			}

			tier := jsl.Tier.ValueString()
			if tier == "" || jsl.Tier.IsNull() {
				// Global limits
				claims.Limits.JetStreamLimits = limit
			} else {
				// Tiered limits
				if claims.Limits.JetStreamTieredLimits == nil {
					claims.Limits.JetStreamTieredLimits = make(map[string]natsjwt.JetStreamLimits)
				}
				claims.Limits.JetStreamTieredLimits[tier] = limit
			}
		}
	}

	// Default permissions
	if !data.DefaultPermissions.IsNull() {
		var dp DefaultPermissionsModel
		resp.Diagnostics.Append(data.DefaultPermissions.As(ctx, &dp, objectAsOptions)...)
		if resp.Diagnostics.HasError() {
			return nil, "", fmt.Errorf("failed to read default permissions")
		}
		var pubAllow, pubDeny, subAllow, subDeny []string
		if !dp.PubAllow.IsNull() {
			resp.Diagnostics.Append(dp.PubAllow.ElementsAs(ctx, &pubAllow, false)...)
		}
		if !dp.PubDeny.IsNull() {
			resp.Diagnostics.Append(dp.PubDeny.ElementsAs(ctx, &pubDeny, false)...)
		}
		if !dp.SubAllow.IsNull() {
			resp.Diagnostics.Append(dp.SubAllow.ElementsAs(ctx, &subAllow, false)...)
		}
		if !dp.SubDeny.IsNull() {
			resp.Diagnostics.Append(dp.SubDeny.ElementsAs(ctx, &subDeny, false)...)
		}
		if resp.Diagnostics.HasError() {
			return nil, "", fmt.Errorf("failed to read permissions lists")
		}
		claims.DefaultPermissions.Pub = buildPermission(pubAllow, pubDeny)
		claims.DefaultPermissions.Sub = buildPermission(subAllow, subDeny)
	}

	// Trace
	if !data.Trace.IsNull() {
		var t TraceModel
		resp.Diagnostics.Append(data.Trace.As(ctx, &t, objectAsOptions)...)
		if resp.Diagnostics.HasError() {
			return nil, "", fmt.Errorf("failed to read trace")
		}
		if !t.Destination.IsNull() {
			claims.Trace = &natsjwt.MsgTrace{
				Destination: natsjwt.Subject(t.Destination.ValueString()),
			}
			if !t.Sampling.IsNull() {
				claims.Trace.Sampling = int(t.Sampling.ValueInt64())
			}
		}
	}

	return claims, pub, nil
}
