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

var (
	provider *prefabProvider.Provider
	ofClient *of.Client
)

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

	evalCtx := of.NewEvaluationContext(
		"",
		map[string]interface{}{},
	)
	enabled, err := ofClient.BooleanValue(context.Background(), "sample_bool", false, evalCtx)
	require.Nil(t, err)
	require.Equal(t, true, enabled)
}

// TODO handle conditional feature flags based on context where json/yaml parsing is implemented
// func TestBooleanEvaluationByUser(t *testing.T) {
// 	providerConfig := prefabProvider.ProviderConfig{
// 		Sources: []string{"datafile://enabled.yaml"},
// 	}

// 	provider, err := prefabProvider.NewProvider(providerConfig)
// 	require.Nil(t, err)
// 	err = provider.Init(of.EvaluationContext{})
// 	require.Nil(t, err)

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

	evalCtx := of.NewEvaluationContext(
		"",
		map[string]interface{}{},
	)
	value, err := ofClient.FloatValue(context.Background(), "sample_double", 1.2, evalCtx)
	require.Nil(t, err)
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

	evalCtx := of.NewEvaluationContext(
		"",
		map[string]interface{}{},
	)
	value, err := ofClient.IntValue(context.Background(), "sample_int", 1, evalCtx)
	require.Nil(t, err)
	require.Equal(t, int64(123), value)
}

func TestStringEvaluation(t *testing.T) {
	flattenedContext := map[string]interface{}{}

	resolution := provider.StringEvaluation(context.Background(), "sample", "default", flattenedContext)
	if resolution.Value != "test sample value" {
		t.Fatalf("Expected one of the variant payloads")
	}

	t.Run("nil evalCtx", func(t *testing.T) {
		resolution := provider.StringEvaluation(context.Background(), "non-existing", "default", nil)
		require.Equal(t, "default", resolution.Value)
	})

	evalCtx := of.NewEvaluationContext(
		"",
		map[string]interface{}{},
	)
	value, err := ofClient.StringValue(context.Background(), "sample", "default", evalCtx)
	require.Nil(t, err)
	require.Equal(t, "test sample value", value)
}

// TODO test and enable when json/yaml parsing is implemented
func TestObjectEvaluation(t *testing.T) {
	flattenedContext := map[string]interface{}{}

	t.Run("example.nested.path", func(t *testing.T) {
		resolution := provider.ObjectEvaluation(context.Background(), "example.nested.path", "default", flattenedContext)
		require.Equal(t, "hello", resolution.Value)
	})

	evalCtx := of.NewEvaluationContext(
		"",
		map[string]interface{}{},
	)

	t.Run("example.nested.path", func(t *testing.T) {
		value, err := ofClient.ObjectValueDetails(context.Background(), "example.nested.path", "default", evalCtx)
		require.Equal(t, "hello", value.Value)
		require.Nil(t, err)
	})

	t.Run("sample_list", func(t *testing.T) {
		value, err := ofClient.ObjectValueDetails(context.Background(), "sample_list", []string{"a2", "b2"}, evalCtx)
		require.Equal(t, []string{"a", "b"}, value.Value)
		require.Nil(t, err)
	})

	// TODO
	// t.Run("sample_json", func(t *testing.T) {
	// 	value, err := ofClient.ObjectValueDetails(context.Background(), "sample_json", map[string]interface{}{
	// 		"nested": "value",
	// 	}, evalCtx)
	// 	require.Equal(t, []string{"a", "b"}, value.Value)
	// 	require.Nil(t, err)
	// })
}

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
	require.Nil(t, err)

	if !reflect.DeepEqual(result, *expected) {
		t.Errorf("expected %v, got %v", *expected, result)
	}
}

func TestUninitializedProviderStates(t *testing.T) {
	flattenedContext := map[string]interface{}{}

	providerConfig := prefabProvider.ProviderConfig{
		Sources: []string{"datafile://enabled.yaml"},
	}
	uninitializedProvider, _ := prefabProvider.NewProvider(providerConfig)

	boolRes := uninitializedProvider.BooleanEvaluation(context.Background(), "sample_bool", false, flattenedContext)
	require.Equal(t, of.ProviderNotReadyCode, boolRes.ResolutionDetail().ErrorCode)

	intRes := uninitializedProvider.IntEvaluation(context.Background(), "sample_int", 0, flattenedContext)
	require.Equal(t, of.ProviderNotReadyCode, intRes.ResolutionDetail().ErrorCode)

	floatRes := uninitializedProvider.FloatEvaluation(context.Background(), "sample_float", 0, flattenedContext)
	require.Equal(t, of.ProviderNotReadyCode, floatRes.ResolutionDetail().ErrorCode)

	strRes := uninitializedProvider.StringEvaluation(context.Background(), "sample_string", "default", flattenedContext)
	require.Equal(t, of.ProviderNotReadyCode, strRes.ResolutionDetail().ErrorCode)

	objRes := uninitializedProvider.ObjectEvaluation(context.Background(), "sample_string", "default", flattenedContext)
	require.Equal(t, of.ProviderNotReadyCode, objRes.ResolutionDetail().ErrorCode)
}

