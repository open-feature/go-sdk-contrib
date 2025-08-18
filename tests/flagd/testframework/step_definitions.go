package testframework

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/cucumber/godog"
	"github.com/open-feature/go-sdk/openfeature"
)

// All type definitions have been moved to types.go for better organization

// InitializeScenario registers all step definitions for gherkin scenarios
func InitializeScenario(ctx *godog.ScenarioContext) {
	state := &TestState{
		EnvVars:       make(map[string]string),
		EvalContext:   make(map[string]interface{}),
		Events:        []EventRecord{},
		EventHandlers: make(map[string]func(openfeature.EventDetails)),
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
		state.cleanupEventHandlers()
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
	s.Events = []EventRecord{}
	s.EventHandlers = make(map[string]func(openfeature.EventDetails))
	s.ConfigError = nil
	s.FlagKey = ""
	s.FlagType = ""
	s.DefaultValue = nil

	// Reset config state
	s.ProviderOptions = []ProviderOption{}
	s.ProviderConfig = ErrorAwareProviderConfiguration{}

	// Properly cleanup provider and client
	if s.Client != nil {
		s.Client = nil
	}
	if s.Provider != nil {
		// Note: We don't shutdown the provider here since it might be shared
		// The OpenFeature SDK will handle cleanup when new providers are set
		s.Provider = nil
	}
}

// Type conversion utilities are now centralized in utils.go
// Legacy compatibility wrappers
func convertValueForSteps(value string, valueType string) (interface{}, error) {
	return DefaultConverter.ConvertForSteps(value, valueType)
}

// Utility functions for event handling
func (s *TestState) addEvent(eventType string, details openfeature.EventDetails) {
	s.Events = append(s.Events, EventRecord{
		Type:      eventType,
		Timestamp: time.Now(),
		Details:   details,
	})
}

// waitForEvents waits for specific events to occur
func (s *TestState) waitForEvents(eventType string, maxWait time.Duration) error {
	timeout := time.After(maxWait)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return fmt.Errorf("timeout waiting for %s event", eventType)
		case <-ticker.C:
			for _, event := range s.Events {
				if event.Type == eventType {
					return nil
				}
			}
		}
	}
}

// assertEventOccurred checks if a specific event occurred
func (s *TestState) assertEventOccurred(eventType string) error {
	for _, event := range s.Events {
		if event.Type == eventType {
			return nil
		}
	}
	return fmt.Errorf("event %s did not occur", eventType)
}

// assertEventCount verifies the count of specific events
func (s *TestState) assertEventCount(eventType string, expectedCount int) error {
	count := 0
	for _, event := range s.Events {
		if event.Type == eventType {
			count++
		}
	}
	if count != expectedCount {
		return fmt.Errorf("expected %d %s events, got %d", expectedCount, eventType, count)
	}
	return nil
}

// applyDefaults sets default values for TestState fields
func (s *TestState) applyDefaults() {
	if s.EnvVars == nil {
		s.EnvVars = make(map[string]string)
	}
	if s.EvalContext == nil {
		s.EvalContext = make(map[string]interface{})
	}
	if s.Events == nil {
		s.Events = []EventRecord{}
	}
	if s.EventHandlers == nil {
		s.EventHandlers = make(map[string]func(openfeature.EventDetails))
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
