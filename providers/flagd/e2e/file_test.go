//go:build e2e

package e2e

import (
	"os"
	"testing"

	"github.com/open-feature/go-sdk-contrib/tests/flagd/pkg/integration"
)

func TestFileProviderE2E(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e tests in short mode")
	}

	// Create temporary directory for flag files that will be mounted in container
	tempDir, err := os.MkdirTemp("", "flagd-file-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Setup testbed runner for file provider
	// The testbed will mount the flagsDir and launchpad will generate allFlags.json in it
	runner := NewTestbedRunner(TestbedConfig{
		ResolverType:  integration.File,
		FlagsDir:      tempDir,
		TestbedConfig: "default", // Use default config to ensure flags are available
	})
	defer runner.Cleanup()

	// Define feature paths
	featurePaths := []string{
		"../flagd-testbed/gherkin",
	}

	// Run tests with file-specific tags, focusing on core evaluation scenarios
	// Skip complex connection-related and synchronization scenarios for file provider
	tags := "@file && ~@reconnect && ~@sync && ~@grace && ~@events"

	if err := runner.RunGherkinTestsWithSubtests(t, featurePaths, tags); err != nil {
		t.Fatalf("Gherkin tests failed: %v", err)
	}
}
