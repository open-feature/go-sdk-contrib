package integration

import (
	"context"
	"errors"
	"fmt"

	"github.com/cucumber/godog"
	"github.com/open-feature/go-sdk/openfeature"
)

// ctxKeyKey is the key used to pass flag key across tests
type ctxKeyKey struct{}

// ctxKeyKey is the key used to pass the default value across tests
type ctxDefaultKey struct{}

// ctxEvaluationCtxKey is the key used to pass openfeature evaluation context across tests
type ctxEvaluationCtxKey struct{}

// ctxReasonKey is the key used to pass the evaluation reason across tests
type ctxReasonKey struct{}

// InitializeFlagdJsonScenario initializes the flagd json evaluator test scenario
func InitializeFlagdJsonScenario(ctx *godog.ScenarioContext) {
	ctx.Step(`^a string flag with key "([^"]*)" is evaluated with default value "([^"]*)"$`, aFlagdStringFlagWithKeyIsEvaluatedWithDefaultValue)
	ctx.Step(`^an integer flag with key "([^"]*)" is evaluated with default value (\d+)$`, aFlagdIntegerFlagWithKeyIsEvaluatedWithDefaultValue)

	ctx.Step(`^a context containing a key "([^"]*)", with value "([^"]*)"$`, aContextContainingAKeyWitStringValue)
	ctx.Step(`^a context containing a key "([^"]*)", with value (\d+)$`, aContextContainingAKeyWithIntValue)
	ctx.Step(`^a context containing a targeting key with value "([^"]*)"$`, aContextContainingATargetingKey)
	ctx.Step(`^a context containing a nested property with outer key "([^"]*)" and inner key "([^"]*)", with value "([^"]*)"$`, aContextContainingANestedPropertyWithOuterKeyAndInnerKeyWithStringValue)
	ctx.Step(`^a context containing a nested property with outer key "([^"]*)" and inner key "([^"]*)", with value (\d+)$`, aContextContainingANestedPropertyWithOuterKeyAndInnerKeyWithIntValue)

	ctx.Step(`^the returned value should be "([^"]*)"$`, theReturnedValueShouldBeString)
	ctx.Step(`^the returned value should be (\d+)$`, theReturnedValueShouldBeInt)
	ctx.Step(`^the returned value should be -(\d+)$`, theReturnedValueShouldBeNegativeInt)

	ctx.Step(`^the returned reason should be "([^"]*)"$`, theReturnedReasonShouldBe)
}

// setup

func aFlagdStringFlagWithKeyIsEvaluatedWithDefaultValue(ctx context.Context, key, defaultValue string) (context.Context, error) {
	ctx = context.WithValue(ctx, ctxKeyKey{}, key)
	ctx = context.WithValue(ctx, ctxDefaultKey{}, defaultValue)
	return ctx, nil
}

func aFlagdIntegerFlagWithKeyIsEvaluatedWithDefaultValue(ctx context.Context, key string, defaultValue int64) (context.Context, error) {
	ctx = context.WithValue(ctx, ctxKeyKey{}, key)
	ctx = context.WithValue(ctx, ctxDefaultKey{}, defaultValue)
	return ctx, nil
}

// set contexts

func aContextContainingAKeyWitStringValue(ctx context.Context, evalContextKey, evalContextValue string) (context.Context, error) {
	evalCtx := openfeature.NewEvaluationContext("", map[string]interface{}{
		evalContextKey: evalContextValue,
	})

	return context.WithValue(ctx, ctxEvaluationCtxKey{}, evalCtx), nil
}

func aContextContainingAKeyWithIntValue(ctx context.Context, evalContextKey string, evalContextValue int64) (context.Context, error) {
	evalCtx := openfeature.NewEvaluationContext("", map[string]interface{}{
		evalContextKey: evalContextValue,
	})

	return context.WithValue(ctx, ctxEvaluationCtxKey{}, evalCtx), nil
}

