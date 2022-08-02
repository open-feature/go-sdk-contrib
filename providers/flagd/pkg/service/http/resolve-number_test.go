package http_service

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"testing"

	models "github.com/open-feature/flagd/pkg/model"
	mocks "github.com/open-feature/golang-sdk-contrib/providers/flagd/pkg/service/http/mocks"
	of "github.com/open-feature/golang-sdk/pkg/openfeature"
	"github.com/stretchr/testify/assert"
	schemaV1 "go.buf.build/grpc/go/open-feature/flagd/schema/v1"
)

type TestServiceResolveNumberArgs struct {
	name                          string
	ServiceClientMockRequestSetup mocks.ServiceClientMockRequestSetup
	httpServiceConfiguration      httpServiceConfiguration
	httpResponseBody              interface{}
	httpResponseCode              int

	flagKey string
	evCtx   of.EvaluationContext

	value   float32
	variant string
	reason  string
	err     error
}

func TestServiceResolveNumber(t *testing.T) {
	tests := []TestServiceResolveNumberArgs{
		{
			name: "happy path",
			ServiceClientMockRequestSetup: mocks.ServiceClientMockRequestSetup{
				InUrl:    "http://localhost:8080/flags/number/resolve/number",
				InMethod: http.MethodPost,
				OutRes:   &http.Response{},
				OutErr:   nil,
			},
			httpServiceConfiguration: httpServiceConfiguration{
				port:     8080,
				host:     "localhost",
				protocol: "http",
			},
			httpResponseBody: schemaV1.ResolveNumberResponse{
				Value:   12,
				Variant: "on",
				Reason:  models.StaticReason,
			},
			httpResponseCode: 200,

			flagKey: "number",
			evCtx: of.EvaluationContext{
				Attributes: map[string]interface{}{
					"con": "text",
				},
			},

			value:   12,
			variant: "on",
			reason:  models.StaticReason,
			err:     nil,
		},
		{
			name: "handle non 200",
			ServiceClientMockRequestSetup: mocks.ServiceClientMockRequestSetup{
				InUrl:    "http://localhost:8080/flags/number/resolve/number",
				InMethod: http.MethodPost,
				OutRes:   &http.Response{},
				OutErr:   nil,
			},
			httpServiceConfiguration: httpServiceConfiguration{
				port:     8080,
				host:     "localhost",
				protocol: "http",
			},
			httpResponseBody: schemaV1.ErrorResponse{
				Reason:    models.StaticReason,
				ErrorCode: "CUSTOM ERROR MESSAGE",
			},
			httpResponseCode: 400,
			flagKey:          "number",
			evCtx: of.EvaluationContext{
				Attributes: map[string]interface{}{
					"con": "text",
				},
			},
			reason: models.ErrorReason,
			err:    errors.New("CUSTOM ERROR MESSAGE"),
		},
		{
			name: "handle error",
			ServiceClientMockRequestSetup: mocks.ServiceClientMockRequestSetup{
				InUrl:    "http://localhost:8080/flags/number/resolve/number",
				InMethod: http.MethodPost,
				OutRes:   &http.Response{},
				OutErr:   errors.New("Its all gone wrong"),
			},
			httpServiceConfiguration: httpServiceConfiguration{
				port:     8080,
				host:     "localhost",
				protocol: "http",
			},
			flagKey: "number",
			evCtx: of.EvaluationContext{
				Attributes: map[string]interface{}{
					"con": "text",
				},
			},
			reason: models.ErrorReason,
			err:    errors.New(models.GeneralErrorCode),
		},
	}

	for _, test := range tests {
		evCtxM, err := json.Marshal(test.evCtx)
		if err != nil {
			t.Error(err)
		}
		bodyM, err := json.Marshal(test.httpResponseBody)
		if err != nil {
			t.Error(err)
		}
		test.ServiceClientMockRequestSetup.InBody = io.NopCloser(bytes.NewReader(evCtxM))
		test.ServiceClientMockRequestSetup.OutRes = &http.Response{
			StatusCode: test.httpResponseCode,
			Body:       io.NopCloser(bytes.NewReader(bodyM)),
		}
		srv := HTTPService{
			client: &mocks.ServiceClient{
				RequestSetup: test.ServiceClientMockRequestSetup,
				Testing:      t,
			},
			httpServiceConfiguration: &test.httpServiceConfiguration,
		}
		res, err := srv.ResolveNumber(test.flagKey, test.evCtx)
		if test.err != nil && !assert.EqualError(t, err, test.err.Error()) {
			t.Errorf("%s: unexpected error received, expected %v, got %v", test.name, test.err, err)
		}
		if res.Reason != test.reason {
			t.Errorf("%s: unexpected reason received, expected %v, got %v", test.name, test.reason, res.Reason)
		}
		if res.Value != test.value {
			t.Errorf("%s: unexpected value received, expected %v, got %v", test.name, test.value, res.Value)
		}
		if res.Variant != test.variant {
			t.Errorf("%s: unexpected variant received, expected %v, got %v", test.name, test.variant, res.Variant)
		}
	}
}
