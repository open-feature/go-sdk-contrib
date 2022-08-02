package http_service_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"
	models "github.com/open-feature/flagd/pkg/model"
	service "github.com/open-feature/golang-sdk-contrib/providers/flagd/pkg/service/http"
	mocks "github.com/open-feature/golang-sdk-contrib/providers/flagd/pkg/service/http/mocks"
	of "github.com/open-feature/golang-sdk/pkg/openfeature"
	"github.com/stretchr/testify/assert"
	schemaV1 "go.buf.build/grpc/go/open-feature/flagd/schema/v1"
	"google.golang.org/protobuf/types/known/structpb"
)

type TestServiceResolveBooleanArgs struct {
	name string

	mockMethod string
	mockUrl    string

	mockHttpResponseCode int
	mockHttpResponseBody interface{}
	mockErr              error

	httpServiceConfiguration service.HttpServiceConfiguration

	flagKey string
	evCtx   of.EvaluationContext

	outErr     error
	outReason  string
	outVariant string
	outValue   bool
}

func TestServiceResolveBoolean(t *testing.T) {
	tests := []TestServiceResolveBooleanArgs{
		{
			name:                 "happy path",
			mockMethod:           "POST",
			mockUrl:              "http://localhost:8080/flags/flag/resolve/boolean",
			mockHttpResponseCode: http.StatusOK,
			mockHttpResponseBody: schemaV1.ResolveBooleanResponse{
				Value:   true,
				Variant: "on",
				Reason:  models.StaticReason,
			},
			mockErr: nil,
			httpServiceConfiguration: service.HttpServiceConfiguration{
				Port:     8080,
				Host:     "localhost",
				Protocol: "http",
			},
			flagKey:    "flag",
			outValue:   true,
			outVariant: "on",
			outReason:  models.StaticReason,
			outErr:     nil,
		},
		{
			name:                 "non 200",
			mockMethod:           "POST",
			mockUrl:              "http://localhost:8080/flags/flag/resolve/boolean",
			mockHttpResponseCode: http.StatusBadRequest,
			mockHttpResponseBody: schemaV1.ErrorResponse{
				Reason:    models.StaticReason,
				ErrorCode: "CUSTOM ERROR MESSAGE",
			},
			mockErr: nil,
			httpServiceConfiguration: service.HttpServiceConfiguration{
				Port:     8080,
				Host:     "localhost",
				Protocol: "http",
			},
			flagKey:   "flag",
			outReason: models.ErrorReason,
			outErr:    errors.New("CUSTOM ERROR MESSAGE"),
		},
		{
			name:                 "non 200",
			mockMethod:           "POST",
			mockUrl:              "http://localhost:8080/flags/flag/resolve/boolean",
			mockHttpResponseCode: http.StatusInternalServerError,
			mockErr:              errors.New("its all gone wrong"),
			httpServiceConfiguration: service.HttpServiceConfiguration{
				Port:     8080,
				Host:     "localhost",
				Protocol: "http",
			},
			flagKey:   "flag",
			outReason: models.ErrorReason,
			outErr:    errors.New(models.GeneralErrorCode),
		},
	}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	for _, test := range tests {
		mock := mocks.NewMockiHTTPClient(ctrl)
		bodyM, err := json.Marshal(test.mockHttpResponseBody)
		if err != nil {
			t.Error(err)
		}
		mock.EXPECT().Request(test.mockMethod, test.mockUrl, gomock.Any()).AnyTimes().Return(
			&http.Response{
				StatusCode: test.mockHttpResponseCode,
				Body:       io.NopCloser(bytes.NewReader(bodyM)),
			},
			test.mockErr,
		)
		srv := service.HTTPService{
			Client:                   mock,
			HttpServiceConfiguration: &test.httpServiceConfiguration,
		}

		res, err := srv.ResolveBoolean(test.flagKey, test.evCtx)
		if test.outErr != nil && !assert.EqualError(t, err, test.outErr.Error()) {
			t.Errorf("%s: unexpected error received, expected %v, got %v", test.name, test.outErr, err)
		}
		if res.Reason != test.outReason {
			t.Errorf("%s: unexpected reason received, expected %v, got %v", test.name, test.outReason, res.Reason)
		}
		if res.Value != test.outValue {
			t.Errorf("%s: unexpected value received, expected %v, got %v", test.name, test.outValue, res.Value)
		}
		if res.Variant != test.outVariant {
			t.Errorf("%s: unexpected variant received, expected %v, got %v", test.name, test.outVariant, res.Variant)
		}
	}
}

