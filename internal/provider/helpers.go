package provider

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	natsjwt "github.com/nats-io/jwt/v2"
	"github.com/nats-io/nkeys"
)

var objectAsOptions = basetypes.ObjectAsOptions{}

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

// applyTemporalClaimsDefaults maps Terraform temporal attributes to JWT claims.
// Defaults are: IssuedAt=0 (Unix epoch), Expires unset (no expiration),
// and NotBefore=IssuedAt when not provided explicitly.
func applyTemporalClaimsDefaults(cd *natsjwt.ClaimsData, issuedAt, expires, notBefore types.Int64) {
	if !issuedAt.IsNull() {
		cd.IssuedAt = issuedAt.ValueInt64()
	} else {
		cd.IssuedAt = 0
	}
	if !expires.IsNull() {
		cd.Expires = expires.ValueInt64()
	}
	if !notBefore.IsNull() {
		cd.NotBefore = notBefore.ValueInt64()
	} else {
		cd.NotBefore = cd.IssuedAt
	}
}
