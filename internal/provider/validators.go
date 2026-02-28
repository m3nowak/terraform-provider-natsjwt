package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/nats-io/nkeys"
)

// seedTypeValidator validates that a string is a valid NKey seed of the expected type.
type seedTypeValidator struct {
	expectedType nkeys.PrefixByte
}

func SeedTypeValidator(expectedType nkeys.PrefixByte) validator.String {
	return seedTypeValidator{expectedType: expectedType}
}

func (v seedTypeValidator) Description(_ context.Context) string {
	return fmt.Sprintf("must be a valid NKey seed of type %s", prefixName(v.expectedType))
}

func (v seedTypeValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v seedTypeValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	seed := req.ConfigValue.ValueString()
	prefix, _, err := nkeys.DecodeSeed([]byte(seed))
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid NKey Seed",
			fmt.Sprintf("Could not decode seed: %s", err),
		)
		return
	}

	if prefix != v.expectedType {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Wrong NKey Seed Type",
			fmt.Sprintf("Expected %s seed, got %s seed", prefixName(v.expectedType), prefixName(prefix)),
		)
	}
}

// publicKeyTypeValidator validates that a string is a valid NKey public key of the expected type.
type publicKeyTypeValidator struct {
	expectedType nkeys.PrefixByte
}

func PublicKeyTypeValidator(expectedType nkeys.PrefixByte) validator.String {
	return publicKeyTypeValidator{expectedType: expectedType}
}

func (v publicKeyTypeValidator) Description(_ context.Context) string {
	return fmt.Sprintf("must be a valid NKey public key of type %s", prefixName(v.expectedType))
}

func (v publicKeyTypeValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v publicKeyTypeValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	key := req.ConfigValue.ValueString()
	if !nkeys.IsValidPublicKey(key) {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid NKey Public Key",
			"The value is not a valid NKey public key",
		)
		return
	}

	var expectedChar byte
	switch v.expectedType {
	case nkeys.PrefixByteOperator:
		expectedChar = 'O'
	case nkeys.PrefixByteAccount:
		expectedChar = 'A'
	case nkeys.PrefixByteUser:
		expectedChar = 'U'
	case nkeys.PrefixByteServer:
		expectedChar = 'N'
	}
	if len(key) > 0 && key[0] != expectedChar {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Wrong NKey Public Key Type",
			fmt.Sprintf("Expected %s public key (starting with %c), got key starting with %c",
				prefixName(v.expectedType), expectedChar, key[0]),
		)
	}
}

// nkeyTypeValidator validates that a string is one of the valid NKey types.
type nkeyTypeValidator struct{}

func NkeyTypeValidator() validator.String {
	return nkeyTypeValidator{}
}

func (v nkeyTypeValidator) Description(_ context.Context) string {
	return "must be one of: operator, account, user"
}

func (v nkeyTypeValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v nkeyTypeValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	val := req.ConfigValue.ValueString()
	switch val {
	case "operator", "account", "user":
		return
	default:
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid NKey Type",
			fmt.Sprintf("Must be one of: operator, account, user. Got: %s", val),
		)
	}
}

// connectionTypeValidator validates allowed connection type strings.
type connectionTypeValidator struct{}

func ConnectionTypeValidator() validator.String {
	return connectionTypeValidator{}
}

func (v connectionTypeValidator) Description(_ context.Context) string {
	return "must be a valid NATS connection type: STANDARD, WEBSOCKET, LEAFNODE, MQTT"
}

func (v connectionTypeValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v connectionTypeValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	val := req.ConfigValue.ValueString()
	validTypes := map[string]bool{
		"STANDARD":  true,
		"WEBSOCKET": true,
		"LEAFNODE":  true,
		"MQTT":      true,
	}

	if !validTypes[val] {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Connection Type",
			fmt.Sprintf("Must be one of: STANDARD, WEBSOCKET, LEAFNODE, MQTT. Got: %s", val),
		)
	}
}

func prefixName(p nkeys.PrefixByte) string {
	switch p {
	case nkeys.PrefixByteOperator:
		return "operator"
	case nkeys.PrefixByteAccount:
		return "account"
	case nkeys.PrefixByteUser:
		return "user"
	case nkeys.PrefixByteServer:
		return "server"
	default:
		return "unknown"
	}
}
