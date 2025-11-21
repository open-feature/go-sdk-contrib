package optimizely

import (
	"testing"

	"github.com/open-feature/go-sdk/openfeature"
	"github.com/optimizely/go-sdk/v2/pkg/client"
)

const testDatafile = `
{
  "version": "4",
  "rollouts": [
    {
      "id": "rollout_1",
      "experiments": [
        {
          "id": "exp_1",
          "key": "exp_1",
          "status": "Running",
          "layerId": "layer_1",
          "audienceIds": [],
          "variations": [
            {
              "id": "var_1",
              "key": "on",
              "featureEnabled": true,
              "variables": [
                {"id": "var_string", "value": "test_value"},
                {"id": "var_int", "value": "42"},
                {"id": "var_float", "value": "3.14"},
                {"id": "var_json", "value": "{\"nested\": \"object\"}"},
                {"id": "var_bool", "value": "true"}
              ]
            }
          ],
          "trafficAllocation": [
            {"entityId": "var_1", "endOfRange": 10000}
          ],
          "forcedVariations": {}
        }
      ]
    }
  ],
  "experiments": [],
  "featureFlags": [
    {
      "id": "flag_1",
      "key": "test_flag",
      "rolloutId": "rollout_1",
      "experimentIds": [],
      "variables": [
        {"id": "var_string", "key": "value", "defaultValue": "default", "type": "string"},
        {"id": "var_int", "key": "int_value", "defaultValue": "0", "type": "integer"},
        {"id": "var_float", "key": "float_value", "defaultValue": "0.0", "type": "double"},
        {"id": "var_json", "key": "json_value", "defaultValue": "{}", "type": "json"},
        {"id": "var_bool", "key": "bool_value", "defaultValue": "false", "type": "boolean"}
      ]
    },
    {
      "id": "flag_2",
      "key": "disabled_flag",
      "rolloutId": "",
      "experimentIds": [],
      "variables": []
    }
  ],
  "events": [],
  "audiences": [],
  "attributes": [],
  "groups": [],
  "projectId": "12345",
  "accountId": "67890",
  "anonymizeIP": true,
  "botFiltering": false
}
`

func TestProvider_Metadata(t *testing.T) {
	p := NewProvider(nil)
	if p.Metadata().Name != "Optimizely" {
		t.Errorf("expected metadata name 'Optimizely', got %s", p.Metadata().Name)
	}
}

func TestProvider_BooleanEvaluation(t *testing.T) {
	optimizelyClient, err := (&client.OptimizelyFactory{
		Datafile: []byte(testDatafile),
	}).Client()
	if err != nil {
		t.Fatalf("failed to create optimizely client: %v", err)
	}

	p := NewProvider(optimizelyClient)
	flattenedCtx := map[string]any{
		openfeature.TargetingKey: "user-1",
		"variableKey":            "bool_value",
	}

	// Test successful evaluation
	res := p.BooleanEvaluation(t.Context(), "test_flag", false, flattenedCtx)
	if res.ResolutionError != (openfeature.ResolutionError{}) {
		t.Errorf("expected no resolution error, got %v", res.ResolutionError)
	}
	if res.Value != true {
		t.Errorf("expected true, got %v", res.Value)
	}

	// Test missing targeting key
	resMissingKey := p.BooleanEvaluation(t.Context(), "test_flag", false, map[string]any{
		"variableKey": "bool_value",
	})
	if resMissingKey.ResolutionError == (openfeature.ResolutionError{}) {
		t.Errorf("expected targeting key missing error, got nil")
	}

	// Test flag not found
	resNotFound := p.BooleanEvaluation(t.Context(), "nonexistent_flag", false, flattenedCtx)
	if resNotFound.ResolutionError == (openfeature.ResolutionError{}) {
		t.Errorf("expected flag not found error, got nil")
	}
	if resNotFound.Value != false {
		t.Errorf("expected default value false, got %v", resNotFound.Value)
	}
}

