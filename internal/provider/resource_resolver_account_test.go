package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	natsjwt "github.com/nats-io/jwt/v2"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nkeys"
)

type mockNatsRequester struct {
	requests []mockRequest
	calls    []mockRequest
	index    int
	closed   bool
}

type mockRequest struct {
	Subject  string
	Data     []byte
	Response *nats.Msg
	Err      error
	Timeout  time.Duration
}

func (m *mockNatsRequester) Request(subj string, data []byte, timeout time.Duration) (*nats.Msg, error) {
	m.calls = append(m.calls, mockRequest{Subject: subj, Data: append([]byte(nil), data...), Timeout: timeout})
	if m.index >= len(m.requests) {
		return nil, fmt.Errorf("unexpected request to %s", subj)
	}
	request := m.requests[m.index]
	m.index++
	if request.Subject != "" && request.Subject != subj {
		return nil, fmt.Errorf("expected subject %s, got %s", request.Subject, subj)
	}
	return request.Response, request.Err
}

func (m *mockNatsRequester) Close() {
	m.closed = true
}

func testAccountJWT(t *testing.T) string {
	t.Helper()
	operatorKP, err := nkeys.CreateOperator()
	if err != nil {
		t.Fatal(err)
	}
	accountKP, err := nkeys.CreateAccount()
	if err != nil {
		t.Fatal(err)
	}
	publicKey, err := accountKP.PublicKey()
	if err != nil {
		t.Fatal(err)
	}
	claims := natsjwt.NewAccountClaims(publicKey)
	claims.Name = "test-acct"
	jwtStr, err := claims.Encode(operatorKP)
	if err != nil {
		t.Fatal(err)
	}
	return jwtStr
}

func TestResolverAccountResourceCreateAndUpdate(t *testing.T) {
	jwtStr := testAccountJWT(t)
	claims, err := natsjwt.DecodeAccountClaims(jwtStr)
	if err != nil {
		t.Fatal(err)
	}

	for _, operation := range []string{"create", "update"} {
		t.Run(operation, func(t *testing.T) {
			requester := successfulMutationRequester(claims.Subject)
			resolver := &ResolverAccountResource{testConnOverride: requester}
			plan := buildResolverAccountPlan(t, resolver, jwtStr, nil)

			switch operation {
			case "create":
				response := resource.CreateResponse{State: emptyResolverAccountState(t, resolver)}
				resolver.Create(context.Background(), resource.CreateRequest{Plan: plan}, &response)
				assertNoDiagnosticErrors(t, response.Diagnostics)
				assertResolverAccountState(t, response.State, jwtStr)
			case "update":
				response := resource.UpdateResponse{State: emptyResolverAccountState(t, resolver)}
				resolver.Update(context.Background(), resource.UpdateRequest{Plan: plan}, &response)
				assertNoDiagnosticErrors(t, response.Diagnostics)
				assertResolverAccountState(t, response.State, jwtStr)
			}

			assertRequest(t, requester, 0, fmt.Sprintf("$SYS.REQ.ACCOUNT.%s.CLAIMS.UPDATE", claims.Subject), []byte(jwtStr))
			if !requester.closed {
				t.Fatal("expected NATS requester to be closed")
			}
		})
	}
}

func TestResolverAccountResourceMutationFailurePreservesStateBehavior(t *testing.T) {
	for _, operation := range []string{"create", "update"} {
		t.Run(operation, func(t *testing.T) {
			requester := &mockNatsRequester{requests: []mockRequest{{
				Response: &nats.Msg{Data: mustJSON(resolverMutationResponse{Error: &resolverMutationError{Code: 500, Description: "bad jwt"}})},
			}}}
			resolver := &ResolverAccountResource{testConnOverride: requester}
			jwtStr := testAccountJWT(t)
			plan := buildResolverAccountPlan(t, resolver, jwtStr, nil)

			if operation == "create" {
				response := resource.CreateResponse{State: emptyResolverAccountState(t, resolver)}
				resolver.Create(context.Background(), resource.CreateRequest{Plan: plan}, &response)
				assertDiagnostic(t, response.Diagnostics, diag.SeverityError, "Update Failed")
				if response.State.Raw.Type() != nil {
					t.Fatal("expected failed create to leave state unset")
				}
				return
			}

			oldJWT := testAccountJWT(t)
			state := buildResolverAccountState(t, resolver, oldJWT, nil)
			response := resource.UpdateResponse{State: state}
			resolver.Update(context.Background(), resource.UpdateRequest{Plan: plan}, &response)
			assertDiagnostic(t, response.Diagnostics, diag.SeverityError, "Update Failed")
			assertResolverAccountState(t, response.State, oldJWT)
		})
	}
}

