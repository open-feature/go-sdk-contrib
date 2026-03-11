package process

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/open-feature/flagd/core/pkg/evaluator"
	"github.com/open-feature/flagd/core/pkg/model"
	"github.com/open-feature/flagd/core/pkg/sync"
	"github.com/open-feature/go-sdk/openfeature"
)

// Tests below use a mock evaluator to test correct wiring of responses

type testDescription struct {
	name      string
	evaluator MockEvaluator
	value     interface{}
	resDetail openfeature.ProviderResolutionDetail
	isError   bool
}

func TestBooleanEvaluation(t *testing.T) {
	tests := []testDescription{
		{
			name: "Boolean Success",
			evaluator: MockEvaluator{
				value:    true,
				variant:  "on",
				reason:   string(openfeature.StaticReason),
				metadata: make(map[string]interface{}),
				err:      nil,
			},
			value: true,
			resDetail: openfeature.ProviderResolutionDetail{
				Reason:       openfeature.StaticReason,
				Variant:      "on",
				FlagMetadata: make(map[string]interface{}),
			},
		},
		{
			name: "Boolean Error",
			evaluator: MockEvaluator{
				value:    false,
				variant:  "off",
				reason:   string(openfeature.ErrorReason),
				metadata: make(map[string]interface{}),
				err:      errors.New("SomeError"),
			},
			value: false,
			resDetail: openfeature.ProviderResolutionDetail{
				Reason:       openfeature.ErrorReason,
				Variant:      "off",
				FlagMetadata: make(map[string]interface{}),
			},
			isError: true,
		},
		{
			name: "Boolean Fallback - no default variant returns code default",
			evaluator: MockEvaluator{
				value:    false,
				variant:  "",
				reason:   model.FallbackReason,
				metadata: make(map[string]interface{}),
				err:      nil,
			},
			value: true,
			resDetail: openfeature.ProviderResolutionDetail{
				Reason:       openfeature.DefaultReason,
				Variant:      "",
				FlagMetadata: make(map[string]interface{}),
			},
		},
	}

	for _, test := range tests {
		inProcessService := InProcess{evaluator: test.evaluator}
		defaultValue := false
		if test.evaluator.reason == model.FallbackReason {
			defaultValue = true
		}
		booleanEval := inProcessService.ResolveBoolean(context.Background(), "any", defaultValue, make(map[string]interface{}))
		commonValidator(t, test, booleanEval.Value, booleanEval.ProviderResolutionDetail)
	}
}

func TestStringEvaluation(t *testing.T) {
	tests := []testDescription{
		{
			name: "String Success",
			evaluator: MockEvaluator{
				value:    "Hello",
				variant:  "v1",
				reason:   string(openfeature.StaticReason),
				metadata: make(map[string]interface{}),
				err:      nil,
			},
			value: "Hello",
			resDetail: openfeature.ProviderResolutionDetail{
				Reason:       openfeature.StaticReason,
				Variant:      "v1",
				FlagMetadata: make(map[string]interface{}),
			},
		},
		{
			name: "String Error",
			evaluator: MockEvaluator{
				value:    "Hello",
				variant:  "v1",
				reason:   string(openfeature.ErrorReason),
				metadata: make(map[string]interface{}),
				err:      errors.New("SomeError"),
			},
			value: "",
			resDetail: openfeature.ProviderResolutionDetail{
				Reason:       openfeature.ErrorReason,
				Variant:      "v1",
				FlagMetadata: make(map[string]interface{}),
			},
			isError: true,
		},
		{
			name: "String Fallback - no default variant returns code default",
			evaluator: MockEvaluator{
				value:    "",
				variant:  "",
				reason:   model.FallbackReason,
				metadata: make(map[string]interface{}),
				err:      nil,
			},
			value: "my-default",
			resDetail: openfeature.ProviderResolutionDetail{
				Reason:       openfeature.DefaultReason,
				Variant:      "",
				FlagMetadata: make(map[string]interface{}),
			},
		},
	}

	for _, test := range tests {
		inProcessService := InProcess{evaluator: test.evaluator}
		defaultValue := ""
		if test.evaluator.reason == model.FallbackReason {
			defaultValue = "my-default"
		}
		stringEval := inProcessService.ResolveString(context.Background(), "any", defaultValue, make(map[string]interface{}))
		commonValidator(t, test, stringEval.Value, stringEval.ProviderResolutionDetail)
	}
}

