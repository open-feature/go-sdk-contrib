//go:build e2e

package e2e

import (
	"testing"

	"github.com/open-feature/go-sdk-contrib/tests/flagd/pkg/integration"
)

func TestRPCProviderE2E(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e tests in short mode")
	}

	// Setup testbed runner for RPC provider
	runner := NewTestbedRunner(TestbedConfig{
		ResolverType:  integration.RPC,
		TestbedConfig: "default", // Use default testbed configuration
	})
	defer runner.Cleanup()

	// Define feature paths - using flagd-testbed gherkin files
	featurePaths := []string{
		"../flagd-testbed/gherkin",
	}

	// Run tests with RPC-specific tags - using subtests for better IntelliJ integration
	tags := "@rpc && ~@targetURI && ~@unixsocket && ~@sync && ~@metadata && ~@grace"

	if err := runner.RunGherkinTestsWithSubtests(t, featurePaths, tags); err != nil {
		t.Fatalf("Gherkin tests failed: %v", err)
	}
}
