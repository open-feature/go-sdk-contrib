package from_env_test

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	fromEnv "github.com/open-feature/golang-sdk-contrib/providers/from-env/pkg"
	"github.com/open-feature/golang-sdk/pkg/openfeature"
)

// this line will fail linting if this provider is no longer compatible with the openfeature sdk
var _ openfeature.FeatureProvider = &fromEnv.FromEnvProvider{}

func TestBoolFromEnv(t *testing.T) {
	tests := map[string]struct {
		flagKey           string
		defaultValue      bool
		expectedValue     bool
		expectedReason    string
		expectedVariant   string
		expectedErrorCode string
		EvaluationContext map[string]interface{}
		flagValue         fromEnv.StoredFlag
	}{
		"bool happy path": {
			flagKey:           "MY_BOOL_FLAG",
			defaultValue:      false,
			expectedValue:     true,
			expectedReason:    openfeature.TARGETING_MATCH,
			expectedVariant:   "yellow",
			expectedErrorCode: "",
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
			flagKey:           "MY_BOOL_FLAG",
			defaultValue:      true,
			expectedValue:     true,
			expectedReason:    openfeature.ERROR,
			expectedVariant:   "",
			expectedErrorCode: fromEnv.ErrorTypeMismatch,
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
			flagKey:           "MY_BOOL_FLAG",
			defaultValue:      true,
			expectedValue:     true,
			expectedReason:    openfeature.ERROR,
			expectedVariant:   "",
			expectedErrorCode: fromEnv.ErrorParse,
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
			flagKey:           "MY_BOOL_FLAG",
			defaultValue:      false,
			expectedValue:     true,
			expectedReason:    fromEnv.ReasonStatic,
			expectedVariant:   "default",
			expectedErrorCode: "",
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
			flagKey:           "MY_BOOL_FLAG",
			defaultValue:      true,
			expectedValue:     true,
			expectedReason:    openfeature.TARGETING_MATCH,
			expectedVariant:   "targeting_key",
			expectedErrorCode: "",
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
			res := p.BooleanEvaluation(test.flagKey, test.defaultValue, test.EvaluationContext)
			if res.Value != test.expectedValue {
				t.Errorf("unexpected Value received, expected %v, got %v", test.expectedValue, res.Value)
				t.FailNow()
			}
			if res.Reason != test.expectedReason {
				t.Errorf("unexpected Reason received, expected %v, got %v", test.expectedReason, res.Reason)
				t.FailNow()
			}
			if res.Variant != test.expectedVariant {
				t.Errorf("unexpected Variant received, expected %v, got %v", test.expectedVariant, res.Variant)
				t.FailNow()
			}
			if res.ErrorCode != test.expectedErrorCode {
				t.Errorf("unexpected Error received, expected %v, got %v", test.expectedErrorCode, res.ErrorCode)
				t.FailNow()
			}
		})
	}
}

func TestStringFromEnv(t *testing.T) {
	tests := map[string]struct {
		flagKey           string
		defaultValue      string
		expectedValue     string
		expectedReason    string
		expectedVariant   string
		expectedErrorCode string
		EvaluationContext map[string]interface{}
		flagValue         fromEnv.StoredFlag
	}{
		"string happy path": {
			flagKey:           "MY_STRING_FLAG",
			defaultValue:      "default value",
			expectedValue:     "yellow",
			expectedReason:    openfeature.TARGETING_MATCH,
			expectedVariant:   "yellow",
			expectedErrorCode: "",
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
			flagKey:           "MY_STRING_FLAG",
			defaultValue:      "default value",
			expectedValue:     "default value",
			expectedReason:    openfeature.ERROR,
			expectedVariant:   "",
			expectedErrorCode: fromEnv.ErrorTypeMismatch,
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
			res := p.StringEvaluation(test.flagKey, test.defaultValue, test.EvaluationContext)
			if res.Value != test.expectedValue {
				t.Errorf("unexpected Value received, expected %v, got %v", test.expectedValue, res.Value)
				t.FailNow()
			}
			if res.Reason != test.expectedReason {
				t.Errorf("unexpected Reason received, expected %v, got %v", test.expectedReason, res.Reason)
				t.FailNow()
			}
			if res.Variant != test.expectedVariant {
				t.Errorf("unexpected Variant received, expected %v, got %v", test.expectedVariant, res.Variant)
				t.FailNow()
			}
			if res.ErrorCode != test.expectedErrorCode {
				t.Errorf("unexpected Error received, expected %v, got %v", test.expectedErrorCode, res.ErrorCode)
				t.FailNow()
			}
		})
	}
}