func TestErrorProviderStates(t *testing.T) {
	flattenedContext := map[string]interface{}{}

	providerConfig := prefabProvider.ProviderConfig{
		Sources: []string{"datafile://non-existing.yaml"},
	}
	errorProvider, _ := prefabProvider.NewProvider(providerConfig)
	errorProvider.Init(of.EvaluationContext{})

	boolRes := errorProvider.BooleanEvaluation(context.Background(), "sample_bool", false, flattenedContext)
	require.Equal(t, of.GeneralCode, boolRes.ResolutionDetail().ErrorCode)

	intRes := errorProvider.IntEvaluation(context.Background(), "sample_int", 0, flattenedContext)
	require.Equal(t, of.GeneralCode, intRes.ResolutionDetail().ErrorCode)

	floatRes := errorProvider.FloatEvaluation(context.Background(), "sample_float", 0, flattenedContext)
	require.Equal(t, of.GeneralCode, floatRes.ResolutionDetail().ErrorCode)

	strRes := errorProvider.StringEvaluation(context.Background(), "sample_string", "default", flattenedContext)
	require.Equal(t, of.GeneralCode, strRes.ResolutionDetail().ErrorCode)

	objRes := errorProvider.ObjectEvaluation(context.Background(), "sample_string", "default", flattenedContext)
	require.Equal(t, of.GeneralCode, objRes.ResolutionDetail().ErrorCode)

	providerConfig = prefabProvider.ProviderConfig{}
	errorProvider, _ = prefabProvider.NewProvider(providerConfig)
	errorProvider.Init(of.EvaluationContext{})
}

func TestEvaluationMethods(t *testing.T) {
	err := of.SetProvider(provider)
	require.Nil(t, err)

	evalCtx := of.NewEvaluationContext(
		"",
		map[string]interface{}{
			"user.id": "123",
		},
	)

	tests := []struct {
		flag              string
		defaultValue      interface{}
		evalCtx           of.EvaluationContext
		expected          interface{}
		expectedErrorCode of.ErrorCode
	}{
		{"sample_bool", false, evalCtx, true, ""},
		{"sample_double", 0.0, evalCtx, 12.12, ""},
		{"sample_int", int64(42999), evalCtx, int64(123), ""},
		{"sample", "default_value", evalCtx, "test sample value", ""},
		// TODO
		// {"flag_with_a_value", map[string]interface{}{"key": "value999"}, evalCtx, map[string]interface{}{"key1": "value1"}, ""},
		{"sample_list", []string{"fallback1", "fallback2"}, evalCtx, []string{"a", "b"}, ""},
		{"invalid_user_context_bool", false, of.NewEvaluationContext(
			"",
			map[string]interface{}{
				"invalid": "123",
			},
		), false, of.InvalidContextCode},
		{"invalid_user_context_int", int64(43), of.NewEvaluationContext(
			"",
			map[string]interface{}{
				"invalid": "123",
			},
		), int64(43), of.InvalidContextCode},
		{"invalid_user_context_float", 1.2, of.NewEvaluationContext(
			"",
			map[string]interface{}{
				"invalid": "123",
			},
		), 1.2, of.InvalidContextCode},
		{"invalid_user_context_string", "a", of.NewEvaluationContext(
			"",
			map[string]interface{}{
				"invalid": "123",
			},
		), "a", of.InvalidContextCode},
		// {"invalid_user_context_object", "a", of.NewEvaluationContext(
		// 	"",
		// 	map[string]interface{}{
		// 		"invalid": "123",
		// 	},
		// ), "a", of.InvalidContextCode},
		{"empty_context", int64(43), evalCtx, int64(43), ""},
	}

	for _, test := range tests {
		fmt.Println("test: {}", test)
		rt := reflect.TypeOf(test.expected)
		switch rt.Kind() {
		case reflect.Bool:
			res, _ := ofClient.BooleanValueDetails(context.Background(), test.flag, test.defaultValue.(bool), test.evalCtx)
			require.Equal(t, test.expected, res.Value)
			require.Equal(t, test.expectedErrorCode, res.ErrorCode)
		case reflect.Int, reflect.Int8, reflect.Int32, reflect.Int64:
			res, _ := ofClient.IntValueDetails(context.Background(), test.flag, test.defaultValue.(int64), test.evalCtx)
			require.Equal(t, test.expected, res.Value)
			require.Equal(t, test.expectedErrorCode, res.ErrorCode)
		case reflect.Float32, reflect.Float64:
			res, _ := ofClient.FloatValueDetails(context.Background(), test.flag, test.defaultValue.(float64), test.evalCtx)
			require.Equal(t, test.expected, res.Value)
			require.Equal(t, test.expectedErrorCode, res.ErrorCode)
		case reflect.String:
			res, _ := ofClient.StringValueDetails(context.Background(), test.flag, test.defaultValue.(string), test.evalCtx)
			require.Equal(t, test.expected, res.Value)
			require.Equal(t, test.expectedErrorCode, res.ErrorCode)
		default:
			res, _ := ofClient.ObjectValueDetails(context.Background(), test.flag, test.defaultValue, test.evalCtx)
			require.Equal(t, test.expected, res.Value)
			require.Equal(t, test.expectedErrorCode, res.ErrorCode)
		}
	}
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

	of.SetProviderAndWait(provider)
	ofClient = of.NewClient("my-app")

	fmt.Printf("provider: %v\n", provider)

	// Run the tests
	exitCode := m.Run()

	cleanup()

	os.Exit(exitCode)
}
