package testframework

import (
	"context"
	"errors"
	"fmt"
	"os"
	"reflect"
	"slices"

	"github.com/cucumber/godog"
	flagd "github.com/open-feature/go-sdk-contrib/providers/flagd/pkg"
)

// ignoredOptions a list of options that are currently not supported
var ignoredOptions = []string{
	"deadlineMs",
	"streamDeadlineMs",
	"keepAliveTime",
	"offlinePollIntervalMs",
}

// InitializeConfigScenario initializes the config test scenario
func InitializeConfigScenario(ctx *godog.ScenarioContext) {
	ctx.Step(`^a config was initialized$`, withStateNoArgs((*TestState).aConfigWasInitialized))
	ctx.Step(`^an environment variable "([^"]*)" with value "([^"]*)"$`, withState2Args((*TestState).anEnvironmentVariableWithValue))
	ctx.Step(`^an option "([^"]*)" of type "([^"]*)" with value "([^"]*)"$`, withState3Args((*TestState).anOptionOfTypeWithValue))
	ctx.Step(
		`^the option "([^"]*)" of type "([^"]*)" should have the value "([^"]*)"$`,
		withState3ArgsReturningContext((*TestState).theOptionOfTypeShouldHaveTheValue),
	)
	ctx.Step(`^we should have an error$`, withStateNoArgsReturningContext((*TestState).weShouldHaveAnError))
}

// State methods - these now expect state as first parameter

func (s *TestState) aConfigWasInitialized(ctx context.Context) error {
	opts := s.GenerateOpts()
	providerConfiguration, err := flagd.NewProviderConfiguration(opts)
	s.ProviderConfig = ErrorAwareProviderConfiguration{
		Configuration: providerConfiguration,
		Error:         err,
	}
	return nil
}

func (s *TestState) GenerateOpts() []flagd.ProviderOption {
	providerOptions := s.ProviderOptions
	var opts []flagd.ProviderOption
	for _, providerOption := range providerOptions {
		if !slices.Contains(ignoredOptions, providerOption.Option) {
			opts = append(
				opts,
				genericProviderOption(providerOption.Option, providerOption.ValueType, providerOption.Value),
			)
		}
	}
	return opts
}

func (s *TestState) anEnvironmentVariableWithValue(ctx context.Context, key, value string) error {
	s.EnvVars[key] = os.Getenv(key)
	err := os.Setenv(key, value)
	if err != nil {
		return fmt.Errorf("failed to set environment variable %s: %w", key, err)
	}
	return nil
}

func (s *TestState) anOptionOfTypeWithValue(ctx context.Context, option, valueType, value string) error {
	providerOptions := s.ProviderOptions
	data := ProviderOption{
		Option:    option,
		ValueType: valueType,
		Value:     value,
	}
	s.ProviderOptions = append(providerOptions, data)
	return nil
}

func (s *TestState) theOptionOfTypeShouldHaveTheValue(
	ctx context.Context, option, valueType, expectedValueS string,
) (context.Context, error) {
	if slices.Contains(ignoredOptions, option) {
		return ctx, nil
	}

	errorAwareConfiguration := s.ProviderConfig
	// gherkins null value needs to converted to an empty string
	if expectedValueS == "null" {
		expectedValueS = ""
	}

	config := errorAwareConfiguration.Configuration
	currentValue := reflect.ValueOf(config).Elem().FieldByName(ToFieldName(option))
	converter := NewValueConverter()
	var expectedValue = converter.ConvertToReflectValue(valueType, expectedValueS, currentValue.Type())

	if ValueToString(currentValue) != ValueToString(expectedValue) {
		return ctx, fmt.Errorf(
			"expected config of type '%s' with value '%s', got '%s'",
			valueType,
			expectedValueS,
			fmt.Sprintf("%v", currentValue),
		)
	}
	return ctx, nil
}

func (s *TestState) weShouldHaveAnError(ctx context.Context) (context.Context, error) {
	errorAwareConfiguration := s.ProviderConfig
	if errorAwareConfiguration.Error == nil {
		return ctx, errors.New("configuration check succeeded, but should not")
	}
	return ctx, nil
}

func genericProviderOption(option, valueType, value string) flagd.ProviderOption {
	converter := NewValueConverter()
	return func(p *flagd.ProviderConfiguration) {
		field := reflect.ValueOf(p).Elem().FieldByName(ToFieldName(option))
		field.Set(converter.ConvertToReflectValue(valueType, value, field.Type()))
	}
}
