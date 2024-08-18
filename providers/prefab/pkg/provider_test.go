package prefab_test

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/open-feature/go-sdk-contrib/providers/prefab/internal"
	prefabProvider "github.com/open-feature/go-sdk-contrib/providers/prefab/pkg"
	of "github.com/open-feature/go-sdk/openfeature"
	prefab "github.com/prefab-cloud/prefab-cloud-go/pkg"

	"github.com/stretchr/testify/require"
)

// based on prefab test

const ()

var provider *prefabProvider.Provider

func TestBooleanEvaluation(t *testing.T) {

	flattenedContext := map[string]interface{}{}

	resolution := provider.BooleanEvaluation(context.Background(), "sample_bool", false, flattenedContext)
	if resolution.Value != true {
		t.Fatalf("Expected one of the variant payloads")
	}

	t.Run("evalCtx empty", func(t *testing.T) {
		resolution := provider.BooleanEvaluation(context.Background(), "non-existing", false, nil)
		require.Equal(t, false, resolution.Value)
	})

	of.SetProvider(provider)
	ofClient := of.NewClient("my-app")

	evalCtx := of.NewEvaluationContext(
		"",
		map[string]interface{}{},
	)
	enabled, err := ofClient.BooleanValue(context.Background(), "sample_bool", false, evalCtx)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	require.Equal(t, true, enabled)

}

// TODO handle conditional feature flags based on context where json/yaml parsing is implemented
// func TestBooleanEvaluationByUser(t *testing.T) {
// 	providerConfig := prefabProvider.ProviderConfig{
// 		Sources: []string{"datafile://enabled.yaml"},
// 	}

// 	provider, err := prefabProvider.NewProvider(providerConfig)
// 	if err != nil {
// 		t.Fail()
// 	}
// 	err = provider.Init(of.EvaluationContext{})
// 	if err != nil {
// 		t.Fail()
// 	}

// 	ctx := context.Background()

// 	evalCtx := map[string]interface{}{
// 		"user.key":         "key1",
// 		"team.domain":      "prefab.cloud",
// 		"team.description": "team1",
// 	}

// 	resolution := provider.BooleanEvaluation(ctx, "test1", false, evalCtx)
// 	if resolution.Value != true {
// 		t.Fatalf("Expected one of the variant payloads")
// 	}

// 	evalCtx = map[string]interface{}{
// 		"user.key":    "key1",
// 		"team.domain": "other.com",
// 	}

// 	resolution = provider.BooleanEvaluation(ctx, "test1", false, evalCtx)
// 	if resolution.Value != false {
// 		t.Fatalf("Expected false")
// 	}

// 	// of.SetProvider(provider)
// 	// ofClient := of.NewClient("my-app")

// 	// evalCtx = of.NewEvaluationContext(
// 	// 	"john",
// 	// 	map[string]interface{}{
// 	// 		"Firstname": "John",
// 	// 		"Lastname":  "Doe",
// 	// 		"Email":     "john@doe.com",
// 	// 	},
// 	// )
// 	// enabled, err := ofClient.BooleanValue(context.Background(), "TestTrueOn", false, evalCtx)
// 	// if enabled == false {
// 	// 	t.Fatalf("Expected feature to be enabled")
// 	// }

// }

func TestFloatEvaluation(t *testing.T) {

	flattenedContext := map[string]interface{}{}

	resolution := provider.FloatEvaluation(context.Background(), "sample_double", 1.2, flattenedContext)
	if resolution.Value != 12.12 {
		t.Fatalf("Expected one of the variant payloads")
	}

	t.Run("evalCtx empty", func(t *testing.T) {
		resolution := provider.FloatEvaluation(context.Background(), "non-existing", 1.2, nil)
		require.Equal(t, 1.2, resolution.Value)
	})

	of.SetProvider(provider)
	ofClient := of.NewClient("my-app")

	evalCtx := of.NewEvaluationContext(
		"",
		map[string]interface{}{},
	)
	value, err := ofClient.FloatValue(context.Background(), "sample_double", 1.2, evalCtx)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	require.Equal(t, 12.12, value)

}

