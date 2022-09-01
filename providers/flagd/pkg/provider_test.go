package flagd_test

import (
	"errors"
	"fmt"
	reflect "reflect"
	"testing"

	gomock "github.com/golang/mock/gomock"
	flagdModels "github.com/open-feature/flagd/pkg/model"
	flagd "github.com/open-feature/golang-sdk-contrib/providers/flagd/pkg"
	of "github.com/open-feature/golang-sdk/pkg/openfeature"
	schemav1 "go.buf.build/grpc/go/open-feature/flagd/schema/v1"
	"google.golang.org/protobuf/types/known/structpb"
)

type TestConstructorArgs struct {
	name       string
	port       uint16
	host       string
	service    flagd.ServiceType
	options    []flagd.ProviderOption
	env        bool
	envPort    uint16
	envHost    string
	envService flagd.ServiceType
}

func TestNewProvider(t *testing.T) {
	tests := []TestConstructorArgs{
		{
			name:    "happy path",
			port:    8013,
			host:    "localhost",
			service: flagd.HTTP,
		},
		{
			name:    "with service https",
			port:    8013,
			host:    "localhost",
			service: flagd.HTTPS,
			options: []flagd.ProviderOption{
				flagd.WithService(flagd.HTTPS),
			},
		},
		{
			name:    "with service grpc",
			port:    8013,
			host:    "localhost",
			service: flagd.GRPC,
			options: []flagd.ProviderOption{
				flagd.WithService(flagd.GRPC),
			},
		},
		{
			name:    "with port",
			port:    1,
			host:    "localhost",
			service: flagd.HTTP,
			options: []flagd.ProviderOption{
				flagd.WithPort(1),
			},
		},
		{
			name:    "with hostname",
			port:    8013,
			host:    "not localhost",
			service: flagd.HTTP,
			options: []flagd.ProviderOption{
				flagd.WithHost("not localhost"),
			},
		},
		{
			name:    "from env - maintain default port preventing overwrite",
			port:    8013,
			host:    "not localhost",
			service: flagd.HTTPS,
			options: []flagd.ProviderOption{
				flagd.WithPort(8013), //matched default
				flagd.FromEnv(),
			},
			env:        true,
			envService: flagd.HTTPS,
			envPort:    1,
			envHost:    "not localhost",
		},
		{
			name:    "from env - maintain default port with explicit overwrite",
			port:    8013,
			host:    "not localhost",
			service: flagd.HTTPS,
			options: []flagd.ProviderOption{
				flagd.FromEnv(),
				flagd.WithPort(8013), //matched default
			},
			env:        true,
			envService: flagd.HTTPS,
			envPort:    1,
			envHost:    "not localhost",
		},
	}

	for _, test := range tests {
		if test.env {
			t.Setenv("FLAGD_PORT", fmt.Sprintf("%d", test.envPort))
			if test.envService == flagd.HTTP {
				t.Setenv("FLAGD_SERVICE_PROVIDER", "http")
			}
			if test.envService == flagd.HTTPS {
				t.Setenv("FLAGD_SERVICE_PROVIDER", "https")
			}
			if test.envService == flagd.GRPC {
				t.Setenv("FLAGD_SERVICE_PROVIDER", "grpc")
			}
			t.Setenv("FLAGD_HOST", test.envHost)
		}
		svc := flagd.NewProvider(test.options...)
		if svc == nil {
			t.Fatalf("%s received nil service from NewProvider", test.name)
		}
		metadata := svc.Metadata()
		if metadata.Name != "flagd" {
			t.Errorf(
				"%s received unexpected metadata from NewProvider, expected %s got %s",
				test.name,
				"flagd",
				metadata.Name,
			)
		}
		config := svc.Configuration()
		if config == nil {
			t.Fatal("config is nil")
		}
		if config.Host != test.host {
			t.Errorf(
				"%s received unexpected ProviderConfiguration.Host from NewProvider, expected %s got %s",
				test.name,
				test.host,
				config.Host,
			)
		}
		if config.Port != test.port {
			t.Errorf(
				"%s received unexpected ProviderConfiguration.Port from NewProvider, expected %d got %d",
				test.name,
				test.port,
				config.Port,
			)
		}
		if config.ServiceName != test.service {
			t.Errorf(
				"%s received unexpected ProviderConfiguration.Port from NewProvider, expected %d got %d",
				test.name,
				test.service,
				config.ServiceName,
			)
		}

		// this line will fail linting if this provider is no longer compatible with the openfeature sdk
		var _ of.FeatureProvider = svc
	}
}

