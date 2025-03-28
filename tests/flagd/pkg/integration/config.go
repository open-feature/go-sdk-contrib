package integration

import (
	"context"
	"errors"
	"fmt"
	"github.com/cucumber/godog"
	flagd "github.com/open-feature/go-sdk-contrib/providers/flagd/pkg"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"unicode"
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

// ignoredOptions a list of options that are currently not supported and cannot be ignored with tags
var ignoredOptions = []string{
	"deadlineMs",
}

// InitializeConfigScenario initializes the config test scenario
func InitializeConfigScenario(ctx *godog.ScenarioContext) {
	ctx.Step(`^a config was initialized$`, aConfigWasInitialized)
	ctx.Step(`^an environment variable "([^"]*)" with value "([^"]*)"$`, anEnvironmentVariableWithValue)
	ctx.Step(`^an option "([^"]*)" of type "([^"]*)" with value "([^"]*)"$`, anOptionOfTypeWithValue)
	ctx.Step(
		`^the option "([^"]*)" of type "([^"]*)" should have the value "([^"]*)"$`,
		theOptionOfTypeShouldHaveTheValue,
	)
	ctx.Step(`^we should have an error$`, weShouldHaveAnError)
}

func aConfigWasInitialized(ctx context.Context) (context.Context, error) {
	providerOptions, _ := ctx.Value(ctxProviderOptionsKey{}).([]providerOption)

	var opts []flagd.ProviderOption

	for _, providerOption := range providerOptions {
		if !slices.Contains(ignoredOptions, providerOption.option) {
			opts = append(
				opts,
				genericProviderOption(providerOption.option, providerOption.valueType, providerOption.value),
			)
		}
	}

	providerConfiguration, err := flagd.NewProviderConfiguration(opts)

	errorAwareProviderConfiguration := errorAwareProviderConfiguration{
		configuration: providerConfiguration,
		error:         err,
	}

	return context.WithValue(ctx, ctxErrorAwareProviderConfigurationKey{}, errorAwareProviderConfiguration), nil
}

func anEnvironmentVariableWithValue(key, value string) {
	setEnvVar(key, value)
}

func anOptionOfTypeWithValue(ctx context.Context, option, valueType, value string) context.Context {
	providerOptions, _ := ctx.Value(ctxProviderOptionsKey{}).([]providerOption)

	data := providerOption{
		option:    option,
		valueType: valueType,
		value:     value,
	}

	providerOptions = append(providerOptions, data)

	return context.WithValue(ctx, ctxProviderOptionsKey{}, providerOptions)
}

func theOptionOfTypeShouldHaveTheValue(
	ctx context.Context, option, valueType, expectedValueS string,
) (context.Context, error) {
	if slices.Contains(ignoredOptions, option) {
		return ctx, nil
	}

	errorAwareConfiguration, ok := ctx.Value(ctxErrorAwareProviderConfigurationKey{}).(errorAwareProviderConfiguration)
	if !ok {
		return ctx, errors.New("no errorAwareProviderConfiguration available")
	}

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

func weShouldHaveAnError(ctx context.Context) (context.Context, error) {
	errorAwareConfiguration, ok := ctx.Value(ctxErrorAwareProviderConfigurationKey{}).(errorAwareProviderConfiguration)
	if !ok {
		return ctx, errors.New("no ProviderConfiguration found")
	}

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