func TestIntEvaluation(t *testing.T) {

	flattenedContext := map[string]interface{}{}

	resolution := provider.IntEvaluation(context.Background(), "sample_int", 1, flattenedContext)
	if resolution.Value != 123 {
		t.Fatalf("Expected one of the variant payloads")
	}

	t.Run("evalCtx empty", func(t *testing.T) {
		resolution := provider.IntEvaluation(context.Background(), "non-existing", 1, nil)
		require.Equal(t, int64(1), resolution.Value)
	})

	of.SetProvider(provider)
	ofClient := of.NewClient("my-app")

	evalCtx := of.NewEvaluationContext(
		"",
		map[string]interface{}{},
	)
	value, err := ofClient.IntValue(context.Background(), "sample_int", 1, evalCtx)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	require.Equal(t, int64(123), value)

}

func TestStringEvaluation(t *testing.T) {

	flattenedContext := map[string]interface{}{}

	resolution := provider.StringEvaluation(context.Background(), "sample", "default", flattenedContext)
	if resolution.Value != "test sample value" {
		t.Fatalf("Expected one of the variant payloads")
	}

	t.Run("evalCtx empty", func(t *testing.T) {
		resolution := provider.StringEvaluation(context.Background(), "non-existing", "default", nil)
		require.Equal(t, "default", resolution.Value)
	})

	of.SetProvider(provider)
	ofClient := of.NewClient("my-app")

	evalCtx := of.NewEvaluationContext(
		"",
		map[string]interface{}{},
	)
	value, err := ofClient.StringValue(context.Background(), "sample", "default", evalCtx)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	require.Equal(t, "test sample value", value)

}

// TODO test and enable when json/yaml parsing is implemented
// func TestObjectEvaluation(t *testing.T) {

// 	flattenedContext := map[string]interface{}{}

// 	resolution := provider.ObjectEvaluation(context.Background(), "flag_with_a_value", "default", flattenedContext)
// 	require.Equal(t, "all-features", resolution.Value)

// 	t.Run("evalCtx empty", func(t *testing.T) {
// 		resolution := provider.ObjectEvaluation(context.Background(), "non-existing", "default", nil)
// 		require.Equal(t, "default", resolution.Value)
// 	})

// 	of.SetProvider(provider)
// 	ofClient := of.NewClient("my-app")

// 	evalCtx := of.NewEvaluationContext(
// 		"",
// 		map[string]interface{}{},
// 	)
// 	value, err := ofClient.ObjectValue(context.Background(), "sample", "default", evalCtx)
// 	if err != nil {
// 		t.Fatalf("expected no error, got %v", err)
// 	}
// 	require.Equal(t, "all-features", value)

// }

// Converts non-empty FlattenedContext to ContextSet correctly
func TestConvertsNonEmptyFlattenedContextToContextSet(t *testing.T) {
	evalCtx := of.FlattenedContext{
		"user.name":   "John",
		"user.age":    30,
		"device.type": "mobile",
	}
	expected := prefab.NewContextSet()
	expected.WithNamedContextValues("user", map[string]interface{}{
		"name": "John",
		"age":  30,
	})
	expected.WithNamedContextValues("device", map[string]interface{}{
		"type": "mobile",
	})

	result, err := internal.ToPrefabContext(evalCtx)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !reflect.DeepEqual(result, *expected) {
		t.Errorf("expected %v, got %v", *expected, result)
	}
}

// Handles keys without a dot separator by panicking
func TestHandlesKeysWithoutDotSeparatorByPanicking(t *testing.T) {
	evalCtx := of.FlattenedContext{
		"invalidKey": "value",
	}

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic for key without dot separator, but did not panic")
		}
	}()

	internal.ToPrefabContext(evalCtx)
}

func cleanup() {
	provider.Shutdown()
}

func TestMain(m *testing.M) {

	providerConfig := prefabProvider.ProviderConfig{
		Sources: []string{"datafile://enabled.yaml"},
	}

	var err error
	provider, err = prefabProvider.NewProvider(providerConfig)
	if err != nil {
		fmt.Printf("Error during new provider: %v\n", err)
		os.Exit(1)
	}
	err = provider.Init(of.EvaluationContext{})
	if err != nil {
		fmt.Printf("Error during provider init: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("provider: %v\n", provider)

	// Run the tests
	exitCode := m.Run()

	cleanup()

	os.Exit(exitCode)
}
