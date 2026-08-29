package provider

import (
	"errors"
	"fmt"
	"testing"

	natsjwt "github.com/nats-io/jwt/v2"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nkeys"
)

func TestResolverAccountProtocolUpdate(t *testing.T) {
	jwtStr := testAccountJWT(t)
	claims, err := natsjwt.DecodeAccountClaims(jwtStr)
	if err != nil {
		t.Fatal(err)
	}

	requester := &mockNatsRequester{requests: []mockRequest{{
		Response: &nats.Msg{Data: mustJSON(resolverMutationResponse{
			Data: &resolverMutationData{Account: claims.Subject, Code: 200, Message: "jwt updated"},
		})},
	}}}
	protocol := newResolverAccountProtocol(requester)

	if err := protocol.Update(jwtStr); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertRequest(t, requester, 0, fmt.Sprintf("$SYS.REQ.ACCOUNT.%s.CLAIMS.UPDATE", claims.Subject), []byte(jwtStr))
}

func TestResolverAccountProtocolUpdateResponses(t *testing.T) {
	tests := map[string]struct {
		response *nats.Msg
		err      error
		kind     resolverAccountErrorKind
	}{
		"request failure": {
			err:  errors.New("request failed"),
			kind: resolverAccountRequestError,
		},
		"malformed response": {
			response: &nats.Msg{Data: []byte("not json")},
			kind:     resolverAccountResponseParseError,
		},
		"server error": {
			response: &nats.Msg{Data: mustJSON(resolverMutationResponse{Error: &resolverMutationError{Code: 500, Description: "bad jwt"}})},
			kind:     resolverAccountServerError,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			requester := &mockNatsRequester{requests: []mockRequest{{Response: test.response, Err: test.err}}}
			err := newResolverAccountProtocol(requester).Update(testAccountJWT(t))
			assertResolverAccountErrorKind(t, err, test.kind)
		})
	}
}

func TestResolverAccountProtocolLookup(t *testing.T) {
	jwtStr := testAccountJWT(t)
	claims, err := natsjwt.DecodeAccountClaims(jwtStr)
	if err != nil {
		t.Fatal(err)
	}

	tests := map[string]struct {
		response *nats.Msg
		err      error
		want     resolverAccountLookupOutcome
		wantErr  resolverAccountErrorKind
	}{
		"found": {
			response: &nats.Msg{Data: []byte(jwtStr)},
			want:     resolverAccountFound,
		},
		"drifted": {
			response: &nats.Msg{Data: []byte(testAccountJWT(t))},
			want:     resolverAccountDrifted,
		},
		"missing": {
			response: &nats.Msg{},
			want:     resolverAccountMissing,
		},
		"no responder": {
			err:  nats.ErrNoResponders,
			want: resolverAccountNoResponder,
		},
		"failed": {
			err:     errors.New("lookup failed"),
			wantErr: resolverAccountRequestError,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			requester := &mockNatsRequester{requests: []mockRequest{{Response: test.response, Err: test.err}}}
			outcome, err := newResolverAccountProtocol(requester).Lookup(jwtStr)
			if test.wantErr != 0 {
				assertResolverAccountErrorKind(t, err, test.wantErr)
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if outcome != test.want {
				t.Fatalf("expected outcome %v, got %v", test.want, outcome)
			}
			assertRequest(t, requester, 0, fmt.Sprintf("$SYS.REQ.ACCOUNT.%s.CLAIMS.LOOKUP", claims.Subject), nil)
		})
	}
}

