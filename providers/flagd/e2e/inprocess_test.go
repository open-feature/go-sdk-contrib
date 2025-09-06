//go:build e2e

package e2e

import (
	"testing"

	"github.com/open-feature/go-sdk-contrib/tests/flagd/testframework"
)

func TestInProcessProviderE2E(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e tests in short mode")
	}

	// Setup testbed runner for in-process provider
	runner := testframework.NewTestbedRunner(testframework.TestbedConfig{
		ResolverType:  testframework.InProcess,
		TestbedConfig: "default",
	})
	defer runner.Cleanup()

	// Define feature paths
	featurePaths := []string{
		"./",
	}

	// Run tests with in-process specific tags - exclude connection/event issues we won't tackle
	tags := "@in-process && ~@unixsocket && ~@metadata && ~@grace && ~@customCert && ~@reconnect && ~@contextEnrichment && ~@sync-payload && ~@events"

	if err := runner.RunGherkinTestsWithSubtests(t, featurePaths, tags); err != nil {
		t.Fatalf("Gherkin tests failed: %v", err)
	}
}
