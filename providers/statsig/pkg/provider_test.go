package statsig_test

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	statsigProvider "github.com/open-feature/go-sdk-contrib/providers/statsig/pkg"
	of "github.com/open-feature/go-sdk/openfeature"
	statsig "github.com/statsig-io/go-sdk"
)

var provider *statsigProvider.Provider

func TestBooleanEvaluation(t *testing.T) {
	err := of.SetProvider(provider)
	if err != nil {
		t.Fatalf("error setting provider %s", err)
	}
	ofClient := of.NewClient("my-app")

	evalCtx := of.NewEvaluationContext(
		"",
		map[string]interface{}{
			"UserID": "123",
		},
	)
	enabled, _ := ofClient.BooleanValue(context.Background(), "always_on_gate", false, evalCtx)
	if enabled == false {
		t.Fatalf("Expected feature to be enabled")
	}
}

func TestStringConfigEvaluation(t *testing.T) {
	err := of.SetProvider(provider)
	if err != nil {
		t.Fatalf("error setting provider %s", err)
	}
	ofClient := of.NewClient("my-app")

	featureConfig := statsigProvider.FeatureConfig{
		FeatureConfigType: statsigProvider.FeatureConfigType("CONFIG"),
		Name:              "test_config",
	}

	evalCtx := of.NewEvaluationContext(
		"",
		map[string]interface{}{
			"UserID":         "123",
			"Email":          "testuser1@statsig.com",
			"feature_config": featureConfig,
		},
	)
	expected := "statsig"
	value, _ := ofClient.StringValue(context.Background(), "string", "fallback", evalCtx)
	if value != expected {
		t.Fatalf("Expected: %s, actual: %s", expected, value)
	}
}

func TestBoolLayerEvaluation(t *testing.T) {
	err := of.SetProvider(provider)
	if err != nil {
		t.Fatalf("error setting provider %s", err)
	}
	ofClient := of.NewClient("my-app")

	featureConfig := statsigProvider.FeatureConfig{
		FeatureConfigType: statsigProvider.FeatureConfigType("LAYER"),
		Name:              "b_layer_no_alloc",
	}

	evalCtx := of.NewEvaluationContext(
		"",
		map[string]interface{}{
			"UserID":         "123",
			"feature_config": featureConfig,
		},
	)
	expected := "layer_default"
	value, _ := ofClient.StringValue(context.Background(), "b_param", "fallback", evalCtx)
	if value != expected {
		t.Fatalf("Expected: %s, actual: %s", expected, value)
	}
}

func cleanup() {
	provider.Shutdown()
}

