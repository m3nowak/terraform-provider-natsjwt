package provider

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	natsjwt "github.com/nats-io/jwt/v2"
	"github.com/nats-io/nkeys"
)

var objectAsOptions = basetypes.ObjectAsOptions{}

// encodeDeterministic encodes claims with stable deterministic fields.
// The standard jwt library always sets IssuedAt to current time, so we encode,
// then decode, patch fields, re-serialize and re-sign.
func encodeDeterministic(claims natsjwt.Claims, kp nkeys.KeyPair) (string, error) {
	// First, do a normal encode to get a valid JWT structure
	cd := claims.Claims()
	issuedAt := cd.IssuedAt
	cd.ID = ""

	// We need to manually construct the JWT with deterministic fields.
	// Build header
	header := map[string]string{
		"typ": "JWT",
		"alg": "ed25519-nkey",
	}
	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", fmt.Errorf("failed to marshal header: %w", err)
	}
	headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)

	// Set issuer from keypair
	pub, err := kp.PublicKey()
	if err != nil {
		return "", fmt.Errorf("failed to get public key: %w", err)
	}
	cd.Issuer = pub

	// Ensure updateVersion is called by doing a trial encode first
	claims.Encode(kp)

	// Now reset deterministic fields
	cd.IssuedAt = issuedAt
	cd.ID = ""

	// Serialize payload
	payloadJSON, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("failed to marshal claims: %w", err)
	}
	payloadB64 := base64.RawURLEncoding.EncodeToString(payloadJSON)

	// Sign
	toSign := headerB64 + "." + payloadB64
	sig, err := kp.Sign([]byte(toSign))
	if err != nil {
		return "", fmt.Errorf("failed to sign: %w", err)
	}
	sigB64 := base64.RawURLEncoding.EncodeToString(sig)

	return toSign + "." + sigB64, nil
}

// prefixByteFromType converts a string type name to an nkeys.PrefixByte.
func prefixByteFromType(keyType string) (nkeys.PrefixByte, error) {
	switch keyType {
	case "operator":
		return nkeys.PrefixByteOperator, nil
	case "account":
		return nkeys.PrefixByteAccount, nil
	case "user":
		return nkeys.PrefixByteUser, nil
	default:
		return 0, fmt.Errorf("unknown key type: %s", keyType)
	}
}

// keypairFromSeed parses a seed string and returns the keypair.
func keypairFromSeed(seed string) (nkeys.KeyPair, error) {
	kp, err := nkeys.FromSeed([]byte(seed))
	if err != nil {
		return nil, fmt.Errorf("failed to parse seed: %w", err)
	}
	return kp, nil
}

// publicKeyFromSeed extracts the public key from a seed.
func publicKeyFromSeed(seed string) (string, error) {
	kp, err := keypairFromSeed(seed)
	if err != nil {
		return "", err
	}
	pub, err := kp.PublicKey()
	if err != nil {
		return "", fmt.Errorf("failed to get public key: %w", err)
	}
	return pub, nil
}

// stringListFromTF converts a slice of string values to []string, filtering nulls/unknowns.
func stringListFromTF(values []string) []string {
	if values == nil {
		return nil
	}
	result := make([]string, 0, len(values))
	result = append(result, values...)
	return result
}

// buildPermission creates a natsjwt.Permission from allow/deny lists.
func buildPermission(allow, deny []string) natsjwt.Permission {
	p := natsjwt.Permission{}
	if len(allow) > 0 {
		p.Allow = natsjwt.StringList(allow)
	}
	if len(deny) > 0 {
		p.Deny = natsjwt.StringList(deny)
	}
	return p
}