func TestResolverAccountResourceReadOutcomes(t *testing.T) {
	jwtStr := testAccountJWT(t)
	tests := map[string]struct {
		response   *nats.Msg
		err        error
		removed    bool
		diagnostic string
	}{
		"matching":     {response: &nats.Msg{Data: []byte(jwtStr)}},
		"drifted":      {response: &nats.Msg{Data: []byte(testAccountJWT(t))}, removed: true},
		"missing":      {response: &nats.Msg{}, removed: true},
		"no responder": {err: nats.ErrNoResponders, removed: true},
		"failed":       {err: errors.New("lookup failed"), diagnostic: "Failed to read account claims"},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			requester := &mockNatsRequester{requests: []mockRequest{{Response: test.response, Err: test.err}}}
			resolver := &ResolverAccountResource{testConnOverride: requester}
			state := buildResolverAccountState(t, resolver, jwtStr, nil)
			response := resource.ReadResponse{State: state}
			resolver.Read(context.Background(), resource.ReadRequest{State: state}, &response)

			if test.diagnostic != "" {
				assertDiagnostic(t, response.Diagnostics, diag.SeverityError, test.diagnostic)
			} else {
				assertNoDiagnosticErrors(t, response.Diagnostics)
			}
			if response.State.Raw.IsNull() != test.removed {
				t.Fatalf("expected removed=%t, state null=%t", test.removed, response.State.Raw.IsNull())
			}
			if name == "matching" {
				assertResolverAccountState(t, response.State, jwtStr)
			}
			if !requester.closed {
				t.Fatal("expected NATS requester to be closed")
			}
		})
	}
}

func TestResolverAccountResourceDelete(t *testing.T) {
	jwtStr := testAccountJWT(t)
	operatorKP, err := nkeys.CreateOperator()
	if err != nil {
		t.Fatal(err)
	}
	operatorSeed, err := operatorKP.Seed()
	if err != nil {
		t.Fatal(err)
	}
	operatorSeedString := string(operatorSeed)

	t.Run("deletes with operator seed", func(t *testing.T) {
		requester := successfulMutationRequester("")
		resolver := &ResolverAccountResource{testConnOverride: requester}
		state := buildResolverAccountState(t, resolver, jwtStr, &operatorSeedString)
		response := resource.DeleteResponse{}
		resolver.Delete(context.Background(), resource.DeleteRequest{State: state}, &response)

		assertNoDiagnosticErrors(t, response.Diagnostics)
		assertRequestSubjectAndTimeout(t, requester, 0, "$SYS.REQ.CLAIMS.DELETE")
		if !requester.closed {
			t.Fatal("expected NATS requester to be closed")
		}
	})

	t.Run("warns without operator seed", func(t *testing.T) {
		requester := &mockNatsRequester{}
		resolver := &ResolverAccountResource{testConnOverride: requester}
		state := buildResolverAccountState(t, resolver, jwtStr, nil)
		response := resource.DeleteResponse{}
		resolver.Delete(context.Background(), resource.DeleteRequest{State: state}, &response)

		assertDiagnostic(t, response.Diagnostics, diag.SeverityWarning, "Account Not Deleted from Server")
		if len(requester.calls) != 0 {
			t.Fatalf("expected no NATS requests, got %d", len(requester.calls))
		}
	})

	t.Run("reports server failure", func(t *testing.T) {
		requester := &mockNatsRequester{requests: []mockRequest{{
			Response: &nats.Msg{Data: mustJSON(resolverMutationResponse{Error: &resolverMutationError{Code: 500, Description: "delete disabled"}})},
		}}}
		resolver := &ResolverAccountResource{testConnOverride: requester}
		state := buildResolverAccountState(t, resolver, jwtStr, &operatorSeedString)
		response := resource.DeleteResponse{}
		resolver.Delete(context.Background(), resource.DeleteRequest{State: state}, &response)

		assertDiagnostic(t, response.Diagnostics, diag.SeverityError, "Delete Failed")
	})
}

