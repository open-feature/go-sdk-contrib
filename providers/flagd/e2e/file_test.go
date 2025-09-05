//go:build e2e

package e2e

import (
	"os"
	"testing"

	"github.com/open-feature/go-sdk-contrib/tests/flagd/testframework"
)

func TestFileProviderE2E(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e tests in short mode")
	}

	// Create temporary directory for flag files
	// This directory will be mounted as /flags inside the testbed container
	tempDir, err := os.MkdirTemp("", "flagd-file-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Setup testbed runner for file provider
	// The testbed will mount tempDir to /flags in the container
	// Launchpad will generate allFlags.json inside the container at /flags/allFlags.json
	// Which corresponds to tempDir/allFlags.json on the host
	// The file provider will read from the local path tempDir/allFlags.json
	runner := testframework.NewTestbedRunner(testframework.TestbedConfig{
		ResolverType:  testframework.File,
		FlagsDir:      tempDir,
		TestbedConfig: "default", // Use default config to ensure flags are available
	})
	defer runner.Cleanup()

	// Define feature paths
	featurePaths := []string{
		"./",
	}

	// Run tests with file-specific tags, focusing on core evaluation scenarios
	// Skip complex connection-related and synchronization scenarios for file provider
	tags := "@file && ~@reconnect && ~@sync && ~@grace && ~@events && ~@unixsocket && ~@metadata"

	if err := runner.RunGherkinTestsWithSubtests(t, featurePaths, tags); err != nil {
		t.Fatalf("Gherkin tests failed: %v", err)
	}
}
