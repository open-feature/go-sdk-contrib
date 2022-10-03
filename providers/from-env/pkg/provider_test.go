package from_env_test

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	fromEnv "github.com/open-feature/go-sdk-contrib/providers/from-env/pkg"
	"github.com/open-feature/go-sdk/pkg/openfeature"
)

// this line will fail linting if this provider is no longer compatible with the openfeature sdk
var _ openfeature.FeatureProvider = &fromEnv.FromEnvProvider{}

func TestBoolFromEnv(t *testing.T) {
	tests := map[string]struct {
		flagKey                 string
		defaultValue            bool
		expectedValue           bool
		expectedReason          openfeature.Reason
		expectedVariant         string
		expectedResolutionError openfeature.ResolutionError
		EvaluationContext       map[string]interface{}
		flagValue               fromEnv.StoredFlag
	}{
		"bool happy path": {
			flagKey:                 "MY_BOOL_FLAG",
			defaultValue:            false,
			expectedValue:           true,
			expectedReason:          openfeature.TargetingMatchReason,
			expectedVariant:         "yellow",
			expectedResolutionError: openfeature.ResolutionError{},
			EvaluationContext: map[string]interface{}{
				"color":                  "yellow",
				openfeature.TargetingKey: "user1"},
			flagValue: fromEnv.StoredFlag{
				DefaultVariant: "not-yellow",
				Variants: []fromEnv.Variant{
					{
						Name:         "yellow-with-extras",
						TargetingKey: "",
						Value:        false,
						Criteria: []fromEnv.Criteria{
							{
								Key:   "color-extra",
								Value: "blue",
							},
							{
								Key:   "color",
								Value: "yellow",
							},
						},
					},
					{
						Name:         "yellow",
						TargetingKey: "",
						Value:        true,
						Criteria: []fromEnv.Criteria{
							{
								Key:   "color",
								Value: "yellow",
							},
						},
					},
					{
						Name:         "not-yellow",
						TargetingKey: "",
						Value:        false,
						Criteria: []fromEnv.Criteria{
							{
								Key:   "color",
								Value: "not yellow",
							},
						},
					},
				},
			},
		},
		"flag is not bool": {
			flagKey:                 "MY_BOOL_FLAG",
			defaultValue:            true,
			expectedValue:           true,
			expectedReason:          openfeature.ErrorReason,
			expectedVariant:         "",
			expectedResolutionError: openfeature.NewTypeMismatchResolutionError(""),
			EvaluationContext: map[string]interface{}{
				"color": "yellow",
			},
			flagValue: fromEnv.StoredFlag{
				DefaultVariant: "default",
				Variants: []fromEnv.Variant{
					{
						Name:         "default",
						TargetingKey: "",
						Value:        "false",
						Criteria:     []fromEnv.Criteria{},
					},
				},
			},
		},
		"variant does not exist": {
			flagKey:                 "MY_BOOL_FLAG",
			defaultValue:            true,
			expectedValue:           true,
			expectedReason:          openfeature.ErrorReason,
			expectedVariant:         "",
			expectedResolutionError: openfeature.NewParseErrorResolutionError(""),
			EvaluationContext: map[string]interface{}{
				"color": "yellow",
			},
			flagValue: fromEnv.StoredFlag{
				DefaultVariant: "not-default",
				Variants: []fromEnv.Variant{
					{
						Name:         "default",
						TargetingKey: "",
						Value:        false,
						Criteria: []fromEnv.Criteria{
							{
								Key:   "color",
								Value: "not yellow",
							},
						},
					},
				},
			},
		},
		"hit default value": {
			flagKey:                 "MY_BOOL_FLAG",
			defaultValue:            false,
			expectedValue:           true,
			expectedReason:          openfeature.DefaultReason,
			expectedVariant:         "default",
			expectedResolutionError: openfeature.ResolutionError{},
			EvaluationContext: map[string]interface{}{
				"color": "yellow",
			},
			flagValue: fromEnv.StoredFlag{
				DefaultVariant: "default",
				Variants: []fromEnv.Variant{
					{
						Name:         "default",
						TargetingKey: "",
						Value:        true,
						Criteria: []fromEnv.Criteria{
							{
								Key:   "color",
								Value: "not yellow",
							},
						},
					},
				},
			},
		},
		"targeting key match": {
			flagKey:                 "MY_BOOL_FLAG",
			defaultValue:            true,
			expectedValue:           true,
			expectedReason:          openfeature.TargetingMatchReason,
			expectedVariant:         "targeting_key",
			expectedResolutionError: openfeature.ResolutionError{},
			EvaluationContext: map[string]interface{}{
				"color":                  "yellow",
				openfeature.TargetingKey: "user1",
			},
			flagValue: fromEnv.StoredFlag{
				DefaultVariant: "default",
				Variants: []fromEnv.Variant{
					{
						Name:         "targeting_key_2",
						TargetingKey: "user2",
						Value:        true,
						Criteria: []fromEnv.Criteria{
							{
								Key:   "color",
								Value: "yellow",
							},
						},
					},
					{
						Name:         "targeting_key",
						TargetingKey: "user1",
						Value:        true,
						Criteria: []fromEnv.Criteria{
							{
								Key:   "color",
								Value: "yellow",
							},
						},
					},
					{
						Name:         "default",
						TargetingKey: "",
						Value:        false,
						Criteria: []fromEnv.Criteria{
							{
								Key:   "color",
								Value: "not yellow",
							},
						},
					},
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			p := fromEnv.FromEnvProvider{}
			flagM, _ := json.Marshal(test.flagValue)
			t.Setenv(test.flagKey, string(flagM))
			res := p.BooleanEvaluation(context.Background(), test.flagKey, test.defaultValue, test.EvaluationContext)
			if res.Value != test.expectedValue {
				t.Fatalf("unexpected Value received, expected %v, got %v", test.expectedValue, res.Value)
			}
			if res.Reason != test.expectedReason {
				t.Fatalf("unexpected Reason received, expected %v, got %v", test.expectedReason, res.Reason)
			}
			if res.Variant != test.expectedVariant {
				t.Fatalf("unexpected Variant received, expected %v, got %v", test.expectedVariant, res.Variant)
			}
			if res.ResolutionError.Error() != test.expectedResolutionError.Error() {
				t.Fatalf(
					"unexpected ResolutionError received, expected %v, got %v", test.expectedResolutionError, res.ResolutionError,
				)
			}
		})
	}
}

func TestStringFromEnv(t *testing.T) {
	tests := map[string]struct {
		flagKey                 string
		defaultValue            string
		expectedValue           string
		expectedReason          openfeature.Reason
		expectedVariant         string
		expectedResolutionError openfeature.ResolutionError
		EvaluationContext       map[string]interface{}
		flagValue               fromEnv.StoredFlag
	}{
		"string happy path": {
			flagKey:                 "MY_STRING_FLAG",
			defaultValue:            "default value",
			expectedValue:           "yellow",
			expectedReason:          openfeature.TargetingMatchReason,
			expectedVariant:         "yellow",
			expectedResolutionError: openfeature.ResolutionError{},
			EvaluationContext: map[string]interface{}{
				"color": "yellow",
			},
			flagValue: fromEnv.StoredFlag{
				DefaultVariant: "not-yellow",
				Variants: []fromEnv.Variant{
					{
						Name:         "yellow-with-extras",
						TargetingKey: "",
						Value:        "not yellow",
						Criteria: []fromEnv.Criteria{
							{
								Key:   "color-extra",
								Value: "blue",
							},
							{
								Key:   "color",
								Value: "yellow",
							},
						},
					},
					{
						Name:         "yellow",
						TargetingKey: "",
						Value:        "yellow",
						Criteria: []fromEnv.Criteria{
							{
								Key:   "color",
								Value: "yellow",
							},
						},
					},
					{
						Name:         "not-yellow",
						TargetingKey: "",
						Value:        "not yellow",
						Criteria: []fromEnv.Criteria{
							{
								Key:   "color",
								Value: "not yellow",
							},
						},
					},
				},
			},
		},
		"flag is not string": {
			flagKey:                 "MY_STRING_FLAG",
			defaultValue:            "default value",
			expectedValue:           "default value",
			expectedReason:          openfeature.ErrorReason,
			expectedVariant:         "",
			expectedResolutionError: openfeature.NewTypeMismatchResolutionError(""),
			EvaluationContext: map[string]interface{}{
				"color": "yellow",
			},
			flagValue: fromEnv.StoredFlag{
				DefaultVariant: "default",
				Variants: []fromEnv.Variant{
					{
						Name:         "default",
						TargetingKey: "",
						Value:        true,
						Criteria:     []fromEnv.Criteria{},
					},
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			p := fromEnv.FromEnvProvider{}
			flagM, _ := json.Marshal(test.flagValue)
			t.Setenv(test.flagKey, string(flagM))
			res := p.StringEvaluation(context.Background(), test.flagKey, test.defaultValue, test.EvaluationContext)
			if res.Value != test.expectedValue {
				t.Fatalf("unexpected Value received, expected %v, got %v", test.expectedValue, res.Value)
			}
			if res.Reason != test.expectedReason {
				t.Fatalf("unexpected Reason received, expected %v, got %v", test.expectedReason, res.Reason)
			}
			if res.Variant != test.expectedVariant {
				t.Fatalf("unexpected Variant received, expected %v, got %v", test.expectedVariant, res.Variant)
			}
			if res.ResolutionError.Error() != test.expectedResolutionError.Error() {
				t.Fatalf(
					"unexpected ResolutionError received, expected %v, got %v",
					test.expectedResolutionError.Error(), res.ResolutionError.Error(),
				)
			}
		})
	}
}

func TestFloatFromEnv(t *testing.T) {
	tests := map[string]struct {
		flagKey                 string
		defaultValue            float64
		expectedValue           float64
		expectedReason          openfeature.Reason
		expectedVariant         string
		expectedResolutionError openfeature.ResolutionError
		EvaluationContext       map[string]interface{}
		flagValue               fromEnv.StoredFlag
	}{
		"string happy path": {
			flagKey:                 "MY_FLOAT_FLAG",
			defaultValue:            1,
			expectedValue:           10,
			expectedReason:          openfeature.TargetingMatchReason,
			expectedVariant:         "yellow",
			expectedResolutionError: openfeature.ResolutionError{},
			EvaluationContext: map[string]interface{}{
				"color": "yellow",
			},
			flagValue: fromEnv.StoredFlag{
				DefaultVariant: "not-yellow",
				Variants: []fromEnv.Variant{
					{
						Name:         "yellow-with-extras",
						TargetingKey: "",
						Value:        100,
						Criteria: []fromEnv.Criteria{
							{
								Key:   "color-extra",
								Value: "blue",
							},
							{
								Key:   "color",
								Value: "yellow",
							},
						},
					},
					{
						Name:         "yellow",
						TargetingKey: "",
						Value:        10,
						Criteria: []fromEnv.Criteria{
							{
								Key:   "color",
								Value: "yellow",
							},
						},
					},
					{
						Name:         "not-yellow",
						TargetingKey: "",
						Value:        100,
						Criteria: []fromEnv.Criteria{
							{
								Key:   "color",
								Value: "not yellow",
							},
						},
					},
				},
			},
		},
		"flag is not float64": {
			flagKey:                 "MY_FLOAT_FLAG",
			defaultValue:            1,
			expectedValue:           1,
			expectedReason:          openfeature.ErrorReason,
			expectedVariant:         "",
			expectedResolutionError: openfeature.NewTypeMismatchResolutionError(""),
			EvaluationContext: map[string]interface{}{
				"color": "yellow",
			},
			flagValue: fromEnv.StoredFlag{
				DefaultVariant: "default",
				Variants: []fromEnv.Variant{
					{
						Name:         "default",
						TargetingKey: "",
						Value:        "10",
						Criteria:     []fromEnv.Criteria{},
					},
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			p := fromEnv.FromEnvProvider{}
			flagM, _ := json.Marshal(test.flagValue)
			t.Setenv(test.flagKey, string(flagM))
			res := p.FloatEvaluation(context.Background(), test.flagKey, test.defaultValue, test.EvaluationContext)
			if res.Value != test.expectedValue {
				t.Fatalf("unexpected Value received, expected %v, got %v", test.expectedValue, res.Value)
			}
			if res.Reason != test.expectedReason {
				t.Fatalf("unexpected Reason received, expected %v, got %v", test.expectedReason, res.Reason)
			}
			if res.Variant != test.expectedVariant {
				t.Fatalf("unexpected Variant received, expected %v, got %v", test.expectedVariant, res.Variant)
			}
			if res.ResolutionError.Error() != test.expectedResolutionError.Error() {
				t.Fatalf(
					"unexpected Error received, expected %v, got %v",
					test.expectedResolutionError.Error(), res.ResolutionError.Error(),
				)
			}
		})
	}
}

func TestIntFromEnv(t *testing.T) {
	tests := map[string]struct {
		flagKey                 string
		defaultValue            int64
		expectedValue           int64
		expectedReason          openfeature.Reason
		expectedVariant         string
		expectedResolutionError openfeature.ResolutionError
		EvaluationContext       map[string]interface{}
		flagValue               fromEnv.StoredFlag
	}{
		"int happy path": {
			flagKey:                 "MY_INT_FLAG",
			defaultValue:            1,
			expectedValue:           10,
			expectedReason:          openfeature.TargetingMatchReason,
			expectedVariant:         "yellow",
			expectedResolutionError: openfeature.ResolutionError{},
			EvaluationContext: map[string]interface{}{
				"color": "yellow",
			},
			flagValue: fromEnv.StoredFlag{
				DefaultVariant: "not-yellow",
				Variants: []fromEnv.Variant{
					{
						Name:         "yellow-with-extras",
						TargetingKey: "",
						Value:        100,
						Criteria: []fromEnv.Criteria{
							{
								Key:   "color-extra",
								Value: "blue",
							},
							{
								Key:   "color",
								Value: "yellow",
							},
						},
					},
					{
						Name:         "yellow",
						TargetingKey: "",
						Value:        10,
						Criteria: []fromEnv.Criteria{
							{
								Key:   "color",
								Value: "yellow",
							},
						},
					},
					{
						Name:         "not-yellow",
						TargetingKey: "",
						Value:        100,
						Criteria: []fromEnv.Criteria{
							{
								Key:   "color",
								Value: "not yellow",
							},
						},
					},
				},
			},
		},
		"flag is not int64": {
			flagKey:                 "MY_INT_FLAG",
			defaultValue:            1,
			expectedValue:           1,
			expectedReason:          openfeature.ErrorReason,
			expectedVariant:         "",
			expectedResolutionError: openfeature.NewTypeMismatchResolutionError(""),
			EvaluationContext: map[string]interface{}{
				"color": "yellow",
			},
			flagValue: fromEnv.StoredFlag{
				DefaultVariant: "default",
				Variants: []fromEnv.Variant{
					{
						Name:         "default",
						TargetingKey: "",
						Value:        "10",
						Criteria:     []fromEnv.Criteria{},
					},
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			p := fromEnv.FromEnvProvider{}
			flagM, _ := json.Marshal(test.flagValue)
			t.Setenv(test.flagKey, string(flagM))
			res := p.IntEvaluation(context.Background(), test.flagKey, test.defaultValue, test.EvaluationContext)
			if res.Value != test.expectedValue {
				t.Fatalf("unexpected Value received, expected %v, got %v", test.expectedValue, res.Value)
			}
			if res.Reason != test.expectedReason {
				t.Fatalf("unexpected Reason received, expected %v, got %v", test.expectedReason, res.Reason)
			}
			if res.Variant != test.expectedVariant {
				t.Fatalf("unexpected Variant received, expected %v, got %v", test.expectedVariant, res.Variant)
			}
			if res.ResolutionError.Error() != test.expectedResolutionError.Error() {
				t.Fatalf(
					"unexpected ResolutionError received, expected %v, got %v",
					test.expectedResolutionError.Error(), res.ResolutionError.Error(),
				)
			}
		})
	}
}

func TestObjectFromEnv(t *testing.T) {
	tests := map[string]struct {
		flagKey                 string
		defaultValue            interface{}
		expectedValue           interface{}
		expectedReason          openfeature.Reason
		expectedVariant         string
		expectedResolutionError openfeature.ResolutionError
		EvaluationContext       map[string]interface{}
		flagValue               fromEnv.StoredFlag
	}{
		"object happy path": {
			flagKey: "MY_OBJECT_FLAG",
			defaultValue: map[string]interface{}{
				"key": "value",
			},
			expectedValue: map[string]interface{}{
				"key": "value2",
			},
			expectedReason:          openfeature.TargetingMatchReason,
			expectedVariant:         "yellow",
			expectedResolutionError: openfeature.ResolutionError{},
			EvaluationContext: map[string]interface{}{
				"color": "yellow",
			},
			flagValue: fromEnv.StoredFlag{
				DefaultVariant: "not-yellow",
				Variants: []fromEnv.Variant{
					{
						Name:         "yellow-with-extras",
						TargetingKey: "",
						Value: map[string]interface{}{
							"key": "value3",
						},
						Criteria: []fromEnv.Criteria{
							{
								Key:   "color-extra",
								Value: "blue",
							},
							{
								Key:   "color",
								Value: "yellow",
							},
						},
					},
					{
						Name:         "yellow",
						TargetingKey: "",
						Value: map[string]interface{}{
							"key": "value2",
						},
						Criteria: []fromEnv.Criteria{
							{
								Key:   "color",
								Value: "yellow",
							},
						},
					},
					{
						Name:         "not-yellow",
						TargetingKey: "",
						Value:        100,
						Criteria: []fromEnv.Criteria{
							{
								Key:   "color",
								Value: "not yellow",
							},
						},
					},
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			p := fromEnv.FromEnvProvider{}
			flagM, _ := json.Marshal(test.flagValue)
			t.Setenv(test.flagKey, string(flagM))
			res := p.ObjectEvaluation(context.Background(), test.flagKey, test.defaultValue, test.EvaluationContext)
			if !reflect.DeepEqual(res.Value, test.expectedValue) {
				t.Fatalf("unexpected Value received, expected %v, got %v", test.expectedValue, res.Value)
			}
			if res.Reason != test.expectedReason {
				t.Fatalf("unexpected Reason received, expected %v, got %v", test.expectedReason, res.Reason)
			}
			if res.Variant != test.expectedVariant {
				t.Fatalf("unexpected Variant received, expected %v, got %v", test.expectedVariant, res.Variant)
			}
			if res.ResolutionError.Error() != test.expectedResolutionError.Error() {
				t.Fatalf(
					"unexpected ResolutionError received, expected %v, got %v",
					test.expectedResolutionError.Error(), res.ResolutionError.Error(),
				)
			}
		})
	}
}
