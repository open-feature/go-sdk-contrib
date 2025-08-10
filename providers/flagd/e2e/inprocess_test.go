//go:build e2e

package e2e

import (
	"testing"

	"github.com/open-feature/go-sdk-contrib/tests/flagd/pkg/integration"
)

func TestInProcessProviderE2E(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e tests in short mode")
	}

	// Setup testbed runner for in-process provider
	runner := NewTestbedRunner(TestbedConfig{
		ResolverType:  integration.InProcess,
		TestbedConfig: "default",
	})
	defer runner.Cleanup()

	// Define feature paths
	featurePaths := []string{
		"../flagd-testbed/gherkin",
	}

	// Run tests with in-process specific tags
	tags := "@in-process && ~@grace"

	if err := runner.RunGherkinTestsWithSubtests(t, featurePaths, tags); err != nil {
		t.Fatalf("Gherkin tests failed: %v", err)
	}
}
