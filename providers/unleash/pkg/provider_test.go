package unleash_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/Unleash/unleash-client-go/v3"
	unleashProvider "github.com/open-feature/go-sdk-contrib/providers/unleash/pkg"
	of "github.com/open-feature/go-sdk/pkg/openfeature"
	"github.com/stretchr/testify/require"
)

func TestBooleanEvaluation(t *testing.T) {
	demoReader, err := os.Open("demo_app_toggles.json")
	if err != nil {
		t.Fail()
	}

	providerConfig := unleashProvider.ProviderConfig{
		Options: []unleash.ConfigOption{
			unleash.WithListener(&unleash.DebugListener{}),
			unleash.WithAppName("my-application"),
			unleash.WithRefreshInterval(5 * time.Second),
			unleash.WithMetricsInterval(5 * time.Second),
			unleash.WithStorage(&unleash.BootstrapStorage{Reader: demoReader}),
			unleash.WithUrl("https://localhost:4242"),
		},
	}

	provider, err := unleashProvider.NewProvider(providerConfig)
	if err != nil {
		t.Fail()
	}
	err = provider.Init(of.EvaluationContext{})
	if err != nil {
		t.Fail()
	}

	ctx := context.Background()

	resolution := provider.BooleanEvaluation(ctx, "variant-flag", false, nil)
	enabled, _ := resolution.ProviderResolutionDetail.FlagMetadata.GetBool("enabled")
	if enabled == false {
		t.Fatalf("Expected feature to be enabled")
	}
	if resolution.Value != true {
		t.Fatalf("Expected one of the variant payloads")
	}

	t.Run("evalCtx empty", func(t *testing.T) {

		resolution := provider.BooleanEvaluation(ctx, "non-existing-flag", false, nil)
		require.Equal(t, false, resolution.Value)
	})

}

func TestStringEvaluation(t *testing.T) {
	demoReader, err := os.Open("demo_app_toggles.json")
	if err != nil {
		t.Fail()
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

	provider, err := unleashProvider.NewProvider(providerOptions)
	if err != nil {
		t.Fail()
	}
	err = provider.Init(of.EvaluationContext{})
	if err != nil {
		t.Fail()
	}

	ctx := context.Background()

	resolution := provider.StringEvaluation(ctx, "variant-flag", "", nil)
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
	value, err := ofClient.StringValue(context.Background(), "variant-flag", "", evalCtx)
	if value == "" {
		t.Fatalf("Expected a value")
	}

}

func TestBooleanEvaluationByUser(t *testing.T) {
	demoReader, err := os.Open("demo_app_toggles.json")
	if err != nil {
		t.Fail()
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

	provider, err := unleashProvider.NewProvider(providerOptions)
	if err != nil {
		t.Fail()
	}
	err = provider.Init(of.EvaluationContext{})
	if err != nil {
		t.Fail()
	}

	ctx := context.Background()

	resolution := provider.BooleanEvaluation(ctx, "users-flag", false, map[string]interface{}{
		"UserId": "111",
	})
	enabled, _ := resolution.ProviderResolutionDetail.FlagMetadata.GetBool("enabled")
	if enabled == false {
		t.Fatalf("Expected feature to be enabled")
	}

	resolution = provider.BooleanEvaluation(ctx, "users-flag", false, map[string]interface{}{
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
	enabled, err = ofClient.BooleanValue(context.Background(), "users-flag", false, evalCtx)
	if enabled == false {
		t.Fatalf("Expected feature to be enabled")
	}
}
