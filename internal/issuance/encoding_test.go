package issuance_test

import (
	"encoding/json"
	"testing"

	"github.com/m3nowak/terraform-provider-natsjwt/internal/issuance"
	natsjwt "github.com/nats-io/jwt/v2"
	"github.com/nats-io/nkeys"
)

func TestEncodeDeterministicDoesNotMutateClaims(t *testing.T) {
	kp, err := nkeys.CreateOperator()
	if err != nil {
		t.Fatal(err)
	}
	publicKey, err := kp.PublicKey()
	if err != nil {
		t.Fatal(err)
	}
	claims := natsjwt.NewOperatorClaims(publicKey)
	claims.IssuedAt = 100
	claims.Expires = 200
	claims.NotBefore = 50

	before, err := json.Marshal(claims)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := issuance.EncodeDeterministic(claims, kp); err != nil {
		t.Fatal(err)
	}
	after, err := json.Marshal(claims)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Fatalf("encoding mutated caller-owned claims\nbefore: %s\n after: %s", before, after)
	}
}

func TestEncodeDeterministicProducesStableVerifiableJWT(t *testing.T) {
	kp, err := nkeys.CreateOperator()
	if err != nil {
		t.Fatal(err)
	}
	publicKey, err := kp.PublicKey()
	if err != nil {
		t.Fatal(err)
	}
	newClaims := func() *natsjwt.OperatorClaims {
		claims := natsjwt.NewOperatorClaims(publicKey)
		claims.Name = "deterministic"
		claims.IssuedAt = 100
		claims.Expires = 200
		claims.NotBefore = 50
		return claims
	}

	first, err := issuance.EncodeDeterministic(newClaims(), kp)
	if err != nil {
		t.Fatal(err)
	}
	second, err := issuance.EncodeDeterministic(newClaims(), kp)
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatal("equivalent claims produced different JWTs")
	}
	decoded, err := natsjwt.DecodeOperatorClaims(first)
	if err != nil {
		t.Fatalf("JWT failed signature verification: %v", err)
	}
	if decoded.Issuer != publicKey || decoded.Version != 2 {
		t.Fatalf("unexpected issuer/version: %q/%d", decoded.Issuer, decoded.Version)
	}
	if decoded.IssuedAt != 100 || decoded.Expires != 200 || decoded.NotBefore != 50 {
		t.Fatalf("temporal claims changed: iat=%d exp=%d nbf=%d", decoded.IssuedAt, decoded.Expires, decoded.NotBefore)
	}
}
