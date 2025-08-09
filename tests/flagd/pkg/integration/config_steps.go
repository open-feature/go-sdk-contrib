package integration

import (
	"context"
	"errors"
	"fmt"
	"os"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"github.com/cucumber/godog"
	flagd "github.com/open-feature/go-sdk-contrib/providers/flagd/pkg"
)

// providerOption is a struct to store the defined options between steps
type providerOption struct {
	option    string
	valueType string
	value     string
}

// errorAwareProviderConfiguration is a struct that contains a ProviderConfiguration and an error in case the
// configuration is invalid
type errorAwareProviderConfiguration struct {
	configuration *flagd.ProviderConfiguration
	error         error
}

// ctxProviderOptionsKey is the key used to pass the provider options across tests
type ctxProviderOptionsKey struct{}

// ctxErrorAwareProviderConfigurationKey is the key used to pass the errorAwareProviderConfiguration across tests
type ctxErrorAwareProviderConfigurationKey struct{}

// ignoredOptions a list of options that are currently not supported
var ignoredOptions = []string{
	"deadlineMs",
	"streamDeadlineMs",
	"keepAliveTime",
	"retryBackoffMs",
	"retryBackoffMaxMs",
	"retryGracePeriod",
	"offlinePollIntervalMs",
}

// InitializeConfigScenario initializes the config test scenario
func InitializeConfigScenario(ctx *godog.ScenarioContext, state *TestState) {
	ctx.Step(`^a config was initialized$`, state.aConfigWasInitialized)
	ctx.Step(`^an environment variable "([^"]*)" with value "([^"]*)"$`, state.anEnvironmentVariableWithValue)
	ctx.Step(`^an option "([^"]*)" of type "([^"]*)" with value "([^"]*)"$`, state.anOptionOfTypeWithValue)
	ctx.Step(
		`^the option "([^"]*)" of type "([^"]*)" should have the value "([^"]*)"$`,
		state.theOptionOfTypeShouldHaveTheValue,
	)
	ctx.Step(`^we should have an error$`, state.weShouldHaveAnError)
}

func (s *TestState) aConfigWasInitialized(ctx context.Context) {
	opts := s.GenerateOpts()

	providerConfiguration, err := flagd.NewProviderConfiguration(opts)

	s.ProviderConfig = errorAwareProviderConfiguration{
		configuration: providerConfiguration,
		error:         err,
	}
}

func (s *TestState) GenerateOpts() []flagd.ProviderOption {
	providerOptions := s.ProviderOptions

	var opts []flagd.ProviderOption

	for _, providerOption := range providerOptions {
		if !slices.Contains(ignoredOptions, providerOption.option) {
			opts = append(
				opts,
				genericProviderOption(providerOption.option, providerOption.valueType, providerOption.value),
			)
		}
	}
	return opts
}

func (s *TestState) anEnvironmentVariableWithValue(key, value string) {

	s.EnvVars[key] = os.Getenv(key)
	err := os.Setenv(key, value)
	if err != nil {
		panic(err)
	}
}

func (s *TestState) anOptionOfTypeWithValue(ctx context.Context, option, valueType, value string) {
	providerOptions := s.ProviderOptions

	data := providerOption{
		option:    option,
		valueType: valueType,
		value:     value,
	}

	s.ProviderOptions = append(providerOptions, data)
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

	config := errorAwareConfiguration.configuration
	currentValue := reflect.ValueOf(config).Elem().FieldByName(toFieldName(option))

	var expectedValue = convertValue(valueType, expectedValueS, currentValue.Type())

	if valueToString(currentValue) != valueToString(expectedValue) {
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

	if errorAwareConfiguration.error == nil {
		return ctx, errors.New("configuration check succeeded, but should not")
	} else {
		return ctx, nil
	}
}

func stringToInt(str string) int {
	i, err := strconv.Atoi(str)
	if err != nil {
		panic(err)
	}
	return i
}

func stringToBoolean(str string) bool {
	b, err := strconv.ParseBool(str)
	if err != nil {
		panic(err)
	}
	return b
}

func valueToString(value interface{}) string {
	return fmt.Sprintf("%v", value)
}

func toFieldName(option string) string {
	r := []rune(option)
	return string(append([]rune{unicode.ToUpper(r[0])}, r[1:]...))
}

func genericProviderOption(option, valueType, value string) flagd.ProviderOption {
	return func(p *flagd.ProviderConfiguration) {
		field := reflect.ValueOf(p).Elem().FieldByName(toFieldName(option))

		field.Set(convertValue(valueType, value, field.Type()))
	}
}

func convertValue(valueType, value string, fieldType reflect.Type) reflect.Value {
	if valueType == "Integer" {
		return reflect.ValueOf(stringToInt(value)).Convert(fieldType)
	} else if valueType == "Boolean" {
		return reflect.ValueOf(stringToBoolean(value)).Convert(fieldType)
	} else if valueType == "ResolverType" {
		return reflect.ValueOf(flagd.ResolverType(strings.ToLower(value))).Convert(fieldType)
	} else {
		return reflect.ValueOf(value).Convert(fieldType)
	}
}
