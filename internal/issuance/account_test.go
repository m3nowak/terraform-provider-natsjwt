package issuance_test

import (
	"testing"

	"github.com/m3nowak/terraform-provider-natsjwt/internal/issuance"
	natsjwt "github.com/nats-io/jwt/v2"
	"github.com/nats-io/nkeys"
)

func TestIssueAccountMapsDomainInputAndSystemPolicy(t *testing.T) {
	accountSeed := seedFor(t, nkeys.PrefixByteAccount)
	operatorSeed := seedFor(t, nkeys.PrefixByteOperator)
	operatorPublicKey := publicKeyFromSeed(t, operatorSeed)
	input := issuance.AccountInput{
		Kind:         issuance.SystemAccount,
		Name:         "SYS",
		Seed:         accountSeed,
		OperatorSeed: operatorSeed,
		SigningKeys:  []string{publicKeyFor(t, nkeys.PrefixByteAccount)},
		Temporal: issuance.Temporal{
			IssuedAt:  int64Pointer(10),
			Expires:   int64Pointer(30),
			NotBefore: int64Pointer(20),
		},
		Description: stringPointer("system account"),
		InfoURL:     stringPointer("https://example.com/system"),
		Tags:        []string{"system"},
		NATSLimits: &issuance.NATSLimits{
			Subscriptions: int64Pointer(100),
			Data:          int64Pointer(200),
			Payload:       int64Pointer(300),
		},
		AccountLimits: &issuance.AccountLimits{
			Imports:         int64Pointer(1),
			Exports:         int64Pointer(2),
			WildcardExports: boolPointer(false),
			DisallowBearer:  boolPointer(true),
			Connections:     int64Pointer(3),
			LeafConnections: int64Pointer(4),
		},
		JetStreamLimits: []issuance.JetStreamLimits{
			{Tier: stringPointer("R3"), DiskStorage: int64Pointer(2000), Consumers: int64Pointer(6), MaxAckPending: int64Pointer(7), MemoryMaxStreamBytes: int64Pointer(8), DiskMaxStreamBytes: int64Pointer(9), MaxBytesRequired: boolPointer(true)},
		},
		DefaultPermissions: &issuance.Permissions{
			Publish:   issuance.Permission{Allow: []string{"pub.>"}, Deny: []string{"pub.private.>"}},
			Subscribe: issuance.Permission{Allow: []string{"sub.>"}, Deny: []string{"sub.private.>"}},
		},
		Trace: &issuance.Trace{Destination: stringPointer("trace.>"), Sampling: int64Pointer(25)},
	}

	artifacts, err := issuance.IssueAccount(input)
	if err != nil {
		t.Fatal(err)
	}
	claims, err := natsjwt.DecodeAccountClaims(artifacts.JWT)
	if err != nil {
		t.Fatalf("decode account JWT: %v", err)
	}
	if claims.Subject != artifacts.PublicKey || claims.Issuer != operatorPublicKey {
		t.Fatalf("unexpected subject/issuer: %q/%q", claims.Subject, claims.Issuer)
	}
	if claims.Name != "SYS" || claims.Description != "system account" || claims.InfoURL != "https://example.com/system" {
		t.Fatalf("account fields were not mapped: %+v", claims)
	}
	if claims.IssuedAt != 10 || claims.Expires != 30 || claims.NotBefore != 20 {
		t.Fatalf("unexpected temporal claims: iat=%d exp=%d nbf=%d", claims.IssuedAt, claims.Expires, claims.NotBefore)
	}
	if claims.Limits.Subs != 100 || claims.Limits.Data != 200 || claims.Limits.Payload != 300 {
		t.Fatalf("NATS limits were not mapped: %+v", claims.Limits.NatsLimits)
	}
	if claims.Limits.Imports != 1 || claims.Limits.Exports != 2 || claims.Limits.WildcardExports || !claims.Limits.DisallowBearer || claims.Limits.Conn != 3 || claims.Limits.LeafNodeConn != 4 {
		t.Fatalf("account limits were not mapped: %+v", claims.Limits.AccountLimits)
	}
	r3 := claims.Limits.JetStreamTieredLimits["R3"]
	if r3.DiskStorage != 2000 || r3.Consumer != 6 || r3.MaxAckPending != 7 || r3.MemoryMaxStreamBytes != 8 || r3.DiskMaxStreamBytes != 9 || !r3.MaxBytesRequired {
		t.Fatalf("tiered JetStream limits were not mapped: %+v", r3)
	}
	if len(claims.DefaultPermissions.Pub.Allow) != 1 || claims.DefaultPermissions.Pub.Allow[0] != "pub.>" || len(claims.DefaultPermissions.Sub.Deny) != 1 || claims.DefaultPermissions.Sub.Deny[0] != "sub.private.>" {
		t.Fatalf("default permissions were not mapped: %+v", claims.DefaultPermissions)
	}
	if claims.Trace == nil || claims.Trace.Destination != "trace.>" || claims.Trace.Sampling != 25 {
		t.Fatalf("trace was not mapped: %+v", claims.Trace)
	}
	exports := make(map[string]bool, len(claims.Exports))
	for _, export := range claims.Exports {
		exports[string(export.Subject)] = true
	}
	if len(claims.Exports) != 2 || !exports["$SYS.REQ.ACCOUNT.*.*"] || !exports["$SYS.ACCOUNT.*.>"] {
		t.Fatalf("system exports were not applied: %v", exports)
	}
}

func TestIssueStandardAccountDoesNotApplySystemPolicy(t *testing.T) {
	artifacts, err := issuance.IssueAccount(issuance.AccountInput{
		Kind:         issuance.StandardAccount,
		Seed:         seedFor(t, nkeys.PrefixByteAccount),
		OperatorSeed: seedFor(t, nkeys.PrefixByteOperator),
	})
	if err != nil {
		t.Fatal(err)
	}
	claims, err := natsjwt.DecodeAccountClaims(artifacts.JWT)
	if err != nil {
		t.Fatal(err)
	}
	if len(claims.Exports) != 0 {
		t.Fatalf("standard account received system exports: %v", claims.Exports)
	}
}

func publicKeyFromSeed(t *testing.T, seed string) string {
	t.Helper()
	kp, err := nkeys.FromSeed([]byte(seed))
	if err != nil {
		t.Fatal(err)
	}
	publicKey, err := kp.PublicKey()
	if err != nil {
		t.Fatal(err)
	}
	return publicKey
}

func int64Pointer(value int64) *int64 {
	return &value
}

func boolPointer(value bool) *bool {
	return &value
}
