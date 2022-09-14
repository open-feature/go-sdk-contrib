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

	httpServiceConfiguration service.HTTPServiceConfiguration

	flagKey string
	evCtx   map[string]interface{}

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
			mockUrl:              "http://localhost:8013/flags/flag/resolve/boolean",
			mockHttpResponseCode: http.StatusOK,
			mockHttpResponseBody: schemaV1.ResolveBooleanResponse{
				Value:   true,
				Variant: "on",
				Reason:  models.StaticReason,
			},
			mockErr: nil,
			httpServiceConfiguration: service.HTTPServiceConfiguration{
				Port:     8013,
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
			mockUrl:              "http://localhost:8013/flags/flag/resolve/boolean",
			mockHttpResponseCode: http.StatusBadRequest,
			mockHttpResponseBody: schemaV1.ErrorResponse{
				Reason:    models.StaticReason,
				ErrorCode: "CUSTOM ERROR MESSAGE",
			},
			mockErr: nil,
			httpServiceConfiguration: service.HTTPServiceConfiguration{
				Port:     8013,
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
			mockUrl:              "http://localhost:8013/flags/flag/resolve/boolean",
			mockHttpResponseCode: http.StatusInternalServerError,
			mockErr:              errors.New("its all gone wrong"),
			httpServiceConfiguration: service.HTTPServiceConfiguration{
				Port:     8013,
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
		mock := NewMockiHTTPClient(ctrl)
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
			HTTPServiceConfiguration: &test.httpServiceConfiguration,
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

	httpServiceConfiguration service.HTTPServiceConfiguration

	flagKey string
	evCtx   map[string]interface{}

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
			mockUrl:              "http://localhost:8013/flags/flag/resolve/string",
			mockHttpResponseCode: http.StatusOK,
			mockHttpResponseBody: schemaV1.ResolveStringResponse{
				Value:   "value",
				Variant: "on",
				Reason:  models.StaticReason,
			},
			mockErr: nil,
			httpServiceConfiguration: service.HTTPServiceConfiguration{
				Port:     8013,
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
			mockUrl:              "http://localhost:8013/flags/flag/resolve/string",
			mockHttpResponseCode: http.StatusBadRequest,
			mockHttpResponseBody: schemaV1.ErrorResponse{
				Reason:    models.StaticReason,
				ErrorCode: "CUSTOM ERROR MESSAGE",
			},
			mockErr: nil,
			httpServiceConfiguration: service.HTTPServiceConfiguration{
				Port:     8013,
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
			mockUrl:              "http://localhost:8013/flags/flag/resolve/string",
			mockHttpResponseCode: http.StatusInternalServerError,
			mockErr:              errors.New("its all gone wrong"),
			httpServiceConfiguration: service.HTTPServiceConfiguration{
				Port:     8013,
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
		mock := NewMockiHTTPClient(ctrl)
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
			HTTPServiceConfiguration: &test.httpServiceConfiguration,
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

type TestServiceResolveFloatArgs struct {
	name string

	mockMethod string
	mockUrl    string

	mockHttpResponseCode int
	mockHttpResponseBody interface{}
	mockErr              error

	httpServiceConfiguration service.HTTPServiceConfiguration

	flagKey string
	evCtx   map[string]interface{}

	outErr     error
	outReason  string
	outVariant string
	outValue   float64
}

func TestServiceResolveFloat(t *testing.T) {
	tests := []TestServiceResolveFloatArgs{
		{
			name:                 "happy path",
			mockMethod:           "POST",
			mockUrl:              "http://localhost:8013/flags/flag/resolve/float",
			mockHttpResponseCode: http.StatusOK,
			mockHttpResponseBody: schemaV1.ResolveFloatResponse{
				Value:   32,
				Variant: "on",
				Reason:  models.StaticReason,
			},
			mockErr: nil,
			httpServiceConfiguration: service.HTTPServiceConfiguration{
				Port:     8013,
				Host:     "localhost",
				Protocol: "http",
			},
			flagKey:    "flag",
			outValue:   32,
			outVariant: "on",
			outReason:  models.StaticReason,
			outErr:     nil,
		},
		{
			name:                 "non 200",
			mockMethod:           "POST",
			mockUrl:              "http://localhost:8013/flags/flag/resolve/float",
			mockHttpResponseCode: http.StatusBadRequest,
			mockHttpResponseBody: schemaV1.ErrorResponse{
				Reason:    models.StaticReason,
				ErrorCode: "CUSTOM ERROR MESSAGE",
			},
			mockErr: nil,
			httpServiceConfiguration: service.HTTPServiceConfiguration{
				Port:     8013,
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
			mockUrl:              "http://localhost:8013/flags/flag/resolve/float",
			mockHttpResponseCode: http.StatusInternalServerError,
			mockErr:              errors.New("its all gone wrong"),
			httpServiceConfiguration: service.HTTPServiceConfiguration{
				Port:     8013,
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
		mock := NewMockiHTTPClient(ctrl)
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
			HTTPServiceConfiguration: &test.httpServiceConfiguration,
		}

		res, err := srv.ResolveFloat(test.flagKey, test.evCtx)
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

type TestServiceResolveIntArgs struct {
	name string

	mockMethod string
	mockUrl    string

	mockHttpResponseCode int
	mockHttpResponseBody interface{}
	mockErr              error

	httpServiceConfiguration service.HTTPServiceConfiguration

	flagKey string
	evCtx   map[string]interface{}

	outErr     error
	outReason  string
	outVariant string
	outValue   int64
}

func TestServiceResolveInt(t *testing.T) {
	tests := []TestServiceResolveIntArgs{
		{
			name:                 "happy path",
			mockMethod:           "POST",
			mockUrl:              "http://localhost:8013/flags/flag/resolve/int",
			mockHttpResponseCode: http.StatusOK,
			mockHttpResponseBody: service.IntDecodeIntermediate{
				Value:   "32",
				Variant: "on",
				Reason:  models.StaticReason,
			},
			mockErr: nil,
			httpServiceConfiguration: service.HTTPServiceConfiguration{
				Port:     8013,
				Host:     "localhost",
				Protocol: "http",
			},
			flagKey:    "flag",
			outValue:   32,
			outVariant: "on",
			outReason:  models.StaticReason,
			outErr:     nil,
		},
		{
			name:                 "non 200",
			mockMethod:           "POST",
			mockUrl:              "http://localhost:8013/flags/flag/resolve/int",
			mockHttpResponseCode: http.StatusBadRequest,
			mockHttpResponseBody: schemaV1.ErrorResponse{
				Reason:    models.StaticReason,
				ErrorCode: "CUSTOM ERROR MESSAGE",
			},
			mockErr: nil,
			httpServiceConfiguration: service.HTTPServiceConfiguration{
				Port:     8013,
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
			mockUrl:              "http://localhost:8013/flags/flag/resolve/int",
			mockHttpResponseCode: http.StatusInternalServerError,
			mockErr:              errors.New("its all gone wrong"),
			httpServiceConfiguration: service.HTTPServiceConfiguration{
				Port:     8013,
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
		mock := NewMockiHTTPClient(ctrl)
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
			HTTPServiceConfiguration: &test.httpServiceConfiguration,
		}

		res, err := srv.ResolveInt(test.flagKey, test.evCtx)
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

	httpServiceConfiguration service.HTTPServiceConfiguration

	flagKey string
	evCtx   map[string]interface{}

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
			mockUrl:              "http://localhost:8013/flags/flag/resolve/object",
			mockHttpResponseCode: http.StatusOK,
			mockErr:              nil,
			httpServiceConfiguration: service.HTTPServiceConfiguration{
				Port:     8013,
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
			mockUrl:              "http://localhost:8013/flags/flag/resolve/object",
			mockHttpResponseCode: http.StatusBadRequest,
			mockHttpResponseBody: schemaV1.ErrorResponse{
				Reason:    models.StaticReason,
				ErrorCode: "CUSTOM ERROR MESSAGE",
			},
			mockErr: nil,
			httpServiceConfiguration: service.HTTPServiceConfiguration{
				Port:     8013,
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
			mockUrl:              "http://localhost:8013/flags/flag/resolve/object",
			mockHttpResponseCode: http.StatusInternalServerError,
			mockErr:              errors.New("its all gone wrong"),
			httpServiceConfiguration: service.HTTPServiceConfiguration{
				Port:     8013,
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
		mock := NewMockiHTTPClient(ctrl)
		if test.outValue != nil {
			f, err := structpb.NewStruct(test.outValue)
			if err != nil {
				t.Fatal(err)
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
			HTTPServiceConfiguration: &test.httpServiceConfiguration,
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

type TestFetchFlagArgs struct {
	name string

	mockHttpResponseCode int
	mockHttpResponseBody interface{}
	mockErr              error

	body interface{}
	url  string
	ctx  map[string]interface{}
	err  error
}

func TestFetchFlag(t *testing.T) {
	tests := []TestFetchFlagArgs{
		{
			name: "happy path",
			body: map[string]interface{}{
				"food": "bars",
			},
			url: "GET/MY/FLAG",
			ctx: map[string]interface{}{
				"targetingKey": "target",
			},
			mockHttpResponseCode: 200,
			err:                  nil,
		},
		{
			name: "200 response cannot unmarshal",
			body: "string",
			url:  "GET/MY/FLAG",
			ctx: map[string]interface{}{
				"con": "text",
			},
			mockHttpResponseCode: 200,
			err:                  errors.New(models.ParseErrorCode),
		},
		{
			name: "non 200 response cannot unmarshal",
			body: "string",
			url:  "GET/MY/FLAG",
			ctx: map[string]interface{}{
				"con": "text",
			},
			mockHttpResponseCode: 400,
			err:                  errors.New(models.ParseErrorCode),
		},
		{
			name: "non 200 response",
			body: schemaV1.ErrorResponse{
				ErrorCode: models.FlagNotFoundErrorCode,
			},
			url: "GET/MY/FLAG",
			ctx: map[string]interface{}{
				"con": "text",
			},
			mockHttpResponseCode: 404,
			err:                  errors.New(models.FlagNotFoundErrorCode),
		},
		{
			name: "500 response",
			url:  "GET/MY/FLAG",
			ctx: map[string]interface{}{
				"con": "text",
			},
			mockHttpResponseCode: 500,
			err:                  errors.New(models.GeneralErrorCode),
		},
		{
			name: "fall through",
			body: schemaV1.ErrorResponse{
				ErrorCode: "",
			},
			url:                  "GET/MY/FLAG",
			mockHttpResponseCode: 400,
			err:                  errors.New(models.GeneralErrorCode),
		},
		{
			name: "context marshal failure",
			body: schemaV1.ErrorResponse{
				ErrorCode: "",
			},
			url:                  "GET/MY/FLAG",
			mockHttpResponseCode: 400,
			err:                  errors.New(models.ParseErrorCode),
			ctx: map[string]interface{}{
				"will fail": make(chan error, 5),
			},
		},
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	for _, test := range tests {
		mock := NewMockiHTTPClient(ctrl)
		bodyM, err := json.Marshal(test.body)
		if err != nil {
			t.Error(err)
		}
		test.mockHttpResponseBody = &http.Response{
			StatusCode: test.mockHttpResponseCode,
			Body:       io.NopCloser(bytes.NewReader(bodyM)),
		}
		mock.EXPECT().Request(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(
			&http.Response{
				StatusCode: test.mockHttpResponseCode,
				Body:       io.NopCloser(bytes.NewReader(bodyM)),
			},
			test.mockErr,
		)
		svc := service.HTTPService{
			Client: mock,
		}
		target := map[string]interface{}{}
		err = svc.FetchFlag(test.url, test.ctx, &target)

		if test.err != nil && !assert.EqualError(t, err, test.err.Error()) {
			t.Errorf("%s: unexpected value for error expected %v received %v", test.name, test.err, err)
		}
	}
}
