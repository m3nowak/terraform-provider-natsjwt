package issuance_test

import (
	"testing"
	"time"

	"github.com/m3nowak/terraform-provider-natsjwt/internal/issuance"
	natsjwt "github.com/nats-io/jwt/v2"
	"github.com/nats-io/nkeys"
)

func TestIssueUserMapsDomainInputAndFormatsCredentials(t *testing.T) {
	userSeed := seedFor(t, nkeys.PrefixByteUser)
	accountSeed := seedFor(t, nkeys.PrefixByteAccount)
	accountPublicKey := publicKeyFromSeed(t, accountSeed)
	issuerAccount := publicKeyFor(t, nkeys.PrefixByteAccount)
	input := issuance.UserInput{
		Name:          "user",
		Seed:          userSeed,
		AccountSeed:   accountSeed,
		IssuerAccount: &issuerAccount,
		Temporal:      issuance.Temporal{IssuedAt: int64Pointer(100)},
		Permissions: &issuance.UserPermissions{
			Permissions: issuance.Permissions{
				Publish:   issuance.Permission{Allow: []string{"pub.>"}, Deny: []string{"pub.private.>"}},
				Subscribe: issuance.Permission{Allow: []string{"sub.>"}, Deny: []string{"sub.private.>"}},
			},
			ResponseMaxMessages: int64Pointer(5),
			ResponseTTL:         stringPointer("30s"),
		},
		Limits:                 &issuance.UserLimits{Subscriptions: int64Pointer(10), Data: int64Pointer(20), Payload: int64Pointer(30)},
		BearerToken:            boolPointer(true),
		AllowedConnectionTypes: []string{"STANDARD", "WEBSOCKET"},
		SourceNetworks:         []string{"10.0.0.0/8"},
		TimeRestrictions:       []issuance.TimeRange{{Start: "08:00:00", End: "17:00:00"}},
		Locale:                 stringPointer("America/New_York"),
		Tags:                   []string{"team:platform"},
	}

	artifacts, err := issuance.IssueUser(input)
	if err != nil {
		t.Fatal(err)
	}
	claims, err := natsjwt.DecodeUserClaims(artifacts.JWT)
	if err != nil {
		t.Fatalf("decode user JWT: %v", err)
	}
	if claims.Subject != artifacts.PublicKey || claims.Issuer != accountPublicKey || claims.IssuerAccount != issuerAccount {
		t.Fatalf("unexpected subject/issuer/issuer account: %q/%q/%q", claims.Subject, claims.Issuer, claims.IssuerAccount)
	}
	if claims.Name != "user" || claims.IssuedAt != 100 || claims.NotBefore != 100 || claims.Expires != 0 {
		t.Fatalf("identity or temporal fields were not mapped: %+v", claims.ClaimsData)
	}
	if len(claims.Pub.Allow) != 1 || claims.Pub.Allow[0] != "pub.>" || len(claims.Sub.Deny) != 1 || claims.Sub.Deny[0] != "sub.private.>" {
		t.Fatalf("permissions were not mapped: %+v", claims.Permissions)
	}
	if claims.Resp == nil || claims.Resp.MaxMsgs != 5 || claims.Resp.Expires != 30*time.Second {
		t.Fatalf("response permission was not mapped: %+v", claims.Resp)
	}
	if claims.Subs != 10 || claims.Data != 20 || claims.Limits.NatsLimits.Payload != 30 {
		t.Fatalf("user limits were not mapped: %+v", claims.Limits)
	}
	if !claims.BearerToken || len(claims.AllowedConnectionTypes) != 2 || len(claims.Src) != 1 || len(claims.Times) != 1 || claims.Locale != "America/New_York" || len(claims.Tags) != 1 {
		t.Fatalf("user restrictions were not mapped: %+v", claims.User)
	}

	decoratedJWT, err := natsjwt.ParseDecoratedJWT([]byte(artifacts.Creds))
	if err != nil {
		t.Fatalf("parse credentials JWT: %v", err)
	}
	if decoratedJWT != artifacts.JWT {
		t.Fatal("credentials JWT does not match returned JWT")
	}
	credentialsKeyPair, err := natsjwt.ParseDecoratedUserNKey([]byte(artifacts.Creds))
	if err != nil {
		t.Fatalf("parse credentials seed: %v", err)
	}
	credentialsSeed, err := credentialsKeyPair.Seed()
	if err != nil {
		t.Fatal(err)
	}
	if string(credentialsSeed) != userSeed {
		t.Fatal("credentials do not contain the input user seed")
	}
}

func TestIssueUserRejectsInvalidResponseTTL(t *testing.T) {
	_, err := issuance.IssueUser(issuance.UserInput{
		Seed:        seedFor(t, nkeys.PrefixByteUser),
		AccountSeed: seedFor(t, nkeys.PrefixByteAccount),
		Permissions: &issuance.UserPermissions{ResponseTTL: stringPointer("not-a-duration")},
	})
	if err == nil {
		t.Fatal("expected invalid response TTL to fail")
	}
}
