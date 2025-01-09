package unleash_test

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/Unleash/unleash-client-go/v3"
	unleashProvider "github.com/open-feature/go-sdk-contrib/providers/unleash/pkg"
	of "github.com/open-feature/go-sdk/openfeature"
	"github.com/stretchr/testify/require"
)

var provider *unleashProvider.Provider
var ofClient *of.Client

func TestBooleanEvaluation(t *testing.T) {
	resolution := provider.BooleanEvaluation(context.Background(), "variant-flag", false, nil)
	enabled, _ := resolution.ProviderResolutionDetail.FlagMetadata.GetBool("enabled")
	if !enabled {
		t.Fatalf("Expected feature to be enabled")
	}
	if !resolution.Value {
		t.Fatalf("Expected one of the variant payloads")
	}

	t.Run("evalCtx empty", func(t *testing.T) {
		resolution := provider.BooleanEvaluation(context.Background(), "non-existing-flag", false, nil)
		require.Equal(t, false, resolution.Value)
	})

	t.Run("evalCtx empty fallback to default", func(t *testing.T) {
		resolution := provider.BooleanEvaluation(context.Background(), "non-existing-flag", true, nil)
		require.Equal(t, true, resolution.Value)
	})
}

func TestIntEvaluation(t *testing.T) {
	defaultValue := int64(0)

	t.Run("int-flag", func(t *testing.T) {
		resolution := provider.IntEvaluation(context.Background(), "int-flag", defaultValue, nil)
		enabled, _ := resolution.ProviderResolutionDetail.FlagMetadata.GetBool("enabled")
		require.True(t, enabled)
		require.Equal(t, "int-flag-variant", resolution.ProviderResolutionDetail.Variant)
		require.Equal(t, int64(123), resolution.Value)
		require.Equal(t, of.ErrorCode(""), resolution.ResolutionDetail().ErrorCode)
	})

	t.Run("disabled-flag", func(t *testing.T) {
		resolution := provider.IntEvaluation(context.Background(), "disabled-flag", defaultValue, nil)
		enabled, _ := resolution.ProviderResolutionDetail.FlagMetadata.GetBool("enabled")
		require.False(t, enabled)
		require.Equal(t, "", resolution.ProviderResolutionDetail.Variant)
		require.Equal(t, defaultValue, resolution.Value)
		require.Equal(t, of.ErrorCode(""), resolution.ResolutionDetail().ErrorCode)
	})

	t.Run("non-existing-flag", func(t *testing.T) {
		resolution := provider.IntEvaluation(context.Background(), "non-existing-flag", defaultValue, nil)
		enabled, _ := resolution.ProviderResolutionDetail.FlagMetadata.GetBool("enabled")
		require.False(t, enabled)
		require.Equal(t, "", resolution.ProviderResolutionDetail.Variant)
		require.Equal(t, defaultValue, resolution.Value)
		require.Equal(t, of.ErrorCode(""), resolution.ResolutionDetail().ErrorCode)
	})
}

func TestFloatEvaluation(t *testing.T) {
	defaultValue := 0.0

	t.Run("int-flag", func(t *testing.T) {
		resolution := provider.FloatEvaluation(context.Background(), "double-flag", defaultValue, nil)
		enabled, _ := resolution.ProviderResolutionDetail.FlagMetadata.GetBool("enabled")
		require.True(t, enabled)
		require.Equal(t, "double-flag-variant", resolution.ProviderResolutionDetail.Variant)
		require.Equal(t, 1.23, resolution.Value)
		require.Equal(t, of.ErrorCode(""), resolution.ResolutionDetail().ErrorCode)
	})

	t.Run("disabled-flag", func(t *testing.T) {
		resolution := provider.FloatEvaluation(context.Background(), "disabled-flag", defaultValue, nil)
		enabled, _ := resolution.ProviderResolutionDetail.FlagMetadata.GetBool("enabled")
		require.False(t, enabled)
		require.Equal(t, "", resolution.ProviderResolutionDetail.Variant)
		require.Equal(t, defaultValue, resolution.Value)
		require.Equal(t, of.ErrorCode(""), resolution.ResolutionDetail().ErrorCode)
	})

	t.Run("non-existing-flag", func(t *testing.T) {
		resolution := provider.FloatEvaluation(context.Background(), "non-existing-flag", defaultValue, nil)
		enabled, _ := resolution.ProviderResolutionDetail.FlagMetadata.GetBool("enabled")
		require.False(t, enabled)
		require.Equal(t, "", resolution.ProviderResolutionDetail.Variant)
		require.Equal(t, defaultValue, resolution.Value)
		require.Equal(t, of.ErrorCode(""), resolution.ResolutionDetail().ErrorCode)
	})
}

