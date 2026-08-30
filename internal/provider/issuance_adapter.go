package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/m3nowak/terraform-provider-natsjwt/internal/issuance"
)

func optionalString(value types.String) *string {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	result := value.ValueString()
	return &result
}

func optionalInt64(value types.Int64) *int64 {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	result := value.ValueInt64()
	return &result
}

func optionalBool(value types.Bool) *bool {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	result := value.ValueBool()
	return &result
}

func temporalInput(issuedAt, expires, notBefore types.Int64) issuance.Temporal {
	return issuance.Temporal{
		IssuedAt:  optionalInt64(issuedAt),
		Expires:   optionalInt64(expires),
		NotBefore: optionalInt64(notBefore),
	}
}

func decodeStringList(ctx context.Context, value types.List, diagnostics *diag.Diagnostics) []string {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	var result []string
	diagnostics.Append(value.ElementsAs(ctx, &result, false)...)
	return result
}