func TestProvider_StringEvaluation(t *testing.T) {
	optimizelyClient, err := (&client.OptimizelyFactory{
		Datafile: []byte(testDatafile),
	}).Client()
	if err != nil {
		t.Fatalf("failed to create optimizely client: %v", err)
	}

	p := NewProvider(optimizelyClient)
	flattenedCtx := map[string]any{
		openfeature.TargetingKey: "user-1",
	}

	// Test successful evaluation with default variableKey
	res := p.StringEvaluation(t.Context(), "test_flag", "default", flattenedCtx)
	if res.ResolutionError != (openfeature.ResolutionError{}) {
		t.Errorf("expected no resolution error, got %v", res.ResolutionError)
	}
	if res.Value != "test_value" {
		t.Errorf("expected 'test_value', got %s", res.Value)
	}

	// Test missing targeting key
	resMissingKey := p.StringEvaluation(t.Context(), "test_flag", "", map[string]any{})
	if resMissingKey.ResolutionError == (openfeature.ResolutionError{}) {
		t.Errorf("expected targeting key missing error, got nil")
	}

	// Test flag not found
	resNotFound := p.StringEvaluation(t.Context(), "nonexistent_flag", "default", flattenedCtx)
	if resNotFound.ResolutionError == (openfeature.ResolutionError{}) {
		t.Errorf("expected flag not found error, got nil")
	}
}

func TestProvider_IntEvaluation(t *testing.T) {
	optimizelyClient, err := (&client.OptimizelyFactory{
		Datafile: []byte(testDatafile),
	}).Client()
	if err != nil {
		t.Fatalf("failed to create optimizely client: %v", err)
	}

	p := NewProvider(optimizelyClient)

	// Test with custom variableKey
	flattenedCtx := map[string]any{
		openfeature.TargetingKey: "user-1",
		"variableKey":            "int_value",
	}

	res := p.IntEvaluation(t.Context(), "test_flag", 0, flattenedCtx)
	if res.ResolutionError != (openfeature.ResolutionError{}) {
		t.Errorf("expected no resolution error, got %v", res.ResolutionError)
	}
	if res.Value != 42 {
		t.Errorf("expected 42, got %d", res.Value)
	}

	// Test missing targeting key
	resMissingKey := p.IntEvaluation(t.Context(), "test_flag", 0, map[string]any{})
	if resMissingKey.ResolutionError == (openfeature.ResolutionError{}) {
		t.Errorf("expected targeting key missing error, got nil")
	}

	// Test flag not found
	resNotFound := p.IntEvaluation(t.Context(), "nonexistent_flag", 99, map[string]any{
		openfeature.TargetingKey: "user-1",
	})
	if resNotFound.ResolutionError == (openfeature.ResolutionError{}) {
		t.Errorf("expected flag not found error, got nil")
	}
	if resNotFound.Value != 99 {
		t.Errorf("expected default value 99, got %d", resNotFound.Value)
	}
}

func TestProvider_FloatEvaluation(t *testing.T) {
	optimizelyClient, err := (&client.OptimizelyFactory{
		Datafile: []byte(testDatafile),
	}).Client()
	if err != nil {
		t.Fatalf("failed to create optimizely client: %v", err)
	}

	p := NewProvider(optimizelyClient)

	// Test with custom variableKey
	flattenedCtx := map[string]any{
		openfeature.TargetingKey: "user-1",
		"variableKey":            "float_value",
	}

	res := p.FloatEvaluation(t.Context(), "test_flag", 0.0, flattenedCtx)
	if res.ResolutionError != (openfeature.ResolutionError{}) {
		t.Errorf("expected no resolution error, got %v", res.ResolutionError)
	}
	if res.Value != 3.14 {
		t.Errorf("expected 3.14, got %f", res.Value)
	}

	// Test missing targeting key
	resMissingKey := p.FloatEvaluation(t.Context(), "test_flag", 0.0, map[string]any{})
	if resMissingKey.ResolutionError == (openfeature.ResolutionError{}) {
		t.Errorf("expected targeting key missing error, got nil")
	}

	// Test flag not found
	resNotFound := p.FloatEvaluation(t.Context(), "nonexistent_flag", 1.5, map[string]any{
		openfeature.TargetingKey: "user-1",
	})
	if resNotFound.ResolutionError == (openfeature.ResolutionError{}) {
		t.Errorf("expected flag not found error, got nil")
	}
	if resNotFound.Value != 1.5 {
		t.Errorf("expected default value 1.5, got %f", resNotFound.Value)
	}
}

