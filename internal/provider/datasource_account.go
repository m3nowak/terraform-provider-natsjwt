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

var _ datasource.DataSource = &AccountDataSource{}

type AccountDataSource struct{}

// Shared model types used by both account and system_account data sources.

type NatsLimitsModel struct {
	Subs    types.Int64 `tfsdk:"subs"`
	Data    types.Int64 `tfsdk:"data"`
	Payload types.Int64 `tfsdk:"payload"`
}

type AccountLimitsModel struct {
	Imports         types.Int64 `tfsdk:"imports"`
	Exports         types.Int64 `tfsdk:"exports"`
	WildcardExports types.Bool  `tfsdk:"wildcard_exports"`
	DisallowBearer  types.Bool  `tfsdk:"disallow_bearer"`
	Conn            types.Int64 `tfsdk:"conn"`
	LeafNodeConn    types.Int64 `tfsdk:"leaf_node_conn"`
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

	generateAccount(ctx, &data, false, &resp.Diagnostics)
	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	}
}

func generateAccount(ctx context.Context, data *AccountDataSourceModel, system bool, diagnostics *diag.Diagnostics) {
	input := accountInputFromModel(ctx, *data, diagnostics)
	if system {
		input.Kind = issuance.SystemAccount
	}
	if diagnostics.HasError() {
		return
	}
	artifacts, err := issuance.IssueAccount(input)
	if err != nil {
		diagnostics.AddError("JWT Issuance Error", err.Error())
		return
	}
	data.PublicKey = types.StringValue(artifacts.PublicKey)
	data.JWT = types.StringValue(artifacts.JWT)
}

func accountInputFromModel(ctx context.Context, data AccountDataSourceModel, diagnostics *diag.Diagnostics) issuance.AccountInput {
	input := issuance.AccountInput{
		Kind:         issuance.StandardAccount,
		Name:         data.Name.ValueString(),
		Seed:         data.Seed.ValueString(),
		OperatorSeed: data.OperatorSeed.ValueString(),
		SigningKeys:  decodeStringList(ctx, data.SigningKeys, diagnostics),
		Temporal:     temporalInput(data.IssuedAt, data.Expires, data.NotBefore),
		Description:  optionalString(data.Description),
		InfoURL:      optionalString(data.InfoURL),
		Tags:         decodeStringList(ctx, data.Tags, diagnostics),
	}
	if !data.NatsLimits.IsNull() {
		var nl NatsLimitsModel
		diagnostics.Append(data.NatsLimits.As(ctx, &nl, objectAsOptions)...)
		input.NATSLimits = &issuance.NATSLimits{
			Subscriptions: optionalInt64(nl.Subs),
			Data:          optionalInt64(nl.Data),
			Payload:       optionalInt64(nl.Payload),
		}
	}
	if !data.AccountLimits.IsNull() {
		var al AccountLimitsModel
		diagnostics.Append(data.AccountLimits.As(ctx, &al, objectAsOptions)...)
		input.AccountLimits = &issuance.AccountLimits{
			Imports:         optionalInt64(al.Imports),
			Exports:         optionalInt64(al.Exports),
			WildcardExports: optionalBool(al.WildcardExports),
			DisallowBearer:  optionalBool(al.DisallowBearer),
			Connections:     optionalInt64(al.Conn),
			LeafConnections: optionalInt64(al.LeafNodeConn),
		}
	}
	if !data.JetStreamLimits.IsNull() {
		var jsLimits []JetStreamLimitsModel
		diagnostics.Append(data.JetStreamLimits.ElementsAs(ctx, &jsLimits, false)...)
		for _, jsl := range jsLimits {
			input.JetStreamLimits = append(input.JetStreamLimits, issuance.JetStreamLimits{
				Tier:                 optionalString(jsl.Tier),
				MemoryStorage:        optionalInt64(jsl.MemStorage),
				DiskStorage:          optionalInt64(jsl.DiskStorage),
				Streams:              optionalInt64(jsl.Streams),
				Consumers:            optionalInt64(jsl.Consumer),
				MaxAckPending:        optionalInt64(jsl.MaxAckPending),
				MemoryMaxStreamBytes: optionalInt64(jsl.MemMaxStreamBytes),
				DiskMaxStreamBytes:   optionalInt64(jsl.DiskMaxStreamBytes),
				MaxBytesRequired:     optionalBool(jsl.MaxBytesRequired),
			})
		}
	}
	if !data.DefaultPermissions.IsNull() {
		var dp DefaultPermissionsModel
		diagnostics.Append(data.DefaultPermissions.As(ctx, &dp, objectAsOptions)...)
		input.DefaultPermissions = &issuance.Permissions{
			Publish: issuance.Permission{
				Allow: decodeStringList(ctx, dp.PubAllow, diagnostics),
				Deny:  decodeStringList(ctx, dp.PubDeny, diagnostics),
			},
			Subscribe: issuance.Permission{
				Allow: decodeStringList(ctx, dp.SubAllow, diagnostics),
				Deny:  decodeStringList(ctx, dp.SubDeny, diagnostics),
			},
		}
	}
	if !data.Trace.IsNull() {
		var t TraceModel
		diagnostics.Append(data.Trace.As(ctx, &t, objectAsOptions)...)
		input.Trace = &issuance.Trace{
			Destination: optionalString(t.Destination),
			Sampling:    optionalInt64(t.Sampling),
		}
	}
	return input
}
