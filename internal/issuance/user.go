package issuance

import (
	"fmt"
	"time"

	natsjwt "github.com/nats-io/jwt/v2"
	"github.com/nats-io/nkeys"
)

type UserInput struct {
	Name                   string
	Seed                   string
	AccountSeed            string
	IssuerAccount          *string
	Temporal               Temporal
	Permissions            *UserPermissions
	Limits                 *UserLimits
	BearerToken            *bool
	AllowedConnectionTypes []string
	SourceNetworks         []string
	TimeRestrictions       []TimeRange
	Locale                 *string
	Tags                   []string
}

type UserPermissions struct {
	Permissions
	ResponseMaxMessages *int64
	ResponseTTL         *string
}

type UserLimits struct {
	Subscriptions *int64
	Data          *int64
	Payload       *int64
}

type TimeRange struct {
	Start string
	End   string
}

func IssueUser(input UserInput) (Artifacts, error) {
	_, publicKey, err := keyPair(input.Seed, nkeys.PrefixByteUser)
	if err != nil {
		return Artifacts{}, fmt.Errorf("invalid user seed: %w", err)
	}
	accountKP, _, err := keyPair(input.AccountSeed, nkeys.PrefixByteAccount)
	if err != nil {
		return Artifacts{}, fmt.Errorf("invalid account seed: %w", err)
	}

	claims := natsjwt.NewUserClaims(publicKey)
	claims.Name = input.Name
	applyTemporal(claims.Claims(), input.Temporal)
	if input.IssuerAccount != nil {
		claims.IssuerAccount = *input.IssuerAccount
	}
	if err := applyUserPermissions(claims, input.Permissions); err != nil {
		return Artifacts{}, err
	}
	if input.Limits != nil {
		claims.Subs = valueOr(input.Limits.Subscriptions, -1)
		claims.Limits.Data = valueOr(input.Limits.Data, -1)
		claims.Limits.NatsLimits.Payload = valueOr(input.Limits.Payload, -1)
	}
	claims.BearerToken = boolValueOr(input.BearerToken, false)
	claims.AllowedConnectionTypes = input.AllowedConnectionTypes
	claims.Src = input.SourceNetworks
	for _, timeRange := range input.TimeRestrictions {
		claims.Times = append(claims.Times, natsjwt.TimeRange{Start: timeRange.Start, End: timeRange.End})
	}
	if input.Locale != nil {
		claims.Locale = *input.Locale
	}
	claims.Tags = input.Tags

	token, err := EncodeDeterministic(claims, accountKP)
	if err != nil {
		return Artifacts{}, fmt.Errorf("encode user JWT: %w", err)
	}
	credentials, err := natsjwt.FormatUserConfig(token, []byte(input.Seed))
	if err != nil {
		return Artifacts{}, fmt.Errorf("format user credentials: %w", err)
	}
	return Artifacts{PublicKey: publicKey, JWT: token, Creds: string(credentials)}, nil
}

func applyUserPermissions(claims *natsjwt.UserClaims, input *UserPermissions) error {
	if input == nil {
		return nil
	}
	claims.Pub = buildPermission(input.Publish)
	claims.Sub = buildPermission(input.Subscribe)
	if input.ResponseMaxMessages == nil && input.ResponseTTL == nil {
		return nil
	}
	claims.Resp = &natsjwt.ResponsePermission{}
	if input.ResponseMaxMessages != nil {
		claims.Resp.MaxMsgs = int(*input.ResponseMaxMessages)
	}
	if input.ResponseTTL != nil {
		ttl, err := time.ParseDuration(*input.ResponseTTL)
		if err != nil {
			return fmt.Errorf("invalid response TTL: %w", err)
		}
		claims.Resp.Expires = ttl
	}
	return nil
}
