package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/m3nowak/terraform-provider-natsjwt/internal/issuance"
)

func TestGenerateOperatorDecodesAndProjectsArtifacts(t *testing.T) {
	seed := testOperatorSeed(t)
	data := OperatorDataSourceModel{
		Name:                types.StringValue("operator"),
		Seed:                types.StringValue(seed),
		OperatorServiceURLs: types.ListValueMust(types.StringType, []attr.Value{types.StringValue("nats://localhost:4222")}),
		Tags:                types.ListValueMust(types.StringType, []attr.Value{types.StringValue("env:test")}),
	}
	var diagnostics diag.Diagnostics
	generateOperator(context.Background(), &data, &diagnostics)
	if diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diagnostics)
	}

	expected, err := issuance.IssueOperator(issuance.OperatorInput{
		Name:                "operator",
		Seed:                seed,
		OperatorServiceURLs: []string{"nats://localhost:4222"},
		Tags:                []string{"env:test"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if data.PublicKey.ValueString() != expected.PublicKey || data.JWT.ValueString() != expected.JWT {
		t.Fatal("adapter did not project issuance artifacts")
	}
}

func TestGenerateUserProjectsIssuanceErrorsAsDiagnostics(t *testing.T) {
	permissionsType := map[string]attr.Type{
		"pub_allow":     types.ListType{ElemType: types.StringType},
		"pub_deny":      types.ListType{ElemType: types.StringType},
		"sub_allow":     types.ListType{ElemType: types.StringType},
		"sub_deny":      types.ListType{ElemType: types.StringType},
		"resp_max_msgs": types.Int64Type,
		"resp_ttl":      types.StringType,
	}
	data := UserDataSourceModel{
		Seed:        types.StringValue(testUserSeed(t)),
		AccountSeed: types.StringValue(testAccountSeed(t)),
		Permissions: types.ObjectValueMust(permissionsType, map[string]attr.Value{
			"pub_allow":     types.ListNull(types.StringType),
			"pub_deny":      types.ListNull(types.StringType),
			"sub_allow":     types.ListNull(types.StringType),
			"sub_deny":      types.ListNull(types.StringType),
			"resp_max_msgs": types.Int64Null(),
			"resp_ttl":      types.StringValue("invalid"),
		}),
	}
	var diagnostics diag.Diagnostics
	generateUser(context.Background(), &data, &diagnostics)
	if !diagnostics.HasError() {
		t.Fatal("expected issuance error diagnostic")
	}
	if !data.JWT.IsNull() || !data.Creds.IsNull() {
		t.Fatal("adapter projected artifacts after issuance failure")
	}
}

func TestGenerateAccountDecodesNestedValuesAndSystemKind(t *testing.T) {
	accountSeed := testAccountSeed(t)
	operatorSeed := testOperatorSeed(t)
	natsLimitsType := map[string]attr.Type{
		"subs":    types.Int64Type,
		"data":    types.Int64Type,
		"payload": types.Int64Type,
	}
	data := AccountDataSourceModel{
		Name:         types.StringValue("SYS"),
		Seed:         types.StringValue(accountSeed),
		OperatorSeed: types.StringValue(operatorSeed),
		NatsLimits: types.ObjectValueMust(natsLimitsType, map[string]attr.Value{
			"subs":    types.Int64Value(10),
			"data":    types.Int64Null(),
			"payload": types.Int64Value(20),
		}),
	}
	var diagnostics diag.Diagnostics
	generateAccount(context.Background(), &data, true, &diagnostics)
	if diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diagnostics)
	}

	expected, err := issuance.IssueAccount(issuance.AccountInput{
		Kind:         issuance.SystemAccount,
		Name:         "SYS",
		Seed:         accountSeed,
		OperatorSeed: operatorSeed,
		NATSLimits: &issuance.NATSLimits{
			Subscriptions: int64Pointer(10),
			Payload:       int64Pointer(20),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if data.PublicKey.ValueString() != expected.PublicKey || data.JWT.ValueString() != expected.JWT {
		t.Fatal("account adapter did not decode nested values or project system artifacts")
	}
}

func int64Pointer(value int64) *int64 {
	return &value
}
