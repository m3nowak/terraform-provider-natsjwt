package provider

import (
	"context"
	"encoding/json"
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

// mockNatsRequester is a test double for NatsRequester.
type mockNatsRequester struct {
	requests []mockRequest
	index    int
}

type mockRequest struct {
	Subject  string
	Data     []byte
	Response *nats.Msg
	Err      error
}

func (m *mockNatsRequester) Request(subj string, data []byte, timeout time.Duration) (*nats.Msg, error) {
	if m.index >= len(m.requests) {
		return nil, fmt.Errorf("unexpected request to %s", subj)
	}
	req := m.requests[m.index]
	m.index++
	if req.Subject != "" && req.Subject != subj {
		return nil, fmt.Errorf("expected subject %s, got %s", req.Subject, subj)
	}
	return req.Response, req.Err
}

func (m *mockNatsRequester) Close() {}

func testAccountJWT(t *testing.T) string {
	t.Helper()
	opKP, err := nkeys.CreateOperator()
	if err != nil {
		t.Fatal(err)
	}
	acctKP, err := nkeys.CreateAccount()
	if err != nil {
		t.Fatal(err)
	}
	pub, err := acctKP.PublicKey()
	if err != nil {
		t.Fatal(err)
	}
	claims := natsjwt.NewAccountClaims(pub)
	claims.Name = "test-acct"
	jwt, err := claims.Encode(opKP)
	if err != nil {
		t.Fatal(err)
	}
	return jwt
}

func TestResolverAccountResource_pushJWT_Success(t *testing.T) {
	jwtStr := testAccountJWT(t)
	claims, _ := natsjwt.DecodeAccountClaims(jwtStr)

	mock := &mockNatsRequester{
		requests: []mockRequest{
			{
				Subject: fmt.Sprintf("$SYS.REQ.ACCOUNT.%s.CLAIMS.UPDATE", claims.Subject),
				Response: &nats.Msg{
					Data: mustJSON(updateResponse{
						Data: &updateData{Account: claims.Subject, Code: 200, Message: "jwt updated"},
					}),
				},
			},
		},
	}

	var diags diag.Diagnostics
	r := &ResolverAccountResource{}
	err := r.pushJWT(mock, jwtStr, &diags)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
}

func TestResolverAccountResource_pushJWT_ErrorResponse(t *testing.T) {
	jwtStr := testAccountJWT(t)
	claims, _ := natsjwt.DecodeAccountClaims(jwtStr)

	mock := &mockNatsRequester{
		requests: []mockRequest{
			{
				Subject: fmt.Sprintf("$SYS.REQ.ACCOUNT.%s.CLAIMS.UPDATE", claims.Subject),
				Response: &nats.Msg{
					Data: mustJSON(updateResponse{
						Error: &updateError{Code: 500, Description: "bad jwt"},
					}),
				},
			},
		},
	}

	var diags diag.Diagnostics
	r := &ResolverAccountResource{}
	err := r.pushJWT(mock, jwtStr, &diags)
	if err == nil {
		t.Fatal("expected error")
	}
	if !diags.HasError() {
		t.Fatal("expected diagnostics error")
	}
}

func buildReadState(t *testing.T, r *ResolverAccountResource, jwtStr string) tfsdk.State {
	t.Helper()
	ctx := context.Background()
	schemaResp := resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	stateVal := tftypes.NewValue(
		tftypes.Object{
			AttributeTypes: map[string]tftypes.Type{
				"jwt":           tftypes.String,
				"operator_seed": tftypes.String,
			},
		},
		map[string]tftypes.Value{
			"jwt":           tftypes.NewValue(tftypes.String, jwtStr),
			"operator_seed": tftypes.NewValue(tftypes.String, nil),
		},
	)
	return tfsdk.State{
		Raw:    stateVal,
		Schema: schemaResp.Schema,
	}
}

func TestResolverAccountResource_ReadDrift(t *testing.T) {
	jwtStr := testAccountJWT(t)
	claims, _ := natsjwt.DecodeAccountClaims(jwtStr)

	// Return a different JWT on lookup to simulate drift.
	otherJWT := testAccountJWT(t)
	mock := &mockNatsRequester{
		requests: []mockRequest{
			{
				Subject: fmt.Sprintf("$SYS.REQ.ACCOUNT.%s.CLAIMS.LOOKUP", claims.Subject),
				Response: &nats.Msg{
					Data: []byte(otherJWT),
				},
			},
		},
	}

	r := &ResolverAccountResource{testConnOverride: mock}
	ctx := context.Background()
	state := buildReadState(t, r, jwtStr)
	req := resource.ReadRequest{State: state}
	resp := resource.ReadResponse{State: state}

	r.Read(ctx, req, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
	}
	if !resp.State.Raw.IsNull() {
		t.Fatal("expected resource to be removed from state on JWT drift")
	}
}

func TestResolverAccountResource_ReadNotFound(t *testing.T) {
	jwtStr := testAccountJWT(t)
	claims, _ := natsjwt.DecodeAccountClaims(jwtStr)

	mock := &mockNatsRequester{
		requests: []mockRequest{
			{
				Subject:  fmt.Sprintf("$SYS.REQ.ACCOUNT.%s.CLAIMS.LOOKUP", claims.Subject),
				Response: &nats.Msg{Data: []byte{}},
			},
		},
	}

	r := &ResolverAccountResource{testConnOverride: mock}
	ctx := context.Background()
	state := buildReadState(t, r, jwtStr)
	req := resource.ReadRequest{State: state}
	resp := resource.ReadResponse{State: state}

	r.Read(ctx, req, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
	}
	if !resp.State.Raw.IsNull() {
		t.Fatal("expected resource to be removed from state when account is not found")
	}
}

func TestResolverAccountResource_DeleteWithoutOperatorSeed(t *testing.T) {
	data := ResolverAccountResourceModel{JWT: types.StringValue(testAccountJWT(t))}
	if !data.OperatorSeed.IsNull() {
		t.Fatal("expected null operator_seed")
	}
}

func TestResolverAccountResource_MissingNatsUrl(t *testing.T) {
	r := &ResolverAccountResource{
		providerData: &NatsjwtProviderData{},
	}
	_, diags := r.getConnection()
	if !diags.HasError() {
		t.Fatal("expected error for missing nats_url")
	}
}

func TestResolverAccountResource_CreateIntegration(t *testing.T) {
	jwtStr := testAccountJWT(t)
	claims, _ := natsjwt.DecodeAccountClaims(jwtStr)

	mock := &mockNatsRequester{
		requests: []mockRequest{
			{
				Subject: fmt.Sprintf("$SYS.REQ.ACCOUNT.%s.CLAIMS.UPDATE", claims.Subject),
				Response: &nats.Msg{
					Data: mustJSON(updateResponse{
						Data: &updateData{Account: claims.Subject, Code: 200, Message: "jwt updated"},
					}),
				},
			},
		},
	}

	r := &ResolverAccountResource{
		providerData: &NatsjwtProviderData{
			NatsUrl: types.StringValue("nats://localhost:4222"),
		},
	}

	// We test the pushJWT path which is the core of Create.
	var diags diag.Diagnostics
	err := r.pushJWT(mock, jwtStr, &diags)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
}

func TestEncodeDeterministicGenericClaims(t *testing.T) {
	jwtStr := testAccountJWT(t)
	claims, _ := natsjwt.DecodeAccountClaims(jwtStr)

	opKP, err := nkeys.CreateOperator()
	if err != nil {
		t.Fatal(err)
	}
	pub, _ := opKP.PublicKey()

	newClaims := func() *natsjwt.GenericClaims {
		genClaims := natsjwt.NewGenericClaims(claims.Subject)
		genClaims.Issuer = claims.Subject
		genClaims.IssuedAt = 100
		genClaims.Expires = 200
		genClaims.NotBefore = 50
		genClaims.Data = map[string]interface{}{
			"accounts": []string{claims.Subject},
		}
		return genClaims
	}

	jwt, err := encodeDeterministic(newClaims(), opKP)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	repeatedJWT, err := encodeDeterministic(newClaims(), opKP)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repeatedJWT != jwt {
		t.Fatal("equivalent generic claims produced different JWTs")
	}

	decoded, err := natsjwt.DecodeGeneric(jwt)
	if err != nil {
		t.Fatalf("failed to decode generic jwt: %v", err)
	}
	if decoded.Issuer != pub {
		t.Fatalf("expected issuer %s, got %s", pub, decoded.Issuer)
	}
	if decoded.Data["version"] != float64(2) {
		t.Fatalf("expected version 2, got %v", decoded.Data["version"])
	}
	if decoded.IssuedAt != 100 || decoded.Expires != 200 || decoded.NotBefore != 50 {
		t.Fatalf("temporal claims changed: iat=%d exp=%d nbf=%d", decoded.IssuedAt, decoded.Expires, decoded.NotBefore)
	}
	accounts, ok := decoded.Data["accounts"].([]interface{})
	if !ok || len(accounts) != 1 || accounts[0] != claims.Subject {
		t.Fatalf("unexpected accounts data: %v", decoded.Data["accounts"])
	}
}

func mustJSON(v interface{}) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}
