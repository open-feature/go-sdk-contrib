package optimizely

import (
	"encoding/json"
	"testing"

	"github.com/open-feature/go-sdk/openfeature"
	"github.com/optimizely/go-sdk/v2/pkg/client"
	"github.com/optimizely/go-sdk/v2/pkg/config/datafileprojectconfig/entities"
	coreEntities "github.com/optimizely/go-sdk/v2/pkg/entities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper to build a datafile with common defaults
func buildDatafile(flags []entities.FeatureFlag, rollouts []entities.Rollout) entities.Datafile {
	return entities.Datafile{
		Version:      "4",
		ProjectID:    "12345",
		AccountID:    "67890",
		AnonymizeIP:  true,
		BotFiltering: false,
		FeatureFlags: flags,
		Rollouts:     rollouts,
		Experiments:  []entities.Experiment{},
		Events:       []entities.Event{},
		Audiences:    []entities.Audience{},
		Attributes:   []entities.Attribute{},
		Groups:       []entities.Group{},
	}
}

// Helper to build a simple rollout with one variation
func buildRollout(id string, variables []entities.VariationVariable) entities.Rollout {
	return entities.Rollout{
		ID: "rollout_" + id,
		Experiments: []entities.Experiment{
			{
				ID:          "exp_" + id,
				Key:         "exp_" + id,
				Status:      "Running",
				LayerID:     "layer_" + id,
				AudienceIds: []string{},
				Variations: []entities.Variation{
					{
						ID:             "var_1",
						Key:            "on",
						FeatureEnabled: true,
						Variables:      variables,
					},
				},
				TrafficAllocation: []entities.TrafficAllocation{
					{EntityID: "var_1", EndOfRange: 10000},
				},
				ForcedVariations: map[string]string{},
			},
		},
	}
}

// Helper to create a client from a datafile struct
func createClient(t *testing.T, datafile entities.Datafile) *client.OptimizelyClient {
	jsonData, err := json.Marshal(datafile)
	require.NoError(t, err, "failed to marshal datafile")

	optimizelyClient, err := (&client.OptimizelyFactory{
		Datafile: jsonData,
	}).Client()
	require.NoError(t, err, "failed to create optimizely client")

	return optimizelyClient
}

// Test datafiles built from structs
var (
	datafileNoVars = buildDatafile(
		[]entities.FeatureFlag{
			{
				ID:            "flag_no_vars",
				Key:           "flag_no_vars",
				RolloutID:     "rollout_no_vars",
				ExperimentIDs: []string{},
				Variables:     []entities.Variable{},
			},
		},
		[]entities.Rollout{
			buildRollout("no_vars", []entities.VariationVariable{}),
		},
	)

	datafileSingleString = buildDatafile(
		[]entities.FeatureFlag{
			{
				ID:            "flag_single_string",
				Key:           "flag_single_string",
				RolloutID:     "rollout_single_string",
				ExperimentIDs: []string{},
				Variables: []entities.Variable{
					{ID: "var_string", Key: "my_string", DefaultValue: "default", Type: coreEntities.String},
				},
			},
		},
		[]entities.Rollout{
			buildRollout("single_string", []entities.VariationVariable{
				{ID: "var_string", Value: "hello_world"},
			}),
		},
	)

	datafileSingleInt = buildDatafile(
		[]entities.FeatureFlag{
			{
				ID:            "flag_single_int",
				Key:           "flag_single_int",
				RolloutID:     "rollout_single_int",
				ExperimentIDs: []string{},
				Variables: []entities.Variable{
					{ID: "var_int", Key: "my_int", DefaultValue: "0", Type: coreEntities.Integer},
				},
			},
		},
		[]entities.Rollout{
			buildRollout("single_int", []entities.VariationVariable{
				{ID: "var_int", Value: "42"},
			}),
		},
	)

	datafileSingleFloat = buildDatafile(
		[]entities.FeatureFlag{
			{
				ID:            "flag_single_float",
				Key:           "flag_single_float",
				RolloutID:     "rollout_single_float",
				ExperimentIDs: []string{},
				Variables: []entities.Variable{
					{ID: "var_float", Key: "my_float", DefaultValue: "0.0", Type: coreEntities.Double},
				},
			},
		},
		[]entities.Rollout{
			buildRollout("single_float", []entities.VariationVariable{
				{ID: "var_float", Value: "3.14"},
			}),
		},
	)

	datafileSingleBool = buildDatafile(
		[]entities.FeatureFlag{
			{
				ID:            "flag_single_bool",
				Key:           "flag_single_bool",
				RolloutID:     "rollout_single_bool",
				ExperimentIDs: []string{},
				Variables: []entities.Variable{
					{ID: "var_bool", Key: "my_bool", DefaultValue: "false", Type: coreEntities.Boolean},
				},
			},
		},
		[]entities.Rollout{
			buildRollout("single_bool", []entities.VariationVariable{
				{ID: "var_bool", Value: "true"},
			}),
		},
	)

	datafileMultipleVars = buildDatafile(
		[]entities.FeatureFlag{
			{
				ID:            "flag_multi",
				Key:           "flag_multi",
				RolloutID:     "rollout_multi",
				ExperimentIDs: []string{},
				Variables: []entities.Variable{
					{ID: "var_string", Key: "string_val", DefaultValue: "default", Type: coreEntities.String},
					{ID: "var_int", Key: "int_val", DefaultValue: "0", Type: coreEntities.Integer},
				},
			},
		},
		[]entities.Rollout{
			buildRollout("multi", []entities.VariationVariable{
				{ID: "var_string", Value: "test_value"},
				{ID: "var_int", Value: "42"},
			}),
		},
	)

	datafileDisabled = buildDatafile(
		[]entities.FeatureFlag{
			{
				ID:            "flag_disabled",
				Key:           "disabled_flag",
				RolloutID:     "",
				ExperimentIDs: []string{},
				Variables:     []entities.Variable{},
			},
		},
		[]entities.Rollout{},
	)
)

func TestProvider_BooleanEvaluation_NoVars(t *testing.T) {
	p := NewProvider(createClient(t, datafileNoVars))
	ctx := map[string]any{openfeature.TargetingKey: "user-1"}

	// Test successful evaluation - returns decision.Enabled (true)
	res := p.BooleanEvaluation(t.Context(), "flag_no_vars", false, ctx)
	assert.Empty(t, res.ResolutionError)
	assert.True(t, res.Value)
	assert.Equal(t, openfeature.TargetingMatchReason, res.Reason)

	// Test missing targeting key
	resMissingKey := p.BooleanEvaluation(t.Context(), "flag_no_vars", false, nil)
	assert.NotEmpty(t, resMissingKey.ResolutionError)

	// Test flag not found
	resNotFound := p.BooleanEvaluation(t.Context(), "nonexistent_flag", false, ctx)
	assert.NotEmpty(t, resNotFound.ResolutionError)
	assert.False(t, resNotFound.Value)
}

func TestProvider_BooleanEvaluation_SingleBoolVar(t *testing.T) {
	p := NewProvider(createClient(t, datafileSingleBool))
	ctx := map[string]any{openfeature.TargetingKey: "user-1"}

	res := p.BooleanEvaluation(t.Context(), "flag_single_bool", false, ctx)
	assert.Empty(t, res.ResolutionError)
	assert.True(t, res.Value)
}

func TestProvider_BooleanEvaluation_MultipleVars_Error(t *testing.T) {
	p := NewProvider(createClient(t, datafileMultipleVars))
	ctx := map[string]any{openfeature.TargetingKey: "user-1"}

	res := p.BooleanEvaluation(t.Context(), "flag_multi", false, ctx)
	assert.NotEmpty(t, res.ResolutionError)
	assert.False(t, res.Value)
}

func TestProvider_BooleanEvaluation_TypeMismatch(t *testing.T) {
	p := NewProvider(createClient(t, datafileSingleString))
	ctx := map[string]any{openfeature.TargetingKey: "user-1"}

	res := p.BooleanEvaluation(t.Context(), "flag_single_string", false, ctx)
	assert.NotEmpty(t, res.ResolutionError)
}

func TestProvider_StringEvaluation_SingleStringVar(t *testing.T) {
	p := NewProvider(createClient(t, datafileSingleString))
	ctx := map[string]any{openfeature.TargetingKey: "user-1"}

	res := p.StringEvaluation(t.Context(), "flag_single_string", "default", ctx)
	assert.Empty(t, res.ResolutionError)
	assert.Equal(t, "hello_world", res.Value)

	// Test missing targeting key
	resMissingKey := p.StringEvaluation(t.Context(), "flag_single_string", "", map[string]any{})
	assert.NotEmpty(t, resMissingKey.ResolutionError)

	// Test flag not found
	resNotFound := p.StringEvaluation(t.Context(), "nonexistent_flag", "default", ctx)
	assert.NotEmpty(t, resNotFound.ResolutionError)
}

func TestProvider_StringEvaluation_NoVars_Error(t *testing.T) {
	p := NewProvider(createClient(t, datafileNoVars))
	ctx := map[string]any{openfeature.TargetingKey: "user-1"}

	res := p.StringEvaluation(t.Context(), "flag_no_vars", "default", ctx)
	assert.NotEmpty(t, res.ResolutionError)
}

func TestProvider_StringEvaluation_MultipleVars_Error(t *testing.T) {
	p := NewProvider(createClient(t, datafileMultipleVars))
	ctx := map[string]any{openfeature.TargetingKey: "user-1"}

	res := p.StringEvaluation(t.Context(), "flag_multi", "default", ctx)
	assert.NotEmpty(t, res.ResolutionError)
}

func TestProvider_IntEvaluation_SingleIntVar(t *testing.T) {
	p := NewProvider(createClient(t, datafileSingleInt))
	ctx := map[string]any{openfeature.TargetingKey: "user-1"}

	res := p.IntEvaluation(t.Context(), "flag_single_int", 0, ctx)
	assert.Empty(t, res.ResolutionError)
	assert.Equal(t, int64(42), res.Value)

	// Test missing targeting key
	resMissingKey := p.IntEvaluation(t.Context(), "flag_single_int", 0, map[string]any{})
	assert.NotEmpty(t, resMissingKey.ResolutionError)

	// Test flag not found
	resNotFound := p.IntEvaluation(t.Context(), "nonexistent_flag", 99, map[string]any{
		openfeature.TargetingKey: "user-1",
	})
	assert.NotEmpty(t, resNotFound.ResolutionError)
	assert.Equal(t, int64(99), resNotFound.Value)
}

func TestProvider_IntEvaluation_NoVars_Error(t *testing.T) {
	p := NewProvider(createClient(t, datafileNoVars))
	ctx := map[string]any{openfeature.TargetingKey: "user-1"}

	res := p.IntEvaluation(t.Context(), "flag_no_vars", 0, ctx)
	assert.NotEmpty(t, res.ResolutionError)
}

func TestProvider_IntEvaluation_MultipleVars_Error(t *testing.T) {
	p := NewProvider(createClient(t, datafileMultipleVars))
	ctx := map[string]any{openfeature.TargetingKey: "user-1"}

	res := p.IntEvaluation(t.Context(), "flag_multi", 0, ctx)
	assert.NotEmpty(t, res.ResolutionError)
}

func TestProvider_FloatEvaluation_SingleFloatVar(t *testing.T) {
	p := NewProvider(createClient(t, datafileSingleFloat))
	ctx := map[string]any{openfeature.TargetingKey: "user-1"}

	res := p.FloatEvaluation(t.Context(), "flag_single_float", 0.0, ctx)
	assert.Empty(t, res.ResolutionError)
	assert.Equal(t, 3.14, res.Value)

	// Test missing targeting key
	resMissingKey := p.FloatEvaluation(t.Context(), "flag_single_float", 0.0, map[string]any{})
	assert.NotEmpty(t, resMissingKey.ResolutionError)

	// Test flag not found
	resNotFound := p.FloatEvaluation(t.Context(), "nonexistent_flag", 1.5, map[string]any{
		openfeature.TargetingKey: "user-1",
	})
	assert.NotEmpty(t, resNotFound.ResolutionError)
	assert.Equal(t, 1.5, resNotFound.Value)
}

func TestProvider_FloatEvaluation_NoVars_Error(t *testing.T) {
	p := NewProvider(createClient(t, datafileNoVars))
	ctx := map[string]any{openfeature.TargetingKey: "user-1"}

	res := p.FloatEvaluation(t.Context(), "flag_no_vars", 0.0, ctx)
	assert.NotEmpty(t, res.ResolutionError)
}

func TestProvider_FloatEvaluation_MultipleVars_Error(t *testing.T) {
	p := NewProvider(createClient(t, datafileMultipleVars))
	ctx := map[string]any{openfeature.TargetingKey: "user-1"}

	res := p.FloatEvaluation(t.Context(), "flag_multi", 0.0, ctx)
	assert.NotEmpty(t, res.ResolutionError)
}

func TestProvider_ObjectEvaluation_SingleVar(t *testing.T) {
	p := NewProvider(createClient(t, datafileSingleString))
	ctx := map[string]any{openfeature.TargetingKey: "user-1"}

	res := p.ObjectEvaluation(t.Context(), "flag_single_string", nil, ctx)
	assert.Empty(t, res.ResolutionError)
	assert.Equal(t, "hello_world", res.Value)
}

func TestProvider_ObjectEvaluation_MultipleVars(t *testing.T) {
	p := NewProvider(createClient(t, datafileMultipleVars))
	ctx := map[string]any{openfeature.TargetingKey: "user-1"}

	res := p.ObjectEvaluation(t.Context(), "flag_multi", nil, ctx)
	assert.Empty(t, res.ResolutionError)

	resultMap, ok := res.Value.(map[string]any)
	require.True(t, ok, "expected map[string]any, got %T", res.Value)
	assert.Equal(t, "test_value", resultMap["string_val"])
	assert.Equal(t, 42, resultMap["int_val"])

	// Test missing targeting key
	resMissingKey := p.ObjectEvaluation(t.Context(), "flag_multi", nil, map[string]any{})
	assert.NotEmpty(t, resMissingKey.ResolutionError)

	// Test flag not found
	defaultVal := map[string]any{"default": true}
	resNotFound := p.ObjectEvaluation(t.Context(), "nonexistent_flag", defaultVal, map[string]any{
		openfeature.TargetingKey: "user-1",
	})
	assert.NotEmpty(t, resNotFound.ResolutionError)
}

func TestProvider_ObjectEvaluation_NoVars_Error(t *testing.T) {
	p := NewProvider(createClient(t, datafileNoVars))
	ctx := map[string]any{openfeature.TargetingKey: "user-1"}

	res := p.ObjectEvaluation(t.Context(), "flag_no_vars", nil, ctx)
	assert.NotEmpty(t, res.ResolutionError)
}

func TestProvider_TypeMismatch(t *testing.T) {
	p := NewProvider(createClient(t, datafileSingleString))
	ctx := map[string]any{openfeature.TargetingKey: "user-1"}

	// Try to get string variable as int
	resInt := p.IntEvaluation(t.Context(), "flag_single_string", 0, ctx)
	assert.NotEmpty(t, resInt.ResolutionError)

	// Try to get string variable as float
	resFloat := p.FloatEvaluation(t.Context(), "flag_single_string", 0.0, ctx)
	assert.NotEmpty(t, resFloat.ResolutionError)
}

func TestProvider_DisabledFlag(t *testing.T) {
	p := NewProvider(createClient(t, datafileDisabled))
	ctx := map[string]any{openfeature.TargetingKey: "user-1"}

	res := p.BooleanEvaluation(t.Context(), "disabled_flag", true, ctx)
	assert.Empty(t, res.ResolutionError)
	assert.True(t, res.Value)
	assert.Equal(t, openfeature.DisabledReason, res.Reason)
}