func TestFloatFromEnv(t *testing.T) {
	tests := map[string]struct {
		flagKey           string
		defaultValue      float64
		expectedValue     float64
		expectedReason    string
		expectedVariant   string
		expectedErrorCode string
		EvaluationContext map[string]interface{}
		flagValue         fromEnv.StoredFlag
	}{
		"string happy path": {
			flagKey:           "MY_FLOAT_FLAG",
			defaultValue:      1,
			expectedValue:     10,
			expectedReason:    openfeature.TARGETING_MATCH,
			expectedVariant:   "yellow",
			expectedErrorCode: "",
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
			flagKey:           "MY_FLOAT_FLAG",
			defaultValue:      1,
			expectedValue:     1,
			expectedReason:    openfeature.ERROR,
			expectedVariant:   "",
			expectedErrorCode: fromEnv.ErrorTypeMismatch,
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
			res := p.FloatEvaluation(test.flagKey, test.defaultValue, test.EvaluationContext)
			if res.Value != test.expectedValue {
				t.Errorf("unexpected Value received, expected %v, got %v", test.expectedValue, res.Value)
				t.FailNow()
			}
			if res.Reason != test.expectedReason {
				t.Errorf("unexpected Reason received, expected %v, got %v", test.expectedReason, res.Reason)
				t.FailNow()
			}
			if res.Variant != test.expectedVariant {
				t.Errorf("unexpected Variant received, expected %v, got %v", test.expectedVariant, res.Variant)
				t.FailNow()
			}
			if res.ErrorCode != test.expectedErrorCode {
				t.Errorf("unexpected Error received, expected %v, got %v", test.expectedErrorCode, res.ErrorCode)
				t.FailNow()
			}
		})
	}
}

func TestIntFromEnv(t *testing.T) {
	tests := map[string]struct {
		flagKey           string
		defaultValue      int64
		expectedValue     int64
		expectedReason    string
		expectedVariant   string
		expectedErrorCode string
		EvaluationContext map[string]interface{}
		flagValue         fromEnv.StoredFlag
	}{
		"int happy path": {
			flagKey:           "MY_INT_FLAG",
			defaultValue:      1,
			expectedValue:     10,
			expectedReason:    openfeature.TARGETING_MATCH,
			expectedVariant:   "yellow",
			expectedErrorCode: "",
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
			flagKey:           "MY_INT_FLAG",
			defaultValue:      1,
			expectedValue:     1,
			expectedReason:    openfeature.ERROR,
			expectedVariant:   "",
			expectedErrorCode: fromEnv.ErrorTypeMismatch,
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
			res := p.IntEvaluation(test.flagKey, test.defaultValue, test.EvaluationContext)
			fmt.Println(res)
			if res.Value != test.expectedValue {
				t.Errorf("unexpected Value received, expected %v, got %v", test.expectedValue, res.Value)
				t.FailNow()
			}
			if res.Reason != test.expectedReason {
				t.Errorf("unexpected Reason received, expected %v, got %v", test.expectedReason, res.Reason)
				t.FailNow()
			}
			if res.Variant != test.expectedVariant {
				t.Errorf("unexpected Variant received, expected %v, got %v", test.expectedVariant, res.Variant)
				t.FailNow()
			}
			if res.ErrorCode != test.expectedErrorCode {
				t.Errorf("unexpected Error received, expected %v, got %v", test.expectedErrorCode, res.ErrorCode)
				t.FailNow()
			}
		})
	}
}

func TestObjectFromEnv(t *testing.T) {
	tests := map[string]struct {
		flagKey           string
		defaultValue      interface{}
		expectedValue     interface{}
		expectedReason    string
		expectedVariant   string
		expectedErrorCode string
		EvaluationContext map[string]interface{}
		flagValue         fromEnv.StoredFlag
	}{
		"object happy path": {
			flagKey: "MY_OBJECT_FLAG",
			defaultValue: map[string]interface{}{
				"key": "value",
			},
			expectedValue: map[string]interface{}{
				"key": "value2",
			},
			expectedReason:    openfeature.TARGETING_MATCH,
			expectedVariant:   "yellow",
			expectedErrorCode: "",
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
			res := p.ObjectEvaluation(test.flagKey, test.defaultValue, test.EvaluationContext)
			if !reflect.DeepEqual(res.Value, test.expectedValue) {
				t.Errorf("unexpected Value received, expected %v, got %v", test.expectedValue, res.Value)
				t.FailNow()
			}
			if res.Reason != test.expectedReason {
				t.Errorf("unexpected Reason received, expected %v, got %v", test.expectedReason, res.Reason)
				t.FailNow()
			}
			if res.Variant != test.expectedVariant {
				t.Errorf("unexpected Variant received, expected %v, got %v", test.expectedVariant, res.Variant)
				t.FailNow()
			}
			if res.ErrorCode != test.expectedErrorCode {
				t.Errorf("unexpected Error received, expected %v, got %v", test.expectedErrorCode, res.ErrorCode)
				t.FailNow()
			}
		})
	}
}
