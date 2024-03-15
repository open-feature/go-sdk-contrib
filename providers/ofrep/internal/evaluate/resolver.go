package evaluate

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	of "github.com/open-feature/go-sdk/openfeature"
)

type Outbound interface {
	Post(context.Context, string, []byte) (*http.Response, error)
}

type resolver interface {
	resolve(ctx context.Context, key string, evalCtx map[string]interface{}) (*successDto, *of.ResolutionError)
}

type OutboundResolver struct {
	client Outbound
}

func NewOutboundResolver(client Outbound) *OutboundResolver {
	return &OutboundResolver{client}
}

func (g *OutboundResolver) resolve(ctx context.Context, key string, evalCtx map[string]interface{}) (
	*successDto, *of.ResolutionError) {

	b, err := json.Marshal(requestFrom(evalCtx))
	if err != nil {
		resErr := of.NewGeneralResolutionError(fmt.Sprintf("context marshelling error: %v", err))
		return nil, &resErr
	}

	rsp, err := g.client.Post(ctx, key, b)
	if err != nil {
		resErr := of.NewGeneralResolutionError(fmt.Sprintf("ofrep request error: %v", err))
		return nil, &resErr
	}

	// detect handler based on known ofrep status codes
	switch rsp.StatusCode {
	case 200:
		var success evaluationSuccess
		err := json.NewDecoder(rsp.Body).Decode(&success)
		if err != nil {
			resErr := of.NewGeneralResolutionError(fmt.Sprintf("error parsing the response: %v", err))
			return nil, &resErr
		}
		return toSuccessDto(success), nil
	case 400:
		return nil, parseError400(rsp.Body)
	case 401, 403:
		resErr := of.NewGeneralResolutionError("authentication/authorization error")
		return nil, &resErr
	case 404:
		resErr := of.NewFlagNotFoundResolutionError(fmt.Sprintf("flag for key '%s' does not exist", key))
		return nil, &resErr
	case 429:
		resErr := of.NewGeneralResolutionError("rate limit exceeded, try again later")
		return nil, &resErr
	case 500:
		return nil, parseError500(rsp.Body)
	default:
		resErr := of.NewGeneralResolutionError("invalid response")
		return nil, &resErr
	}
}

func parseError400(body io.ReadCloser) *of.ResolutionError {
	var evalError evaluationError
	err := json.NewDecoder(body).Decode(&evalError)
	if err != nil {
		resErr := of.NewGeneralResolutionError(fmt.Sprintf("error parsing error payload: %v", err))
		return &resErr
	}

	var resErr of.ResolutionError
	switch evalError.ErrorCode {
	case string(of.ParseErrorCode):
		resErr = of.NewParseErrorResolutionError(evalError.ErrorDetails)
	case string(of.TargetingKeyMissingCode):
		resErr = of.NewTargetingKeyMissingResolutionError(evalError.ErrorDetails)
	case string(of.InvalidContextCode):
		resErr = of.NewInvalidContextResolutionError(evalError.ErrorDetails)
	case string(of.GeneralCode):
		resErr = of.NewGeneralResolutionError(evalError.ErrorDetails)
	default:
		// we do not expect other error codes from ofrep, hence wrap as a general error
		resErr = of.NewGeneralResolutionError(evalError.ErrorDetails)
	}

	return &resErr
}

func parseError500(body io.ReadCloser) *of.ResolutionError {
	var evalError errorResponse
	var resErr of.ResolutionError

	err := json.NewDecoder(body).Decode(&evalError)
	if err != nil {
		resErr = of.NewGeneralResolutionError(fmt.Sprintf("error parsing error payload: %v", err))
	} else {
		resErr = of.NewGeneralResolutionError(evalError.ErrorDetails)
	}

	return &resErr
}

type successDto struct {
	Value    interface{}
	Reason   string
	Variant  string
	Metadata map[string]interface{}
}

func toSuccessDto(e evaluationSuccess) *successDto {
	m, _ := e.Metadata.(map[string]interface{})

	return &successDto{
		Value:    e.Value,
		Reason:   e.Reason,
		Variant:  e.Variant,
		Metadata: m,
	}
}

type request struct {
	Context interface{} `json:"context"`
}

func requestFrom(ctx map[string]interface{}) request {
	return request{
		Context: ctx,
	}
}

type evaluationSuccess struct {
	Value    interface{} `json:"value"`
	Key      string      `json:"key"`
	Reason   string      `json:"reason"`
	Variant  string      `json:"variant"`
	Metadata interface{} `json:"metadata"`
}

type evaluationError struct {
	Key          string `json:"key"`
	ErrorCode    string `json:"errorCode"`
	ErrorDetails string `json:"errorDetails"`
}

type errorResponse struct {
	ErrorDetails string `json:"errorDetails"`
}