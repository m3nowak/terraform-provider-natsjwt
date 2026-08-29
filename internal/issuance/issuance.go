package issuance

import (
	"fmt"

	natsjwt "github.com/nats-io/jwt/v2"
	"github.com/nats-io/nkeys"
)

type Temporal struct {
	IssuedAt  *int64
	Expires   *int64
	NotBefore *int64
}

type Artifacts struct {
	PublicKey string
	JWT       string
	Creds     string
}

type OperatorInput struct {
	Name                  string
	Seed                  string
	SigningKeys           []string
	AccountServerURL      *string
	OperatorServiceURLs   []string
	SystemAccount         *string
	StrictSigningKeyUsage *bool
	Temporal              Temporal
	Tags                  []string
}

func IssueOperator(input OperatorInput) (Artifacts, error) {
	kp, publicKey, err := keyPair(input.Seed, nkeys.PrefixByteOperator)
	if err != nil {
		return Artifacts{}, fmt.Errorf("invalid operator seed: %w", err)
	}

	claims := natsjwt.NewOperatorClaims(publicKey)
	claims.Name = input.Name
	for _, signingKey := range input.SigningKeys {
		claims.SigningKeys.Add(signingKey)
	}
	if input.AccountServerURL != nil {
		claims.AccountServerURL = *input.AccountServerURL
	}
	claims.OperatorServiceURLs = input.OperatorServiceURLs
	if input.SystemAccount != nil {
		claims.SystemAccount = *input.SystemAccount
	}
	if input.StrictSigningKeyUsage != nil {
		claims.StrictSigningKeyUsage = *input.StrictSigningKeyUsage
	}
	applyTemporal(claims.Claims(), input.Temporal)
	claims.Tags = input.Tags

	token, err := EncodeDeterministic(claims, kp)
	if err != nil {
		return Artifacts{}, fmt.Errorf("encode operator JWT: %w", err)
	}
	return Artifacts{PublicKey: publicKey, JWT: token}, nil
}

func keyPair(seed string, expectedPrefix nkeys.PrefixByte) (nkeys.KeyPair, string, error) {
	prefix, _, err := nkeys.DecodeSeed([]byte(seed))
	if err != nil {
		return nil, "", fmt.Errorf("decode seed: %w", err)
	}
	if prefix != expectedPrefix {
		return nil, "", fmt.Errorf("expected %s seed", prefixName(expectedPrefix))
	}
	kp, err := nkeys.FromSeed([]byte(seed))
	if err != nil {
		return nil, "", fmt.Errorf("parse seed: %w", err)
	}
	publicKey, err := kp.PublicKey()
	if err != nil {
		return nil, "", fmt.Errorf("get public key: %w", err)
	}
	return kp, publicKey, nil
}

func prefixName(prefix nkeys.PrefixByte) string {
	switch prefix {
	case nkeys.PrefixByteOperator:
		return "operator"
	case nkeys.PrefixByteAccount:
		return "account"
	case nkeys.PrefixByteUser:
		return "user"
	default:
		return "unknown"
	}
}

func applyTemporal(claims *natsjwt.ClaimsData, temporal Temporal) {
	if temporal.IssuedAt != nil {
		claims.IssuedAt = *temporal.IssuedAt
	} else {
		claims.IssuedAt = 0
	}
	if temporal.Expires != nil {
		claims.Expires = *temporal.Expires
	}
	if temporal.NotBefore != nil {
		claims.NotBefore = *temporal.NotBefore
	} else {
		claims.NotBefore = claims.IssuedAt
	}
}
