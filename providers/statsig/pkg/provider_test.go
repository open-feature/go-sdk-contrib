package statsig_test

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	statsig "github.com/statsig-io/go-sdk"
	of "go.openfeature.dev/openfeature/v2"

	statsigProvider "go.openfeature.dev/contrib/providers/statsig/v2/pkg"
)

var provider *statsigProvider.Provider

func TestBooleanEvaluation(t *testing.T) {
	err := of.SetProviderAndWait(t.Context(), provider)
	if err != nil {
		t.Fatalf("error setting provider %s", err)
	}
	ofClient := of.NewClient(of.WithDomain("my-app"))

	evalCtx := of.NewEvaluationContext(
		"",
		map[string]any{
			"UserID": "123",
		},
	)
	enabled := ofClient.Boolean(t.Context(), "always_on_gate", false, evalCtx)
	if enabled == false {
		t.Fatalf("Expected feature to be enabled")
	}
}

func TestStringConfigEvaluation(t *testing.T) {
	err := of.SetProviderAndWait(t.Context(), provider)
	if err != nil {
		t.Fatalf("error setting provider %s", err)
	}
	ofClient := of.NewClient(of.WithDomain("my-app"))

	featureConfig := statsigProvider.FeatureConfig{
		FeatureConfigType: statsigProvider.FeatureConfigType("CONFIG"),
		Name:              "test_config",
	}

	evalCtx := of.NewEvaluationContext(
		"",
		map[string]any{
			"UserID":         "123",
			"Email":          "testuser1@statsig.com",
			"feature_config": featureConfig,
		},
	)
	expected := "statsig"
	value := ofClient.String(t.Context(), "string", "fallback", evalCtx)
	if value != expected {
		t.Fatalf("Expected: %s, actual: %s", expected, value)
	}
}

func TestBoolLayerEvaluation(t *testing.T) {
	err := of.SetProviderAndWait(t.Context(), provider)
	if err != nil {
		t.Fatalf("error setting provider %s", err)
	}
	ofClient := of.NewClient(of.WithDomain("my-app"))

	featureConfig := statsigProvider.FeatureConfig{
		FeatureConfigType: statsigProvider.FeatureConfigType("LAYER"),
		Name:              "b_layer_no_alloc",
	}

	evalCtx := of.NewEvaluationContext(
		"",
		map[string]any{
			"UserID":         "123",
			"feature_config": featureConfig,
		},
	)
	expected := "layer_default"
	value := ofClient.String(t.Context(), "b_param", "fallback", evalCtx)
	if value != expected {
		t.Fatalf("Expected: %s, actual: %s", expected, value)
	}
}

func TestConvertsValidUserIDToString(t *testing.T) {
	evalCtx := of.FlattenedContext{
		"UserID": "test_user",
	}

	user, err := statsigProvider.ToStatsigUser(evalCtx)
	assert.NoError(t, err)
	assert.Equal(t, "test_user", user.UserID)
}

// Converts valid EvaluationContext with all fields correctly to statsig.User
func TestConvertsValidEvaluationContextToStatsigUser(t *testing.T) {
	evalCtx := of.FlattenedContext{
		of.TargetingKey:      "test-key",
		"Email":              "user@example.com",
		"IpAddress":          "192.168.1.1",
		"UserAgent":          "Mozilla/5.0",
		"Country":            "US",
		"Locale":             "en-US",
		"AppVersion":         "1.0.0",
		"Custom":             map[string]any{"customKey": "customValue"},
		"PrivateAttributes":  map[string]any{"privateKey": "privateValue"},
		"StatsigEnvironment": map[string]string{"envKey": "envValue"},
		"CustomIDs":          map[string]string{"customIDKey": "customIDValue"},
		"custom-key":         "custom-value",
	}

	user, err := statsigProvider.ToStatsigUser(evalCtx)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if user.UserID != "test-key" {
		t.Errorf("expected UserID to be 'test-key', got %v", user.UserID)
	}
	if user.Email != "user@example.com" {
		t.Errorf("expected Email to be 'user@example.com', got %v", user.Email)
	}
	if user.IpAddress != "192.168.1.1" {
		t.Errorf("expected IpAddress to be '192.168.1.1', got %v", user.IpAddress)
	}
	if user.UserAgent != "Mozilla/5.0" {
		t.Errorf("expected UserAgent to be 'Mozilla/5.0', got %v", user.UserAgent)
	}
	if user.Country != "US" {
		t.Errorf("expected Country to be 'US', got %v", user.Country)
	}
	if user.Locale != "en-US" {
		t.Errorf("expected Locale to be 'en-US', got %v", user.Locale)
	}
	if user.AppVersion != "1.0.0" {
		t.Errorf("expected AppVersion to be '1.0.0', got %v", user.AppVersion)
	}
	if user.Custom["customKey"] != "customValue" {
		t.Errorf("expected Custom['customKey'] to be 'customValue', got %v", user.Custom["customKey"])
	}
	if user.PrivateAttributes["privateKey"] != "privateValue" {
		t.Errorf("expected PrivateAttributes['privateKey'] to be 'privateValue', got %v", user.PrivateAttributes["privateKey"])
	}
	if user.StatsigEnvironment["envKey"] != "envValue" {
		t.Errorf("expected StatsigEnvironment['envKey'] to be 'envValue', got %v", user.StatsigEnvironment["envKey"])
	}
	if user.CustomIDs["customIDKey"] != "customIDValue" {
		t.Errorf("expected CustomIDs['customIDKey'] to be 'customIDValue', got %v", user.CustomIDs["customIDKey"])
	}
	if user.Custom["custom-key"] != "custom-value" {
		t.Errorf("expected CustomIDs['custom-key'] to be 'custom_value', got %v", user.Custom["custom-key"])
	}
}

