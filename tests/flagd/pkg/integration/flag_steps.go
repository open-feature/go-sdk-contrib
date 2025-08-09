package integration

import (
	"fmt"
	"time"
	
	"github.com/cucumber/godog"
	"github.com/open-feature/go-sdk/openfeature"
)

// initializeFlagSteps registers flag evaluation step definitions
func initializeFlagSteps(ctx *godog.ScenarioContext, state *TestState) {
	ctx.Step(`^a ([^-]*)-flag with key "([^"]*)" and a default value "([^"]*)"$`, state.setFlagForEvaluation)
	ctx.Step(`^the flag was evaluated with details$`, state.evaluateFlagWithDetails)
	ctx.Step(`^the resolved details value should be "([^"]*)"$`, state.assertResolvedValue)
	ctx.Step(`^the reason should be "([^"]*)"$`, state.assertReason)
	ctx.Step(`^the error-code should be "([^"]*)"$`, state.assertErrorCode)
	ctx.Step(`^the flag should be part of the event payload$`, state.assertFlagInEventPayload)
	ctx.Step(`^the flag was modified$`, state.modifyFlag)
	ctx.Step(`^a change event was fired$`, state.triggerChangeEvent)
}

// setFlagForEvaluation prepares a flag for evaluation
func (s *TestState) setFlagForEvaluation(flagType, flagKey, defaultValue string) error {
	s.FlagType = flagType
	s.FlagKey = flagKey
	
	// Convert the default value based on flag type
	converted, err := s.convertDefaultValue(flagType, defaultValue)
	if err != nil {
		return fmt.Errorf("failed to convert default value: %w", err)
	}
	
	s.DefaultValue = converted
	return nil
}

// convertDefaultValue converts string default value to appropriate type
func (s *TestState) convertDefaultValue(flagType, value string) (interface{}, error) {
	switch flagType {
	case "Boolean":
		return convertValue(value, "Boolean")
	case "String":
		return value, nil
	case "Integer":
		return convertValue(value, "Integer")
	case "Float":
		return convertValue(value, "Float")
	case "Object":
		return convertValue(value, "Object")
	default:
		return nil, fmt.Errorf("unknown flag type: %s", flagType)
	}
}

// evaluateFlagWithDetails evaluates the current flag with details
func (s *TestState) evaluateFlagWithDetails() error {
	if s.Client == nil {
		return fmt.Errorf("no client available for evaluation")
	}
	
	if s.FlagKey == "" {
		return fmt.Errorf("no flag key set for evaluation")
	}
	
	// Create evaluation context from current context map
	evalCtx := openfeature.NewEvaluationContext()
	for key, value := range s.EvalContext {
		evalCtx = evalCtx.WithValue(key, value)
	}
	
	// Evaluate based on flag type
	var details openfeature.EvaluationDetails
	var err error
	
	switch s.FlagType {
	case "Boolean":
		if defaultVal, ok := s.DefaultValue.(bool); ok {
			details = s.Client.BooleanValueDetails(s.FlagKey, defaultVal, evalCtx)
		} else {
			return fmt.Errorf("default value is not a boolean")
		}
	case "String":
		if defaultVal, ok := s.DefaultValue.(string); ok {
			details = s.Client.StringValueDetails(s.FlagKey, defaultVal, evalCtx)
		} else {
			return fmt.Errorf("default value is not a string")
		}
	case "Integer":
		if defaultVal, ok := s.DefaultValue.(int); ok {
			details = s.Client.IntValueDetails(s.FlagKey, defaultVal, evalCtx)
		} else {
			return fmt.Errorf("default value is not an integer")
		}
	case "Float":
		if defaultVal, ok := s.DefaultValue.(float64); ok {
			details = s.Client.FloatValueDetails(s.FlagKey, defaultVal, evalCtx)
		} else {
			return fmt.Errorf("default value is not a float")
		}
	case "Object":
		if defaultVal := s.DefaultValue; defaultVal != nil {
			details = s.Client.ObjectValueDetails(s.FlagKey, defaultVal, evalCtx)
		} else {
			return fmt.Errorf("default value is not an object")
		}
	default:
		return fmt.Errorf("unknown flag type: %s", s.FlagType)
	}
	
	s.LastEvaluation = details
	return err
}

// assertResolvedValue checks that the resolved value matches expected
func (s *TestState) assertResolvedValue(expectedValue string) error {
	if s.LastEvaluation.FlagKey == "" {
		return fmt.Errorf("no evaluation details available")
	}
	
	// Convert expected value to appropriate type for comparison
	expected, err := s.convertDefaultValue(s.FlagType, expectedValue)
	if err != nil {
		return fmt.Errorf("failed to convert expected value: %w", err)
	}
	
	actualValue := s.LastEvaluation.Value
	
	// Handle special cases for zero values and empty strings
	if s.FlagType == "String" && expectedValue == "" {
		if actualValue != "" {
			return fmt.Errorf("expected empty string, got: %v", actualValue)
		}
		return nil
	}
	
	if actualValue != expected {
		return fmt.Errorf("expected value %v, got %v", expected, actualValue)
	}
	
	return nil
}

