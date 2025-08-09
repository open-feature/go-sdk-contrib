//go:build e2e

package e2e

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/open-feature/go-sdk-contrib/tests/flagd/pkg/integration"
)

func TestFileProviderE2E(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e tests in short mode")
	}
	
	// Create temporary directory for flag files
	tempDir, err := os.MkdirTemp("", "flagd-file-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	// Create test flag file
	if err := createTestFlagFile(tempDir); err != nil {
		t.Fatalf("Failed to create test flag file: %v", err)
	}
	
	// Setup testbed runner for file provider
	runner := NewTestbedRunner(TestbedConfig{
		ResolverType: integration.File,
		FlagsDir:     tempDir,
	})
	defer runner.Cleanup()
	
	// For file provider, we don't need a container since it reads files directly
	// But we still set up the runner for consistency
	ctx := context.Background()
	
	// Define feature paths
	featurePaths := []string{
		"../../flagd-testbed/gherkin",
	}
	
	// Run tests with file-specific tags
	tags := "@file && ~@rpc && ~@in-process && ~@events && ~@reconnect"
	
	if err := runner.RunGherkinTests(featurePaths, tags); err != nil {
		t.Fatalf("Gherkin tests failed: %v", err)
	}
}

func TestFileProviderConfiguration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e tests in short mode")
	}
	
	tempDir, err := os.MkdirTemp("", "flagd-file-config-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	if err := createTestFlagFile(tempDir); err != nil {
		t.Fatalf("Failed to create test flag file: %v", err)
	}
	
	runner := NewTestbedRunner(TestbedConfig{
		ResolverType: integration.File,
		FlagsDir:     tempDir,
	})
	defer runner.Cleanup()
	
	featurePaths := []string{
		"../../flagd-testbed/gherkin",
	}
	
	// Run configuration-specific tests for file provider
	tags := "@file && config"
	
	if err := runner.RunGherkinTests(featurePaths, tags); err != nil {
		t.Fatalf("Configuration tests failed: %v", err)
	}
}

func TestFileProviderPolling(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e tests in short mode")
	}
	
	tempDir, err := os.MkdirTemp("", "flagd-file-polling-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	// Create initial flag file
	if err := createTestFlagFile(tempDir); err != nil {
		t.Fatalf("Failed to create test flag file: %v", err)
	}
	
	runner := NewTestbedRunner(TestbedConfig{
		ResolverType: integration.File,
		FlagsDir:     tempDir,
	})
	defer runner.Cleanup()
	
	// Test flag file modification and polling
	go func() {
		// Simulate flag file changes after a delay
		// This would modify the flag file to test polling behavior
		// Implementation would depend on specific test requirements
	}()
	
	featurePaths := []string{
		"../../flagd-testbed/gherkin",
	}
	
	// Run offline/polling specific tests
	tags := "@file && @offline"
	
	if err := runner.RunGherkinTests(featurePaths, tags); err != nil {
		t.Fatalf("Polling tests failed: %v", err)
	}
}

// createTestFlagFile creates a test flag configuration file
func createTestFlagFile(dir string) error {
	flagContent := `{
		"flags": {
			"boolean-flag": {
				"state": "ENABLED",
				"variants": {
					"on": true,
					"off": false
				},
				"defaultVariant": "on"
			},
			"string-flag": {
				"state": "ENABLED",
				"variants": {
					"greeting": "hi",
					"parting": "bye"
				},
				"defaultVariant": "greeting"
			},
			"integer-flag": {
				"state": "ENABLED",
				"variants": {
					"one": 1,
					"ten": 10
				},
				"defaultVariant": "ten"
			},
			"float-flag": {
				"state": "ENABLED",
				"variants": {
					"half": 0.5,
					"tenth": 0.1
				},
				"defaultVariant": "half"
			},
			"boolean-zero-flag": {
				"state": "ENABLED",
				"variants": {
					"on": true,
					"off": false
				},
				"defaultVariant": "off"
			},
			"string-zero-flag": {
				"state": "ENABLED",
				"variants": {
					"empty": "",
					"greeting": "hi"
				},
				"defaultVariant": "empty"
			},
			"integer-zero-flag": {
				"state": "ENABLED",
				"variants": {
					"zero": 0,
					"one": 1
				},
				"defaultVariant": "zero"
			},
			"float-zero-flag": {
				"state": "ENABLED",
				"variants": {
					"zero": 0.0,
					"half": 0.5
				},
				"defaultVariant": "zero"
			},
			"boolean-targeted-zero-flag": {
				"state": "ENABLED",
				"variants": {
					"on": true,
					"off": false
				},
				"defaultVariant": "off",
				"targeting": {
					"if": [
						{
							"==": [
								{ "var": "email" },
								"ballmer@macrosoft.com"
							]
						},
						"off",
						null
					]
				}
			},
			"string-targeted-zero-flag": {
				"state": "ENABLED",
				"variants": {
					"empty": "",
					"greeting": "hi"
				},
				"defaultVariant": "empty",
				"targeting": {
					"if": [
						{
							"==": [
								{ "var": "email" },
								"ballmer@macrosoft.com"
							]
						},
						"empty",
						null
					]
				}
			},
			"integer-targeted-zero-flag": {
				"state": "ENABLED",
				"variants": {
					"zero": 0,
					"one": 1
				},
				"defaultVariant": "zero",
				"targeting": {
					"if": [
						{
							"==": [
								{ "var": "email" },
								"ballmer@macrosoft.com"
							]
						},
						"zero",
						null
					]
				}
			},
			"float-targeted-zero-flag": {
				"state": "ENABLED",
				"variants": {
					"zero": 0.0,
					"half": 0.5
				},
				"defaultVariant": "zero",
				"targeting": {
					"if": [
						{
							"==": [
								{ "var": "email" },
								"ballmer@macrosoft.com"
							]
						},
						"zero",
						null
					]
				}
			}
		}
	}`
	
	flagFile := filepath.Join(dir, "testing-flags.json")
	return os.WriteFile(flagFile, []byte(flagContent), 0644)
}