type TestBooleanEvaluationArgs struct {
	name         string
	flagKey      string
	defaultValue bool
	evalCtx      of.EvaluationContext

	mockOut   *schemav1.ResolveBooleanResponse
	mockError error

	response of.BoolResolutionDetail
}

func TestBooleanEvaluation(t *testing.T) {
	tests := []TestBooleanEvaluationArgs{
		{
			name:         "happy path",
			flagKey:      "flag",
			defaultValue: true,
			evalCtx: of.EvaluationContext{
				Attributes: map[string]interface{}{
					"food": "bars",
				},
			},
			mockOut: &schemav1.ResolveBooleanResponse{
				Value:   true,
				Variant: "on",
				Reason:  flagdModels.StaticReason,
			},
			mockError: nil,
			response: of.BoolResolutionDetail{
				Value: true,
				ResolutionDetail: of.ResolutionDetail{
					Value:   true,
					Variant: "on",
					Reason:  flagdModels.StaticReason,
				},
			},
		},
		{
			name:         "error response",
			flagKey:      "flag",
			defaultValue: true,
			evalCtx: of.EvaluationContext{
				Attributes: map[string]interface{}{
					"food": "bars",
				},
			},
			mockOut: &schemav1.ResolveBooleanResponse{
				Reason: flagdModels.StaticReason,
			},
			mockError: errors.New(flagdModels.FlagNotFoundErrorCode),
			response: of.BoolResolutionDetail{
				Value: true,
				ResolutionDetail: of.ResolutionDetail{
					Value:     true,
					Reason:    flagdModels.StaticReason,
					ErrorCode: flagdModels.FlagNotFoundErrorCode,
				},
			},
		},
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	for _, test := range tests {
		mock := NewMockIService(ctrl)
		mock.EXPECT().ResolveBoolean(test.flagKey, test.evalCtx).Return(test.mockOut, test.mockError)

		provider := flagd.Provider{
			Service: mock,
		}

		res := provider.BooleanEvaluation(test.flagKey, test.defaultValue, test.evalCtx)

		if res.ErrorCode != test.response.ErrorCode {
			t.Errorf("%s: unexpected ErrorCode received, expected %v, got %v", test.name, test.response.ErrorCode, res.ErrorCode)
		}
		if res.Variant != test.response.Variant {
			t.Errorf("%s: unexpected Variant received, expected %v, got %v", test.name, test.response.Variant, res.Variant)
		}
		if res.Value != test.response.Value {
			t.Errorf("%s: unexpected Value received, expected %v, got %v", test.name, test.response.Value, res.Value)
		}
		if res.Reason != test.response.Reason {
			t.Errorf("%s: unexpected Reason received, expected %v, got %v", test.name, test.response.Reason, res.Reason)
		}
	}
}

type TestStringEvaluationArgs struct {
	name         string
	flagKey      string
	defaultValue string
	evalCtx      of.EvaluationContext

	mockOut   *schemav1.ResolveStringResponse
	mockError error

	response of.StringResolutionDetail
}

func TestStringEvaluation(t *testing.T) {
	tests := []TestStringEvaluationArgs{
		{
			name:         "happy path",
			flagKey:      "flag",
			defaultValue: "true",
			evalCtx: of.EvaluationContext{
				Attributes: map[string]interface{}{
					"food": "bars",
				},
			},
			mockOut: &schemav1.ResolveStringResponse{
				Value:   "true",
				Variant: "on",
				Reason:  flagdModels.StaticReason,
			},
			mockError: nil,
			response: of.StringResolutionDetail{
				Value: "true",
				ResolutionDetail: of.ResolutionDetail{
					Value:   true,
					Variant: "on",
					Reason:  flagdModels.StaticReason,
				},
			},
		},
		{
			name:         "error response",
			flagKey:      "flag",
			defaultValue: "true",
			evalCtx: of.EvaluationContext{
				Attributes: map[string]interface{}{
					"food": "bars",
				},
			},
			mockOut: &schemav1.ResolveStringResponse{
				Reason: flagdModels.StaticReason,
			},
			mockError: errors.New(flagdModels.FlagNotFoundErrorCode),
			response: of.StringResolutionDetail{
				Value: "true",
				ResolutionDetail: of.ResolutionDetail{
					Value:     true,
					Reason:    flagdModels.StaticReason,
					ErrorCode: flagdModels.FlagNotFoundErrorCode,
				},
			},
		},
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	for _, test := range tests {
		mock := NewMockIService(ctrl)
		mock.EXPECT().ResolveString(test.flagKey, test.evalCtx).Return(test.mockOut, test.mockError)

		provider := flagd.Provider{
			Service: mock,
		}

		res := provider.StringEvaluation(test.flagKey, test.defaultValue, test.evalCtx)

		if res.ErrorCode != test.response.ErrorCode {
			t.Errorf("%s: unexpected ErrorCode received, expected %v, got %v", test.name, test.response.ErrorCode, res.ErrorCode)
		}
		if res.Variant != test.response.Variant {
			t.Errorf("%s: unexpected Variant received, expected %v, got %v", test.name, test.response.Variant, res.Variant)
		}
		if res.Value != test.response.Value {
			t.Errorf("%s: unexpected Value received, expected %v, got %v", test.name, test.response.Value, res.Value)
		}
		if res.Reason != test.response.Reason {
			t.Errorf("%s: unexpected Reason received, expected %v, got %v", test.name, test.response.Reason, res.Reason)
		}
	}
}

type TestFloatEvaluationArgs struct {
	name         string
	flagKey      string
	defaultValue float64
	evalCtx      of.EvaluationContext

	mockOut   *schemav1.ResolveFloatResponse
	mockError error

	response of.FloatResolutionDetail
}

func TestFloatEvaluation(t *testing.T) {
	tests := []TestFloatEvaluationArgs{
		{
			name:         "happy path",
			flagKey:      "flag",
			defaultValue: float64(1),
			evalCtx: of.EvaluationContext{
				Attributes: map[string]interface{}{
					"food": "bars",
				},
			},
			mockOut: &schemav1.ResolveFloatResponse{
				Value:   1,
				Variant: "on",
				Reason:  flagdModels.StaticReason,
			},
			mockError: nil,
			response: of.FloatResolutionDetail{
				Value: 1,
				ResolutionDetail: of.ResolutionDetail{
					Value:   true,
					Variant: "on",
					Reason:  flagdModels.StaticReason,
				},
			},
		},
		{
			name:         "error response",
			flagKey:      "flag",
			defaultValue: float64(1),
			evalCtx: of.EvaluationContext{
				Attributes: map[string]interface{}{
					"food": "bars",
				},
			},
			mockOut: &schemav1.ResolveFloatResponse{
				Reason: flagdModels.StaticReason,
			},
			mockError: errors.New(flagdModels.FlagNotFoundErrorCode),
			response: of.FloatResolutionDetail{
				Value: 1,
				ResolutionDetail: of.ResolutionDetail{
					Value:     true,
					Reason:    flagdModels.StaticReason,
					ErrorCode: flagdModels.FlagNotFoundErrorCode,
				},
			},
		},
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	for _, test := range tests {
		mock := NewMockIService(ctrl)
		mock.EXPECT().ResolveFloat(test.flagKey, test.evalCtx).Return(test.mockOut, test.mockError)

		provider := flagd.Provider{
			Service: mock,
		}

		res := provider.FloatEvaluation(test.flagKey, test.defaultValue, test.evalCtx)

		if res.ErrorCode != test.response.ErrorCode {
			t.Errorf("%s: unexpected ErrorCode received, expected %v, got %v", test.name, test.response.ErrorCode, res.ErrorCode)
		}
		if res.Variant != test.response.Variant {
			t.Errorf("%s: unexpected Variant received, expected %v, got %v", test.name, test.response.Variant, res.Variant)
		}
		if res.Value != test.response.Value {
			t.Errorf("%s: unexpected Value received, expected %v, got %v", test.name, test.response.Value, res.Value)
		}
		if res.Reason != test.response.Reason {
			t.Errorf("%s: unexpected Reason received, expected %v, got %v", test.name, test.response.Reason, res.Reason)
		}
	}
}

type TestIntEvaluationArgs struct {
	name         string
	flagKey      string
	defaultValue int64
	evalCtx      of.EvaluationContext

	mockOut   *schemav1.ResolveIntResponse
	mockError error

	response of.IntResolutionDetail
}

func TestIntEvaluation(t *testing.T) {
	tests := []TestIntEvaluationArgs{
		{
			name:         "happy path",
			flagKey:      "flag",
			defaultValue: 1,
			evalCtx: of.EvaluationContext{
				Attributes: map[string]interface{}{
					"food": "bars",
				},
			},
			mockOut: &schemav1.ResolveIntResponse{
				Value:   1,
				Variant: "on",
				Reason:  flagdModels.StaticReason,
			},
			mockError: nil,
			response: of.IntResolutionDetail{
				Value: 1,
				ResolutionDetail: of.ResolutionDetail{
					Value:   true,
					Variant: "on",
					Reason:  flagdModels.StaticReason,
				},
			},
		},
		{
			name:         "error response",
			flagKey:      "flag",
			defaultValue: 1,
			evalCtx: of.EvaluationContext{
				Attributes: map[string]interface{}{
					"food": "bars",
				},
			},
			mockOut: &schemav1.ResolveIntResponse{
				Reason: flagdModels.StaticReason,
			},
			mockError: errors.New(flagdModels.FlagNotFoundErrorCode),
			response: of.IntResolutionDetail{
				Value: 1,
				ResolutionDetail: of.ResolutionDetail{
					Value:     true,
					Reason:    flagdModels.StaticReason,
					ErrorCode: flagdModels.FlagNotFoundErrorCode,
				},
			},
		},
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	for _, test := range tests {
		mock := NewMockIService(ctrl)
		mock.EXPECT().ResolveInt(test.flagKey, test.evalCtx).Return(test.mockOut, test.mockError)

		provider := flagd.Provider{
			Service: mock,
		}

		res := provider.IntEvaluation(test.flagKey, test.defaultValue, test.evalCtx)

		if res.ErrorCode != test.response.ErrorCode {
			t.Errorf("%s: unexpected ErrorCode received, expected %v, got %v", test.name, test.response.ErrorCode, res.ErrorCode)
		}
		if res.Variant != test.response.Variant {
			t.Errorf("%s: unexpected Variant received, expected %v, got %v", test.name, test.response.Variant, res.Variant)
		}
		if res.Value != test.response.Value {
			t.Errorf("%s: unexpected Value received, expected %v, got %v", test.name, test.response.Value, res.Value)
		}
		if res.Reason != test.response.Reason {
			t.Errorf("%s: unexpected Reason received, expected %v, got %v", test.name, test.response.Reason, res.Reason)
		}
	}
}

type TestObjectEvaluationArgs struct {
	name         string
	flagKey      string
	defaultValue map[string]interface{}
	evalCtx      of.EvaluationContext

	mockOut   *schemav1.ResolveObjectResponse
	mockError error

	response of.ResolutionDetail
}

func TestObjectEvaluation(t *testing.T) {
	tests := []TestObjectEvaluationArgs{
		{
			name:    "happy path",
			flagKey: "flag",
			defaultValue: map[string]interface{}{
				"ping": "pong",
			},
			evalCtx: of.EvaluationContext{
				Attributes: map[string]interface{}{
					"food": "bars",
				},
			},
			mockOut: &schemav1.ResolveObjectResponse{
				Variant: "on",
				Reason:  flagdModels.StaticReason,
			},
			mockError: nil,
			response: of.ResolutionDetail{
				Value: map[string]interface{}{
					"this": "that",
				},
				Variant: "on",
				Reason:  flagdModels.StaticReason,
			},
		},
		{
			name:    "error response",
			flagKey: "flag",
			evalCtx: of.EvaluationContext{
				Attributes: map[string]interface{}{
					"food": "bars",
				},
			},
			mockOut: &schemav1.ResolveObjectResponse{
				Reason: flagdModels.StaticReason,
			},
			mockError: errors.New(flagdModels.FlagNotFoundErrorCode),
			response: of.ResolutionDetail{
				Reason:    flagdModels.StaticReason,
				ErrorCode: flagdModels.FlagNotFoundErrorCode,
			},
		},
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	for _, test := range tests {
		mock := NewMockIService(ctrl)

		if test.response.Value != nil {
			f, err := structpb.NewStruct(test.response.Value.(map[string]interface{}))
			if err != nil {
				t.Fatal(err)
			}
			test.mockOut.Value = f
		}

		mock.EXPECT().ResolveObject(test.flagKey, test.evalCtx).Return(test.mockOut, test.mockError)

		provider := flagd.Provider{
			Service: mock,
		}

		res := provider.ObjectEvaluation(test.flagKey, test.defaultValue, test.evalCtx)

		if res.ErrorCode != test.response.ErrorCode {
			t.Errorf("%s: unexpected ErrorCode received, expected %v, got %v", test.name, test.response.ErrorCode, res.ErrorCode)
		}
		if res.Variant != test.response.Variant {
			t.Errorf("%s: unexpected Variant received, expected %v, got %v", test.name, test.response.Variant, res.Variant)
		}
		if res.Value != nil && test.mockOut.Value != nil && !reflect.DeepEqual(res.Value.(*structpb.Struct).AsMap(), test.response.Value.(map[string]interface{})) {
			t.Errorf("%s: unexpected Value received, expected %v, got %v", test.name, test.response.Value, res.Value)
		}
		if res.Reason != test.response.Reason {
			t.Errorf("%s: unexpected Reason received, expected %v, got %v", test.name, test.response.Reason, res.Reason)
		}
	}
}
