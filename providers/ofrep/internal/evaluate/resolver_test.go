package evaluate

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	of "github.com/open-feature/go-sdk/openfeature"
)

type resolverTest struct {
	name   string
	client mockOutbound
}

var success = evaluationSuccess{
	Value:   true,
	Key:     "flagA",
	Reason:  string(of.StaticReason),
	Variant: "true",
	Metadata: map[string]interface{}{
		"key": "value",
	},
}

func TestSuccess200(t *testing.T) {
	t.Run("success evaluation response", func(t *testing.T) {
		successBytes, err := json.Marshal(success)
		if err != nil {
			t.Fatal(err)
		}

		client := mockOutbound{
			rsp: http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader(successBytes)),
			},
		}

		successDto, resolutionError := NewOutboundResolver(client).resolveSingle(context.Background(), "", make(map[string]interface{}))

		if resolutionError != nil {
			t.Error(fmt.Sprintf("expected no errors, but got error: %v", err))
		}

		if successDto == nil {
			t.Fatal("expected non empty success response")
		}

		if successDto.Value != success.Value {
			t.Errorf(fmt.Sprintf("expected value %v, but got %v", success.Value, successDto.Value))
		}

		if successDto.Variant != success.Variant {
			t.Errorf(fmt.Sprintf("expected variant %v, but got %v", success.Variant, successDto.Variant))
		}

		if successDto.Reason != success.Reason {
			t.Errorf(fmt.Sprintf("expected reason %s, but got %s", success.Reason, successDto.Reason))
		}
	})

	t.Run("invalid payload type results in general error", func(t *testing.T) {
		client := mockOutbound{
			rsp: http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader([]byte("some payload"))),
			},
		}

		success, resolutionError := NewOutboundResolver(client).resolveSingle(context.Background(), "", make(map[string]interface{}))
		validateErrorCode(success, resolutionError, of.GeneralCode, t)
	})
}

func TestResolveGeneralErrors(t *testing.T) {
	tests := []resolverTest{
		{
			name: "http error results in a general error",
			client: mockOutbound{
				err: errors.New("some http error"),
				rsp: http.Response{},
			},
		},
		{
			name: "non ofrep http status codes results in general error",
			client: mockOutbound{
				rsp: http.Response{
					StatusCode: http.StatusServiceUnavailable,
				},
			},
		},
		{
			name: "http 401",
			client: mockOutbound{
				rsp: http.Response{
					StatusCode: http.StatusUnauthorized,
				},
			},
		},
		{
			name: "http 403",
			client: mockOutbound{
				rsp: http.Response{
					StatusCode: http.StatusForbidden,
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// when
			success, resolutionError := NewOutboundResolver(test.client).resolveSingle(context.Background(), "key", map[string]interface{}{})
			validateErrorCode(success, resolutionError, of.GeneralCode, t)
		})
	}
}

func TestEvaluationError4xx(t *testing.T) {
	tests := []struct {
		name       string
		errorCode  of.ErrorCode
		expectCode of.ErrorCode
	}{
		{
			name:       "validate parse error",
			errorCode:  of.ParseErrorCode,
			expectCode: of.ParseErrorCode,
		},
		{
			name:       "validate targeting key missing error",
			errorCode:  of.TargetingKeyMissingCode,
			expectCode: of.TargetingKeyMissingCode,
		},
		{
			name:       "validate invalid context error",
			errorCode:  of.InvalidContextCode,
			expectCode: of.InvalidContextCode,
		},
		{
			name:       "validate general error",
			errorCode:  of.GeneralCode,
			expectCode: of.GeneralCode,
		},
		{
			name:       "validate ofrep unhandled code",
			errorCode:  of.ProviderNotReadyCode,
			expectCode: of.GeneralCode,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// derive test specific error response
			errBytes, err := json.Marshal(evaluationError{
				ErrorCode: string(test.errorCode),
			})
			if err != nil {
				t.Fatal(err)
			}

			client := mockOutbound{
				rsp: http.Response{
					StatusCode: http.StatusBadRequest,
					Body:       io.NopCloser(bytes.NewReader(errBytes)),
				},
			}
			// when
			success, resolutionError := NewOutboundResolver(client).resolveSingle(context.Background(), "", make(map[string]interface{}))

			// then
			validateErrorCode(success, resolutionError, test.expectCode, t)
		})
	}
}