func TestResolverAccountResourceMissingNatsURL(t *testing.T) {
	resolver := &ResolverAccountResource{providerData: &NatsjwtProviderData{}}
	_, diagnostics := resolver.getConnection()
	if !diagnostics.HasError() {
		t.Fatal("expected error for missing nats_url")
	}
}

func successfulMutationRequester(account string) *mockNatsRequester {
	return &mockNatsRequester{requests: []mockRequest{{
		Response: &nats.Msg{Data: mustJSON(resolverMutationResponse{
			Data: &resolverMutationData{Account: account, Code: 200, Message: "ok"},
		})},
	}}}
}

func resolverAccountSchema(t *testing.T, resolver *ResolverAccountResource) resource.SchemaResponse {
	t.Helper()
	response := resource.SchemaResponse{}
	resolver.Schema(context.Background(), resource.SchemaRequest{}, &response)
	return response
}

func resolverAccountValue(jwtStr string, operatorSeed *string) tftypes.Value {
	var operatorSeedValue interface{}
	if operatorSeed != nil {
		operatorSeedValue = *operatorSeed
	}
	return tftypes.NewValue(
		tftypes.Object{AttributeTypes: map[string]tftypes.Type{
			"jwt": tftypes.String, "operator_seed": tftypes.String,
		}},
		map[string]tftypes.Value{
			"jwt":           tftypes.NewValue(tftypes.String, jwtStr),
			"operator_seed": tftypes.NewValue(tftypes.String, operatorSeedValue),
		},
	)
}

func buildResolverAccountPlan(t *testing.T, resolver *ResolverAccountResource, jwtStr string, operatorSeed *string) tfsdk.Plan {
	t.Helper()
	return tfsdk.Plan{Raw: resolverAccountValue(jwtStr, operatorSeed), Schema: resolverAccountSchema(t, resolver).Schema}
}

func buildResolverAccountState(t *testing.T, resolver *ResolverAccountResource, jwtStr string, operatorSeed *string) tfsdk.State {
	t.Helper()
	return tfsdk.State{Raw: resolverAccountValue(jwtStr, operatorSeed), Schema: resolverAccountSchema(t, resolver).Schema}
}

func emptyResolverAccountState(t *testing.T, resolver *ResolverAccountResource) tfsdk.State {
	t.Helper()
	return tfsdk.State{Schema: resolverAccountSchema(t, resolver).Schema}
}

func assertResolverAccountState(t *testing.T, state tfsdk.State, jwtStr string) {
	t.Helper()
	var model ResolverAccountResourceModel
	diagnostics := state.Get(context.Background(), &model)
	assertNoDiagnosticErrors(t, diagnostics)
	if model.JWT != types.StringValue(jwtStr) {
		t.Fatalf("expected JWT to be retained in state")
	}
}

func assertNoDiagnosticErrors(t *testing.T, diagnostics diag.Diagnostics) {
	t.Helper()
	if diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diagnostics)
	}
}

func assertDiagnostic(t *testing.T, diagnostics diag.Diagnostics, severity diag.Severity, summary string) {
	t.Helper()
	for _, diagnostic := range diagnostics {
		if diagnostic.Severity() == severity && diagnostic.Summary() == summary {
			return
		}
	}
	t.Fatalf("expected %s diagnostic %q, got %v", severity, summary, diagnostics)
}

func mustJSON(value interface{}) []byte {
	data, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return data
}
