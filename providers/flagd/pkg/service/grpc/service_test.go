package grpc_service_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"
	models "github.com/open-feature/flagd/pkg/model"
	service "github.com/open-feature/golang-sdk-contrib/providers/flagd/pkg/service/grpc"
	of "github.com/open-feature/golang-sdk/pkg/openfeature"
	"github.com/stretchr/testify/assert"
	schemaV1 "go.buf.build/grpc/go/open-feature/flagd/schema/v1"
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
	evCtx   of.EvaluationContext

	outErr     error
	outReason  string
	outVariant string
	outValue   bool

	structFormatCheck bool
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
				Reason:  models.StaticReason,
			},
			mockErr: nil,
			flagKey: "flag",
			evCtx: of.EvaluationContext{
				Attributes: map[string]interface{}{
					"this": "that",
				},
			},
			outErr:     nil,
			outReason:  models.StaticReason,
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
				Reason:  models.StaticReason,
			},
			mockErr:   status.Error(codes.NotFound, "custom message"),
			customErr: "CUSTOM_ERROR",
			flagKey:   "flag",
			evCtx: of.EvaluationContext{
				Attributes: map[string]interface{}{
					"this": "that",
				},
			},
			outErr:    errors.New("CUSTOM_ERROR"),
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
				Reason:  models.StaticReason,
			},
			mockErr: status.Error(codes.NotFound, "custom message"),
			flagKey: "flag",
			evCtx: of.EvaluationContext{
				Attributes: map[string]interface{}{
					"this": "that",
				},
			},
			outErr:    errors.New("CONNECTION_ERROR"),
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
				Reason:  models.StaticReason,
			},
			mockErr: status.Error(codes.NotFound, "custom message"),
			flagKey: "flag",
			evCtx: of.EvaluationContext{
				Attributes: map[string]interface{}{
					"this": "that",
				},
			},
			outErr:    errors.New(models.GeneralErrorCode),
			outReason: models.ErrorReason,
		},
		{
			name: "formatStructAsPbFails",
			mockIn: &schemaV1.ResolveBooleanRequest{
				FlagKey: "flag",
				Context: nil,
			},
			mockOut: &schemaV1.ResolveBooleanResponse{
				Value:   true,
				Variant: "on",
				Reason:  models.StaticReason,
			},
			mockErr: nil,
			flagKey: "flag",
			evCtx: of.EvaluationContext{
				Attributes: map[string]interface{}{
					"this": make(chan error, 5),
				},
			},
			outErr:            nil,
			outReason:         models.ErrorReason,
			structFormatCheck: true,
		},
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	for _, test := range tests {
		mock := NewMockServiceClient(ctrl)

		if test.customErr != "" {
			st, ok := status.FromError(test.mockErr)
			if !ok {
				t.Errorf("%s: malformed error status recieved, cannot attach custom properties", test.name)
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

		if !reflect.DeepEqual(of.EvaluationContext{}, test.evCtx) && !test.structFormatCheck {
			f, err := service.FormatAsStructpb(test.evCtx)
			if err != nil {
				t.Error(err)
				t.FailNow()
			}
			test.mockIn.Context = f
		}

		mock.EXPECT().ResolveBoolean(gomock.Any(), test.mockIn).AnyTimes().Return(test.mockOut, test.mockErr)
		srv := service.GRPCService{
			Client: &MockClient{
				Client:    mock,
				NilClient: test.nilClient,
			},
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

type TestServiceResolveNumberArgs struct {
	name string

	mockIn    *schemaV1.ResolveNumberRequest
	nilClient bool

	mockOut   *schemaV1.ResolveNumberResponse
	mockErr   error
	customErr string

	flagKey string
	evCtx   of.EvaluationContext

	outErr     error
	outReason  string
	outVariant string
	outValue   float32

	structFormatCheck bool
}

func TestServiceResolveNumber(t *testing.T) {
	tests := []TestServiceResolveNumberArgs{
		{
			name: "happy path",
			mockIn: &schemaV1.ResolveNumberRequest{
				FlagKey: "flag",
				Context: nil,
			},
			mockOut: &schemaV1.ResolveNumberResponse{
				Value:   float32(12),
				Variant: "on",
				Reason:  models.StaticReason,
			},
			mockErr: nil,
			flagKey: "flag",
			evCtx: of.EvaluationContext{
				TargetingKey: "me",
				Attributes: map[string]interface{}{
					"this": "that",
				},
			},
			outErr:     nil,
			outReason:  models.StaticReason,
			outValue:   float32(12),
			outVariant: "on",
		},
		{
			name: "custom error response",
			mockIn: &schemaV1.ResolveNumberRequest{
				FlagKey: "flag",
				Context: nil,
			},
			mockOut: &schemaV1.ResolveNumberResponse{
				Value:   float32(12),
				Variant: "on",
				Reason:  models.StaticReason,
			},
			mockErr:   status.Error(codes.NotFound, "custom message"),
			customErr: "CUSTOM_ERROR",
			flagKey:   "flag",
			evCtx: of.EvaluationContext{
				Attributes: map[string]interface{}{
					"this": "that",
				},
			},
			outErr:    errors.New("CUSTOM_ERROR"),
			outReason: models.ErrorReason,
		},
		{
			name: "nil client",
			mockIn: &schemaV1.ResolveNumberRequest{
				FlagKey: "flag",
				Context: nil,
			},
			nilClient: true,
			mockOut: &schemaV1.ResolveNumberResponse{
				Value:   float32(12),
				Variant: "on",
				Reason:  models.StaticReason,
			},
			mockErr: status.Error(codes.NotFound, "custom message"),
			flagKey: "flag",
			evCtx: of.EvaluationContext{
				Attributes: map[string]interface{}{
					"this": "that",
				},
			},
			outErr:    errors.New("CONNECTION_ERROR"),
			outReason: models.ErrorReason,
		},
		{
			name: "parseError helper fails",
			mockIn: &schemaV1.ResolveNumberRequest{
				FlagKey: "flag",
				Context: nil,
			},
			mockOut: &schemaV1.ResolveNumberResponse{
				Value:   float32(12),
				Variant: "on",
				Reason:  models.StaticReason,
			},
			mockErr: status.Error(codes.NotFound, "custom message"),
			flagKey: "flag",
			evCtx: of.EvaluationContext{
				Attributes: map[string]interface{}{
					"this": "that",
				},
			},
			outErr:    errors.New(models.GeneralErrorCode),
			outReason: models.ErrorReason,
		},
		{
			name: "formatStructAsPb Fails",
			mockIn: &schemaV1.ResolveNumberRequest{
				FlagKey: "flag",
				Context: nil,
			},
			mockOut: &schemaV1.ResolveNumberResponse{
				Value:   float32(12),
				Variant: "on",
				Reason:  models.StaticReason,
			},
			mockErr: nil,
			flagKey: "flag",
			evCtx: of.EvaluationContext{
				Attributes: map[string]interface{}{
					"this": make(chan error, 5),
				},
			},
			outErr:            nil,
			outReason:         models.ErrorReason,
			structFormatCheck: true,
		},
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	for _, test := range tests {
		mock := NewMockServiceClient(ctrl)

		if test.customErr != "" {
			st, ok := status.FromError(test.mockErr)
			if !ok {
				t.Errorf("%s: malformed error status recieved, cannot attach custom properties", test.name)
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

		if !reflect.DeepEqual(of.EvaluationContext{}, test.evCtx) && !test.structFormatCheck {
			f, err := service.FormatAsStructpb(test.evCtx)
			if err != nil {
				t.Error(err)
				t.FailNow()
			}
			test.mockIn.Context = f
		}

		mock.EXPECT().ResolveNumber(gomock.Any(), test.mockIn).AnyTimes().Return(test.mockOut, test.mockErr)
		srv := service.GRPCService{
			Client: &MockClient{
				Client:    mock,
				NilClient: test.nilClient,
			},
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

type TestServiceResolveStringArgs struct {
	name string

	mockIn    *schemaV1.ResolveStringRequest
	nilClient bool

	mockOut   *schemaV1.ResolveStringResponse
	mockErr   error
	customErr string

	flagKey string
	evCtx   of.EvaluationContext

	outErr     error
	outReason  string
	outVariant string
	outValue   string

	structFormatCheck bool
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
				Reason:  models.StaticReason,
			},
			mockErr: nil,
			flagKey: "flag",
			evCtx: of.EvaluationContext{
				Attributes: map[string]interface{}{
					"this": "that",
				},
			},
			outErr:     nil,
			outReason:  models.StaticReason,
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
				Reason:  models.StaticReason,
			},
			mockErr:   status.Error(codes.NotFound, "custom message"),
			customErr: "CUSTOM_ERROR",
			flagKey:   "flag",
			evCtx: of.EvaluationContext{
				Attributes: map[string]interface{}{
					"this": "that",
				},
			},
			outErr:    errors.New("CUSTOM_ERROR"),
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
				Reason:  models.StaticReason,
			},
			mockErr: status.Error(codes.NotFound, "custom message"),
			flagKey: "flag",
			evCtx: of.EvaluationContext{
				Attributes: map[string]interface{}{
					"this": "that",
				},
			},
			outErr:    errors.New("CONNECTION_ERROR"),
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
				Reason:  models.StaticReason,
			},
			mockErr: status.Error(codes.NotFound, "custom message"),
			flagKey: "flag",
			evCtx: of.EvaluationContext{
				Attributes: map[string]interface{}{
					"this": "that",
				},
			},
			outErr:    errors.New(models.GeneralErrorCode),
			outReason: models.ErrorReason,
		},
		{
			name: "formatStructAsPb Fails",
			mockIn: &schemaV1.ResolveStringRequest{
				FlagKey: "flag",
				Context: nil,
			},
			mockOut: &schemaV1.ResolveStringResponse{
				Value:   "ok",
				Variant: "on",
				Reason:  models.StaticReason,
			},
			mockErr: nil,
			flagKey: "flag",
			evCtx: of.EvaluationContext{
				Attributes: map[string]interface{}{
					"this": make(chan error, 5),
				},
			},
			outErr:            nil,
			outReason:         models.ErrorReason,
			structFormatCheck: true,
		},
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	for _, test := range tests {
		mock := NewMockServiceClient(ctrl)

		if test.customErr != "" {
			st, ok := status.FromError(test.mockErr)
			if !ok {
				t.Errorf("%s: malformed error status recieved, cannot attach custom properties", test.name)
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

		if !reflect.DeepEqual(of.EvaluationContext{}, test.evCtx) && !test.structFormatCheck {
			f, err := service.FormatAsStructpb(test.evCtx)
			if err != nil {
				t.Error(err)
				t.FailNow()
			}
			test.mockIn.Context = f
		}

		mock.EXPECT().ResolveString(gomock.Any(), test.mockIn).AnyTimes().Return(test.mockOut, test.mockErr)
		srv := service.GRPCService{
			Client: &MockClient{
				Client:    mock,
				NilClient: test.nilClient,
			},
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

type TestServiceResolveObjectArgs struct {
	name string

	mockIn    *schemaV1.ResolveObjectRequest
	nilClient bool

	mockOut   *schemaV1.ResolveObjectResponse
	mockErr   error
	customErr string

	flagKey string
	evCtx   of.EvaluationContext

	outErr     error
	outReason  string
	outVariant string
	outValue   map[string]interface{}

	structFormatCheck bool
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
				Reason:  models.StaticReason,
			},
			mockErr: nil,
			flagKey: "flag",
			evCtx: of.EvaluationContext{
				Attributes: map[string]interface{}{
					"this": "that",
				},
			},
			outErr:    nil,
			outReason: models.StaticReason,
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
				Reason:  models.StaticReason,
			},
			mockErr:   status.Error(codes.NotFound, "custom message"),
			customErr: "CUSTOM_ERROR",
			flagKey:   "flag",
			evCtx: of.EvaluationContext{
				Attributes: map[string]interface{}{
					"this": "that",
				},
			},
			outErr:    errors.New("CUSTOM_ERROR"),
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
				Reason:  models.StaticReason,
			},
			mockErr: status.Error(codes.NotFound, "custom message"),
			flagKey: "flag",
			evCtx: of.EvaluationContext{
				Attributes: map[string]interface{}{
					"this": "that",
				},
			},
			outErr:    errors.New("CONNECTION_ERROR"),
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
				Reason:  models.StaticReason,
			},
			mockErr: status.Error(codes.NotFound, "custom message"),
			flagKey: "flag",
			evCtx: of.EvaluationContext{
				Attributes: map[string]interface{}{
					"this": "that",
				},
			},
			outErr:    errors.New(models.GeneralErrorCode),
			outReason: models.ErrorReason,
		},

		{
			name: "formatStructasPb fails",
			mockIn: &schemaV1.ResolveObjectRequest{
				FlagKey: "flag",
				Context: nil,
			},
			mockOut: &schemaV1.ResolveObjectResponse{
				Variant: "on",
				Reason:  models.StaticReason,
			},
			mockErr: nil,
			flagKey: "flag",
			evCtx: of.EvaluationContext{
				Attributes: map[string]interface{}{
					"this": make(chan error, 5),
				},
			},
			outErr:            nil,
			outReason:         models.ErrorReason,
			structFormatCheck: true,
		},
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	for _, test := range tests {
		mock := NewMockServiceClient(ctrl)

		if test.customErr != "" {
			st, ok := status.FromError(test.mockErr)
			if !ok {
				t.Errorf("%s: malformed error status recieved, cannot attach custom properties", test.name)
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

		if !reflect.DeepEqual(of.EvaluationContext{}, test.evCtx) && !test.structFormatCheck {
			f, err := service.FormatAsStructpb(test.evCtx)
			if err != nil {
				t.Error(err)
				t.FailNow()
			}
			test.mockIn.Context = f
		}

		if test.outValue != nil {
			f, err := structpb.NewStruct(test.outValue)
			if err != nil {
				t.Error(err)
				t.FailNow()
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
		res, err := srv.ResolveObject(test.flagKey, test.evCtx)
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
