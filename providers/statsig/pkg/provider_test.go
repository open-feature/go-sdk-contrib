package statsig_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	statsigProvider "github.com/open-feature/go-sdk-contrib/providers/statsig/pkg"
	of "github.com/open-feature/go-sdk/openfeature"
	statsig "github.com/statsig-io/go-sdk"
)

var provider *statsigProvider.Provider

func TestBooleanEvaluation(t *testing.T) {
	of.SetProvider(provider)
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
	of.SetProvider(provider)
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
	of.SetProvider(provider)
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

// global cleanup
func cleanup() {
	provider.Shutdown()
}

func TestMain(m *testing.M) {

	bytes, err := os.ReadFile("download_config_specs.json")

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
