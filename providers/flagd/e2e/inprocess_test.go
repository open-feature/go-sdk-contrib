//go:build e2e

package e2e

import (
	flagd "github.com/open-feature/go-sdk-contrib/providers/flagd/pkg"
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
		ExtraOptions: []flagd.ProviderOption{
			flagd.WithRetryBackoffMaxMs(5000),
		},
	})
	defer runner.Cleanup()

	// Define feature paths
	featurePaths := []string{
		"./",
	}

	// Run tests with in-process specific tags
	tags := "@in-process && ~@unixsocket && ~@metadata && ~@customCert && ~@contextEnrichment && ~@sync-payload"

	if err := runner.RunGherkinTestsWithSubtests(t, featurePaths, tags); err != nil {
		t.Fatalf("Gherkin tests failed: %v", err)
	}
}
