package evaluate

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/open-feature/go-sdk-contrib/providers/ofrep/internal/outbound"
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

var successWithInvalidMetadata = evaluationSuccess{
	Value:    true,
	Key:      "flagA",
	Reason:   string(of.StaticReason),
	Variant:  "true",
	Metadata: "metadata",
}

var successWithoutMetadata = evaluationSuccess{
	Value:   true,
	Key:     "flagA",
	Reason:  string(of.StaticReason),
	Variant: "true",
}

func TestSuccess200(t *testing.T) {
	t.Run("success evaluation response", func(t *testing.T) {
		successBytes, err := json.Marshal(success)
		if err != nil {
			t.Fatal(err)
		}

		resolver := OutboundResolver{client: mockOutbound{
			rsp: outbound.Resolution{
				Status: http.StatusOK,
				Data:   successBytes,
			},
		}}

		successDto, resolutionError := resolver.resolveSingle(context.Background(), "", make(map[string]interface{}))

		if resolutionError != nil {
			t.Errorf("expected no errors, but got error: %v", err)
		}

		if successDto == nil {
			t.Fatal("expected non empty success response")
		}

		if successDto.Value != success.Value {
			t.Errorf("expected value %v, but got %v", success.Value, successDto.Value)
		}

		if successDto.Variant != success.Variant {
			t.Errorf("expected variant %v, but got %v", success.Variant, successDto.Variant)
		}

		if successDto.Reason != success.Reason {
			t.Errorf("expected reason %s, but got %s", success.Reason, successDto.Reason)
		}

		if successDto.Metadata["key"] != "value" {
			t.Errorf("expected key to contain value %s, but got %s", "value", successDto.Metadata["key"])
		}
	})

	t.Run("invalid payload type results in general error", func(t *testing.T) {
		resolver := OutboundResolver{client: mockOutbound{
			rsp: outbound.Resolution{
				Status: http.StatusOK,
				Data:   []byte("some payload"),
			},
		}}
		success, resolutionError := resolver.resolveSingle(context.Background(), "", make(map[string]interface{}))

		validateErrorCode(success, resolutionError, of.ParseErrorCode, t)
	})

	t.Run("invalid metadata results in a parsing error", func(t *testing.T) {
		b, err := json.Marshal(successWithInvalidMetadata)
		if err != nil {
			t.Fatal(err)
		}

		resolver := OutboundResolver{client: mockOutbound{
			rsp: outbound.Resolution{
				Status: http.StatusOK,
				Data:   b,
			},
		}}
		success, resolutionError := resolver.resolveSingle(context.Background(), "", make(map[string]interface{}))

		validateErrorCode(success, resolutionError, of.ParseErrorCode, t)
	})
	t.Run("no metadata in the ofrep response", func(t *testing.T) {
		b, err := json.Marshal(successWithoutMetadata)
		if err != nil {
			t.Fatal(err)
		}

		resolver := OutboundResolver{client: mockOutbound{
			rsp: outbound.Resolution{
				Status: http.StatusOK,
				Data:   b,
			},
		}}
		success, _ := resolver.resolveSingle(context.Background(), "", make(map[string]interface{}))

		if len(success.Metadata) > 0 {
			t.Errorf("should not have metadata, but got %v", success.Metadata)
		}
	})
}

func TestResolveGeneralErrors(t *testing.T) {
	tests := []resolverTest{
		{
			name: "http error results in a general error",
			client: mockOutbound{
				err: errors.New("some http error"),
				rsp: outbound.Resolution{},
			},
		},
		{
			name: "non ofrep http status codes results in general error",
			client: mockOutbound{
				rsp: outbound.Resolution{
					Status: http.StatusServiceUnavailable,
				},
			},
		},
		{
			name: "http 401",
			client: mockOutbound{
				rsp: outbound.Resolution{
					Status: http.StatusUnauthorized,
				},
			},
		},
		{
			name: "http 403",
			client: mockOutbound{
				rsp: outbound.Resolution{
					Status: http.StatusForbidden,
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resolver := OutboundResolver{client: test.client}
			success, resolutionError := resolver.resolveSingle(context.Background(), "key", map[string]interface{}{})

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

			resolver := OutboundResolver{client: mockOutbound{
				rsp: outbound.Resolution{
					Status: http.StatusBadRequest,
					Data:   errBytes,
				},
			}}
			success, resolutionError := resolver.resolveSingle(context.Background(), "", make(map[string]interface{}))

			validateErrorCode(success, resolutionError, test.expectCode, t)
		})
	}
}

func TestFlagNotFound404(t *testing.T) {
	resolver := OutboundResolver{client: mockOutbound{
		rsp: outbound.Resolution{
			Status: http.StatusNotFound,
		},
	}}
	success, resolutionError := resolver.resolveSingle(context.Background(), "", make(map[string]interface{}))

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
			response := outbound.Resolution{
				Status: http.StatusTooManyRequests,
			}

			if test.retryAfter != "" {
				response.Headers = map[string][]string{
					"Retry-After": {test.retryAfter},
				}
			}

			resolver := OutboundResolver{client: mockOutbound{rsp: response}}
			success, resolutionError := resolver.resolveSingle(context.Background(), "", make(map[string]interface{}))

			validateErrorCode(success, resolutionError, of.GeneralCode, t)
		})
	}
}

func TestEvaluationError5xx(t *testing.T) {
	t.Run("without body", func(t *testing.T) {
		resolver := OutboundResolver{client: mockOutbound{
			rsp: outbound.Resolution{
				Status: http.StatusInternalServerError,
				Data:   []byte{},
			},
		}}
		success, resolutionError := resolver.resolveSingle(context.Background(), "", make(map[string]interface{}))

		validateErrorCode(success, resolutionError, of.GeneralCode, t)
	})

	t.Run("with valid body", func(t *testing.T) {
		errorBytes, err := json.Marshal(errorResponse{ErrorDetails: "some error detail"})
		if err != nil {
			t.Fatal(err)
		}

		resolver := OutboundResolver{client: mockOutbound{
			rsp: outbound.Resolution{
				Status: http.StatusInternalServerError,
				Data:   errorBytes,
			},
		}}
		success, resolutionError := resolver.resolveSingle(context.Background(), "", make(map[string]interface{}))

		validateErrorCode(success, resolutionError, of.GeneralCode, t)
	})

	t.Run("with invalid body", func(t *testing.T) {
		resolver := OutboundResolver{client: mockOutbound{
			rsp: outbound.Resolution{
				Status: http.StatusInternalServerError,
				Data:   []byte("some error"),
			},
		}}
		success, resolutionError := resolver.resolveSingle(context.Background(), "", make(map[string]interface{}))

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
	rsp outbound.Resolution
}

func (m mockOutbound) Single(_ context.Context, _ string, _ []byte) (*outbound.Resolution, error) {
	return &m.rsp, m.err
}
