package evaluate

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	of "github.com/open-feature/go-sdk/openfeature"
)

type mockResolver struct {
	success *successDto
	err     *of.ResolutionError
}

func (m mockResolver) resolveSingle(ctx context.Context, key string, evalCtx map[string]interface{}) (*successDto, *of.ResolutionError) {
	return m.success, m.err
}

type knownTypes interface {
	int64 | bool | float64 | string | interface{}
}

type testDefinition[T knownTypes] struct {
	name         string
	resolver     mockResolver
	isError      bool
	defaultValue T
	expect       T
}

var successBoolean = successDto{
	Value:    true,
	Reason:   string(of.StaticReason),
	Variant:  "true",
	Metadata: nil,
}

var successInt = successDto{
	Value:    10,
	Reason:   string(of.StaticReason),
	Variant:  "10",
	Metadata: nil,
}

var successInt64 = successDto{
	Value:    int64(10),
	Reason:   string(of.StaticReason),
	Variant:  "10",
	Metadata: nil,
}

var successFloat = successDto{
	Value:    float32(1.10),
	Reason:   string(of.StaticReason),
	Variant:  "1.10",
	Metadata: nil,
}

var successFloat64 = successDto{
	Value:    float64(1.10),
	Reason:   string(of.StaticReason),
	Variant:  "1.10",
	Metadata: nil,
}

var successString = successDto{
	Value:    "pass",
	Reason:   string(of.StaticReason),
	Variant:  "pass",
	Metadata: nil,
}

var successObject = successDto{
	Value: map[string]string{
		"key": "value",
	},
	Reason:   string(of.StaticReason),
	Variant:  "pass",
	Metadata: nil,
}

var parseError = of.NewParseErrorResolutionError("flag parsing error")

func TestBooleanEvaluation(t *testing.T) {
	ctx := context.Background()

	tests := []testDefinition[bool]{
		{
			name: "Success evaluation",
			resolver: mockResolver{
				success: &successBoolean,
			},
			defaultValue: false,
			expect:       successBoolean.Value.(bool),
		},
		{
			name: "Error evaluation",
			resolver: mockResolver{
				err: &parseError,
			},
			isError:      true,
			defaultValue: false,
			expect:       false,
		},
		{
			name:    "Type conversion error",
			isError: true,
			resolver: mockResolver{
				success: &successInt,
			},
			defaultValue: false,
			expect:       false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			flags := Flags{resolver: test.resolver}
			resolutionDetail := flags.ResolveBoolean(ctx, "booleanFlag", test.defaultValue, nil)
			genericValidator[bool](test, resolutionDetail.Value, resolutionDetail.Reason, resolutionDetail.Error(), t)
		})
	}
}

func TestIntegerEvaluation(t *testing.T) {
	ctx := context.Background()

	tests := []testDefinition[int64]{
		{
			name: "Success evaluation",
			resolver: mockResolver{
				success: &successInt,
			},
			defaultValue: 1,
			expect:       int64(successInt.Value.(int)),
		},
		{
			name: "Success evaluation - int64",
			resolver: mockResolver{
				success: &successInt64,
			},
			defaultValue: 1,
			expect:       successInt64.Value.(int64),
		},
		{
			name: "Error evaluation",
			resolver: mockResolver{
				err: &parseError,
			},
			isError:      true,
			defaultValue: 1,
			expect:       1,
		},
		{
			name: "Type conversion error",
			resolver: mockResolver{
				success: &successBoolean,
			},
			isError:      true,
			defaultValue: 1,
			expect:       1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			flags := Flags{resolver: test.resolver}
			resolutionDetail := flags.ResolveInt(ctx, "booleanFlag", test.defaultValue, nil)
			genericValidator[int64](test, resolutionDetail.Value, resolutionDetail.Reason, resolutionDetail.Error(), t)
		})
	}
}

func TestFloatEvaluation(t *testing.T) {
	ctx := context.Background()

	tests := []testDefinition[float64]{
		{
			name: "Success evaluation",
			resolver: mockResolver{
				success: &successFloat,
			},
			defaultValue: 1.05,
			expect:       float64(successFloat.Value.(float32)),
		},
		{
			name: "Success evaluation - float64",
			resolver: mockResolver{
				success: &successFloat64,
			},
			defaultValue: 1.05,
			expect:       successFloat64.Value.(float64),
		},
		{
			name: "Error evaluation",
			resolver: mockResolver{
				err: &parseError,
			},
			isError:      true,
			defaultValue: 1.05,
			expect:       1.05,
		},
		{
			name: "Type conversion error",
			resolver: mockResolver{
				success: &successBoolean,
			},
			isError:      true,
			defaultValue: 1.05,
			expect:       1.05,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			flags := Flags{resolver: test.resolver}
			resolutionDetail := flags.ResolveFloat(ctx, "booleanFlag", test.defaultValue, nil)
			genericValidator[float64](test, resolutionDetail.Value, resolutionDetail.Reason, resolutionDetail.Error(), t)
		})
	}
}

func TestStringEvaluation(t *testing.T) {
	ctx := context.Background()

	tests := []testDefinition[string]{
		{
			name: "Success evaluation",
			resolver: mockResolver{
				success: &successString,
			},
			defaultValue: "fail",
			expect:       successString.Value.(string),
		},
		{
			name: "Error evaluation",
			resolver: mockResolver{
				err: &parseError,
			},
			isError:      true,
			defaultValue: "fail",
			expect:       "fail",
		},
		{
			name: "Type conversion error",
			resolver: mockResolver{
				success: &successBoolean,
			},
			isError:      true,
			defaultValue: "fail",
			expect:       "fail",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			flags := Flags{resolver: test.resolver}
			resolutionDetail := flags.ResolveString(ctx, "booleanFlag", test.defaultValue, nil)
			genericValidator[string](test, resolutionDetail.Value, resolutionDetail.Reason, resolutionDetail.Error(), t)
		})
	}
}

func TestObjectEvaluation(t *testing.T) {
	ctx := context.Background()

	tests := []testDefinition[interface{}]{
		{
			name: "Success evaluation",
			resolver: mockResolver{
				success: &successObject,
			},
			defaultValue: map[string]interface{}{},
			expect:       successObject.Value,
		},
		{
			name: "Error evaluation",
			resolver: mockResolver{
				err: &parseError,
			},
			isError:      true,
			defaultValue: map[string]interface{}{},
			expect:       map[string]interface{}{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			flags := Flags{resolver: test.resolver}
			resolutionDetail := flags.ResolveObject(ctx, "booleanFlag", test.defaultValue, nil)
			genericValidator[interface{}](test, resolutionDetail.Value, resolutionDetail.Reason, resolutionDetail.Error(), t)
		})
	}
}

func genericValidator[T knownTypes](test testDefinition[T], resolvedValue T, reason of.Reason, err error, t *testing.T) {
	if test.isError {
		if err == nil {
			t.Error("expected error but got nil")
		}

		if !reflect.DeepEqual(test.defaultValue, resolvedValue) {
			t.Error(fmt.Sprintf("expected deafault value %v, but got %v", test.defaultValue, resolvedValue))
		}

		if reason != of.ErrorReason {
			t.Error(fmt.Sprintf("expected reason %v, but got %v", of.ErrorReason, reason))
		}
	} else {
		if err != nil {
			t.Fatal(fmt.Sprintf("expected no error, but got none nil error: %v", err))
		}

		if !reflect.DeepEqual(test.expect, resolvedValue) {
			t.Error(fmt.Sprintf("expected value %v, but got %v", test.expect, resolvedValue))
		}

	}

}