type TestServiceResolveStringArgs struct {
	name string

	mockMethod string
	mockUrl    string

	mockHttpResponseCode int
	mockHttpResponseBody interface{}
	mockErr              error

	httpServiceConfiguration service.HttpServiceConfiguration

	flagKey string
	evCtx   of.EvaluationContext

	outErr     error
	outReason  string
	outVariant string
	outValue   string
}

func TestServiceResolveString(t *testing.T) {
	tests := []TestServiceResolveStringArgs{
		{
			name:                 "happy path",
			mockMethod:           "POST",
			mockUrl:              "http://localhost:8080/flags/flag/resolve/string",
			mockHttpResponseCode: http.StatusOK,
			mockHttpResponseBody: schemaV1.ResolveStringResponse{
				Value:   "value",
				Variant: "on",
				Reason:  models.StaticReason,
			},
			mockErr: nil,
			httpServiceConfiguration: service.HttpServiceConfiguration{
				Port:     8080,
				Host:     "localhost",
				Protocol: "http",
			},
			flagKey:    "flag",
			outValue:   "value",
			outVariant: "on",
			outReason:  models.StaticReason,
			outErr:     nil,
		},
		{
			name:                 "non 200",
			mockMethod:           "POST",
			mockUrl:              "http://localhost:8080/flags/flag/resolve/string",
			mockHttpResponseCode: http.StatusBadRequest,
			mockHttpResponseBody: schemaV1.ErrorResponse{
				Reason:    models.StaticReason,
				ErrorCode: "CUSTOM ERROR MESSAGE",
			},
			mockErr: nil,
			httpServiceConfiguration: service.HttpServiceConfiguration{
				Port:     8080,
				Host:     "localhost",
				Protocol: "http",
			},
			flagKey:   "flag",
			outReason: models.ErrorReason,
			outErr:    errors.New("CUSTOM ERROR MESSAGE"),
		},
		{
			name:                 "non 200",
			mockMethod:           "POST",
			mockUrl:              "http://localhost:8080/flags/flag/resolve/string",
			mockHttpResponseCode: http.StatusInternalServerError,
			mockErr:              errors.New("its all gone wrong"),
			httpServiceConfiguration: service.HttpServiceConfiguration{
				Port:     8080,
				Host:     "localhost",
				Protocol: "http",
			},
			flagKey:   "flag",
			outReason: models.ErrorReason,
			outErr:    errors.New(models.GeneralErrorCode),
		},
	}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	for _, test := range tests {
		mock := mocks.NewMockiHTTPClient(ctrl)
		bodyM, err := json.Marshal(test.mockHttpResponseBody)
		if err != nil {
			t.Error(err)
		}
		mock.EXPECT().Request(test.mockMethod, test.mockUrl, gomock.Any()).AnyTimes().Return(
			&http.Response{
				StatusCode: test.mockHttpResponseCode,
				Body:       io.NopCloser(bytes.NewReader(bodyM)),
			},
			test.mockErr,
		)
		srv := service.HTTPService{
			Client:                   mock,
			HttpServiceConfiguration: &test.httpServiceConfiguration,
		}

		res, err := srv.ResolveString(test.flagKey, test.evCtx)
		if test.outErr != nil && !assert.EqualError(t, err, test.outErr.Error()) {
			t.Errorf("%s: unexpected error received, expected %v, got %v", test.name, test.outErr, err)
		}
		if res.Reason != test.outReason {
			t.Errorf("%s: unexpected reason received, expected %v, got %v", test.name, test.outReason, res.Reason)
		}
		if res.Value != test.outValue {
			t.Errorf("%s: unexpected value received, expected %v, got %v", test.name, test.outValue, res.Value)
		}
		if res.Variant != test.outVariant {
			t.Errorf("%s: unexpected variant received, expected %v, got %v", test.name, test.outVariant, res.Variant)
		}
	}
}

