package optimizely_test

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/open-feature/go-sdk/openfeature"
	"github.com/optimizely/go-sdk/v2/pkg/client"
	"github.com/optimizely/go-sdk/v2/pkg/config/datafileprojectconfig/entities"
	coreEntities "github.com/optimizely/go-sdk/v2/pkg/entities"

	optimizely "github.com/open-feature/go-sdk-contrib/providers/optimizely"
)

// buildExampleDatafile creates a datafile with various flag configurations for examples.
func buildExampleDatafile() []byte {
	datafile := entities.Datafile{
		Version:      "4",
		ProjectID:    "12345",
		AccountID:    "67890",
		AnonymizeIP:  true,
		BotFiltering: false,
		FeatureFlags: []entities.FeatureFlag{
			{
				ID:            "flag_bool",
				Key:           "feature_enabled",
				RolloutID:     "rollout_bool",
				ExperimentIDs: []string{},
				Variables:     []entities.Variable{},
			},
			{
				ID:            "flag_string",
				Key:           "welcome_message",
				RolloutID:     "rollout_string",
				ExperimentIDs: []string{},
				Variables: []entities.Variable{
					{ID: "var_string", Key: "message", DefaultValue: "Hello", Type: coreEntities.String},
				},
			},
			{
				ID:            "flag_config",
				Key:           "ui_config",
				RolloutID:     "rollout_config",
				ExperimentIDs: []string{},
				Variables: []entities.Variable{
					{ID: "var_color", Key: "button_color", DefaultValue: "blue", Type: coreEntities.String},
					{ID: "var_size", Key: "font_size", DefaultValue: "14", Type: coreEntities.Integer},
				},
			},
		},
		Rollouts: []entities.Rollout{
			{
				ID: "rollout_bool",
				Experiments: []entities.Experiment{{
					ID: "exp_bool", Key: "exp_bool", Status: "Running", LayerID: "layer_bool",
					AudienceIds: []string{},
					Variations: []entities.Variation{{
						ID: "var_1", Key: "on", FeatureEnabled: true, Variables: []entities.VariationVariable{},
					}},
					TrafficAllocation:  []entities.TrafficAllocation{{EntityID: "var_1", EndOfRange: 10000}},
					ForcedVariations: map[string]string{},
				}},
			},
			{
				ID: "rollout_string",
				Experiments: []entities.Experiment{{
					ID: "exp_string", Key: "exp_string", Status: "Running", LayerID: "layer_string",
					AudienceIds: []string{},
					Variations: []entities.Variation{{
						ID: "var_1", Key: "on", FeatureEnabled: true,
						Variables: []entities.VariationVariable{{ID: "var_string", Value: "Welcome to our app!"}},
					}},
					TrafficAllocation:  []entities.TrafficAllocation{{EntityID: "var_1", EndOfRange: 10000}},
					ForcedVariations: map[string]string{},
				}},
			},
			{
				ID: "rollout_config",
				Experiments: []entities.Experiment{{
					ID: "exp_config", Key: "exp_config", Status: "Running", LayerID: "layer_config",
					AudienceIds: []string{},
					Variations: []entities.Variation{{
						ID: "var_1", Key: "on", FeatureEnabled: true,
						Variables: []entities.VariationVariable{
							{ID: "var_color", Value: "green"},
							{ID: "var_size", Value: "16"},
						},
					}},
					TrafficAllocation:  []entities.TrafficAllocation{{EntityID: "var_1", EndOfRange: 10000}},
					ForcedVariations: map[string]string{},
				}},
			},
		},
		Experiments: []entities.Experiment{},
		Events:      []entities.Event{},
		Audiences:   []entities.Audience{},
		Attributes:  []entities.Attribute{},
		Groups:      []entities.Group{},
	}
	data, _ := json.Marshal(datafile)
	return data
}

