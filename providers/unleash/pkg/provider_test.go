package unleash_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/Unleash/unleash-client-go/v3"
	unleashProvider "github.com/open-feature/go-sdk-contrib/providers/unleash/pkg"
	of "github.com/open-feature/go-sdk/pkg/openfeature"
	"github.com/stretchr/testify/require"
)

var provider *unleashProvider.Provider

func TestBooleanEvaluation(t *testing.T) {
	resolution := provider.BooleanEvaluation(context.Background(), "variant-flag", false, nil)
	enabled, _ := resolution.ProviderResolutionDetail.FlagMetadata.GetBool("enabled")
	if enabled == false {
		t.Fatalf("Expected feature to be enabled")
	}
	if resolution.Value != true {
		t.Fatalf("Expected one of the variant payloads")
	}

	t.Run("evalCtx empty", func(t *testing.T) {
		resolution := provider.BooleanEvaluation(context.Background(), "non-existing-flag", false, nil)
		require.Equal(t, false, resolution.Value)
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

	of.SetProvider(provider)
	ofClient := of.NewClient("my-app")

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

	of.SetProvider(provider)
	ofClient := of.NewClient("my-app")

	evalCtx := of.NewEvaluationContext(
		"",
		map[string]interface{}{
			"UserId": "111",
		},
	)
	enabled, _ = ofClient.BooleanValue(context.Background(), "users-flag", false, evalCtx)
	if enabled == false {
		t.Fatalf("Expected feature to be enabled")
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
	}

	providerOptions := unleashProvider.ProviderConfig{
		Options: []unleash.ConfigOption{
			unleash.WithListener(&unleash.DebugListener{}),
			unleash.WithAppName("my-application"),
			unleash.WithRefreshInterval(5 * time.Second),
			unleash.WithMetricsInterval(5 * time.Second),
			unleash.WithStorage(&unleash.BootstrapStorage{Reader: demoReader}),
			unleash.WithUrl("https://localhost:4242"),
		},
	}

	provider, err = unleashProvider.NewProvider(providerOptions)

	if err != nil {
		fmt.Printf("Error during provider open: %v\n", err)
	}
	err = provider.Init(of.EvaluationContext{})
	if err != nil {
		fmt.Printf("Error during provider init: %v\n", err)
	}

	fmt.Printf("provider: %v\n", provider)

	// Run the tests
	exitCode := m.Run()

	cleanup()

	os.Exit(exitCode)
}