func TestProvider_ObjectEvaluation(t *testing.T) {
	optimizelyClient, err := (&client.OptimizelyFactory{
		Datafile: []byte(testDatafile),
	}).Client()
	if err != nil {
		t.Fatalf("failed to create optimizely client: %v", err)
	}

	p := NewProvider(optimizelyClient)

	// Test with default variableKey
	flattenedCtx := map[string]any{
		openfeature.TargetingKey: "user-1",
	}

	res := p.ObjectEvaluation(t.Context(), "test_flag", nil, flattenedCtx)
	if res.ResolutionError != (openfeature.ResolutionError{}) {
		t.Errorf("expected no resolution error, got %v", res.ResolutionError)
	}
	if res.Value != "test_value" {
		t.Errorf("expected 'test_value', got %v", res.Value)
	}

	// Test missing targeting key
	resMissingKey := p.ObjectEvaluation(t.Context(), "test_flag", nil, map[string]any{})
	if resMissingKey.ResolutionError == (openfeature.ResolutionError{}) {
		t.Errorf("expected targeting key missing error, got nil")
	}

	// Test flag not found
	defaultVal := map[string]any{"default": true}
	resNotFound := p.ObjectEvaluation(t.Context(), "nonexistent_flag", defaultVal, map[string]any{
		openfeature.TargetingKey: "user-1",
	})
	if resNotFound.ResolutionError == (openfeature.ResolutionError{}) {
		t.Errorf("expected flag not found error, got nil")
	}
}

func TestProvider_TypeMismatch(t *testing.T) {
	optimizelyClient, err := (&client.OptimizelyFactory{
		Datafile: []byte(testDatafile),
	}).Client()
	if err != nil {
		t.Fatalf("failed to create optimizely client: %v", err)
	}

	p := NewProvider(optimizelyClient)
	flattenedCtx := map[string]any{
		openfeature.TargetingKey: "user-1",
	}

	// Try to get string variable as int (should fail with type mismatch)
	res := p.IntEvaluation(t.Context(), "test_flag", 0, flattenedCtx)
	if res.ResolutionError == (openfeature.ResolutionError{}) {
		t.Errorf("expected type mismatch error when getting string as int")
	}

	// Try to get string variable as float (should fail with type mismatch)
	resFloat := p.FloatEvaluation(t.Context(), "test_flag", 0.0, flattenedCtx)
	if resFloat.ResolutionError == (openfeature.ResolutionError{}) {
		t.Errorf("expected type mismatch error when getting string as float")
	}
}

func TestProvider_CustomVariableKey(t *testing.T) {
	optimizelyClient, err := (&client.OptimizelyFactory{
		Datafile: []byte(testDatafile),
	}).Client()
	if err != nil {
		t.Fatalf("failed to create optimizely client: %v", err)
	}

	p := NewProvider(optimizelyClient)

	// Test custom variableKey for string
	flattenedCtx := map[string]any{
		openfeature.TargetingKey: "user-1",
		"variableKey":            "int_value",
	}

	// Get int_value as string should fail (type mismatch)
	res := p.StringEvaluation(t.Context(), "test_flag", "default", flattenedCtx)
	if res.ResolutionError == (openfeature.ResolutionError{}) {
		t.Errorf("expected type mismatch error when getting int as string")
	}
}

func TestProvider_Hooks(t *testing.T) {
	p := NewProvider(nil)
	hooks := p.Hooks()
	if len(hooks) != 0 {
		t.Errorf("expected empty hooks slice, got %d hooks", len(hooks))
	}
}

func TestProvider_StateHandler(t *testing.T) {
	optimizelyClient, err := (&client.OptimizelyFactory{
		Datafile: []byte(testDatafile),
	}).Client()
	if err != nil {
		t.Fatalf("failed to create optimizely client: %v", err)
	}

	p := NewProvider(optimizelyClient)

	// Test Init
	if err := p.Init(openfeature.EvaluationContext{}); err != nil {
		t.Errorf("expected Init to return nil, got %v", err)
	}

	// Test Status
	if p.Status() != openfeature.ReadyState {
		t.Errorf("expected ReadyState, got %v", p.Status())
	}

	// Test Shutdown
	p.Shutdown()
}
