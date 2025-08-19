package testframework

import (
	"context"
	"os"

	"github.com/cucumber/godog"
)

// All type definitions have been moved to types.go for better organization

// InitializeScenario registers all step definitions for gherkin scenarios
func InitializeScenario(ctx *godog.ScenarioContext) {
	state := &TestState{
		EnvVars:      make(map[string]string),
		EvalContext:  make(map[string]interface{}),
		EventChannel: make(chan EventRecord, 100),
	}

	// Configuration steps (existing config_steps.go steps work fine with TestState via context)
	InitializeConfigScenario(ctx, state)

	// Provider lifecycle steps
	initializeProviderSteps(ctx, state)

	// Flag evaluation steps
	initializeFlagSteps(ctx, state)

	// Context management steps
	initializeContextSteps(ctx, state)

	// Event handling steps
	initializeEventSteps(ctx, state)

	// Setup scenario hooks
	ctx.Before(func(ctx context.Context, sc *godog.Scenario) (context.Context, error) {
		// Reset state for each scenario
		state.resetState()
		state.ProviderType = ctx.Value("resolver").(ProviderType)
		state.FlagDir = ctx.Value("flagDir").(string)
		// Store state in context for steps that need it
		return context.WithValue(ctx, TestStateKey{}, state), nil
	})

	ctx.After(func(ctx context.Context, sc *godog.Scenario, err error) (context.Context, error) {
		// Clean up per-scenario state, but keep the container running
		state.CleanupEnvironmentVariables()

		// Properly cleanup provider and client to prevent event contamination
		state.cleanupProvider()

		// Clear events after provider cleanup to ensure no residual events
		state.clearEvents()

		// NOTE: We do NOT stop the container here - it should run for the entire test suite
		return ctx, nil
	})
}

// resetState clears test state between scenarios
func (s *TestState) resetState() {
	s.EnvVars = make(map[string]string)
	s.LastEvaluation = EvaluationResult{}
	s.EvalContext = make(map[string]interface{})
	s.TargetingKey = ""
	s.ConfigError = nil
	s.FlagKey = ""
	s.FlagType = ""
	s.DefaultValue = nil

	// Reset config state
	s.ProviderOptions = []ProviderOption{}
	s.ProviderConfig = ErrorAwareProviderConfiguration{}

	// Create a fresh event channel for this scenario
	// This ensures no events from previous scenarios leak through
	s.EventChannel = make(chan EventRecord, 100)
	s.LastEvent = nil

	// Note: Provider and client cleanup is handled in the After hook
	// to ensure proper shutdown sequencing
	s.Client = nil
	s.Provider = nil
}

// Type conversion utilities are now centralized in utils.go
// Legacy compatibility wrappers
func convertValueForSteps(value string, valueType string) (interface{}, error) {
	return DefaultConverter.ConvertForSteps(value, valueType)
}

// applyDefaults sets default values for TestState fields
func (s *TestState) applyDefaults() {
	if s.EnvVars == nil {
		s.EnvVars = make(map[string]string)
	}
	if s.EvalContext == nil {
		s.EvalContext = make(map[string]interface{})
	}
	if s.EventChannel == nil {
		s.EventChannel = make(chan EventRecord, 100) // Buffered channel to prevent blocking
	}
	if s.ProviderType == 0 {
		s.ProviderType = RPC // Default to RPC
	}
}

// cleanupEnvironmentVariables restores original environment variables
func (s *TestState) CleanupEnvironmentVariables() {
	for envVar, originalValue := range s.EnvVars {
		if originalValue == "" {
			os.Unsetenv(envVar)
		} else {
			os.Setenv(envVar, originalValue)
		}
	}
	s.EnvVars = make(map[string]string)
}

// cleanupProvider properly shuts down the provider and client to prevent event contamination
func (s *TestState) cleanupProvider() {
	// Remove all event handlers from client to prevent lingering events
	if s.Client != nil {
		// Note: OpenFeature Go SDK doesn't have a RemoveAllHandlers method,
		// but setting the client to nil will prevent further event handling
		s.Client = nil
	}

	// Shutdown the provider if it has a shutdown method
	if s.Provider != nil {
		// Try to cast to common provider interfaces that might have shutdown methods
		// This is defensive - not all providers will have explicit shutdown
		if shutdownable, ok := s.Provider.(interface{ Shutdown() }); ok {
			shutdownable.Shutdown()
		}
		s.Provider = nil
	}
}