func TestFloatEvaluation(t *testing.T) {
	tests := []testDescription{
		{
			name: "Float Success",
			evaluator: MockEvaluator{
				value:    1.01,
				variant:  "v1",
				reason:   string(openfeature.StaticReason),
				metadata: make(map[string]interface{}),
				err:      nil,
			},
			value: 1.01,
			resDetail: openfeature.ProviderResolutionDetail{
				Reason:       openfeature.StaticReason,
				Variant:      "v1",
				FlagMetadata: make(map[string]interface{}),
			},
		},
		{
			name: "Float Error",
			evaluator: MockEvaluator{
				value:    1.0,
				variant:  "",
				reason:   string(openfeature.ErrorReason),
				metadata: make(map[string]interface{}),
				err:      errors.New("SomeError"),
			},
			value: 0.0,
			resDetail: openfeature.ProviderResolutionDetail{
				Reason:       openfeature.ErrorReason,
				Variant:      "",
				FlagMetadata: make(map[string]interface{}),
			},
			isError: true,
		},
		{
			name: "Float Fallback - no default variant returns code default",
			evaluator: MockEvaluator{
				value:    0.0,
				variant:  "",
				reason:   model.FallbackReason,
				metadata: make(map[string]interface{}),
				err:      nil,
			},
			value: 42.5,
			resDetail: openfeature.ProviderResolutionDetail{
				Reason:       openfeature.DefaultReason,
				Variant:      "",
				FlagMetadata: make(map[string]interface{}),
			},
		},
	}

	for _, test := range tests {
		inProcessService := InProcess{evaluator: test.evaluator}
		defaultValue := 0.0
		if test.evaluator.reason == model.FallbackReason {
			defaultValue = 42.5
		}
		floatEval := inProcessService.ResolveFloat(context.Background(), "any", defaultValue, make(map[string]interface{}))
		commonValidator(t, test, floatEval.Value, floatEval.ProviderResolutionDetail)
	}
}

func TestIntEvaluation(t *testing.T) {
	tests := []testDescription{
		{
			name: "Integer Success",
			evaluator: MockEvaluator{
				value:    int64(100),
				variant:  "v1",
				reason:   string(openfeature.StaticReason),
				metadata: make(map[string]interface{}),
				err:      nil,
			},
			value: int64(100),
			resDetail: openfeature.ProviderResolutionDetail{
				Reason:       openfeature.StaticReason,
				Variant:      "v1",
				FlagMetadata: make(map[string]interface{}),
			},
		},
		{
			name: "Integer Error",
			evaluator: MockEvaluator{
				value:    int64(0),
				variant:  "",
				reason:   string(openfeature.ErrorReason),
				metadata: make(map[string]interface{}),
				err:      errors.New("SomeError"),
			},
			value: int64(0),
			resDetail: openfeature.ProviderResolutionDetail{
				Reason:       openfeature.ErrorReason,
				Variant:      "",
				FlagMetadata: make(map[string]interface{}),
			},
			isError: true,
		},
		{
			name: "Integer Fallback - no default variant returns code default",
			evaluator: MockEvaluator{
				value:    int64(0),
				variant:  "",
				reason:   model.FallbackReason,
				metadata: make(map[string]interface{}),
				err:      nil,
			},
			value: int64(99),
			resDetail: openfeature.ProviderResolutionDetail{
				Reason:       openfeature.DefaultReason,
				Variant:      "",
				FlagMetadata: make(map[string]interface{}),
			},
		},
	}

	for _, test := range tests {
		inProcessService := InProcess{evaluator: test.evaluator}
		defaultValue := int64(0)
		if test.evaluator.reason == model.FallbackReason {
			defaultValue = 99
		}
		intEval := inProcessService.ResolveInt(context.Background(), "any", defaultValue, make(map[string]interface{}))
		commonValidator(t, test, intEval.Value, intEval.ProviderResolutionDetail)
	}
}

