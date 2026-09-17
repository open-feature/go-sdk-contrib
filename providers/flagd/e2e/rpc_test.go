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
	runner := testframework.NewTestbedRunner(testframework.TestbedConfig{
		ResolverType:  testframework.RPC,
		TestbedConfig: "default", // Use default testbed configuration
	})
	defer runner.Cleanup()

	// Define feature paths - using flagd-testbed gherkin files
	featurePaths := []string{
		"./",
	}

	// Run tests with RPC-specific tags - exclude unimplemented or inapplicable scenarios
	tags := "@rpc" + // rpc resolver scenarios
		" && ~@unixsocket" + // unix socket channel not supported
		" && ~@targetURI" + // target-uri scenarios not supported
		" && ~@customCert" + // custom TLS cert not wired up
		" && ~@caching" + // caching scenarios not covered here
		" && ~@deprecated" +
		" && ~@fractional-v1" + // legacy fractional algorithm
		" && ~@fractional-v3" // cbor fractional not yet implemented

	if err := runner.RunGherkinTestsWithSubtests(t, featurePaths, tags); err != nil {
		t.Fatalf("Gherkin tests failed: %v", err)
	}
}