// assertReason checks that the evaluation reason matches expected
func (s *TestState) assertReason(expectedReason string) error {
	if s.LastEvaluation.FlagKey == "" {
		return fmt.Errorf("no evaluation details available")
	}
	
	actualReason := string(s.LastEvaluation.Reason)
	if actualReason != expectedReason {
		return fmt.Errorf("expected reason %s, got %s", expectedReason, actualReason)
	}
	
	return nil
}

// assertErrorCode checks that the error code matches expected
func (s *TestState) assertErrorCode(expectedCode string) error {
	if s.LastEvaluation.FlagKey == "" {
		return fmt.Errorf("no evaluation details available")
	}
	
	// If no error code is expected, ensure no error occurred
	if expectedCode == "" {
		if s.LastEvaluation.ErrorCode != "" {
			return fmt.Errorf("expected no error, but got error code: %s", s.LastEvaluation.ErrorCode)
		}
		return nil
	}
	
	if string(s.LastEvaluation.ErrorCode) != expectedCode {
		return fmt.Errorf("expected error code %s, got %s", expectedCode, s.LastEvaluation.ErrorCode)
	}
	
	return nil
}

// assertFlagInEventPayload checks that the current flag is in the latest change event
func (s *TestState) assertFlagInEventPayload() error {
	// Find the most recent CONFIGURATION_CHANGE event
	for i := len(s.Events) - 1; i >= 0; i-- {
		event := s.Events[i]
		if event.Type == "CONFIGURATION_CHANGE" {
			// Check if the flag key is in the event details
			if event.Details.FlagNames != nil {
				for _, flagName := range event.Details.FlagNames {
					if flagName == s.FlagKey {
						return nil
					}
				}
			}
			return fmt.Errorf("flag %s not found in change event payload", s.FlagKey)
		}
	}
	
	return fmt.Errorf("no configuration change event found")
}

// modifyFlag modifies the current flag (typically by calling testbed API)
func (s *TestState) modifyFlag() error {
	if s.Container == nil {
		return fmt.Errorf("no container available to modify flags")
	}
	
	// This would typically call the testbed launchpad API to trigger flag changes
	// Implementation depends on the specific container/testbed interface
	return nil
}

// triggerChangeEvent triggers a flag change event
func (s *TestState) triggerChangeEvent() error {
	// Add change event handler if not already present
	if _, exists := s.EventHandlers["CONFIGURATION_CHANGE"]; !exists {
		handler := func(details openfeature.EventDetails) {
			s.addEvent("CONFIGURATION_CHANGE", details)
		}
		
		s.EventHandlers["CONFIGURATION_CHANGE"] = handler
		s.Client.AddHandler(openfeature.ProviderConfigChange, &handler)
	}
	
	// Wait a moment for the change to propagate
	return s.waitForEvents("CONFIGURATION_CHANGE", 2*time.Second)
}

// Helper methods for flag evaluation
func (s *TestState) evaluateBooleanFlag(flagKey string, defaultValue bool, context openfeature.EvaluationContext) (bool, error) {
	details := s.Client.BooleanValueDetails(flagKey, defaultValue, context)
	s.LastEvaluation = details
	
	if details.ErrorCode != "" {
		return defaultValue, fmt.Errorf("evaluation error: %s", details.ErrorMessage)
	}
	
	if value, ok := details.Value.(bool); ok {
		return value, nil
	}
	
	return defaultValue, fmt.Errorf("evaluation did not return boolean value")
}

func (s *TestState) evaluateStringFlag(flagKey string, defaultValue string, context openfeature.EvaluationContext) (string, error) {
	details := s.Client.StringValueDetails(flagKey, defaultValue, context)
	s.LastEvaluation = details
	
	if details.ErrorCode != "" {
		return defaultValue, fmt.Errorf("evaluation error: %s", details.ErrorMessage)
	}
	
	if value, ok := details.Value.(string); ok {
		return value, nil
	}
	
	return defaultValue, fmt.Errorf("evaluation did not return string value")
}

func (s *TestState) evaluateIntegerFlag(flagKey string, defaultValue int, context openfeature.EvaluationContext) (int, error) {
	details := s.Client.IntValueDetails(flagKey, defaultValue, context)
	s.LastEvaluation = details
	
	if details.ErrorCode != "" {
		return defaultValue, fmt.Errorf("evaluation error: %s", details.ErrorMessage)
	}
	
	if value, ok := details.Value.(int); ok {
		return value, nil
	}
	
	return defaultValue, fmt.Errorf("evaluation did not return integer value")
}

func (s *TestState) evaluateFloatFlag(flagKey string, defaultValue float64, context openfeature.EvaluationContext) (float64, error) {
	details := s.Client.FloatValueDetails(flagKey, defaultValue, context)
	s.LastEvaluation = details
	
	if details.ErrorCode != "" {
		return defaultValue, fmt.Errorf("evaluation error: %s", details.ErrorMessage)
	}
	
	if value, ok := details.Value.(float64); ok {
		return value, nil
	}
	
	return defaultValue, fmt.Errorf("evaluation did not return float value")
}