func TestObjectEvaluation(t *testing.T) {
	structValue := map[string]interface{}{
		"name": "some Name",
	}

	tests := []testDescription{
		{
			name: "Object Success",
			evaluator: MockEvaluator{
				value:    structValue,
				variant:  "v1",
				reason:   string(openfeature.StaticReason),
				metadata: make(map[string]interface{}),
				err:      nil,
			},
			value: structValue,
			resDetail: openfeature.ProviderResolutionDetail{
				Reason:       openfeature.StaticReason,
				Variant:      "v1",
				FlagMetadata: make(map[string]interface{}),
			},
		},
		{
			name: "Object Error",
			evaluator: MockEvaluator{
				value:    make(map[string]interface{}),
				variant:  "",
				reason:   string(openfeature.ErrorReason),
				metadata: make(map[string]interface{}),
				err:      errors.New("SomeError"),
			},
			value: make(map[string]interface{}),
			resDetail: openfeature.ProviderResolutionDetail{
				Reason:       openfeature.ErrorReason,
				Variant:      "",
				FlagMetadata: make(map[string]interface{}),
			},
			isError: true,
		},
		{
			name: "Object Fallback - no default variant returns code default",
			evaluator: MockEvaluator{
				value:    make(map[string]interface{}),
				variant:  "",
				reason:   model.FallbackReason,
				metadata: make(map[string]interface{}),
				err:      nil,
			},
			value: map[string]interface{}{"default": true},
			resDetail: openfeature.ProviderResolutionDetail{
				Reason:       openfeature.DefaultReason,
				Variant:      "",
				FlagMetadata: make(map[string]interface{}),
			},
		},
	}

	defaultObj := map[string]interface{}{"default": true}

	for _, test := range tests {
		inProcessService := InProcess{evaluator: test.evaluator}
		defaultValue := make(map[string]interface{})
		if test.evaluator.reason == model.FallbackReason {
			defaultValue = defaultObj
		}
		objEval := inProcessService.ResolveObject(context.Background(), "any", defaultValue, make(map[string]interface{}))

		if !reflect.DeepEqual(test.value, objEval.Value) {
			t.Logf("Test failed:  %s", test.name)
			t.Fatalf("Expected value %v, but got %v", test.value, objEval.Value)
		}

		if test.resDetail.Variant != objEval.Variant {
			t.Logf("Test failed:  %s", test.name)
			t.Fatalf("Expected reason %s, but got %v", test.resDetail.Variant, objEval.Variant)
		}

		if test.resDetail.Reason != objEval.Reason {
			t.Logf("Test failed:  %s", test.name)
			t.Fatalf("Expected reason %s, but got %v", test.resDetail.Reason, objEval.Reason)
		}

		if test.isError && objEval.Error() == nil {
			t.Logf("Test failed:  %s", test.name)
			t.Fatal("Expected error in resolution but got none")
		}
	}
}

