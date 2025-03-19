package integration

import (
	"context"
	"errors"
	"fmt"
	"github.com/cucumber/godog"
	flagd "github.com/open-feature/go-sdk-contrib/providers/flagd/pkg"
	"slices"
	"strconv"
	"strings"
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

// setEnvVar is a function to define env vars to be used to create provider configurations
var setEnvVar func(key, value string)

// valueProvider is a struct with functions to obtain the current and expected configuration value
type valueProvider struct {
	currentValue  func(config *flagd.ProviderConfiguration) interface{}
	expectedValue func(value string) interface{}
}

// ignoredOptions a list of options that are currently not supported
var ignoredOptions = []string{
	"deadlineMs",
	"streamDeadlineMs",
	"keepAliveTime",
	"retryBackoffMs",
	"retryBackoffMaxMs",
	"retryGracePeriod",
	"offlinePollIntervalMs",
	"tls", // tls option needs to be ignored because the env var FLAGD_TLS is not processed
}

// valueGeneratorMap a map that defines the value generators for supported options
var valueGeneratorMap = map[string]valueProvider{
	"resolver": {
		currentValue:  func(config *flagd.ProviderConfiguration) interface{} { return config.Resolver },
		expectedValue: func(value string) interface{} { return flagd.ResolverType(strings.ToLower(value)) },
	},
	"port": {
		currentValue:  func(config *flagd.ProviderConfiguration) interface{} { return config.Port },
		expectedValue: func(value string) interface{} { return uint16(stringToInt(value)) },
	},
	"host": {
		currentValue: func(config *flagd.ProviderConfiguration) interface{} { return config.Host },
	},
	"targetUri": {
		currentValue: func(config *flagd.ProviderConfiguration) interface{} { return config.TargetUri },
	},
	"certPath": {
		currentValue: func(config *flagd.ProviderConfiguration) interface{} { return config.CertificatePath },
	},
	"socketPath": {
		currentValue: func(config *flagd.ProviderConfiguration) interface{} { return config.SocketPath },
	},
	"cache": {
		currentValue: func(config *flagd.ProviderConfiguration) interface{} {
			return fmt.Sprintf(
				"%s",
				config.CacheType,
			)
		},
	},
	"selector": {
		currentValue: func(config *flagd.ProviderConfiguration) interface{} { return config.Selector },
	},
	"maxCacheSize": {
		currentValue:  func(config *flagd.ProviderConfiguration) interface{} { return config.MaxCacheSize },
		expectedValue: func(value string) interface{} { return stringToInt(value) },
	},
	"offlineFlagSourcePath": {
		currentValue: func(config *flagd.ProviderConfiguration) interface{} {
			return config.OfflineFlagSourcePath
		},
	},
}

// providerOptionGeneratorMap a map that defines the provider options generators for supported options
var providerOptionGeneratorMap = map[string]func(value string) (flagd.ProviderOption, error){
	"resolver": func(value string) (flagd.ProviderOption, error) {
		switch strings.ToLower(value) {
		case "rpc":
			return flagd.WithRPCResolver(), nil
		case "in-process":
			return flagd.WithInProcessResolver(), nil
		case "file":
			return flagd.WithFileResolver(), nil
		}
		return nil, fmt.Errorf("invalid resolver '%s'", value)
	},
	"offlineFlagSourcePath": func(value string) (flagd.ProviderOption, error) {
		return flagd.WithOfflineFilePath(value), nil
	},
	"host": func(value string) (flagd.ProviderOption, error) {
		return flagd.WithHost(value), nil
	},
	"port": func(value string) (flagd.ProviderOption, error) {
		return flagd.WithPort(uint16(stringToInt(value))), nil
	},
	"targetUri": func(value string) (flagd.ProviderOption, error) {
		return flagd.WithTargetUri(value), nil
	},
	"certPath": func(value string) (flagd.ProviderOption, error) {
		return flagd.WithCertificatePath(value), nil
	},
	"socketPath": func(value string) (flagd.ProviderOption, error) {
		return flagd.WithSocketPath(value), nil
	},
	"selector": func(value string) (flagd.ProviderOption, error) {
		return flagd.WithSelector(value), nil
	},
	"cache": func(value string) (flagd.ProviderOption, error) {
		switch strings.ToLower(value) {
		case "lru":
			return flagd.WithLRUCache(2500), nil
		case "mem":
			return flagd.WithBasicInMemoryCache(), nil
		case "disabled":

			return flagd.WithoutCache(), nil
		}
		return nil, fmt.Errorf("invalid cache type '%s'", value)
	},
	"maxCacheSize": func(value string) (flagd.ProviderOption, error) {
		return flagd.WithLRUCache(stringToInt(value)), nil
	},
}

// InitializeConfigTestSuite register provider supplier and register test steps
func InitializeConfigTestSuite(setEnvVarFunc func(key, value string)) func(*godog.TestSuiteContext) {
	setEnvVar = setEnvVarFunc

	return func(suiteContext *godog.TestSuiteContext) {}
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
			optionSupplier, ok := providerOptionGeneratorMap[providerOption.option]

			if !ok {
				return ctx, fmt.Errorf(
					"invalid config with option '%s' with type '%s' and value '%s'",
					providerOption.option,
					providerOption.valueType,
					providerOption.value,
				)
			}

			option, err := optionSupplier(providerOption.value)

			if err != nil {
				return ctx, err
			}

			opts = append(opts, option)
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

	valueGenerator, ok := valueGeneratorMap[option]

	if !ok {
		return ctx, fmt.Errorf(
			"invalid option '%s' with type '%s' and value '%s'",
			option,
			valueType,
			expectedValueS,
		)
	}

	currentValue := valueGenerator.currentValue(config)

	var expectedValue interface{} = expectedValueS
	if valueGenerator.expectedValue != nil {
		expectedValue = valueGenerator.expectedValue(expectedValueS)
	}

	if currentValue != expectedValue {
		return ctx, fmt.Errorf(
			"expected response of type '%s' with value '%s', got '%s'",
			valueType,
			expectedValueS,
			currentValue,
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

func stringToBool(str string) bool {
	b, err := strconv.ParseBool(str)
	if err != nil {
		panic(err)
	}

	return b
}
