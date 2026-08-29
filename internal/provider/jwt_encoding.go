package provider

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"reflect"

	natsjwt "github.com/nats-io/jwt/v2"
	"github.com/nats-io/nkeys"
)

var encodedJWTHeader = base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"ed25519-nkey","typ":"JWT"}`))

// encodeDeterministic applies the jwt library's claim normalization to a copy,
// then restores caller-provided times before producing a stable signature.
func encodeDeterministic(claims natsjwt.Claims, kp nkeys.KeyPair) (string, error) {
	cloned, err := cloneClaims(claims)
	if err != nil {
		return "", err
	}

	claimData := cloned.Claims()
	issuedAt := claimData.IssuedAt
	expires := claimData.Expires
	notBefore := claimData.NotBefore
	if _, err := cloned.Encode(kp); err != nil {
		return "", fmt.Errorf("failed to normalize claims: %w", err)
	}
	claimData.IssuedAt = issuedAt
	claimData.Expires = expires
	claimData.NotBefore = notBefore
	claimData.ID = ""

	payload, err := json.Marshal(cloned)
	if err != nil {
		return "", fmt.Errorf("failed to marshal claims: %w", err)
	}
	toSign := encodedJWTHeader + "." + base64.RawURLEncoding.EncodeToString(payload)
	signature, err := kp.Sign([]byte(toSign))
	if err != nil {
		return "", fmt.Errorf("failed to sign claims: %w", err)
	}
	return toSign + "." + base64.RawURLEncoding.EncodeToString(signature), nil
}

func cloneClaims(claims natsjwt.Claims) (natsjwt.Claims, error) {
	value := reflect.ValueOf(claims)
	if !value.IsValid() || value.Kind() != reflect.Pointer || value.IsNil() {
		return nil, fmt.Errorf("claims must be a non-nil pointer")
	}

	cloned, ok := reflect.New(value.Elem().Type()).Interface().(natsjwt.Claims)
	if !ok {
		return nil, fmt.Errorf("unsupported claims type %T", claims)
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		return nil, fmt.Errorf("failed to clone claims: %w", err)
	}
	if err := json.Unmarshal(payload, cloned); err != nil {
		return nil, fmt.Errorf("failed to clone claims: %w", err)
	}
	return cloned, nil
}