func TestErrorMapping(t *testing.T) {
	// validate correct error mapping from flagd to OF
	tests := []struct {
		name         string
		errorType    string
		expectedCode openfeature.ErrorCode
	}{
		{
			name:         "Flag not found",
			errorType:    model.FlagNotFoundErrorCode,
			expectedCode: openfeature.FlagNotFoundCode,
		},
		{
			name:         "Flag disabled",
			errorType:    model.FlagDisabledErrorCode,
			expectedCode: openfeature.FlagNotFoundCode,
		},
		{
			name:         "Type mismatch",
			errorType:    model.TypeMismatchErrorCode,
			expectedCode: openfeature.TypeMismatchCode,
		},
		{
			name:         "Parsing error",
			errorType:    model.ParseErrorCode,
			expectedCode: openfeature.ParseErrorCode,
		},
		{
			name:         "General error",
			errorType:    model.GeneralErrorCode,
			expectedCode: openfeature.GeneralCode,
		},
	}

	for _, test := range tests {
		resolution := mapError("someFlag", errors.New(test.errorType))

		if !strings.HasPrefix(resolution.Error(), string(test.expectedCode)) {
			t.Errorf("Test %s: Expected resolution error to contain prefix %s, but error was %s",
				test.name, test.expectedCode, resolution.Error())
		}
	}
}

// commonValidator for tests
func commonValidator(t *testing.T, test testDescription, value interface{}, details openfeature.ProviderResolutionDetail) {
	if test.value != value {
		t.Logf("Test failed:  %s", test.name)
		t.Fatalf("Expected value %v, but got %v", test.value, value)
	}

	if test.resDetail.Variant != details.Variant {
		t.Logf("Test failed:  %s", test.name)
		t.Fatalf("Expected reason %s, but got %v", test.resDetail.Variant, details.Variant)
	}

	if test.resDetail.Reason != details.Reason {
		t.Logf("Test failed:  %s", test.name)
		t.Fatalf("Expected reason %s, but got %v", test.resDetail.Reason, details.Reason)
	}

	if test.isError && details.Error() == nil {
		t.Logf("Test failed:  %s", test.name)
		t.Fatal("Expected error in resolution but got none")
	}
}

// Mock Evaluator for testing
type MockEvaluator struct {
	value    interface{}
	variant  string
	reason   string
	metadata model.Metadata
	err      error
}

func (m MockEvaluator) ResolveBooleanValue(ctx context.Context, reqID string, flagKey string, context map[string]any) (value bool, variant string, reason string, metadata model.Metadata, err error) {
	return m.value.(bool), m.variant, m.reason, m.metadata, m.err
}

func (m MockEvaluator) ResolveStringValue(ctx context.Context, reqID string, flagKey string, context map[string]any) (value string, variant string, reason string, metadata model.Metadata, err error) {
	return m.value.(string), m.variant, m.reason, m.metadata, m.err
}

func (m MockEvaluator) ResolveIntValue(ctx context.Context, reqID string, flagKey string, context map[string]any) (value int64, variant string, reason string, metadata model.Metadata, err error) {
	return m.value.(int64), m.variant, m.reason, m.metadata, m.err
}

func (m MockEvaluator) ResolveFloatValue(ctx context.Context, reqID string, flagKey string, context map[string]any) (value float64, variant string, reason string, metadata model.Metadata, err error) {
	return m.value.(float64), m.variant, m.reason, m.metadata, m.err
}

func (m MockEvaluator) ResolveObjectValue(ctx context.Context, reqID string, flagKey string, context map[string]any) (value map[string]any, variant string, reason string, metadata model.Metadata, err error) {
	return m.value.(map[string]any), m.variant, m.reason, m.metadata, m.err
}

func (m MockEvaluator) GetState() (string, error) {
	// ignored
	return "", nil
}

func (m MockEvaluator) SetState(payload sync.DataSync) error {
	// ignored
	return nil
}

func (m MockEvaluator) ResolveAllValues(ctx context.Context, reqID string, context map[string]any) ([]evaluator.AnyValue, model.Metadata, error) {
	// ignored
	return nil, nil, nil
}

func (m MockEvaluator) ResolveAsAnyValue(ctx context.Context, reqID string, flagKey string, context map[string]any) evaluator.AnyValue {
	// ignored
	return evaluator.AnyValue{}
}
