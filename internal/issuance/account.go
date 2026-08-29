package issuance

import (
	"fmt"

	natsjwt "github.com/nats-io/jwt/v2"
	"github.com/nats-io/nkeys"
)

type AccountKind uint8

const (
	StandardAccount AccountKind = iota
	SystemAccount
)

type AccountInput struct {
	Kind               AccountKind
	Name               string
	Seed               string
	OperatorSeed       string
	SigningKeys        []string
	Temporal           Temporal
	Description        *string
	InfoURL            *string
	Tags               []string
	NATSLimits         *NATSLimits
	AccountLimits      *AccountLimits
	JetStreamLimits    []JetStreamLimits
	DefaultPermissions *Permissions
	Trace              *Trace
}

type NATSLimits struct {
	Subscriptions *int64
	Data          *int64
	Payload       *int64
}

type AccountLimits struct {
	Imports         *int64
	Exports         *int64
	WildcardExports *bool
	DisallowBearer  *bool
	Connections     *int64
	LeafConnections *int64
}

type JetStreamLimits struct {
	Tier                 *string
	MemoryStorage        *int64
	DiskStorage          *int64
	Streams              *int64
	Consumers            *int64
	MaxAckPending        *int64
	MemoryMaxStreamBytes *int64
	DiskMaxStreamBytes   *int64
	MaxBytesRequired     *bool
}

type Permission struct {
	Allow []string
	Deny  []string
}

type Permissions struct {
	Publish   Permission
	Subscribe Permission
}

type Trace struct {
	Destination *string
	Sampling    *int64
}

func IssueAccount(input AccountInput) (Artifacts, error) {
	_, publicKey, err := keyPair(input.Seed, nkeys.PrefixByteAccount)
	if err != nil {
		return Artifacts{}, fmt.Errorf("invalid account seed: %w", err)
	}
	operatorKP, _, err := keyPair(input.OperatorSeed, nkeys.PrefixByteOperator)
	if err != nil {
		return Artifacts{}, fmt.Errorf("invalid operator seed: %w", err)
	}

	claims := natsjwt.NewAccountClaims(publicKey)
	claims.Name = input.Name
	applyTemporal(claims.Claims(), input.Temporal)
	for _, signingKey := range input.SigningKeys {
		claims.SigningKeys.Add(signingKey)
	}
	if input.Description != nil {
		claims.Description = *input.Description
	}
	if input.InfoURL != nil {
		claims.InfoURL = *input.InfoURL
	}
	claims.Tags = input.Tags
	applyNATSLimits(claims, input.NATSLimits)
	applyAccountLimits(claims, input.AccountLimits)
	applyJetStreamLimits(claims, input.JetStreamLimits)
	if input.DefaultPermissions != nil {
		claims.DefaultPermissions.Pub = buildPermission(input.DefaultPermissions.Publish)
		claims.DefaultPermissions.Sub = buildPermission(input.DefaultPermissions.Subscribe)
	}
	if input.Trace != nil && input.Trace.Destination != nil {
		claims.Trace = &natsjwt.MsgTrace{Destination: natsjwt.Subject(*input.Trace.Destination)}
		if input.Trace.Sampling != nil {
			claims.Trace.Sampling = int(*input.Trace.Sampling)
		}
	}
	if input.Kind == SystemAccount {
		applySystemAccountPolicy(claims)
	}

	token, err := EncodeDeterministic(claims, operatorKP)
	if err != nil {
		return Artifacts{}, fmt.Errorf("encode account JWT: %w", err)
	}
	return Artifacts{PublicKey: publicKey, JWT: token}, nil
}

func applyNATSLimits(claims *natsjwt.AccountClaims, limits *NATSLimits) {
	if limits == nil {
		return
	}
	claims.Limits.Subs = valueOr(limits.Subscriptions, -1)
	claims.Limits.Data = valueOr(limits.Data, -1)
	claims.Limits.Payload = valueOr(limits.Payload, -1)
}

func applyAccountLimits(claims *natsjwt.AccountClaims, limits *AccountLimits) {
	if limits == nil {
		return
	}
	claims.Limits.Imports = valueOr(limits.Imports, -1)
	claims.Limits.Exports = valueOr(limits.Exports, -1)
	claims.Limits.WildcardExports = boolValueOr(limits.WildcardExports, true)
	claims.Limits.DisallowBearer = boolValueOr(limits.DisallowBearer, false)
	claims.Limits.Conn = valueOr(limits.Connections, -1)
	claims.Limits.LeafNodeConn = valueOr(limits.LeafConnections, -1)
}

func applyJetStreamLimits(claims *natsjwt.AccountClaims, inputs []JetStreamLimits) {
	for _, input := range inputs {
		limit := natsjwt.JetStreamLimits{
			MemoryStorage:        valueOr(input.MemoryStorage, 0),
			DiskStorage:          valueOr(input.DiskStorage, 0),
			Streams:              valueOr(input.Streams, -1),
			Consumer:             valueOr(input.Consumers, -1),
			MaxAckPending:        valueOr(input.MaxAckPending, -1),
			MemoryMaxStreamBytes: valueOr(input.MemoryMaxStreamBytes, 0),
			DiskMaxStreamBytes:   valueOr(input.DiskMaxStreamBytes, 0),
			MaxBytesRequired:     boolValueOr(input.MaxBytesRequired, false),
		}
		if input.Tier == nil || *input.Tier == "" {
			claims.Limits.JetStreamLimits = limit
			continue
		}
		if claims.Limits.JetStreamTieredLimits == nil {
			claims.Limits.JetStreamTieredLimits = make(map[string]natsjwt.JetStreamLimits)
		}
		claims.Limits.JetStreamTieredLimits[*input.Tier] = limit
	}
}

func buildPermission(input Permission) natsjwt.Permission {
	permission := natsjwt.Permission{}
	if len(input.Allow) > 0 {
		permission.Allow = natsjwt.StringList(input.Allow)
	}
	if len(input.Deny) > 0 {
		permission.Deny = natsjwt.StringList(input.Deny)
	}
	return permission
}

func applySystemAccountPolicy(claims *natsjwt.AccountClaims) {
	claims.Exports = append(claims.Exports,
		&natsjwt.Export{
			Name:                 "account-monitoring-services",
			Subject:              "$SYS.REQ.ACCOUNT.*.*",
			Type:                 natsjwt.Service,
			ResponseType:         natsjwt.ResponseTypeStream,
			AccountTokenPosition: 4,
			Info: natsjwt.Info{
				Description: "Request account specific monitoring services for: SUBSZ, CONNZ, LEAFZ, JSZ and INFO",
				InfoURL:     "https://docs.nats.io/nats-server/configuration/sys_accounts",
			},
		},
		&natsjwt.Export{
			Name:                 "account-monitoring-streams",
			Subject:              "$SYS.ACCOUNT.*.>",
			Type:                 natsjwt.Stream,
			AccountTokenPosition: 3,
			Info: natsjwt.Info{
				Description: "Account specific monitoring stream",
				InfoURL:     "https://docs.nats.io/nats-server/configuration/sys_accounts",
			},
		},
	)
}

func valueOr(value *int64, fallback int64) int64 {
	if value == nil {
		return fallback
	}
	return *value
}

func boolValueOr(value *bool, fallback bool) bool {
	if value == nil {
		return fallback
	}
	return *value
}