func TestResolverAccountProtocolDelete(t *testing.T) {
	jwtStr := testAccountJWT(t)
	accountClaims, err := natsjwt.DecodeAccountClaims(jwtStr)
	if err != nil {
		t.Fatal(err)
	}
	operatorKP, err := nkeys.CreateOperator()
	if err != nil {
		t.Fatal(err)
	}
	operatorSeed, err := operatorKP.Seed()
	if err != nil {
		t.Fatal(err)
	}
	operatorPublicKey, err := operatorKP.PublicKey()
	if err != nil {
		t.Fatal(err)
	}

	requester := &mockNatsRequester{requests: []mockRequest{{
		Response: &nats.Msg{Data: mustJSON(resolverMutationResponse{
			Data: &resolverMutationData{Account: accountClaims.Subject, Code: 200, Message: "deleted"},
		})},
	}}}
	protocol := newResolverAccountProtocol(requester)
	if err := protocol.Delete(jwtStr, string(operatorSeed)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertRequestSubjectAndTimeout(t, requester, 0, "$SYS.REQ.CLAIMS.DELETE")
	deleteClaims, err := natsjwt.DecodeGeneric(string(requester.calls[0].Data))
	if err != nil {
		t.Fatalf("failed to decode deletion JWT: %v", err)
	}
	if deleteClaims.Subject != operatorPublicKey || deleteClaims.Issuer != operatorPublicKey {
		t.Fatalf("deletion JWT must be self-signed by the operator, subject=%q issuer=%q", deleteClaims.Subject, deleteClaims.Issuer)
	}
	if deleteClaims.Data["version"] != float64(2) {
		t.Fatalf("expected deletion JWT version 2, got %v", deleteClaims.Data["version"])
	}
	accounts, ok := deleteClaims.Data["accounts"].([]interface{})
	if !ok || len(accounts) != 1 || accounts[0] != accountClaims.Subject {
		t.Fatalf("unexpected accounts data: %v", deleteClaims.Data["accounts"])
	}

	firstPayload := append([]byte(nil), requester.calls[0].Data...)
	requester.requests = append(requester.requests, mockRequest{Response: requester.requests[0].Response})
	if err := protocol.Delete(jwtStr, string(operatorSeed)); err != nil {
		t.Fatalf("unexpected repeated deletion error: %v", err)
	}
	if string(firstPayload) != string(requester.calls[1].Data) {
		t.Fatal("equivalent deletion requests produced different JWTs")
	}
}

func TestResolverAccountProtocolDeleteResponses(t *testing.T) {
	tests := map[string]struct {
		response *nats.Msg
		err      error
		kind     resolverAccountErrorKind
	}{
		"request failure": {
			err:  errors.New("request failed"),
			kind: resolverAccountRequestError,
		},
		"malformed response": {
			response: &nats.Msg{Data: []byte("not json")},
			kind:     resolverAccountResponseParseError,
		},
		"server error": {
			response: &nats.Msg{Data: mustJSON(resolverMutationResponse{Error: &resolverMutationError{Code: 500, Description: "delete disabled"}})},
			kind:     resolverAccountServerError,
		},
	}

	operatorKP, err := nkeys.CreateOperator()
	if err != nil {
		t.Fatal(err)
	}
	operatorSeed, err := operatorKP.Seed()
	if err != nil {
		t.Fatal(err)
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			requester := &mockNatsRequester{requests: []mockRequest{{Response: test.response, Err: test.err}}}
			err := newResolverAccountProtocol(requester).Delete(testAccountJWT(t), string(operatorSeed))
			assertResolverAccountErrorKind(t, err, test.kind)
			assertRequestSubjectAndTimeout(t, requester, 0, "$SYS.REQ.CLAIMS.DELETE")
		})
	}
}

func assertRequest(t *testing.T, requester *mockNatsRequester, index int, subject string, data []byte) {
	t.Helper()
	assertRequestSubjectAndTimeout(t, requester, index, subject)
	call := requester.calls[index]
	if string(call.Data) != string(data) {
		t.Fatalf("expected payload %q, got %q", data, call.Data)
	}
}

func assertRequestSubjectAndTimeout(t *testing.T, requester *mockNatsRequester, index int, subject string) {
	t.Helper()
	if len(requester.calls) <= index {
		t.Fatalf("expected request %d, got %d requests", index, len(requester.calls))
	}
	call := requester.calls[index]
	if call.Subject != subject {
		t.Fatalf("expected subject %q, got %q", subject, call.Subject)
	}
	if call.Timeout != resolverAccountRequestTimeout {
		t.Fatalf("expected timeout %s, got %s", resolverAccountRequestTimeout, call.Timeout)
	}
}

func assertResolverAccountErrorKind(t *testing.T, err error, kind resolverAccountErrorKind) {
	t.Helper()
	var protocolErr *resolverAccountError
	if !errors.As(err, &protocolErr) {
		t.Fatalf("expected resolver account error, got %v", err)
	}
	if protocolErr.kind != kind {
		t.Fatalf("expected error kind %v, got %v", kind, protocolErr.kind)
	}
}