// Handles missing TargetingKey, UserID, and/or CustomID using a table-driven test
func TestHandlesMissingTargetingKeyOrUserIDOrCustomID(t *testing.T) {
	tests := []struct {
		name    string
		evalCtx of.FlattenedContext
		wantErr require.ErrorAssertionFunc
	}{
		{
			name:    "only unrelated key",
			evalCtx: of.FlattenedContext{"dummy-key": "test-key"},
			wantErr: require.Error,
		},
		{
			name:    "has UserID",
			evalCtx: of.FlattenedContext{"UserID": "test_user"},
			wantErr: require.NoError,
		},
		{
			name:    "has targetingKey",
			evalCtx: of.FlattenedContext{of.TargetingKey: "targeting-key"},
			wantErr: require.NoError,
		},
		{
			name:    "has CustomIDs",
			evalCtx: of.FlattenedContext{"CustomIDs": map[string]string{"custom": "id"}},
			wantErr: require.NoError,
		},
		{
			name:    "has UserID and CustomIDs",
			evalCtx: of.FlattenedContext{"UserID": "test_user", "CustomIDs": map[string]string{"custom": "id"}},
			wantErr: require.NoError,
		},
		{
			name:    "has targetingKey and CustomIDs",
			evalCtx: of.FlattenedContext{of.TargetingKey: "targeting-key", "CustomIDs": map[string]string{"custom": "id"}},
			wantErr: require.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := statsigProvider.ToStatsigUser(tt.evalCtx)
			tt.wantErr(t, err)
		})
	}
}

func cleanup() {
	provider.Shutdown()
}

