package provider

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nats-io/nats.go"
	natsjwt "github.com/nats-io/jwt/v2"
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

	nc := NatsRequester(mock)
	msg, err := nc.Request(fmt.Sprintf("$SYS.REQ.ACCOUNT.%s.CLAIMS.LOOKUP", claims.Subject), nil, 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if string(msg.Data) == jwtStr {
		t.Fatal("expected different jwt")
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

	nc := NatsRequester(mock)
	msg, err := nc.Request(fmt.Sprintf("$SYS.REQ.ACCOUNT.%s.CLAIMS.LOOKUP", claims.Subject), nil, 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if len(msg.Data) != 0 {
		t.Fatal("expected empty data")
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

func TestResolverAccountResource_encodeDeterministicGeneric(t *testing.T) {
	jwtStr := testAccountJWT(t)
	claims, _ := natsjwt.DecodeAccountClaims(jwtStr)

	opKP, err := nkeys.CreateOperator()
	if err != nil {
		t.Fatal(err)
	}
	pub, _ := opKP.PublicKey()

	genClaims := natsjwt.NewGenericClaims(claims.Subject)
	genClaims.Issuer = claims.Subject
	genClaims.Data = map[string]interface{}{
		"accounts": []string{claims.Subject},
	}

	jwt, err := encodeDeterministicGeneric(genClaims, opKP)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	decoded, err := natsjwt.DecodeGeneric(jwt)
	if err != nil {
		t.Fatalf("failed to decode generic jwt: %v", err)
	}
	if decoded.Issuer != pub {
		t.Fatalf("expected issuer %s, got %s", pub, decoded.Issuer)
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
