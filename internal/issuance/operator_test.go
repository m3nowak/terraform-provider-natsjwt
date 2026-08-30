package issuance_test

import (
	"testing"

	"github.com/m3nowak/terraform-provider-natsjwt/internal/issuance"
	natsjwt "github.com/nats-io/jwt/v2"
	"github.com/nats-io/nkeys"
)

func TestIssueOperatorMapsDomainInput(t *testing.T) {
	seed := seedFor(t, nkeys.PrefixByteOperator)
	signingKey := publicKeyFor(t, nkeys.PrefixByteOperator)
	systemAccount := publicKeyFor(t, nkeys.PrefixByteAccount)
	issuedAt := int64(100)
	expires := int64(200)
	strict := true
	input := issuance.OperatorInput{
		Name:                  "operator",
		Seed:                  seed,
		SigningKeys:           []string{signingKey},
		AccountServerURL:      stringPointer("nats://localhost:4222"),
		OperatorServiceURLs:   []string{"nats://localhost:4223"},
		SystemAccount:         &systemAccount,
		StrictSigningKeyUsage: &strict,
		Temporal:              issuance.Temporal{IssuedAt: &issuedAt, Expires: &expires},
		Tags:                  []string{"env:test"},
	}

	first, err := issuance.IssueOperator(input)
	if err != nil {
		t.Fatal(err)
	}
	second, err := issuance.IssueOperator(input)
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatal("equivalent operator inputs produced different artifacts")
	}

	claims, err := natsjwt.DecodeOperatorClaims(first.JWT)
	if err != nil {
		t.Fatalf("decode operator JWT: %v", err)
	}
	if claims.Subject != first.PublicKey || claims.Issuer != first.PublicKey {
		t.Fatalf("unexpected subject/issuer: %q/%q", claims.Subject, claims.Issuer)
	}
	if claims.Name != "operator" || claims.AccountServerURL != "nats://localhost:4222" {
		t.Fatalf("operator fields were not mapped: %+v", claims)
	}
	if len(claims.SigningKeys) != 1 || !claims.SigningKeys.Contains(signingKey) {
		t.Fatalf("signing keys were not mapped: %v", claims.SigningKeys)
	}
	if claims.SystemAccount != systemAccount || !claims.StrictSigningKeyUsage {
		t.Fatalf("operator policy fields were not mapped: %+v", claims.Operator)
	}
	if claims.IssuedAt != 100 || claims.Expires != 200 || claims.NotBefore != 100 {
		t.Fatalf("unexpected temporal claims: iat=%d exp=%d nbf=%d", claims.IssuedAt, claims.Expires, claims.NotBefore)
	}
	if len(claims.Tags) != 1 || claims.Tags[0] != "env:test" {
		t.Fatalf("tags were not mapped: %v", claims.Tags)
	}
	if first.Creds != "" {
		t.Fatal("operator issuance returned credentials")
	}
}

func TestIssuanceRejectsWrongSeedRoles(t *testing.T) {
	accountSeed := seedFor(t, nkeys.PrefixByteAccount)
	userSeed := seedFor(t, nkeys.PrefixByteUser)
	operatorSeed := seedFor(t, nkeys.PrefixByteOperator)
	tests := map[string]func() error{
		"operator subject": func() error {
			_, err := issuance.IssueOperator(issuance.OperatorInput{Seed: accountSeed})
			return err
		},
		"account subject": func() error {
			_, err := issuance.IssueAccount(issuance.AccountInput{Seed: userSeed, OperatorSeed: operatorSeed})
			return err
		},
		"account signer": func() error {
			_, err := issuance.IssueAccount(issuance.AccountInput{Seed: accountSeed, OperatorSeed: accountSeed})
			return err
		},
		"user subject": func() error {
			_, err := issuance.IssueUser(issuance.UserInput{Seed: accountSeed, AccountSeed: accountSeed})
			return err
		},
		"user signer": func() error {
			_, err := issuance.IssueUser(issuance.UserInput{Seed: userSeed, AccountSeed: operatorSeed})
			return err
		},
	}
	for name, issue := range tests {
		t.Run(name, func(t *testing.T) {
			if err := issue(); err == nil {
				t.Fatal("expected wrong seed role to fail")
			}
		})
	}
}

func seedFor(t *testing.T, prefix nkeys.PrefixByte) string {
	t.Helper()
	kp, err := nkeys.CreatePair(prefix)
	if err != nil {
		t.Fatal(err)
	}
	seed, err := kp.Seed()
	if err != nil {
		t.Fatal(err)
	}
	return string(seed)
}

func publicKeyFor(t *testing.T, prefix nkeys.PrefixByte) string {
	t.Helper()
	kp, err := nkeys.CreatePair(prefix)
	if err != nil {
		t.Fatal(err)
	}
	publicKey, err := kp.PublicKey()
	if err != nil {
		t.Fatal(err)
	}
	return publicKey
}

func stringPointer(value string) *string {
	return &value
}
