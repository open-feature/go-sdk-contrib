package codereadiness_test

import (
	"context"
	"fmt"

	"github.com/open-feature/go-sdk-contrib/hooks/codereadiness"
	"github.com/open-feature/go-sdk/openfeature"
)

type mockProvider struct {
	openfeature.NoopProvider
	metadata openfeature.FlagMetadata
}

func (p mockProvider) BooleanEvaluation(ctx context.Context, flag string, defaultValue bool, evalCtx openfeature.FlattenedContext) openfeature.BoolResolutionDetail {
	return openfeature.BoolResolutionDetail{
		Value: true,
		ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
			FlagMetadata: p.metadata,
		},
	}
}

func Example() {
	ctx := context.Background()

	// Create a new code readiness hook for application version "v1.2.0".
	hook, err := codereadiness.New("v1.2.0")
	if err != nil {
		fmt.Println("Error creating hook:", err)
		return
	}

	// Register the hook globally.
	openfeature.AddHooks(hook)
	defer openfeature.Shutdown()

	// Set up a provider that returns flag metadata specifying minCodeVersion = "v1.0.0".
	provider := mockProvider{
		metadata: openfeature.FlagMetadata{
			"minCodeVersion": "v1.0.0",
		},
	}
	if err := openfeature.SetProviderAndWait(provider); err != nil {
		fmt.Println("Error setting provider:", err)
		return
	}

	client := openfeature.NewDefaultClient()
	// Since current version (v1.2.0) >= required (v1.0.0), flag evaluation succeeds.
	enabled := client.Boolean(ctx, "feature-a", false, openfeature.EvaluationContext{})
	fmt.Println("feature-a enabled:", enabled)

	// Output:
	// feature-a enabled: true
}

type customComparator struct {
	current string
}

func (c *customComparator) Initialize(current string) error {
	c.current = current
	return nil
}

func (c *customComparator) Compare(required string) (bool, error) {
	if c.current < required {
		return false, nil
	}
	return true, nil
}

func ExampleNew() {
	ctx := context.Background()

	// Create a hook with strict validation enabled, custom metadata key and custom comparator.
	hook, err := codereadiness.New(
		"v1.0.0",
		codereadiness.WithStrictValidation(true),
		codereadiness.WithMetadataMinVerKey("requiredCodeVersion"),
		codereadiness.WithComparator(&customComparator{}),
	)
	if err != nil {
		fmt.Println("Error creating hook:", err)
		return
	}

	openfeature.AddHooks(hook)
	defer openfeature.Shutdown()

	// Set up a provider whose flag metadata is missing the required key ("requiredCodeVersion").
	provider := mockProvider{
		metadata: openfeature.FlagMetadata{
			"minCodeVersion": "v1.0.0",
		},
	}
	if err := openfeature.SetProviderAndWait(provider); err != nil {
		fmt.Println("Error setting provider:", err)
		return
	}

	client := openfeature.NewDefaultClient()
	// Because strict validation is enabled and the expected metadata key is missing,
	// evaluation fails and returns the default value (false).
	enabled, err := client.BooleanValue(ctx, "feature-b", false, openfeature.EvaluationContext{})
	fmt.Println("feature-b enabled:", enabled)
	fmt.Println("evaluation error:", err)
	// Output:
	// feature-b enabled: false
	// evaluation error: after hook: key "requiredCodeVersion" missing in flag's "feature-b" metadata
}