func TestEvaluationMethods(t *testing.T) {
	err := of.SetProvider(provider)
	if err != nil {
		t.Fatalf("error setting provider %s", err)
	}

	tests := []struct {
		flag          string
		defaultValue  interface{}
		evalCtx       of.FlattenedContext
		expected      interface{}
		expectedError string
	}{
		{"always_on_gate", false, of.FlattenedContext{"UserID": "123"}, true, ""},

		{"boolean", false, of.FlattenedContext{"UserID": "123", "feature_config": statsigProvider.FeatureConfig{FeatureConfigType: "CONFIG", Name: "valid_flag"}}, true, ""},
		{"float", 1.5999, of.FlattenedContext{"UserID": "123", "feature_config": statsigProvider.FeatureConfig{FeatureConfigType: "CONFIG", Name: "valid_flag"}}, 1.5, ""},
		{"number", int64(42999), of.FlattenedContext{"UserID": "123", "feature_config": statsigProvider.FeatureConfig{FeatureConfigType: "CONFIG", Name: "valid_flag"}}, int64(42), ""},
		{"object", map[string]interface{}{"key": "value999"}, of.FlattenedContext{"UserID": "123", "feature_config": statsigProvider.FeatureConfig{FeatureConfigType: "CONFIG", Name: "valid_flag"}}, map[string]interface{}{"key1": "value1"}, ""},
		{"string", "default_value", of.FlattenedContext{"UserID": "123", "feature_config": statsigProvider.FeatureConfig{FeatureConfigType: "CONFIG", Name: "valid_flag"}}, "expected_value", ""},
		{"slice", []interface{}{"fallback1", "fallback2"}, of.FlattenedContext{"UserID": "123", "feature_config": statsigProvider.FeatureConfig{FeatureConfigType: "CONFIG", Name: "valid_flag"}}, []interface{}{"v1", "v2"}, ""},

		{"boolean", false, of.FlattenedContext{"UserID": "123", "feature_config": statsigProvider.FeatureConfig{FeatureConfigType: "LAYER", Name: "valid_layer"}}, true, ""},
		{"float", 1.5999, of.FlattenedContext{"UserID": "123", "feature_config": statsigProvider.FeatureConfig{FeatureConfigType: "LAYER", Name: "valid_layer"}}, 1.5, ""},
		{"number", int64(42999), of.FlattenedContext{"UserID": "123", "feature_config": statsigProvider.FeatureConfig{FeatureConfigType: "LAYER", Name: "valid_layer"}}, int64(42), ""},
		{"object", map[string]interface{}{"key": "value999"}, of.FlattenedContext{"UserID": "123", "feature_config": statsigProvider.FeatureConfig{FeatureConfigType: "LAYER", Name: "valid_layer"}}, map[string]interface{}{"key1": "value1"}, ""},
		{"string", "default_value", of.FlattenedContext{"UserID": "123", "feature_config": statsigProvider.FeatureConfig{FeatureConfigType: "LAYER", Name: "valid_layer"}}, "expected_value", ""},
		{"slice", []interface{}{"fallback1", "fallback2"}, of.FlattenedContext{"UserID": "123", "feature_config": statsigProvider.FeatureConfig{FeatureConfigType: "LAYER", Name: "valid_layer"}}, []interface{}{"v1", "v2"}, ""},

		{"invalid_flag", false, of.FlattenedContext{"UserID": "123"}, false, "flag not found"},

		// expected to succeed when https://github.com/statsig-io/go-sdk/issues/32 is resolved and adopted
		// {"invalid_flag", true, of.FlattenedContext{"UserID": "123", "feature_config": statsigProvider.FeatureConfig{FeatureConfigType: "CONFIG", Name: "test"}}, true, "flag not found"},

		{"float", 1.23, of.FlattenedContext{"UserID": "123", "feature_config": statsigProvider.FeatureConfig{FeatureConfigType: "CONFIG", Name: "invalid_flag"}}, 1.23, "flag not found"},
		{"number", int64(43), of.FlattenedContext{"UserID": "123", "feature_config": statsigProvider.FeatureConfig{FeatureConfigType: "CONFIG", Name: "invalid_flag"}}, int64(43), "flag not found"},
		{"object", map[string]interface{}{"key1": "other-value"}, of.FlattenedContext{"UserID": "123", "feature_config": statsigProvider.FeatureConfig{FeatureConfigType: "CONFIG", Name: "invalid_flag"}}, map[string]interface{}{"key1": "other-value"}, "flag not found"},
		{"string", "value2", of.FlattenedContext{"UserID": "123", "feature_config": statsigProvider.FeatureConfig{FeatureConfigType: "CONFIG", Name: "invalid_flag"}}, "value2", "flag not found"},

		{"float", 1.23, of.FlattenedContext{"UserID": "123", "feature_config": statsigProvider.FeatureConfig{FeatureConfigType: "LAYER", Name: "invalid_flag"}}, 1.23, "flag not found"},
		{"number", int64(43), of.FlattenedContext{"UserID": "123", "feature_config": statsigProvider.FeatureConfig{FeatureConfigType: "LAYER", Name: "invalid_flag"}}, int64(43), "flag not found"},
		{"object", map[string]interface{}{"key1": "other-value"}, of.FlattenedContext{"UserID": "123", "feature_config": statsigProvider.FeatureConfig{FeatureConfigType: "LAYER", Name: "invalid_flag"}}, map[string]interface{}{"key1": "other-value"}, "flag not found"},
		{"string", "value2", of.FlattenedContext{"UserID": "123", "feature_config": statsigProvider.FeatureConfig{FeatureConfigType: "LAYER", Name: "invalid_flag"}}, "value2", "flag not found"},

		{"invalid_user_context", false, of.FlattenedContext{"UserID": "123", "invalid": "value"}, false, ""},
		{"enriched_user_context", false, of.FlattenedContext{"UserID": "123", "Email": "v", "IpAddress": "v", "UserAgent": "v", "Country": "v", "Locale": "v"}, false, ""},
		{"missing_feature_config", int64(43), of.FlattenedContext{"UserID": "123"}, int64(43), ""},
		{"empty_context", int64(43), of.FlattenedContext{}, int64(43), ""},
	}

	for _, test := range tests {

		fmt.Println("test: {}", test)

		rt := reflect.TypeOf(test.expected)
		switch rt.Kind() {
		case reflect.Bool:
			res := provider.BooleanEvaluation(context.Background(), test.flag, test.defaultValue.(bool), test.evalCtx)
			require.Equal(t, test.expected, res.Value)
		case reflect.Int, reflect.Int8, reflect.Int32, reflect.Int64:
			res := provider.IntEvaluation(context.Background(), test.flag, test.defaultValue.(int64), test.evalCtx)
			require.Equal(t, test.expected, res.Value)
		case reflect.Float32, reflect.Float64:
			res := provider.FloatEvaluation(context.Background(), test.flag, test.defaultValue.(float64), test.evalCtx)
			require.Equal(t, test.expected, res.Value)
		case reflect.String:
			res := provider.StringEvaluation(context.Background(), test.flag, test.defaultValue.(string), test.evalCtx)
			require.Equal(t, test.expected, res.Value)
		default:
			res := provider.ObjectEvaluation(context.Background(), test.flag, test.defaultValue, test.evalCtx)
			require.Equal(t, test.expected, res.Value)
		}
	}
}

func TestMain(m *testing.M) {

	bytes, err := os.ReadFile("download_config_specs.json")
	if err != nil {
		os.Exit(1)
	}

	providerOptions := statsigProvider.ProviderConfig{
		Options: statsig.Options{
			BootstrapValues: string(bytes[:]),
		},
		SdkKey: "secret-key",
	}

	provider, err = statsigProvider.NewProvider(providerOptions)
	if err != nil {
		fmt.Printf("Error during new provider: %v\n", err)
	}
	provider.Init(of.EvaluationContext{})
	fmt.Printf("provider: %v\n", provider)

	// Run the tests
	exitCode := m.Run()

	cleanup()

	os.Exit(exitCode)
}
