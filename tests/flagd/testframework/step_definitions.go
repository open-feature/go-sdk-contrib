package testframework

import (
	"context"
	"github.com/cucumber/godog"
	"os"
	"sync"
)

// All type definitions have been moved to types.go for better organization
var scenarioMutex sync.Mutex

// InitializeScenario registers all step definitions for gherkin scenarios
func InitializeScenario(ctx *godog.ScenarioContext) {

	// Configuration steps (existing config_steps.go steps work fine with TestState via context)
	InitializeConfigScenario(ctx)

	// Provider lifecycle steps
	InitializeProviderSteps(ctx)

	// Flag evaluation steps
	InitializeFlagSteps(ctx)

	// Context management steps
	InitializeContextSteps(ctx)

	// Event handling steps
	InitializeEventSteps(ctx)

	// Setup scenario hooks
	ctx.Before(func(ctx context.Context, sc *godog.Scenario) (context.Context, error) {
		scenarioMutex.Lock()
		defer scenarioMutex.Unlock()
		state := &TestState{
			EnvVars:      make(map[string]string),
			EvalContext:  make(map[string]interface{}),
			EventChannel: make(chan EventRecord, 100),
		}
		state.ProviderType = ctx.Value("resolver").(ProviderType)
		state.FlagDir = ctx.Value("flagDir").(string)

		return context.WithValue(ctx, TestStateKey{}, state), nil
	})

	ctx.After(func(ctx context.Context, sc *godog.Scenario, err error) (context.Context, error) {
		scenarioMutex.Lock()
		defer scenarioMutex.Unlock()
		if state, ok := ctx.Value(TestStateKey{}).(*TestState); ok {
			state.clearEvents()
			state.CleanupEnvironmentVariables()
			state.cleanupProvider()
		}
		return ctx, nil
	})
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
