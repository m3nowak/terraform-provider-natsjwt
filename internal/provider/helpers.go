package provider

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
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