func TestFlagNotFound404(t *testing.T) {
	client := mockOutbound{
		rsp: http.Response{
			StatusCode: http.StatusNotFound,
		},
	}
	// when
	success, resolutionError := NewOutboundResolver(client).resolveSingle(context.Background(), "", make(map[string]interface{}))

	// then
	validateErrorCode(success, resolutionError, of.FlagNotFoundCode, t)
}

func Test429(t *testing.T) {
	tests := []struct {
		name       string
		retryAfter string
	}{
		{
			name:       "handle 429 with retry after header with seconds",
			retryAfter: "10",
		},
		{
			name:       "handle 429 with retry after header with date",
			retryAfter: "Wed, 21 Oct 2015 07:28:00 GMT",
		},
		{
			name: "handle 429 without retry header",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// derive test specific handler
			response := http.Response{
				StatusCode: http.StatusTooManyRequests,
			}

			if test.retryAfter != "" {
				response.Header = map[string][]string{
					"Retry-After": {test.retryAfter},
				}
			}

			client := mockOutbound{
				rsp: response,
			}

			// when
			success, resolutionError := NewOutboundResolver(client).resolveSingle(context.Background(), "", make(map[string]interface{}))
			validateErrorCode(success, resolutionError, of.GeneralCode, t)
		})
	}
}

func TestEvaluationError5xx(t *testing.T) {
	t.Run("without body", func(t *testing.T) {
		client := mockOutbound{
			rsp: http.Response{
				StatusCode: http.StatusInternalServerError,
				Body:       io.NopCloser(bytes.NewReader([]byte{})),
			},
		}
		// when
		success, resolutionError := NewOutboundResolver(client).resolveSingle(context.Background(), "", make(map[string]interface{}))

		// then
		validateErrorCode(success, resolutionError, of.GeneralCode, t)
	})

	t.Run("with valid body", func(t *testing.T) {
		errorBytes, err := json.Marshal(errorResponse{ErrorDetails: "some error detail"})
		if err != nil {
			t.Fatal(err)
		}

		client := mockOutbound{
			rsp: http.Response{
				StatusCode: http.StatusInternalServerError,
				Body:       io.NopCloser(bytes.NewReader(errorBytes)),
			},
		}
		// when
		success, resolutionError := NewOutboundResolver(client).resolveSingle(context.Background(), "", make(map[string]interface{}))

		// then
		validateErrorCode(success, resolutionError, of.GeneralCode, t)
	})

	t.Run("with invalid body", func(t *testing.T) {
		client := mockOutbound{
			rsp: http.Response{
				StatusCode: http.StatusInternalServerError,
				Body:       io.NopCloser(bytes.NewReader([]byte("some error"))),
			},
		}
		// when
		success, resolutionError := NewOutboundResolver(client).resolveSingle(context.Background(), "", make(map[string]interface{}))

		// then
		validateErrorCode(success, resolutionError, of.GeneralCode, t)
	})
}

func validateErrorCode(success *successDto, resolutionError *of.ResolutionError, errorCode of.ErrorCode, t *testing.T) {
	if success != nil {
		t.Fatal("expected no success result, but got non nil value")
	}

	if resolutionError == nil {
		t.Fatal("expected non nil error, but got empty")
	}

	if !strings.Contains(resolutionError.Error(), string(errorCode)) {
		t.Errorf("expected error to contain error code %s", errorCode)
	}
}

type mockOutbound struct {
	err error
	rsp http.Response
}

func (m mockOutbound) PostSingle(_ context.Context, _ string, _ []byte) (*http.Response, error) {
	return &m.rsp, m.err
}