type TestServiceResolveNumberArgs struct {
	name string

	mockMethod string
	mockUrl    string

	mockHttpResponseCode int
	mockHttpResponseBody interface{}
	mockErr              error

	httpServiceConfiguration service.HttpServiceConfiguration

	flagKey string
	evCtx   of.EvaluationContext

	outErr     error
	outReason  string
	outVariant string
	outValue   float32
}

func TestServiceResolveNumber(t *testing.T) {
	tests := []TestServiceResolveNumberArgs{
		{
			name:                 "happy path",
			mockMethod:           "POST",
			mockUrl:              "http://localhost:8080/flags/flag/resolve/number",
			mockHttpResponseCode: http.StatusOK,
			mockHttpResponseBody: schemaV1.ResolveNumberResponse{
				Value:   float32(32),
				Variant: "on",
				Reason:  models.StaticReason,
			},
			mockErr: nil,
			httpServiceConfiguration: service.HttpServiceConfiguration{
				Port:     8080,
				Host:     "localhost",
				Protocol: "http",
			},
			flagKey:    "flag",
			outValue:   float32(32),
			outVariant: "on",
			outReason:  models.StaticReason,
			outErr:     nil,
		},
		{
			name:                 "non 200",
			mockMethod:           "POST",
			mockUrl:              "http://localhost:8080/flags/flag/resolve/number",
			mockHttpResponseCode: http.StatusBadRequest,
			mockHttpResponseBody: schemaV1.ErrorResponse{
				Reason:    models.StaticReason,
				ErrorCode: "CUSTOM ERROR MESSAGE",
			},
			mockErr: nil,
			httpServiceConfiguration: service.HttpServiceConfiguration{
				Port:     8080,
				Host:     "localhost",
				Protocol: "http",
			},
			flagKey:   "flag",
			outReason: models.ErrorReason,
			outErr:    errors.New("CUSTOM ERROR MESSAGE"),
		},
		{
			name:                 "non 200",
			mockMethod:           "POST",
			mockUrl:              "http://localhost:8080/flags/flag/resolve/number",
			mockHttpResponseCode: http.StatusInternalServerError,
			mockErr:              errors.New("its all gone wrong"),
			httpServiceConfiguration: service.HttpServiceConfiguration{
				Port:     8080,
				Host:     "localhost",
				Protocol: "http",
			},
			flagKey:   "flag",
			outReason: models.ErrorReason,
			outErr:    errors.New(models.GeneralErrorCode),
		},
	}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	for _, test := range tests {
		mock := mocks.NewMockiHTTPClient(ctrl)
		bodyM, err := json.Marshal(test.mockHttpResponseBody)
		if err != nil {
			t.Error(err)
		}
		mock.EXPECT().Request(test.mockMethod, test.mockUrl, gomock.Any()).AnyTimes().Return(
			&http.Response{
				StatusCode: test.mockHttpResponseCode,
				Body:       io.NopCloser(bytes.NewReader(bodyM)),
			},
			test.mockErr,
		)
		srv := service.HTTPService{
			Client:                   mock,
			HttpServiceConfiguration: &test.httpServiceConfiguration,
		}

		res, err := srv.ResolveNumber(test.flagKey, test.evCtx)
		if test.outErr != nil && !assert.EqualError(t, err, test.outErr.Error()) {
			t.Errorf("%s: unexpected error received, expected %v, got %v", test.name, test.outErr, err)
		}
		if res.Reason != test.outReason {
			t.Errorf("%s: unexpected reason received, expected %v, got %v", test.name, test.outReason, res.Reason)
		}
		if res.Value != test.outValue {
			t.Errorf("%s: unexpected value received, expected %v, got %v", test.name, test.outValue, res.Value)
		}
		if res.Variant != test.outVariant {
			t.Errorf("%s: unexpected variant received, expected %v, got %v", test.name, test.outVariant, res.Variant)
		}
	}
}

