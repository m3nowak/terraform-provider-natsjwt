package provider

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/m3nowak/terraform-provider-natsjwt/internal/issuance"
	natsjwt "github.com/nats-io/jwt/v2"
	"github.com/nats-io/nats.go"
)

const resolverAccountRequestTimeout = 5 * time.Second

type resolverAccountLookupOutcome int

const (
	resolverAccountLookupUnknown resolverAccountLookupOutcome = iota
	resolverAccountFound
	resolverAccountDrifted
	resolverAccountMissing
	resolverAccountNoResponder
)

type resolverAccountErrorKind int

const (
	resolverAccountUnknownError resolverAccountErrorKind = iota
	resolverAccountInvalidJWTError
	resolverAccountInvalidOperatorSeedError
	resolverAccountJWTEncodingError
	resolverAccountRequestError
	resolverAccountResponseParseError
	resolverAccountServerError
)

type resolverAccountError struct {
	kind        resolverAccountErrorKind
	subject     string
	code        int
	description string
	err         error
}

func (e *resolverAccountError) Error() string {
	if e.err != nil {
		return e.err.Error()
	}
	return e.description
}

func (e *resolverAccountError) Unwrap() error {
	return e.err
}

type resolverAccountProtocol struct {
	requester NatsRequester
}

func newResolverAccountProtocol(requester NatsRequester) resolverAccountProtocol {
	return resolverAccountProtocol{requester: requester}
}

// resolverMutationResponse matches the JSON returned by resolver update and delete requests.
type resolverMutationResponse struct {
	Server map[string]interface{} `json:"server,omitempty"`
	Data   *resolverMutationData  `json:"data,omitempty"`
	Error  *resolverMutationError `json:"error,omitempty"`
}

type resolverMutationData struct {
	Account string `json:"account,omitempty"`
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type resolverMutationError struct {
	Account     string `json:"account,omitempty"`
	Code        int    `json:"code"`
	Description string `json:"description"`
}

func (p resolverAccountProtocol) Update(jwtStr string) error {
	claims, err := decodeResolverAccountJWT(jwtStr)
	if err != nil {
		return err
	}

	subject := fmt.Sprintf("$SYS.REQ.ACCOUNT.%s.CLAIMS.UPDATE", claims.Subject)
	msg, err := p.requester.Request(subject, []byte(jwtStr), resolverAccountRequestTimeout)
	if err != nil {
		return &resolverAccountError{kind: resolverAccountRequestError, subject: claims.Subject, err: err}
	}
	return parseResolverMutationResponse(msg)
}

func (p resolverAccountProtocol) Lookup(jwtStr string) (resolverAccountLookupOutcome, error) {
	claims, err := decodeResolverAccountJWT(jwtStr)
	if err != nil {
		return resolverAccountLookupUnknown, err
	}

	subject := fmt.Sprintf("$SYS.REQ.ACCOUNT.%s.CLAIMS.LOOKUP", claims.Subject)
	msg, err := p.requester.Request(subject, nil, resolverAccountRequestTimeout)
	if errors.Is(err, nats.ErrNoResponders) {
		return resolverAccountNoResponder, nil
	}
	if err != nil {
		return resolverAccountLookupUnknown, &resolverAccountError{kind: resolverAccountRequestError, subject: claims.Subject, err: err}
	}
	if msg == nil || len(msg.Data) == 0 {
		return resolverAccountMissing, nil
	}
	if string(msg.Data) != jwtStr {
		return resolverAccountDrifted, nil
	}
	return resolverAccountFound, nil
}

func (p resolverAccountProtocol) Delete(jwtStr, operatorSeed string) error {
	claims, err := decodeResolverAccountJWT(jwtStr)
	if err != nil {
		return err
	}
	operatorKP, err := keypairFromSeed(operatorSeed)
	if err != nil {
		return &resolverAccountError{kind: resolverAccountInvalidOperatorSeedError, err: err}
	}
	operatorPublicKey, err := operatorKP.PublicKey()
	if err != nil {
		return &resolverAccountError{kind: resolverAccountInvalidOperatorSeedError, err: err}
	}

	deleteClaims := natsjwt.NewGenericClaims(operatorPublicKey)
	deleteClaims.Data = map[string]interface{}{"accounts": []string{claims.Subject}}
	deleteJWT, err := issuance.EncodeDeterministic(deleteClaims, operatorKP)
	if err != nil {
		return &resolverAccountError{kind: resolverAccountJWTEncodingError, err: err}
	}

	msg, err := p.requester.Request("$SYS.REQ.CLAIMS.DELETE", []byte(deleteJWT), resolverAccountRequestTimeout)
	if err != nil {
		return &resolverAccountError{kind: resolverAccountRequestError, subject: claims.Subject, err: err}
	}
	return parseResolverMutationResponse(msg)
}

func decodeResolverAccountJWT(jwtStr string) (*natsjwt.AccountClaims, error) {
	claims, err := natsjwt.DecodeAccountClaims(jwtStr)
	if err != nil {
		return nil, &resolverAccountError{kind: resolverAccountInvalidJWTError, err: err}
	}
	return claims, nil
}

func parseResolverMutationResponse(msg *nats.Msg) error {
	var response resolverMutationResponse
	if msg == nil {
		return &resolverAccountError{kind: resolverAccountResponseParseError, err: errors.New("empty response")}
	}
	if err := json.Unmarshal(msg.Data, &response); err != nil {
		return &resolverAccountError{kind: resolverAccountResponseParseError, err: err}
	}
	if response.Error != nil {
		return &resolverAccountError{
			kind:        resolverAccountServerError,
			code:        response.Error.Code,
			description: response.Error.Description,
		}
	}
	return nil
}