func TestEvaluationMethods(t *testing.T) {
	err := of.SetProviderAndWait(t.Context(), provider)
	if err != nil {
		t.Fatalf("error setting provider %s", err)
	}

	tests := []struct {
		flag          string
		defaultValue  any
		evalCtx       of.FlattenedContext
		expected      any
		expectedError string
	}{
		{"always_on_gate", false, of.FlattenedContext{"UserID": "123"}, true, ""},

		{"boolean", false, of.FlattenedContext{"UserID": "123", "feature_config": statsigProvider.FeatureConfig{FeatureConfigType: "CONFIG", Name: "valid_flag"}}, true, ""},
		{"float", 1.5999, of.FlattenedContext{"UserID": "123", "feature_config": statsigProvider.FeatureConfig{FeatureConfigType: "CONFIG", Name: "valid_flag"}}, 1.5, ""},
		{"number", int64(42999), of.FlattenedContext{"UserID": "123", "feature_config": statsigProvider.FeatureConfig{FeatureConfigType: "CONFIG", Name: "valid_flag"}}, int64(42), ""},
		{"object", map[string]any{"key": "value999"}, of.FlattenedContext{"UserID": "123", "feature_config": statsigProvider.FeatureConfig{FeatureConfigType: "CONFIG", Name: "valid_flag"}}, map[string]any{"key1": "value1"}, ""},
		{"string", "default_value", of.FlattenedContext{"UserID": "123", "feature_config": statsigProvider.FeatureConfig{FeatureConfigType: "CONFIG", Name: "valid_flag"}}, "expected_value", ""},
		{"slice", []any{"fallback1", "fallback2"}, of.FlattenedContext{"UserID": "123", "feature_config": statsigProvider.FeatureConfig{FeatureConfigType: "CONFIG", Name: "valid_flag"}}, []any{"v1", "v2"}, ""},

		{"boolean", false, of.FlattenedContext{"UserID": "123", "feature_config": statsigProvider.FeatureConfig{FeatureConfigType: "LAYER", Name: "valid_layer"}}, true, ""},
		{"float", 1.5999, of.FlattenedContext{"UserID": "123", "feature_config": statsigProvider.FeatureConfig{FeatureConfigType: "LAYER", Name: "valid_layer"}}, 1.5, ""},
		{"number", int64(42999), of.FlattenedContext{"UserID": "123", "feature_config": statsigProvider.FeatureConfig{FeatureConfigType: "LAYER", Name: "valid_layer"}}, int64(42), ""},
		{"object", map[string]any{"key": "value999"}, of.FlattenedContext{"UserID": "123", "feature_config": statsigProvider.FeatureConfig{FeatureConfigType: "LAYER", Name: "valid_layer"}}, map[string]any{"key1": "value1"}, ""},
		{"string", "default_value", of.FlattenedContext{"UserID": "123", "feature_config": statsigProvider.FeatureConfig{FeatureConfigType: "LAYER", Name: "valid_layer"}}, "expected_value", ""},
		{"slice", []any{"fallback1", "fallback2"}, of.FlattenedContext{"UserID": "123", "feature_config": statsigProvider.FeatureConfig{FeatureConfigType: "LAYER", Name: "valid_layer"}}, []any{"v1", "v2"}, ""},

		{"invalid_flag", false, of.FlattenedContext{"UserID": "123"}, false, "flag not found"},

		// expected to succeed when https://github.com/statsig-io/go-sdk/issues/32 is resolved and adopted
		// {"invalid_flag", true, of.FlattenedContext{"UserID": "123", "feature_config": statsigProvider.FeatureConfig{FeatureConfigType: "CONFIG", Name: "test"}}, true, "flag not found"},

		{"float", 1.23, of.FlattenedContext{"UserID": "123", "feature_config": statsigProvider.FeatureConfig{FeatureConfigType: "CONFIG", Name: "invalid_flag"}}, 1.23, "flag not found"},
		{"number", int64(43), of.FlattenedContext{"UserID": "123", "feature_config": statsigProvider.FeatureConfig{FeatureConfigType: "CONFIG", Name: "invalid_flag"}}, int64(43), "flag not found"},
		{"object", map[string]any{"key1": "other-value"}, of.FlattenedContext{"UserID": "123", "feature_config": statsigProvider.FeatureConfig{FeatureConfigType: "CONFIG", Name: "invalid_flag"}}, map[string]any{"key1": "other-value"}, "flag not found"},
		{"string", "value2", of.FlattenedContext{"UserID": "123", "feature_config": statsigProvider.FeatureConfig{FeatureConfigType: "CONFIG", Name: "invalid_flag"}}, "value2", "flag not found"},

		{"float", 1.23, of.FlattenedContext{"UserID": "123", "feature_config": statsigProvider.FeatureConfig{FeatureConfigType: "LAYER", Name: "invalid_flag"}}, 1.23, "flag not found"},
		{"number", int64(43), of.FlattenedContext{"UserID": "123", "feature_config": statsigProvider.FeatureConfig{FeatureConfigType: "LAYER", Name: "invalid_flag"}}, int64(43), "flag not found"},
		{"object", map[string]any{"key1": "other-value"}, of.FlattenedContext{"UserID": "123", "feature_config": statsigProvider.FeatureConfig{FeatureConfigType: "LAYER", Name: "invalid_flag"}}, map[string]any{"key1": "other-value"}, "flag not found"},
		{"string", "value2", of.FlattenedContext{"UserID": "123", "feature_config": statsigProvider.FeatureConfig{FeatureConfigType: "LAYER", Name: "invalid_flag"}}, "value2", "flag not found"},

		{"invalid_user_context", false, of.FlattenedContext{"UserID": "123", "invalid": "value"}, false, ""},
		{"enriched_user_context", false, of.FlattenedContext{"UserID": "123", "Email": "v", "IpAddress": "v", "UserAgent": "v", "Country": "v", "Locale": "v"}, false, ""},
		{"missing_feature_config", int64(43), of.FlattenedContext{"UserID": "123"}, int64(43), ""},
		{"empty_context", int64(43), of.FlattenedContext{}, int64(43), ""},
	}

	for _, test := range tests {

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
		Options: statsig.Options{BootstrapValues: string(bytes[:])},
		SdkKey:  "secret-key",
	}

	provider, err = statsigProvider.NewProvider(providerOptions)
	if err != nil {
		fmt.Printf("Error during new provider: %v\n", err)
		os.Exit(1)
	}

	if err := provider.Init(of.EvaluationContext{}); err != nil {
		fmt.Printf("Error during provider initialization: %v\n", err)
		os.Exit(1)
	}

	// Run the tests
	exitCode := m.Run()

	cleanup()

	os.Exit(exitCode)
}
