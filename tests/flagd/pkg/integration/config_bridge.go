package integration

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"strings"
	
	"github.com/cucumber/godog"
	flagd "github.com/open-feature/go-sdk-contrib/providers/flagd/pkg"
	"github.com/open-feature/go-sdk/openfeature"
)

// initializeConfigStepsWithState creates enhanced config steps that work with TestState
func initializeConfigStepsWithState(ctx *godog.ScenarioContext, state *TestState) {
	// Register the existing config steps first
	InitializeConfigScenario(ctx)
	
	// Add enhanced step definitions that integrate with TestState
	ctx.Step(`^a stable provider is configured$`, state.configureStableProvider)
	ctx.Step(`^the provider configuration should be valid$`, state.validateProviderConfiguration)
	ctx.Step(`^the provider should use resolver type "([^"]*)"$`, state.assertResolverType)
}

// configureStableProvider creates a provider configuration and stores it in TestState
func (s *TestState) configureStableProvider(ctx context.Context) error {
	// Get provider options from the original config system
	providerOptions, ok := ctx.Value(ctxProviderOptionsKey{}).([]providerOption)
	if !ok {
		// No options set, use defaults
		providerOptions = []providerOption{}
	}
	
	// Convert provider options to our TestState format
	for _, opt := range providerOptions {
		convertedValue, err := s.convertConfigValue(opt.valueType, opt.value)
		if err != nil {
			return fmt.Errorf("failed to convert config option %s: %w", opt.option, err)
		}
		
		// Store in our options map using snake_case
		snakeKey := camelToSnake(opt.option)
		s.Options[snakeKey] = convertedValue
		
		// Handle special cases
		if opt.option == "resolver" {
			if resolverType, ok := convertedValue.(ProviderType); ok {
				s.ProviderType = resolverType
			}
		}
	}
	
	// Get error-aware configuration from original system
	errorAwareConfig, ok := ctx.Value(ctxErrorAwareProviderConfigurationKey{}).(errorAwareProviderConfiguration)
	if ok && errorAwareConfig.error != nil {
		s.ConfigError = errorAwareConfig.error
	}
	
	return nil
}

// validateProviderConfiguration checks that the provider configuration is valid
func (s *TestState) validateProviderConfiguration() error {
	if s.ConfigError != nil {
		return fmt.Errorf("provider configuration has error: %w", s.ConfigError)
	}
	
	// Additional validations can be added here
	return nil
}

// assertResolverType verifies the resolver type is as expected
func (s *TestState) assertResolverType(expectedType string) error {
	expected, err := parseResolverType(expectedType)
	if err != nil {
		return fmt.Errorf("invalid expected resolver type: %w", err)
	}
	
	if s.ProviderType != expected {
		return fmt.Errorf("expected resolver type %s, got %s", expected, s.ProviderType)
	}
	
	return nil
}

// convertConfigValue converts config values using the same logic as config.go
func (s *TestState) convertConfigValue(valueType, value string) (interface{}, error) {
	return convertValueForSteps(value, valueType)
}

// Helper functions to integrate with existing config system

// getStateFromContext retrieves TestState from Go context
func getStateFromContext(ctx context.Context) (*TestState, error) {
	state, ok := ctx.Value(testStateKey{}).(*TestState)
	if !ok {
		return nil, fmt.Errorf("TestState not found in context")
	}
	return state, nil
}

// createProviderFromState creates a flagd provider from TestState configuration
func createProviderFromState(state *TestState) (openfeature.FeatureProvider, error) {
	// Convert TestState options back to flagd.ProviderOption format
	var opts []flagd.ProviderOption
	
	for key, value := range state.Options {
		// Convert snake_case back to camelCase for flagd provider
		camelKey := snakeToCamel(key)
		
		// Skip ignored options (from config.go)
		if contains(ignoredOptions, camelKey) {
			continue
		}
		
		// Create generic provider option
		opts = append(opts, createGenericOption(camelKey, value))
	}
	
	// Create provider with options
	provider, err := flagd.NewProvider(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create provider: %w", err)
	}
	return provider, nil
}

// createGenericOption creates a flagd.ProviderOption from key-value pair
func createGenericOption(key string, value interface{}) flagd.ProviderOption {
	return func(p *flagd.ProviderConfiguration) {
		field := reflect.ValueOf(p).Elem().FieldByName(toFieldName(key))
		if field.IsValid() && field.CanSet() {
			fieldValue := reflect.ValueOf(value)
			if fieldValue.Type().ConvertibleTo(field.Type()) {
				field.Set(fieldValue.Convert(field.Type()))
			}
		}
	}
}

// snakeToCamel converts snake_case to camelCase
func snakeToCamel(input string) string {
	parts := strings.Split(input, "_")
	if len(parts) == 1 {
		return parts[0]
	}
	
	result := parts[0]
	for _, part := range parts[1:] {
		if len(part) > 0 {
			result += strings.ToUpper(part[:1]) + part[1:]
		}
	}
	return result
}

// contains checks if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// Enhanced step to set environment variables with TestState integration
func (s *TestState) setEnvironmentVariableWithState(envVar, value string) error {
	// Store original value for cleanup
	if originalValue, exists := os.LookupEnv(envVar); exists {
		s.EnvVars[envVar] = originalValue
	} else {
		s.EnvVars[envVar] = ""
	}
	
	// Set the environment variable
	return os.Setenv(envVar, value)
}

// cleanupEnvironmentVariables restores original environment variables
func (s *TestState) cleanupEnvironmentVariables() {
	for envVar, originalValue := range s.EnvVars {
		if originalValue == "" {
			os.Unsetenv(envVar)
		} else {
			os.Setenv(envVar, originalValue)
		}
	}
	s.EnvVars = make(map[string]string)
}

// Enhanced configuration validation
func (s *TestState) validateConfigurationWithContext(ctx context.Context) error {
	// Get the error-aware configuration from context
	errorAwareConfig, ok := ctx.Value(ctxErrorAwareProviderConfigurationKey{}).(errorAwareProviderConfiguration)
	if ok {
		s.ConfigError = errorAwareConfig.error
		if errorAwareConfig.configuration != nil {
			// Extract configuration values into our TestState
			s.extractConfigurationValues(errorAwareConfig.configuration)
		}
	}
	
	return s.validateProviderConfiguration()
}

// extractConfigurationValues copies values from flagd.ProviderConfiguration to TestState
func (s *TestState) extractConfigurationValues(config *flagd.ProviderConfiguration) {
	configValue := reflect.ValueOf(config).Elem()
	configType := configValue.Type()
	
	for i := 0; i < configValue.NumField(); i++ {
		field := configValue.Field(i)
		fieldType := configType.Field(i)
		
		if field.IsValid() && field.CanInterface() {
			snakeKey := camelToSnake(fieldType.Name)
			s.Options[snakeKey] = field.Interface()
		}
	}
}