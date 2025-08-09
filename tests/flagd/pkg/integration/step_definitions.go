package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/cucumber/godog"
	"github.com/open-feature/go-sdk/openfeature"
)

// TestStateKey is the key used to pass TestState across context.Context
type TestStateKey struct{}

// EvaluationResult holds the result of flag evaluation in a generic way
type EvaluationResult struct {
	FlagKey      string
	Value        interface{}
	Reason       openfeature.Reason
	Variant      string
	ErrorCode    openfeature.ErrorCode
	ErrorMessage string
}

// TestState holds all test state shared across step definitions
type TestState struct {
	// Provider configuration
	EnvVars      map[string]string
	ProviderType ProviderType
	Provider     openfeature.FeatureProvider
	Client       *openfeature.Client
	ConfigError  error

	// Configuration testing state
	ProviderOptions []providerOption
	ProviderConfig  errorAwareProviderConfiguration

	// Evaluation state
	LastEvaluation EvaluationResult
	EvalContext    map[string]interface{}
	FlagKey        string
	FlagType       string
	DefaultValue   interface{}

	// Event tracking
	Events        []EventRecord
	EventHandlers map[string]func(openfeature.EventDetails)

	// Container/testbed state
	Container    TestContainer
	LaunchpadURL string
}

// EventRecord tracks events for verification
type EventRecord struct {
	Type      string
	Timestamp time.Time
	Details   openfeature.EventDetails
}

// ProviderType represents the type of provider being tested
type ProviderType int

const (
	RPC ProviderType = iota
	InProcess
	File
)

func (p ProviderType) String() string {
	switch p {
	case RPC:
		return "rpc"
	case InProcess:
		return "in-process"
	case File:
		return "file"
	default:
		return "unknown"
	}
}

// TestContainer interface abstracts container operations
type TestContainer interface {
	GetHost() string
	GetPort(service string) int
	GetLaunchpadURL() string
	Start() error
	Stop() error
	Restart(delaySeconds int) error
	IsHealthy() bool
}

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
	s.Events = []EventRecord{}
	s.EventHandlers = make(map[string]func(openfeature.EventDetails))
	s.ConfigError = nil
	s.FlagKey = ""
	s.FlagType = ""
	s.DefaultValue = nil

	// Reset config state
	s.ProviderOptions = []providerOption{}
	s.ProviderConfig = errorAwareProviderConfiguration{}

	if s.Provider != nil {
		s.Provider = nil
	}
	s.Client = nil
}

// Type conversion utilities (similar to Python implementation)
func convertValueForSteps(value string, valueType string) (interface{}, error) {
	switch valueType {
	case "Boolean":
		return strconv.ParseBool(strings.ToLower(value))
	case "Integer":
		// Return int64 to match OpenFeature IntValueDetails return type
		return strconv.ParseInt(value, 10, 64)
	case "Long":
		return strconv.ParseInt(value, 10, 64)
	case "Float":
		return strconv.ParseFloat(value, 64)
	case "String":
		if value == "null" {
			return nil, nil
		}
		return value, nil
	case "Object":
		var obj interface{}
		err := json.Unmarshal([]byte(value), &obj)
		return obj, err
	case "ResolverType":
		return parseResolverType(value)
	case "CacheType":
		return parseCacheType(value)
	default:
		return value, nil
	}
}

// parseResolverType converts string to ProviderType
func parseResolverType(value string) (ProviderType, error) {
	switch strings.ToLower(value) {
	case "rpc":
		return RPC, nil
	case "in-process":
		return InProcess, nil
	case "file":
		return File, nil
	default:
		return RPC, fmt.Errorf("unknown resolver type: %s", value)
	}
}

// parseCacheType handles cache type conversion
func parseCacheType(value string) (string, error) {
	switch strings.ToLower(value) {
	case "lru", "disabled":
		return strings.ToLower(value), nil
	default:
		return "", fmt.Errorf("unknown cache type: %s", value)
	}
}

// camelToSnake converts CamelCase to snake_case
func camelToSnake(input string) string {
	var result strings.Builder
	for i, r := range input {
		if i > 0 && 'A' <= r && r <= 'Z' {
			result.WriteRune('_')
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
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
