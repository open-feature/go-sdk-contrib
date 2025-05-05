package integration

import (
	"context"
	"errors"
	"github.com/cucumber/godog"
	"time"

	"github.com/open-feature/go-sdk/openfeature"
)

var test_provider_supplier func() openfeature.FeatureProvider

// setEnvVar is a function to define env vars to be used to create provider configurations
var setEnvVar func(key, value string)

// ctxStorageKey is the key used to pass test data across context.Context
type ctxStorageKey struct{}

// ctxClientKey is the key used to pass the openfeature client across context.Context
type ctxClientKey struct{}

var domain = "flagd-e2e-tests"

// RegisterProviderSupplier register provider supplier and register test steps
func RegisterProviderSupplier(providerSupplier func() openfeature.FeatureProvider) {
	test_provider_supplier = providerSupplier
}

// RegisterSetEnvVarFunc register function to set env vars
func RegisterSetEnvVarFunc(setEnvVarFunc func(key, value string)) {
	setEnvVar = setEnvVarFunc
}

func InitializeGenericScenario(ctx *godog.ScenarioContext) {
	ctx.Step(`^a flagd provider is set$`, aFlagdProviderIsSet)

	InitializeConfigScenario(ctx)
	InitializeFlagdJsonScenario(ctx)
}

func aFlagdProviderIsSet(ctx context.Context) (context.Context, error) {
	readyChan := make(chan struct{})

	err := openfeature.SetNamedProvider(domain, test_provider_supplier())
	if err != nil {
		return nil, err
	}

	callBack := func(details openfeature.EventDetails) {
		// emit readiness
		close(readyChan)
	}

	client := openfeature.NewClient(domain)
	client.AddHandler(openfeature.ProviderReady, &callBack)

	select {
	case <-readyChan:
	case <-time.After(500 * time.Millisecond):
		return ctx, errors.New("provider not ready after 500 milliseconds")
	}

	return context.WithValue(ctx, ctxClientKey{}, client), nil
}