func TestStringEvaluation(t *testing.T) {
	resolution := provider.StringEvaluation(context.Background(), "variant-flag", "", nil)
	enabled, _ := resolution.ProviderResolutionDetail.FlagMetadata.GetBool("enabled")
	if enabled == false {
		t.Fatalf("Expected feature to be enabled")
	}
	if resolution.ProviderResolutionDetail.Variant != "v1" {
		t.Fatalf("Expected variant name")
	}
	if resolution.Value != "v1" {
		t.Fatalf("Expected one of the variant payloads")
	}

	evalCtx := of.NewEvaluationContext(
		"",
		map[string]interface{}{},
	)
	value, _ := ofClient.StringValue(context.Background(), "variant-flag", "", evalCtx)
	if value == "" {
		t.Fatalf("Expected a value")
	}
}

func TestBooleanEvaluationByUser(t *testing.T) {
	resolution := provider.BooleanEvaluation(context.Background(), "users-flag", false, map[string]interface{}{
		"UserId": "111",
	})
	enabled, _ := resolution.ProviderResolutionDetail.FlagMetadata.GetBool("enabled")
	if enabled == false {
		t.Fatalf("Expected feature to be enabled")
	}

	resolution = provider.BooleanEvaluation(context.Background(), "users-flag", false, map[string]interface{}{
		"UserId": "2",
	})
	enabled, _ = resolution.ProviderResolutionDetail.FlagMetadata.GetBool("enabled")
	if enabled == true {
		t.Fatalf("Expected feature to be disabled")
	}

	evalCtx := of.NewEvaluationContext(
		"",
		map[string]interface{}{
			"UserId":        "111",
			"AppName":       "test-app",
			"CurrentTime":   "2006-01-02T15:04:05Z",
			"Environment":   "test-env",
			"RemoteAddress": "1.2.3.4",
			"SessionId":     "test-session",
		},
	)
	enabled, _ = ofClient.BooleanValue(context.Background(), "users-flag", false, evalCtx)
	if enabled == false {
		t.Fatalf("Expected feature to be enabled")
	}
}

func TestStringEvaluationByCurrentTime(t *testing.T) {
	resolution := provider.StringEvaluation(context.Background(), "variant-flag-by-date", "fallback", map[string]interface{}{
		"UserId":      "2",
		"CurrentTime": "2025-01-02T15:04:05Z",
	})
	enabled, _ := resolution.ProviderResolutionDetail.FlagMetadata.GetBool("enabled")
	if enabled == false {
		t.Fatalf("Expected feature to be enabled")
	}

	if resolution.ProviderResolutionDetail.Variant != "var1" {
		t.Fatalf("Expected variant name")
	}
	if resolution.Value != "v1" {
		t.Fatalf("Expected one of the variant payloads")
	}

	resolution = provider.StringEvaluation(context.Background(), "variant-flag-by-date", "fallback", map[string]interface{}{
		"UserId":      "2",
		"CurrentTime": "2023-01-02T15:04:05Z",
	})
	if resolution.Value != "fallback" {
		t.Fatalf("Expected fallback value")
	}
}

func TestInvalidContextEvaluation(t *testing.T) {
	evalCtx := make(of.FlattenedContext)
	defaultValue := true
	evalCtx["Invalid-key"] = make(chan int)
	resolution := provider.BooleanEvaluation(context.Background(), "non-existing-flag", defaultValue, evalCtx)
	if resolution.Value != defaultValue {
		t.Errorf("Expected value to be %v when evaluation context is invalid, got %v", defaultValue, resolution.Value)
	}
	if resolution.Reason != of.ErrorReason {
		t.Errorf("Expected reason to be %s, got %s", of.ErrorReason, resolution.Reason)
	}
}

