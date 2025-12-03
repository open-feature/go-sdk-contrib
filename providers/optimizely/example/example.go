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

	// Required: Optimizely SDK key
	sdkKey := os.Getenv("OPTIMIZELY_SDK_KEY")
	if sdkKey == "" {
		fmt.Println("OPTIMIZELY_SDK_KEY environment variable is required")
		os.Exit(1)
	}

	// Optional: customize flag key and user ID
	flagKey := os.Getenv("FLAG_KEY")
	if flagKey == "" {
		flagKey = "my_flag"
	}
	userID := os.Getenv("USER_ID")
	if userID == "" {
		userID = "user_123"
	}

	// Create Optimizely client
	optimizelyClient, err := (&client.OptimizelyFactory{
		SDKKey: sdkKey,
	}).Client()
	if err != nil {
		fmt.Printf("failed to create Optimizely client: %v\n", err)
		os.Exit(1)
	}

	// Set up OpenFeature with the Optimizely provider
	provider := optimizely.NewProvider(optimizelyClient)
	err = openfeature.SetProviderAndWait(provider)
	if err != nil {
		fmt.Printf("failed to set provider: %v\n", err)
		os.Exit(1)
	}
	defer openfeature.Shutdown()

	ofClient := openfeature.NewClient("my-app")
	evalCtx := openfeature.NewEvaluationContext(userID, nil)

	fmt.Printf("Testing flag: %s (user: %s)\n\n", flagKey, userID)

	// Boolean evaluation
	// Use for: flags with 0 variables (returns enabled state) OR flags with 1 bool variable
	fmt.Println("=== BooleanEvaluation ===")
	boolResult, err := ofClient.BooleanValueDetails(ctx, flagKey, false, evalCtx)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Value: %v, Reason: %s, Variant: %s\n", boolResult.Value, boolResult.Reason, boolResult.Variant)
	}

	// String evaluation
	// Use for: flags with 1 string variable
	fmt.Println("\n=== StringEvaluation ===")
	stringResult, err := ofClient.StringValueDetails(ctx, flagKey, "default", evalCtx)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Value: %s, Reason: %s, Variant: %s\n", stringResult.Value, stringResult.Reason, stringResult.Variant)
	}

	// Int evaluation
	// Use for: flags with 1 integer variable
	fmt.Println("\n=== IntEvaluation ===")
	intResult, err := ofClient.IntValueDetails(ctx, flagKey, 0, evalCtx)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Value: %d, Reason: %s, Variant: %s\n", intResult.Value, intResult.Reason, intResult.Variant)
	}

	// Float evaluation
	// Use for: flags with 1 double variable
	fmt.Println("\n=== FloatEvaluation ===")
	floatResult, err := ofClient.FloatValueDetails(ctx, flagKey, 0.0, evalCtx)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Value: %.2f, Reason: %s, Variant: %s\n", floatResult.Value, floatResult.Reason, floatResult.Variant)
	}

	// Object evaluation
	// Use for: flags with 1 variable (returns single value) OR flags with multiple variables (returns map)
	fmt.Println("\n=== ObjectEvaluation ===")
	objectResult, err := ofClient.ObjectValueDetails(ctx, flagKey, nil, evalCtx)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Value: %v, Reason: %s, Variant: %s\n", objectResult.Value, objectResult.Reason, objectResult.Variant)
	}
}
