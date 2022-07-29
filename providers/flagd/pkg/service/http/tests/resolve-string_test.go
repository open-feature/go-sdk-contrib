package tests

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"testing"

	models "github.com/open-feature/flagd/pkg/model"
	service "github.com/open-feature/golang-sdk-contrib/providers/flagd/pkg/service/http"
	mocks "github.com/open-feature/golang-sdk-contrib/providers/flagd/pkg/service/http/tests/mocks"
	"github.com/stretchr/testify/assert"
	schemaV1 "go.buf.build/grpc/go/open-feature/flagd/schema/v1"
)

type TestServiceResolveStringArgs struct {
	name                          string
	ServiceClientMockRequestSetup mocks.ServiceClientMockRequestSetup
	HTTPServiceConfiguration      service.HTTPServiceConfiguration
	httpResponseBody              interface{}
	httpResponseCode              int

	flagKey string
	evCtx   interface{}

	value   string
	variant string
	reason  string
	err     error
}

func TestServiceResolveString(t *testing.T) {
	tests := []TestServiceResolveStringArgs{
		{
			name: "happy path",
			ServiceClientMockRequestSetup: mocks.ServiceClientMockRequestSetup{
				InUrl:    "http://localhost:8080/flags/string/resolve/string",
				InMethod: http.MethodPost,
				OutRes:   &http.Response{},
				OutErr:   nil,
			},
			HTTPServiceConfiguration: service.HTTPServiceConfiguration{
				Port: 8080,
				Host: "localhost",
			},
			httpResponseBody: schemaV1.ResolveStringResponse{
				Value:   "value",
				Variant: "on",
				Reason:  models.StaticReason,
			},
			httpResponseCode: 200,
			flagKey:          "string",
			evCtx:            nil,

			value:   "value",
			variant: "on",
			reason:  models.StaticReason,
			err:     nil,
		},
		{
			name: "handle non 200",
			ServiceClientMockRequestSetup: mocks.ServiceClientMockRequestSetup{
				InUrl:    "http://localhost:8080/flags/string/resolve/string",
				InMethod: http.MethodPost,
				OutRes:   &http.Response{},
				OutErr:   nil,
			},
			HTTPServiceConfiguration: service.HTTPServiceConfiguration{
				Port: 8080,
				Host: "localhost",
			},
			httpResponseBody: schemaV1.ErrorResponse{
				Reason:    models.StaticReason,
				ErrorCode: "CUSTOM ERROR MESSAGE",
			},
			httpResponseCode: 400,
			flagKey:          "string",
			evCtx:            nil,
			reason:           models.ErrorReason,
			err:              errors.New("CUSTOM ERROR MESSAGE"),
		},
		{
			name: "handle error",
			ServiceClientMockRequestSetup: mocks.ServiceClientMockRequestSetup{
				InUrl:    "http://localhost:8080/flags/string/resolve/string",
				InMethod: http.MethodPost,
				OutRes:   &http.Response{},
				OutErr:   errors.New("its all gone wrong"),
			},
			HTTPServiceConfiguration: service.HTTPServiceConfiguration{
				Port: 8080,
				Host: "localhost",
			},
			flagKey: "string",
			evCtx:   nil,
			reason:  models.ErrorReason,
			err:     errors.New(models.GeneralErrorCode),
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
		srv := service.HTTPService{
			Client: &mocks.ServiceClient{
				RequestSetup: test.ServiceClientMockRequestSetup,
				Testing:      t,
			},
			HTTPServiceConfiguration: &test.HTTPServiceConfiguration,
		}
		res, err := srv.ResolveString(test.flagKey, test.evCtx)
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
