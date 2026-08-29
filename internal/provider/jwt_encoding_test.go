package provider

import (
	"encoding/json"
	"testing"

	natsjwt "github.com/nats-io/jwt/v2"
	"github.com/nats-io/nkeys"
)

func TestEncodeDeterministicDoesNotMutateClaims(t *testing.T) {
	kp, err := nkeys.CreateOperator()
	if err != nil {
		t.Fatal(err)
	}
	pub, err := kp.PublicKey()
	if err != nil {
		t.Fatal(err)
	}
	claims := natsjwt.NewOperatorClaims(pub)
	claims.IssuedAt = 100
	claims.Expires = 200
	claims.NotBefore = 50

	before, err := json.Marshal(claims)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := encodeDeterministic(claims, kp); err != nil {
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
	pub, err := kp.PublicKey()
	if err != nil {
		t.Fatal(err)
	}
	newClaims := func() *natsjwt.OperatorClaims {
		claims := natsjwt.NewOperatorClaims(pub)
		claims.Name = "deterministic"
		claims.IssuedAt = 100
		claims.Expires = 200
		claims.NotBefore = 50
		return claims
	}

	first, err := encodeDeterministic(newClaims(), kp)
	if err != nil {
		t.Fatal(err)
	}
	second, err := encodeDeterministic(newClaims(), kp)
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
	if decoded.Issuer != pub {
		t.Fatalf("expected issuer %q, got %q", pub, decoded.Issuer)
	}
	if decoded.Version != 2 {
		t.Fatalf("expected version 2, got %d", decoded.Version)
	}
	if decoded.IssuedAt != 100 || decoded.Expires != 200 || decoded.NotBefore != 50 {
		t.Fatalf("temporal claims changed: iat=%d exp=%d nbf=%d", decoded.IssuedAt, decoded.Expires, decoded.NotBefore)
	}
}
