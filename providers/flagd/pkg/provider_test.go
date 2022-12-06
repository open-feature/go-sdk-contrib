package flagd_test

import (
	"context"
	"fmt"
	reflect "reflect"
	"testing"

	schemav1 "buf.build/gen/go/open-feature/flagd/protocolbuffers/go/schema/v1"
	gomock "github.com/golang/mock/gomock"
	flagdModels "github.com/open-feature/flagd/pkg/model"
	flagd "github.com/open-feature/go-sdk-contrib/providers/flagd/pkg"
	of "github.com/open-feature/go-sdk/pkg/openfeature"
	"google.golang.org/protobuf/types/known/structpb"
)

type TestConstructorArgs struct {
	name    string
	port    uint16
	host    string
	options []flagd.ProviderOption
	env     bool
	envPort uint16
	envHost string
}

func TestNewProvider(t *testing.T) {
	tests := []TestConstructorArgs{
		{
			name: "happy path",
			port: 8013,
			host: "localhost",
		},
		{
			name: "with port",
			port: 1,
			host: "localhost",
			options: []flagd.ProviderOption{
				flagd.WithPort(1),
			},
		},
		{
			name: "with hostname",
			port: 8013,
			host: "not localhost",
			options: []flagd.ProviderOption{
				flagd.WithHost("not localhost"),
			},
		},
		{
			name: "from env - maintain default port preventing overwrite",
			port: 8013,
			host: "not localhost",
			options: []flagd.ProviderOption{
				flagd.WithPort(8013), //matched default
				flagd.FromEnv(),
			},
			env:     true,
			envPort: 1,
			envHost: "not localhost",
		},
		{
			name: "from env - maintain default port with explicit overwrite",
			port: 8013,
			host: "not localhost",
			options: []flagd.ProviderOption{
				flagd.FromEnv(),
				flagd.WithPort(8013), //matched default
			},
			env:     true,
			envPort: 1,
			envHost: "not localhost",
		},
	}

	for _, test := range tests {
		if test.env {
			t.Setenv("FLAGD_PORT", fmt.Sprintf("%d", test.envPort))
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

		// this line will fail linting if this provider is no longer compatible with the openfeature sdk
		var _ of.FeatureProvider = svc
	}
}

type TestBooleanEvaluationArgs struct {
	name         string
	flagKey      string
	defaultValue bool
	evalCtx      map[string]interface{}

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
			evalCtx: map[string]interface{}{
				"food": "bars",
			},
			mockOut: &schemav1.ResolveBooleanResponse{
				Value:   true,
				Variant: "on",
				Reason:  flagdModels.DefaultReason,
			},
			mockError: nil,
			response: of.BoolResolutionDetail{
				Value: true,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					Variant: "on",
					Reason:  flagdModels.DefaultReason,
				},
			},
		},
		{
			name:         "error response",
			flagKey:      "flag",
			defaultValue: true,
			evalCtx: map[string]interface{}{
				"food": "bars",
			},
			mockOut: &schemav1.ResolveBooleanResponse{
				Reason: flagdModels.DefaultReason,
			},
			mockError: of.NewFlagNotFoundResolutionError(""),
			response: of.BoolResolutionDetail{
				Value: true,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					Reason:          flagdModels.DefaultReason,
					ResolutionError: of.NewFlagNotFoundResolutionError(""),
				},
			},
		},
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	for _, test := range tests {
		mock := NewMockIService(ctrl)
		mock.EXPECT().ResolveBoolean(context.Background(), test.flagKey, test.evalCtx).Return(test.mockOut, test.mockError)

		provider := flagd.Provider{
			Service: mock,
		}

		res := provider.BooleanEvaluation(context.Background(), test.flagKey, test.defaultValue, test.evalCtx)

		if res.ResolutionError.Error() != test.response.ResolutionError.Error() {
			t.Errorf("%s: unexpected ResolutionError received, expected %v, got %v", test.name, test.response.ResolutionError.Error(), res.ResolutionError.Error())
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
	evalCtx      map[string]interface{}

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
			evalCtx: map[string]interface{}{
				"food": "bars",
			},
			mockOut: &schemav1.ResolveStringResponse{
				Value:   "true",
				Variant: "on",
				Reason:  flagdModels.DefaultReason,
			},
			mockError: nil,
			response: of.StringResolutionDetail{
				Value: "true",
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					Variant: "on",
					Reason:  flagdModels.DefaultReason,
				},
			},
		},
		{
			name:         "error response",
			flagKey:      "flag",
			defaultValue: "true",
			evalCtx: map[string]interface{}{
				"food": "bars",
			},
			mockOut: &schemav1.ResolveStringResponse{
				Reason: flagdModels.DefaultReason,
			},
			mockError: of.NewFlagNotFoundResolutionError(""),
			response: of.StringResolutionDetail{
				Value: "true",
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					Reason:          flagdModels.DefaultReason,
					ResolutionError: of.NewFlagNotFoundResolutionError(""),
				},
			},
		},
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	for _, test := range tests {
		mock := NewMockIService(ctrl)
		mock.EXPECT().ResolveString(context.Background(), test.flagKey, test.evalCtx).Return(test.mockOut, test.mockError)

		provider := flagd.Provider{
			Service: mock,
		}

		res := provider.StringEvaluation(context.Background(), test.flagKey, test.defaultValue, test.evalCtx)

		if res.ResolutionError.Error() != test.response.ResolutionError.Error() {
			t.Errorf("%s: unexpected ResolutionError received, expected %v, got %v", test.name, test.response.ResolutionError.Error(), res.ResolutionError.Error())
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
	evalCtx      map[string]interface{}

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
			evalCtx: map[string]interface{}{
				"food": "bars",
			},
			mockOut: &schemav1.ResolveFloatResponse{
				Value:   1,
				Variant: "on",
				Reason:  flagdModels.DefaultReason,
			},
			mockError: nil,
			response: of.FloatResolutionDetail{
				Value: 1,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					Variant: "on",
					Reason:  flagdModels.DefaultReason,
				},
			},
		},
		{
			name:         "error response",
			flagKey:      "flag",
			defaultValue: float64(1),
			evalCtx: map[string]interface{}{
				"food": "bars",
			},
			mockOut: &schemav1.ResolveFloatResponse{
				Reason: flagdModels.DefaultReason,
			},
			mockError: of.NewFlagNotFoundResolutionError(""),
			response: of.FloatResolutionDetail{
				Value: 1,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					Reason:          flagdModels.DefaultReason,
					ResolutionError: of.NewFlagNotFoundResolutionError(""),
				},
			},
		},
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	for _, test := range tests {
		mock := NewMockIService(ctrl)
		mock.EXPECT().ResolveFloat(context.Background(), test.flagKey, test.evalCtx).Return(test.mockOut, test.mockError)

		provider := flagd.Provider{
			Service: mock,
		}

		res := provider.FloatEvaluation(context.Background(), test.flagKey, test.defaultValue, test.evalCtx)

		if res.ResolutionError.Error() != test.response.ResolutionError.Error() {
			t.Errorf("%s: unexpected ResolutionError received, expected %v, got %v", test.name, test.response.ResolutionError.Error(), res.ResolutionError.Error())
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
	evalCtx      map[string]interface{}

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
			evalCtx: map[string]interface{}{
				"food": "bars",
			},
			mockOut: &schemav1.ResolveIntResponse{
				Value:   1,
				Variant: "on",
				Reason:  flagdModels.DefaultReason,
			},
			mockError: nil,
			response: of.IntResolutionDetail{
				Value: 1,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					Variant: "on",
					Reason:  flagdModels.DefaultReason,
				},
			},
		},
		{
			name:         "error response",
			flagKey:      "flag",
			defaultValue: 1,
			evalCtx: map[string]interface{}{
				"food": "bars",
			},
			mockOut: &schemav1.ResolveIntResponse{
				Reason: flagdModels.DefaultReason,
			},
			mockError: of.NewFlagNotFoundResolutionError(""),
			response: of.IntResolutionDetail{
				Value: 1,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					Reason:          flagdModels.DefaultReason,
					ResolutionError: of.NewFlagNotFoundResolutionError(""),
				},
			},
		},
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	for _, test := range tests {
		mock := NewMockIService(ctrl)
		mock.EXPECT().ResolveInt(context.Background(), test.flagKey, test.evalCtx).Return(test.mockOut, test.mockError)

		provider := flagd.Provider{
			Service: mock,
		}

		res := provider.IntEvaluation(context.Background(), test.flagKey, test.defaultValue, test.evalCtx)

		if res.ResolutionError.Error() != test.response.ResolutionError.Error() {
			t.Errorf("%s: unexpected ResolutionError received, expected %v, got %v", test.name, test.response.ResolutionError.Error(), res.ResolutionError.Error())
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
	evalCtx      map[string]interface{}

	mockOut   *schemav1.ResolveObjectResponse
	mockError error

	response of.InterfaceResolutionDetail
}

func TestObjectEvaluation(t *testing.T) {
	tests := []TestObjectEvaluationArgs{
		{
			name:    "happy path",
			flagKey: "flag",
			defaultValue: map[string]interface{}{
				"ping": "pong",
			},
			evalCtx: map[string]interface{}{
				"food": "bars",
			},
			mockOut: &schemav1.ResolveObjectResponse{
				Variant: "on",
				Reason:  flagdModels.DefaultReason,
			},
			mockError: nil,
			response: of.InterfaceResolutionDetail{
				Value: map[string]interface{}{
					"this": "that",
				},
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					Variant: "on",
					Reason:  flagdModels.DefaultReason,
				},
			},
		},
		{
			name:    "error response",
			flagKey: "flag",
			evalCtx: map[string]interface{}{
				"food": "bars",
			},
			mockOut: &schemav1.ResolveObjectResponse{
				Reason: flagdModels.DefaultReason,
			},
			mockError: of.NewFlagNotFoundResolutionError(""),
			response: of.InterfaceResolutionDetail{
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					Reason:          flagdModels.DefaultReason,
					ResolutionError: of.NewFlagNotFoundResolutionError(""),
				},
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

		mock.EXPECT().ResolveObject(context.Background(), test.flagKey, test.evalCtx).Return(test.mockOut, test.mockError)

		provider := flagd.Provider{
			Service: mock,
		}

		res := provider.ObjectEvaluation(context.Background(), test.flagKey, test.defaultValue, test.evalCtx)

		if res.ResolutionError.Error() != test.response.ResolutionError.Error() {
			t.Errorf("%s: unexpected ResolutionError received, expected %v, got %v", test.name, test.response.ResolutionError.Error(), res.ResolutionError.Error())
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
