package prefab_test

import (
	"context"
	"testing"

	prefabProvider "github.com/open-feature/go-sdk-contrib/providers/prefab/pkg"
	of "github.com/open-feature/go-sdk/openfeature"

	"github.com/stretchr/testify/require"
)

// based on prefab test

const ()

var provider *prefabProvider.Provider

func TestBooleanEvaluation(t *testing.T) {
	providerConfig := prefabProvider.ProviderConfig{
		// Sources: []string{"datafile://features.json"},
		// Sources: []string{"datafile://enabled.yaml"},
		Sources: []string{"datafile://features.yaml"},
	}

	provider, err := prefabProvider.NewProvider(providerConfig)
	if err != nil {
		t.Fail()
	}
	err = provider.Init(of.EvaluationContext{})
	if err != nil {
		t.Fail()
	}

	ctx := context.Background()

	evalCtx := map[string]interface{}{}

	resolution := provider.BooleanEvaluation(ctx, "sample_bool", false, evalCtx)
	if resolution.Value != true {
		t.Fatalf("Expected one of the variant payloads")
	}

	t.Run("evalCtx empty", func(t *testing.T) {
		resolution := provider.BooleanEvaluation(ctx, "non-existing", false, nil)
		require.Equal(t, false, resolution.Value)
	})

	// of.SetProvider(provider)
	// ofClient := of.NewClient("my-app")

	// evalCtx = of.NewEvaluationContext(
	// 	"john",
	// 	map[string]interface{}{
	// 		"Firstname": "John",
	// 		"Lastname":  "Doe",
	// 		"Email":     "john@doe.com",
	// 	},
	// )
	// enabled, err := ofClient.BooleanValue(context.Background(), "TestTrueOn", false, evalCtx)
	// if enabled == false {
	// 	t.Fatalf("Expected feature to be enabled")
	// }

}

func TestBooleanEvaluationByUser(t *testing.T) {
	providerConfig := prefabProvider.ProviderConfig{
		Sources: []string{"datafile://features.yaml"},
	}

	provider, err := prefabProvider.NewProvider(providerConfig)
	if err != nil {
		t.Fail()
	}
	err = provider.Init(of.EvaluationContext{})
	if err != nil {
		t.Fail()
	}

	ctx := context.Background()

	evalCtx := map[string]interface{}{
		"user.key":         "key1",
		"team.domain":      "prefab.cloud",
		"team.description": "team1",
	}

	resolution := provider.BooleanEvaluation(ctx, "test1", false, evalCtx)
	if resolution.Value != true {
		t.Fatalf("Expected one of the variant payloads")
	}

	evalCtx = map[string]interface{}{
		"user.key":    "key1",
		"team.domain": "other.com",
	}

	resolution = provider.BooleanEvaluation(ctx, "test1", false, evalCtx)
	if resolution.Value != false {
		t.Fatalf("Expected false")
	}

	// of.SetProvider(provider)
	// ofClient := of.NewClient("my-app")

	// evalCtx = of.NewEvaluationContext(
	// 	"john",
	// 	map[string]interface{}{
	// 		"Firstname": "John",
	// 		"Lastname":  "Doe",
	// 		"Email":     "john@doe.com",
	// 	},
	// )
	// enabled, err := ofClient.BooleanValue(context.Background(), "TestTrueOn", false, evalCtx)
	// if enabled == false {
	// 	t.Fatalf("Expected feature to be enabled")
	// }

}
