//go:build e2e

package e2e

import (
	"testing"

	"github.com/open-feature/go-sdk-contrib/tests/flagd/testframework"
)

func TestRPCProviderE2E(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e tests in short mode")
	}

	// Setup testbed runner for RPC provider
	runner := NewTestbedRunner(TestbedConfig{
		ResolverType:  testframework.RPC,
		TestbedConfig: "default", // Use default testbed configuration
	})
	defer runner.Cleanup()

	// Define feature paths - using flagd-testbed gherkin files
	featurePaths := []string{
		"../flagd-testbed/gherkin",
	}

	// Run tests with RPC-specific tags - exclude connection/event issues we won't tackle
	tags := "@rpc && ~@targetURI && ~@unixsocket && ~@sync && ~@metadata && ~@grace && ~@events && ~@customCert && ~@reconnect && ~@caching"

	if err := runner.RunGherkinTestsWithSubtests(t, featurePaths, tags); err != nil {
		t.Fatalf("Gherkin tests failed: %v", err)
	}
}