func aContextContainingATargetingKey(ctx context.Context, targetingKet string) (context.Context, error) {
	evalCtx := openfeature.NewEvaluationContext(targetingKet, map[string]interface{}{})

	return context.WithValue(ctx, ctxEvaluationCtxKey{}, evalCtx), nil
}

func aContextContainingANestedPropertyWithOuterKeyAndInnerKeyWithStringValue(ctx context.Context, outerKey, innerKey, value string) (context.Context, error) {
	evalCtx := openfeature.NewEvaluationContext("", map[string]interface{}{
		outerKey: map[string]interface{}{
			innerKey: value,
		},
	})

	return context.WithValue(ctx, ctxEvaluationCtxKey{}, evalCtx), nil
}

func aContextContainingANestedPropertyWithOuterKeyAndInnerKeyWithIntValue(ctx context.Context, outerKey string, innerKey string, value int) (context.Context, error) {
	evalCtx := openfeature.NewEvaluationContext("", map[string]interface{}{
		outerKey: map[string]interface{}{
			innerKey: value,
		},
	})

	return context.WithValue(ctx, ctxEvaluationCtxKey{}, evalCtx), nil
}

// validate

func theReturnedValueShouldBeString(ctx context.Context, expectedValue string) (context.Context, error) {
	client := ctx.Value(ctxClientKey{}).(*openfeature.Client)
	key := ctx.Value(ctxKeyKey{}).(string)
	defaultValue := ctx.Value(ctxDefaultKey{}).(string)

	var evalCtx openfeature.EvaluationContext
	if ctx.Value(ctxEvaluationCtxKey{}) != nil {
		evalCtx = ctx.Value(ctxEvaluationCtxKey{}).(openfeature.EvaluationContext)
	}

	// error from evaluation are ignored as we only check for detail content
	details, _ := client.StringValueDetails(ctx, key, defaultValue, evalCtx)

	if details.Value != expectedValue {
		return ctx, fmt.Errorf("expected resolved int value to be %s, got %s", expectedValue, details.Value)
	}

	return context.WithValue(ctx, ctxReasonKey{}, details.Reason), nil
}

func theReturnedValueShouldBeInt(ctx context.Context, expectedValue int64) (context.Context, error) {
	return validateInteger(ctx, expectedValue, false)
}

func theReturnedValueShouldBeNegativeInt(ctx context.Context, expectedValue int64) (context.Context, error) {
	return validateInteger(ctx, expectedValue, true)
}

func validateInteger(ctx context.Context, expectedValue int64, isNegative bool) (context.Context, error) {
	client := ctx.Value(ctxClientKey{}).(*openfeature.Client)
	key := ctx.Value(ctxKeyKey{}).(string)
	defaultValue := ctx.Value(ctxDefaultKey{}).(int64)

	var evalCtx openfeature.EvaluationContext
	if ctx.Value(ctxEvaluationCtxKey{}) != nil {
		evalCtx = ctx.Value(ctxEvaluationCtxKey{}).(openfeature.EvaluationContext)
	}

	// error from evaluation are ignored as we only check for detail content
	details, _ := client.IntValueDetails(ctx, key, defaultValue, evalCtx)

	if isNegative {
		expectedValue = -(expectedValue)
	}

	if details.Value != expectedValue {
		return ctx, fmt.Errorf("expected resolved int value to be %d, got %d", expectedValue, details.Value)
	}

	return context.WithValue(ctx, ctxReasonKey{}, details.Reason), nil
}

func theReturnedReasonShouldBe(ctx context.Context, expectedReason string) (context.Context, error) {
	evaluatedReason, ok := ctx.Value(ctxReasonKey{}).(openfeature.Reason)
	if !ok {
		return ctx, errors.New("no flag resolution reason set")
	}

	if string(evaluatedReason) != expectedReason {
		return ctx, fmt.Errorf("expected resolved int value to be %s, got %s", expectedReason, evaluatedReason)
	}
	return ctx, nil
}
