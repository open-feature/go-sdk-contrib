package integration

import (
	"context"
	"errors"
	"fmt"

	"github.com/cucumber/godog"
	flagd "github.com/open-feature/go-sdk-contrib/providers/flagd/pkg"
	"github.com/open-feature/go-sdk/pkg/openfeature"
)

// ctxKeyKey is the key used to pass flag key across context.Context
type ctxKeyKey struct{}

// ctxKeyKey is the key used to pass the default across context.Context
type ctxDefaultKey struct{}

// ctxValueKey is the key used to pass the value across context.Context
type ctxValueKey struct{}

func InitializeFlagdJsonScenario(pOptions ...flagd.ProviderOption) func(*godog.ScenarioContext) {

	providerOptions = pOptions

	return initializeFlagdJsonScenario
}

// initializeFlagdJsonScenario initializes the flagd json evaluator test scenario
func initializeFlagdJsonScenario(ctx *godog.ScenarioContext) {
	ctx.Step(`^a flagd provider is set$`, aFlagdProviderIsSet)
	ctx.Step(`^a string flag with key "([^"]*)" is evaluated with default value "([^"]*)"$`, aFlagdStringFlagWithKeyIsEvaluatedWithDefaultValue)
	ctx.Step(`^a context containing a key "([^"]*)", with value "([^"]*)"$`, aContextContainingAKeyWithValue)
	ctx.Step(`^a context containing a nested property with outer key "([^"]*)" and inner key "([^"]*)", with value "([^"]*)"$`, aContextContainingANestedPropertyWithOuterKeyAndInnerKeyWithValue)
	ctx.Step(`^the returned value should be "([^"]*)"$`, theReturnedValueShouldBe)
}

func aFlagdStringFlagWithKeyIsEvaluatedWithDefaultValue(ctx context.Context, key, defaultValue string) (context.Context, error) {
	ctx = context.WithValue(ctx, ctxKeyKey{}, key)
	ctx = context.WithValue(ctx, ctxDefaultKey{}, defaultValue)
	return ctx, nil
}

func aContextContainingAKeyWithValue(ctx context.Context, evalContextKey, evalContextValue string) (context.Context, error) {
	client := ctx.Value(ctxClientKey{}).(*openfeature.Client)
	key := ctx.Value(ctxKeyKey{}).(string)
	defaultValue := ctx.Value(ctxDefaultKey{}).(string)
	ec := openfeature.NewEvaluationContext("", map[string]interface{}{
		evalContextKey: evalContextValue,
	})
	got, err := client.StringValue(ctx, key, defaultValue, ec)
	if err != nil {
		return ctx, fmt.Errorf("error: %w", err)
	}
	return context.WithValue(ctx, ctxValueKey{}, got), nil
}

func aContextContainingANestedPropertyWithOuterKeyAndInnerKeyWithValue(ctx context.Context, outerKey, innerKey, name string) (context.Context, error) {
	client := ctx.Value(ctxClientKey{}).(*openfeature.Client)
	key := ctx.Value(ctxKeyKey{}).(string)
	defaultValue := ctx.Value(ctxDefaultKey{}).(string)
	ec := openfeature.NewEvaluationContext("", map[string]interface{}{
		outerKey: map[string]interface{}{
			innerKey: name,
		},
	})
	got, err := client.StringValue(ctx, key, defaultValue, ec)
	if err != nil {
		return ctx, fmt.Errorf("error: %w", err)
	}
	return context.WithValue(ctx, ctxValueKey{}, got), nil
}

func theReturnedValueShouldBe(ctx context.Context, expectedValue string) (context.Context, error) {
	got, ok := ctx.Value(ctxValueKey{}).(string)
	if !ok {
		return ctx, errors.New("no flag resolution result")
	}
	if got != expectedValue {
		return ctx, fmt.Errorf("expected resolved int value to be %s, got %s", expectedValue, got)
	}
	return ctx, nil
}
