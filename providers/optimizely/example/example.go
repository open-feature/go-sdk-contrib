package main

import (
	"context"
	"fmt"
	"os"

	"github.com/open-feature/go-sdk/openfeature"
	"github.com/optimizely/go-sdk/v2/pkg/client"

	optimizely "github.com/open-feature/go-sdk-contrib/providers/optimizely"
)

func main() {
	ctx := context.Background()
	sdkKey := os.Getenv("OPTIMIZELY_SDK_KEY")
	if sdkKey == "" {
		fmt.Println("OPTIMIZELY_SDK_KEY environment variable is required")
		os.Exit(1)
	}

	optimizelyClient, err := (&client.OptimizelyFactory{
		SDKKey: sdkKey,
	}).Client()
	if err != nil {
		panic(err)
	}

	provider := optimizely.NewProvider(optimizelyClient)
	err = openfeature.SetProviderAndWait(provider)
	if err != nil {
		panic(err)
	}
	defer openfeature.Shutdown()

	ofClient := openfeature.NewClient("my-app")

	const newConst = "user_123"

	// Boolean evaluation
	boolCtx := openfeature.NewEvaluationContext(newConst, map[string]any{
		"variableKey": "boolean_variable",
	})
	boolValue, err := ofClient.BooleanValue(ctx, "flag", false, boolCtx)
	if err != nil {
		fmt.Printf("boolean evaluation error: %v\n", err)
	} else {
		fmt.Printf("boolean value: %v\n", boolValue)
	}

	// String evaluation
	stringCtx := openfeature.NewEvaluationContext(newConst, map[string]any{
		"variableKey": "string_variable",
	})
	stringValue, err := ofClient.StringValue(ctx, "flag", "default", stringCtx)
	if err != nil {
		fmt.Printf("string evaluation error: %v\n", err)
	} else {
		fmt.Printf("string value: %s\n", stringValue)
	}

	// Int evaluation with custom variableKey
	intCtx := openfeature.NewEvaluationContext(newConst, map[string]any{
		"variableKey": "integer_variable",
	})
	intValue, err := ofClient.IntValue(ctx, "flag", 10, intCtx)
	if err != nil {
		fmt.Printf("int evaluation error: %v\n", err)
	} else {
		fmt.Printf("int value: %d\n", intValue)
	}

	// Float evaluation with custom variableKey
	floatCtx := openfeature.NewEvaluationContext(newConst, map[string]any{
		"variableKey": "double_variable",
	})
	floatValue, err := ofClient.FloatValue(ctx, "flag", 0.0, floatCtx)
	if err != nil {
		fmt.Printf("float evaluation error: %v\n", err)
	} else {
		fmt.Printf("float value: %.2f\n", floatValue)
	}

	// Object evaluation with custom variableKey
	objectCtx := openfeature.NewEvaluationContext(newConst, map[string]any{
		"variableKey": "json_variable",
	})
	objectValue, err := ofClient.ObjectValue(ctx, "flag", map[string]any{}, objectCtx)
	if err != nil {
		fmt.Printf("object evaluation error: %v\n", err)
	} else {
		fmt.Printf("object value: %v\n", objectValue)
	}
}
