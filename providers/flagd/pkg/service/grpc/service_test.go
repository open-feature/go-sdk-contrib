package grpc_service_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	of "github.com/open-feature/go-sdk/pkg/openfeature"

	"github.com/golang/mock/gomock"
	models "github.com/open-feature/flagd/pkg/model"
	service "github.com/open-feature/go-sdk-contrib/providers/flagd/pkg/service/grpc"
	"github.com/stretchr/testify/assert"
	schemaV1 "go.buf.build/open-feature/flagd-connect/open-feature/flagd/schema/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

type TestServiceResolveBooleanArgs struct {
	name string

	mockIn    *schemaV1.ResolveBooleanRequest
	nilClient bool

	mockOut   *schemaV1.ResolveBooleanResponse
	mockErr   error
	customErr string

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
			name: "happy path",
			mockIn: &schemaV1.ResolveBooleanRequest{
				FlagKey: "flag",
				Context: nil,
			},
			mockOut: &schemaV1.ResolveBooleanResponse{
				Value:   true,
				Variant: "on",
				Reason:  models.DefaultReason,
			},
			mockErr: nil,
			flagKey: "flag",
			evCtx: map[string]interface{}{
				"this": "that",
			},
			outErr:     nil,
			outReason:  models.DefaultReason,
			outValue:   true,
			outVariant: "on",
		},
		{
			name: "custom error response",
			mockIn: &schemaV1.ResolveBooleanRequest{
				FlagKey: "flag",
				Context: nil,
			},
			mockOut: &schemaV1.ResolveBooleanResponse{
				Value:   true,
				Variant: "on",
				Reason:  models.DefaultReason,
			},
			mockErr:   status.Error(codes.NotFound, "custom message"),
			customErr: "CUSTOM_ERROR",
			flagKey:   "flag",
			evCtx: map[string]interface{}{
				"this": "that",
			},
			outErr:    of.NewGeneralResolutionError(""),
			outReason: models.ErrorReason,
		},
		{
			name: "nil client",
			mockIn: &schemaV1.ResolveBooleanRequest{
				FlagKey: "flag",
				Context: nil,
			},
			nilClient: true,
			mockOut: &schemaV1.ResolveBooleanResponse{
				Value:   true,
				Variant: "on",
				Reason:  models.DefaultReason,
			},
			mockErr: status.Error(codes.NotFound, "custom message"),
			flagKey: "flag",
			evCtx: map[string]interface{}{
				"this": "that",
			},
			outErr:    of.NewProviderNotReadyResolutionError(""),
			outReason: models.ErrorReason,
		},
		{
			name: "parseError helper fails",
			mockIn: &schemaV1.ResolveBooleanRequest{
				FlagKey: "flag",
				Context: nil,
			},
			mockOut: &schemaV1.ResolveBooleanResponse{
				Value:   true,
				Variant: "on",
				Reason:  models.DefaultReason,
			},
			mockErr: status.Error(codes.NotFound, "custom message"),
			flagKey: "flag",
			evCtx: map[string]interface{}{
				"this": "that",
			},
			outErr:    errors.New(models.GeneralErrorCode),
			outReason: models.ErrorReason,
		},
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	for _, test := range tests {
		mock := NewMockServiceClient(ctrl)

		if test.customErr != "" {
			st, ok := status.FromError(test.mockErr)
			if !ok {
				t.Fatalf("%s: malformed error status received, cannot attach custom properties", test.name)
			}
			stWD, err := st.WithDetails(&schemaV1.ErrorResponse{
				ErrorCode: test.customErr,
				Reason:    models.ErrorReason,
			})
			if err != nil {
				t.Error(err)
			}
			test.mockErr = stWD.Err()
		}

		f, err := structpb.NewStruct(test.evCtx)
		if err != nil {
			t.Fatal(err)
		}
		test.mockIn.Context = f

		mock.EXPECT(gomock.Any(), test.mockIn).AnyTimes().Return(test.mockOut, test.mockErr)
		srv := service.GRPCService{
			Client: &MockClient{
				Client:    mock,
				NilClient: test.nilClient,
			},
		}
		res, err := srv.ResolveBoolean(context.Background(), test.flagKey, test.evCtx)
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

	mockIn    *schemaV1.ResolveFloatRequest
	nilClient bool

	mockOut   *schemaV1.ResolveFloatResponse
	mockErr   error
	customErr string

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
			name: "happy path",
			mockIn: &schemaV1.ResolveFloatRequest{
				FlagKey: "flag",
				Context: nil,
			},
			mockOut: &schemaV1.ResolveFloatResponse{
				Value:   12,
				Variant: "on",
				Reason:  models.DefaultReason,
			},
			mockErr: nil,
			flagKey: "flag",
			evCtx: map[string]interface{}{
				"this":         "that",
				"targetingKey": "me",
			},
			outErr:     nil,
			outReason:  models.DefaultReason,
			outValue:   12,
			outVariant: "on",
		},
		{
			name: "custom error response",
			mockIn: &schemaV1.ResolveFloatRequest{
				FlagKey: "flag",
				Context: nil,
			},
			mockOut: &schemaV1.ResolveFloatResponse{
				Value:   12,
				Variant: "on",
				Reason:  models.DefaultReason,
			},
			mockErr:   status.Error(codes.NotFound, "custom message"),
			customErr: "CUSTOM_ERROR",
			flagKey:   "flag",
			evCtx: map[string]interface{}{
				"this": "that",
			},
			outErr:    of.NewGeneralResolutionError(""),
			outReason: models.ErrorReason,
		},
		{
			name: "nil client",
			mockIn: &schemaV1.ResolveFloatRequest{
				FlagKey: "flag",
				Context: nil,
			},
			nilClient: true,
			mockOut: &schemaV1.ResolveFloatResponse{
				Value:   12,
				Variant: "on",
				Reason:  models.DefaultReason,
			},
			mockErr: status.Error(codes.NotFound, "custom message"),
			flagKey: "flag",
			evCtx: map[string]interface{}{
				"this": "that",
			},
			outErr:    of.NewProviderNotReadyResolutionError(""),
			outReason: models.ErrorReason,
		},
		{
			name: "parseError helper fails",
			mockIn: &schemaV1.ResolveFloatRequest{
				FlagKey: "flag",
				Context: nil,
			},
			mockOut: &schemaV1.ResolveFloatResponse{
				Value:   12,
				Variant: "on",
				Reason:  models.DefaultReason,
			},
			mockErr: status.Error(codes.NotFound, "custom message"),
			flagKey: "flag",
			evCtx: map[string]interface{}{
				"this": "that",
			},
			outErr:    errors.New(models.GeneralErrorCode),
			outReason: models.ErrorReason,
		},
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	for _, test := range tests {
		mock := NewMockServiceClient(ctrl)

		if test.customErr != "" {
			st, ok := status.FromError(test.mockErr)
			if !ok {
				t.Errorf("%s: malformed error status received, cannot attach custom properties", test.name)
			}
			stWD, err := st.WithDetails(&schemaV1.ErrorResponse{
				ErrorCode: test.customErr,
				Reason:    models.ErrorReason,
			})
			if err != nil {
				t.Error(err)
			}
			test.mockErr = stWD.Err()
		}

		f, err := structpb.NewStruct(test.evCtx)
		if err != nil {
			t.Fatal(err)
		}
		test.mockIn.Context = f

		mock.EXPECT().ResolveFloat(gomock.Any(), test.mockIn).AnyTimes().Return(test.mockOut, test.mockErr)
		srv := service.GRPCService{
			Client: &MockClient{
				Client:    mock,
				NilClient: test.nilClient,
			},
		}
		res, err := srv.ResolveFloat(context.Background(), test.flagKey, test.evCtx)
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

	mockIn    *schemaV1.ResolveIntRequest
	nilClient bool

	mockOut   *schemaV1.ResolveIntResponse
	mockErr   error
	customErr string

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
			name: "happy path",
			mockIn: &schemaV1.ResolveIntRequest{
				FlagKey: "flag",
				Context: nil,
			},
			mockOut: &schemaV1.ResolveIntResponse{
				Value:   12,
				Variant: "on",
				Reason:  models.DefaultReason,
			},
			mockErr: nil,
			flagKey: "flag",
			evCtx: map[string]interface{}{
				"this":         "that",
				"targetingKey": "me",
			},
			outErr:     nil,
			outReason:  models.DefaultReason,
			outValue:   12,
			outVariant: "on",
		},
		{
			name: "custom error response",
			mockIn: &schemaV1.ResolveIntRequest{
				FlagKey: "flag",
				Context: nil,
			},
			mockOut: &schemaV1.ResolveIntResponse{
				Value:   12,
				Variant: "on",
				Reason:  models.DefaultReason,
			},
			mockErr:   status.Error(codes.NotFound, "custom message"),
			customErr: "CUSTOM_ERROR",
			flagKey:   "flag",
			evCtx: map[string]interface{}{
				"this": "that",
			},
			outErr:    of.NewGeneralResolutionError(""),
			outReason: models.ErrorReason,
		},
		{
			name: "nil client",
			mockIn: &schemaV1.ResolveIntRequest{
				FlagKey: "flag",
				Context: nil,
			},
			nilClient: true,
			mockOut: &schemaV1.ResolveIntResponse{
				Value:   12,
				Variant: "on",
				Reason:  models.DefaultReason,
			},
			mockErr: status.Error(codes.NotFound, "custom message"),
			flagKey: "flag",
			evCtx: map[string]interface{}{
				"this": "that",
			},
			outErr:    of.NewProviderNotReadyResolutionError(""),
			outReason: models.ErrorReason,
		},
		{
			name: "parseError helper fails",
			mockIn: &schemaV1.ResolveIntRequest{
				FlagKey: "flag",
				Context: nil,
			},
			mockOut: &schemaV1.ResolveIntResponse{
				Value:   12,
				Variant: "on",
				Reason:  models.DefaultReason,
			},
			mockErr: status.Error(codes.NotFound, "custom message"),
			flagKey: "flag",
			evCtx: map[string]interface{}{
				"this": "that",
			},
			outErr:    errors.New(models.GeneralErrorCode),
			outReason: models.ErrorReason,
		},
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	for _, test := range tests {
		mock := NewMockServiceClient(ctrl)

		if test.customErr != "" {
			st, ok := status.FromError(test.mockErr)
			if !ok {
				t.Errorf("%s: malformed error status received, cannot attach custom properties", test.name)
			}
			stWD, err := st.WithDetails(&schemaV1.ErrorResponse{
				ErrorCode: test.customErr,
				Reason:    models.ErrorReason,
			})
			if err != nil {
				t.Error(err)
			}
			test.mockErr = stWD.Err()
		}

		f, err := structpb.NewStruct(test.evCtx)
		if err != nil {
			t.Fatal(err)
		}
		test.mockIn.Context = f

		mock.EXPECT().ResolveInt(gomock.Any(), test.mockIn).AnyTimes().Return(test.mockOut, test.mockErr)
		srv := service.GRPCService{
			Client: &MockClient{
				Client:    mock,
				NilClient: test.nilClient,
			},
		}
		res, err := srv.ResolveInt(context.Background(), test.flagKey, test.evCtx)
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

	mockIn    *schemaV1.ResolveStringRequest
	nilClient bool

	mockOut   *schemaV1.ResolveStringResponse
	mockErr   error
	customErr string

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
			name: "happy path",
			mockIn: &schemaV1.ResolveStringRequest{
				FlagKey: "flag",
				Context: nil,
			},
			mockOut: &schemaV1.ResolveStringResponse{
				Value:   "ok",
				Variant: "on",
				Reason:  models.DefaultReason,
			},
			mockErr: nil,
			flagKey: "flag",
			evCtx: map[string]interface{}{
				"this": "that",
			},
			outErr:     nil,
			outReason:  models.DefaultReason,
			outValue:   "ok",
			outVariant: "on",
		},
		{
			name: "custom error response",
			mockIn: &schemaV1.ResolveStringRequest{
				FlagKey: "flag",
				Context: nil,
			},
			mockOut: &schemaV1.ResolveStringResponse{
				Value:   "ok",
				Variant: "on",
				Reason:  models.DefaultReason,
			},
			mockErr:   status.Error(codes.NotFound, "custom message"),
			customErr: "CUSTOM_ERROR",
			flagKey:   "flag",
			evCtx: map[string]interface{}{
				"this": "that",
			},
			outErr:    of.NewGeneralResolutionError(""),
			outReason: models.ErrorReason,
		},
		{
			name: "nil client",
			mockIn: &schemaV1.ResolveStringRequest{
				FlagKey: "flag",
				Context: nil,
			},
			nilClient: true,
			mockOut: &schemaV1.ResolveStringResponse{
				Value:   "ok",
				Variant: "on",
				Reason:  models.DefaultReason,
			},
			mockErr: status.Error(codes.NotFound, "custom message"),
			flagKey: "flag",
			evCtx: map[string]interface{}{
				"this": "that",
			},
			outErr:    of.NewProviderNotReadyResolutionError(""),
			outReason: models.ErrorReason,
		},
		{
			name: "parseError helper fails",
			mockIn: &schemaV1.ResolveStringRequest{
				FlagKey: "flag",
				Context: nil,
			},
			mockOut: &schemaV1.ResolveStringResponse{
				Value:   "ok",
				Variant: "on",
				Reason:  models.DefaultReason,
			},
			mockErr: status.Error(codes.NotFound, "custom message"),
			flagKey: "flag",
			evCtx: map[string]interface{}{
				"this": "that",
			},
			outErr:    errors.New(models.GeneralErrorCode),
			outReason: models.ErrorReason,
		},
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	for _, test := range tests {
		mock := NewMockServiceClient(ctrl)

		if test.customErr != "" {
			st, ok := status.FromError(test.mockErr)
			if !ok {
				t.Errorf("%s: malformed error status received, cannot attach custom properties", test.name)
			}
			stWD, err := st.WithDetails(&schemaV1.ErrorResponse{
				ErrorCode: test.customErr,
				Reason:    models.ErrorReason,
			})
			if err != nil {
				t.Error(err)
			}
			test.mockErr = stWD.Err()
		}

		f, err := structpb.NewStruct(test.evCtx)
		if err != nil {
			t.Fatal(err)
		}
		test.mockIn.Context = f

		mock.EXPECT().ResolveString(gomock.Any(), test.mockIn).AnyTimes().Return(test.mockOut, test.mockErr)
		srv := service.GRPCService{
			Client: &MockClient{
				Client:    mock,
				NilClient: test.nilClient,
			},
		}
		res, err := srv.ResolveString(context.Background(), test.flagKey, test.evCtx)
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

	mockIn    *schemaV1.ResolveObjectRequest
	nilClient bool

	mockOut   *schemaV1.ResolveObjectResponse
	mockErr   error
	customErr string

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
			name: "happy path",
			mockIn: &schemaV1.ResolveObjectRequest{
				FlagKey: "flag",
				Context: nil,
			},
			mockOut: &schemaV1.ResolveObjectResponse{
				Variant: "on",
				Reason:  models.DefaultReason,
			},
			mockErr: nil,
			flagKey: "flag",
			evCtx: map[string]interface{}{
				"this": "that",
			},
			outErr:    nil,
			outReason: models.DefaultReason,
			outValue: map[string]interface{}{
				"food": "bars",
			},
			outVariant: "on",
		},
		{
			name: "custom error response",
			mockIn: &schemaV1.ResolveObjectRequest{
				FlagKey: "flag",
				Context: nil,
			},
			mockOut: &schemaV1.ResolveObjectResponse{
				Variant: "on",
				Reason:  models.DefaultReason,
			},
			mockErr:   status.Error(codes.NotFound, "custom message"),
			customErr: "CUSTOM_ERROR",
			flagKey:   "flag",
			evCtx: map[string]interface{}{
				"this": "that",
			},
			outErr:    of.NewGeneralResolutionError(""),
			outReason: models.ErrorReason,
		},
		{
			name: "nil client",
			mockIn: &schemaV1.ResolveObjectRequest{
				FlagKey: "flag",
				Context: nil,
			},
			nilClient: true,
			mockOut: &schemaV1.ResolveObjectResponse{
				Variant: "on",
				Reason:  models.DefaultReason,
			},
			mockErr: status.Error(codes.NotFound, "custom message"),
			flagKey: "flag",
			evCtx: map[string]interface{}{
				"this": "that",
			},
			outErr:    of.NewProviderNotReadyResolutionError(""),
			outReason: models.ErrorReason,
		},
		{
			name: "parseError helper fails",
			mockIn: &schemaV1.ResolveObjectRequest{
				FlagKey: "flag",
				Context: nil,
			},
			mockOut: &schemaV1.ResolveObjectResponse{
				Variant: "on",
				Reason:  models.DefaultReason,
			},
			mockErr: status.Error(codes.NotFound, "custom message"),
			flagKey: "flag",
			evCtx: map[string]interface{}{
				"this": "that",
			},
			outErr:    errors.New(models.GeneralErrorCode),
			outReason: models.ErrorReason,
		},
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	for _, test := range tests {
		mock := NewMockServiceClient(ctrl)

		if test.customErr != "" {
			st, ok := status.FromError(test.mockErr)
			if !ok {
				t.Errorf("%s: malformed error status received, cannot attach custom properties", test.name)
			}
			stWD, err := st.WithDetails(&schemaV1.ErrorResponse{
				ErrorCode: test.customErr,
				Reason:    models.ErrorReason,
			})
			if err != nil {
				t.Error(err)
			}
			test.mockErr = stWD.Err()
		}

		f, err := structpb.NewStruct(test.evCtx)
		if err != nil {
			t.Fatal(err)
		}
		test.mockIn.Context = f

		if test.outValue != nil {
			f, err := structpb.NewStruct(test.outValue)
			if err != nil {
				t.Fatal(err)
			}
			test.mockOut.Value = f
		}

		mock.EXPECT().ResolveObject(gomock.Any(), test.mockIn).AnyTimes().Return(test.mockOut, test.mockErr)
		srv := service.GRPCService{
			Client: &MockClient{
				Client:    mock,
				NilClient: test.nilClient,
			},
		}
		res, err := srv.ResolveObject(context.Background(), test.flagKey, test.evCtx)
		if test.outErr != nil && !assert.EqualError(t, err, test.outErr.Error()) {
			t.Errorf("%s: unexpected error received, expected %v, got %v", test.name, test.outErr, err)
		}
		if res.Reason != test.outReason {
			t.Errorf("%s: unexpected reason received, expected %v, got %v", test.name, test.outReason, res.Reason)
		}
		if res.Value != nil && test.mockOut.Value != nil && !reflect.DeepEqual(res.Value.AsMap(), test.outValue) {
			t.Errorf("%s: unexpected value received, expected %v, got %v", test.name, test.outValue, res.Value)
		}
		if res.Variant != test.outVariant {
			t.Errorf("%s: unexpected variant received, expected %v, got %v", test.name, test.outVariant, res.Variant)
		}
	}
}