// Example demonstrates basic usage of the Optimizely provider.
func Example() {
	// Create Optimizely client (using datafile for demonstration)
	optimizelyClient, err := (&client.OptimizelyFactory{
		Datafile: buildExampleDatafile(),
	}).Client()
	if err != nil {
		panic(err)
	}

	// Set up OpenFeature with the Optimizely provider
	provider := optimizely.NewProvider(optimizelyClient)
	err = openfeature.SetProviderAndWait(provider)
	if err != nil {
		panic(err)
	}
	defer openfeature.Shutdown()

	// Create client and evaluation context
	ofClient := openfeature.NewDefaultClient()
	evalCtx := openfeature.NewEvaluationContext("user-123", nil)

	// Boolean evaluation for a flag with no variables (returns enabled state)
	enabled, _ := ofClient.BooleanValue(context.Background(), "feature_enabled", false, evalCtx)
	fmt.Printf("Feature enabled: %v\n", enabled)

	// String evaluation for a flag with one string variable
	message, _ := ofClient.StringValue(context.Background(), "welcome_message", "Hello", evalCtx)
	fmt.Printf("Welcome message: %s\n", message)

	// Object evaluation for a flag with multiple variables
	config, _ := ofClient.ObjectValue(context.Background(), "ui_config", nil, evalCtx)
	configMap := config.(map[string]any)
	fmt.Printf("Button color: %s, Font size: %d\n", configMap["button_color"], configMap["font_size"])

	// Output:
	// Feature enabled: true
	// Welcome message: Welcome to our app!
	// Button color: green, Font size: 16
}

// Example_booleanFlag demonstrates evaluating a flag with no variables.
func Example_booleanFlag() {
	optimizelyClient, _ := (&client.OptimizelyFactory{
		Datafile: buildExampleDatafile(),
	}).Client()

	provider := optimizely.NewProvider(optimizelyClient)
	_ = openfeature.SetProviderAndWait(provider)
	defer openfeature.Shutdown()

	ofClient := openfeature.NewDefaultClient()
	evalCtx := openfeature.NewEvaluationContext("user-123", nil)

	// Flags with 0 variables return the enabled state
	enabled, _ := ofClient.BooleanValue(context.Background(), "feature_enabled", false, evalCtx)
	fmt.Printf("enabled: %v\n", enabled)

	// Output:
	// enabled: true
}

// Example_singleVariableFlag demonstrates evaluating a flag with one variable.
func Example_singleVariableFlag() {
	optimizelyClient, _ := (&client.OptimizelyFactory{
		Datafile: buildExampleDatafile(),
	}).Client()

	provider := optimizely.NewProvider(optimizelyClient)
	_ = openfeature.SetProviderAndWait(provider)
	defer openfeature.Shutdown()

	ofClient := openfeature.NewDefaultClient()
	evalCtx := openfeature.NewEvaluationContext("user-123", nil)

	// Flags with 1 variable return the variable's value
	message, _ := ofClient.StringValue(context.Background(), "welcome_message", "default", evalCtx)
	fmt.Printf("message: %s\n", message)

	// Output:
	// message: Welcome to our app!
}

// Example_multipleVariablesFlag demonstrates evaluating a flag with multiple variables.
func Example_multipleVariablesFlag() {
	optimizelyClient, _ := (&client.OptimizelyFactory{
		Datafile: buildExampleDatafile(),
	}).Client()

	provider := optimizely.NewProvider(optimizelyClient)
	_ = openfeature.SetProviderAndWait(provider)
	defer openfeature.Shutdown()

	ofClient := openfeature.NewDefaultClient()
	evalCtx := openfeature.NewEvaluationContext("user-123", nil)

	// Flags with multiple variables return a map of all variable names to values
	config, _ := ofClient.ObjectValue(context.Background(), "ui_config", nil, evalCtx)
	configMap := config.(map[string]any)
	fmt.Printf("button_color: %s\n", configMap["button_color"])
	fmt.Printf("font_size: %d\n", configMap["font_size"])

	// Output:
	// button_color: green
	// font_size: 16
}
