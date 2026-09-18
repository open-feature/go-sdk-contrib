//go:build e2e

package e2e

import (
	"testing"

	flagd "github.com/open-feature/go-sdk-contrib/providers/flagd/pkg"

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
			flagd.WithRetryBackoffMaxMs(3000),
		},
	})
	defer runner.Cleanup()

	// Define feature paths
	featurePaths := []string{
		"./",
	}

	// Run tests with in-process specific tags
	tags := "@in-process" + // in-process resolver scenarios
		" && ~@unixsocket" + // unix socket channel not supported
		" && ~@metadata" + // framework assertResolvedMetadata* steps are unimplemented stubs
		" && ~@customCert" + // testbed server cert is CN-only (no SANs); Go's TLS requires SANs
		" && ~@contextEnrichment" + // in-process context enrichment not merged yet (PR #730)
		" && ~@sync-payload" + // depends on context enrichment (PR #730)
		" && ~@deprecated" +
		" && ~@fractional-v1" + // legacy fractional algorithm
		" && ~@fractional-v3" // cbor fractional not yet implemented

	if err := runner.RunGherkinTestsWithSubtests(t, featurePaths, tags); err != nil {
		t.Fatalf("Gherkin tests failed: %v", err)
	}
}