func TestEvaluationMethods(t *testing.T) {

	tests := []struct {
		flag          string
		defaultValue  interface{}
		evalCtx       of.FlattenedContext
		expected      interface{}
		expectedError string
	}{
		{flag: "DateExample", defaultValue: false, evalCtx: of.FlattenedContext{}, expected: true, expectedError: ""},
		{flag: "variant-flag", defaultValue: false, evalCtx: of.FlattenedContext{}, expected: true, expectedError: ""},
		{flag: "double-flag", defaultValue: 9.9, evalCtx: of.FlattenedContext{}, expected: 1.23, expectedError: ""},
		{flag: "int-flag", defaultValue: int64(1), evalCtx: of.FlattenedContext{}, expected: int64(123), expectedError: ""},
		{flag: "variant-flag", defaultValue: "fallback", evalCtx: of.FlattenedContext{}, expected: "v1", expectedError: ""},
		{flag: "json-flag", defaultValue: "fallback", evalCtx: of.FlattenedContext{}, expected: "{\n  \"k1\": \"v1\"\n}", expectedError: ""},
		{flag: "csv-flag", defaultValue: "fallback", evalCtx: of.FlattenedContext{}, expected: "a,b,c", expectedError: ""},

		{flag: "csv-invalid_flag", defaultValue: false, evalCtx: of.FlattenedContext{}, expected: false, expectedError: ""},
		{flag: "csv-invalid_flag", defaultValue: true, evalCtx: of.FlattenedContext{}, expected: true, expectedError: ""},

		{"float", 1.23, of.FlattenedContext{"UserID": "123"}, 1.23, "flag not found"},
		{"number", int64(43), of.FlattenedContext{"UserID": "123"}, int64(43), "flag not found"},
		{"object", map[string]interface{}{"key1": "other-value"}, of.FlattenedContext{"UserID": "123"}, map[string]interface{}{"key1": "other-value"}, "flag not found"},
		{"string", "value2", of.FlattenedContext{"UserID": "123"}, "value2", "flag not found"},

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
			require.Equal(t, test.expected, res.Value, fmt.Errorf("failed for test flag `%s`", test.flag))
		case reflect.Int, reflect.Int8, reflect.Int32, reflect.Int64:
			res := provider.IntEvaluation(context.Background(), test.flag, test.defaultValue.(int64), test.evalCtx)
			require.Equal(t, test.expected, res.Value, fmt.Errorf("failed for test flag `%s`", test.flag))
		case reflect.Float32, reflect.Float64:
			res := provider.FloatEvaluation(context.Background(), test.flag, test.defaultValue.(float64), test.evalCtx)
			require.Equal(t, test.expected, res.Value, fmt.Errorf("failed for test flag `%s`", test.flag))
		case reflect.String:
			res := provider.StringEvaluation(context.Background(), test.flag, test.defaultValue.(string), test.evalCtx)
			require.Equal(t, test.expected, res.Value, fmt.Errorf("failed for test flag `%s`", test.flag))
		default:
			res := provider.ObjectEvaluation(context.Background(), test.flag, test.defaultValue, test.evalCtx)
			require.Equal(t, test.expected, res.Value, fmt.Errorf("failed for test flag `%s`", test.flag))
		}
	}
}

// global cleanup
func cleanup() {
	provider.Shutdown()
}

func TestMain(m *testing.M) {

	// global init
	demoReader, err := os.Open("demo_app_toggles.json")
	if err != nil {
		fmt.Printf("Error during features file open: %v\n", err)
		os.Exit(1)
	}
	defer demoReader.Close()

	appName := "my-application"
	backupFile := fmt.Sprintf("unleash-repo-schema-v1-%s.json", appName)

	providerOptions := unleashProvider.ProviderConfig{
		Options: []unleash.ConfigOption{
			unleash.WithListener(&unleash.DebugListener{}),
			unleash.WithAppName(appName),
			unleash.WithRefreshInterval(5 * time.Second),
			unleash.WithMetricsInterval(5 * time.Second),
			unleash.WithStorage(&unleash.BootstrapStorage{Reader: demoReader}),
			unleash.WithBackupPath("./"),
			unleash.WithUrl("https://localhost:4242"),
		},
	}

	provider, err = unleashProvider.NewProvider(providerOptions)

	if err != nil {
		fmt.Printf("Error during provider open: %v\n", err)
	}
	err = of.SetProviderAndWait(provider)
	if err != nil {
		fmt.Printf("Error during SetProviderAndWait: %v\n", err)
		os.Exit(1)
	}
	ofClient = of.NewClient("my-app")

	fmt.Printf("provider: %v\n", provider)

	// Run the tests
	exitCode := m.Run()

	cleanup()

	os.Remove(backupFile)
	os.Exit(exitCode)
}