type TestServiceResolveObjectArgs struct {
	name string

	mockMethod string
	mockUrl    string

	mockHttpResponseCode int
	mockHttpResponseBody interface{}
	mockErr              error

	httpServiceConfiguration service.HttpServiceConfiguration

	flagKey string
	evCtx   of.EvaluationContext

	outErr     error
	outReason  string
	outVariant string
	outValue   map[string]interface{}
}

func TestServiceResolveObject(t *testing.T) {
	tests := []TestServiceResolveObjectArgs{
		{
			name:                 "happy path",
			mockMethod:           "POST",
			mockUrl:              "http://localhost:8080/flags/flag/resolve/object",
			mockHttpResponseCode: http.StatusOK,
			mockErr:              nil,
			httpServiceConfiguration: service.HttpServiceConfiguration{
				Port:     8080,
				Host:     "localhost",
				Protocol: "http",
			},
			flagKey: "flag",
			outValue: map[string]interface{}{
				"food": "bars",
			},
			outVariant: "on",
			outReason:  models.StaticReason,
			outErr:     nil,
		},
		{
			name:                 "non 200",
			mockMethod:           "POST",
			mockUrl:              "http://localhost:8080/flags/flag/resolve/object",
			mockHttpResponseCode: http.StatusBadRequest,
			mockHttpResponseBody: schemaV1.ErrorResponse{
				Reason:    models.StaticReason,
				ErrorCode: "CUSTOM ERROR MESSAGE",
			},
			mockErr: nil,
			httpServiceConfiguration: service.HttpServiceConfiguration{
				Port:     8080,
				Host:     "localhost",
				Protocol: "http",
			},
			flagKey:   "flag",
			outReason: models.ErrorReason,
			outErr:    errors.New("CUSTOM ERROR MESSAGE"),
		},
		{
			name:                 "non 200",
			mockMethod:           "POST",
			mockUrl:              "http://localhost:8080/flags/flag/resolve/object",
			mockHttpResponseCode: http.StatusInternalServerError,
			mockErr:              errors.New("its all gone wrong"),
			httpServiceConfiguration: service.HttpServiceConfiguration{
				Port:     8080,
				Host:     "localhost",
				Protocol: "http",
			},
			flagKey:   "flag",
			outReason: models.ErrorReason,
			outErr:    errors.New(models.GeneralErrorCode),
		},
	}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	for _, test := range tests {
		mock := mocks.NewMockiHTTPClient(ctrl)
		if test.outValue != nil {
			f, err := structpb.NewStruct(test.outValue)
			if err != nil {
				t.Error(err)
				t.FailNow()
			}
			test.mockHttpResponseBody = schemaV1.ResolveObjectResponse{
				Variant: "on",
				Reason:  models.StaticReason,
				Value:   f,
			}
		}
		bodyM, err := json.Marshal(test.mockHttpResponseBody)
		if err != nil {
			t.Error(err)
		}
		mock.EXPECT().Request(test.mockMethod, test.mockUrl, gomock.Any()).AnyTimes().Return(
			&http.Response{
				StatusCode: test.mockHttpResponseCode,
				Body:       io.NopCloser(bytes.NewReader(bodyM)),
			},
			test.mockErr,
		)
		srv := service.HTTPService{
			Client:                   mock,
			HttpServiceConfiguration: &test.httpServiceConfiguration,
		}

		res, err := srv.ResolveObject(test.flagKey, test.evCtx)
		if test.outErr != nil && !assert.EqualError(t, err, test.outErr.Error()) {
			t.Errorf("%s: unexpected error received, expected %v, got %v", test.name, test.outErr, err)
		}
		if res.Reason != test.outReason {
			t.Errorf("%s: unexpected reason received, expected %v, got %v", test.name, test.outReason, res.Reason)
		}
		if reflect.DeepEqual(res.Value, test.outValue) {
			t.Errorf("%s: unexpected value received, expected %v, got %v", test.name, test.outValue, res.Value)
		}
		if res.Variant != test.outVariant {
			t.Errorf("%s: unexpected variant received, expected %v, got %v", test.name, test.outVariant, res.Variant)
		}
	}
}
