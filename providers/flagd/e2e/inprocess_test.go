//go:build e2e

package e2e

import (
	"context"
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
	
	// Setup container
	ctx := context.Background()
	if err := runner.SetupContainer(ctx); err != nil {
		t.Fatalf("Failed to setup container: %v", err)
	}
	
	// Define feature paths
	featurePaths := []string{
		"../../flagd-testbed/gherkin",
	}
	
	// Run tests with in-process specific tags
	tags := "@in-process && ~@rpc && ~@file"
	
	if err := runner.RunGherkinTests(featurePaths, tags); err != nil {
		t.Fatalf("Gherkin tests failed: %v", err)
	}
}

func TestInProcessProviderSync(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e tests in short mode")
	}
	
	runner := NewTestbedRunner(TestbedConfig{
		ResolverType:  integration.InProcess,
		TestbedConfig: "sync-payload", // Use sync-payload testbed configuration
	})
	defer runner.Cleanup()
	
	ctx := context.Background()
	if err := runner.SetupContainer(ctx); err != nil {
		t.Fatalf("Failed to setup container: %v", err)
	}
	
	featurePaths := []string{
		"../../flagd-testbed/gherkin",
	}
	
	// Run sync-specific tests
	tags := "@in-process && @sync"
	
	if err := runner.RunGherkinTests(featurePaths, tags); err != nil {
		t.Fatalf("Sync Gherkin tests failed: %v", err)
	}
}

func TestInProcessProviderMetadata(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e tests in short mode")
	}
	
	runner := NewTestbedRunner(TestbedConfig{
		ResolverType:  integration.InProcess,
		TestbedConfig: "metadata", // Use metadata testbed configuration
	})
	defer runner.Cleanup()
	
	ctx := context.Background()
	if err := runner.SetupContainer(ctx); err != nil {
		t.Fatalf("Failed to setup container: %v", err)
	}
	
	featurePaths := []string{
		"../../flagd-testbed/gherkin",
	}
	
	// Run metadata-specific tests
	tags := "@in-process && @metadata"
	
	if err := runner.RunGherkinTests(featurePaths, tags); err != nil {
		t.Fatalf("Metadata Gherkin tests failed: %v", err)
	}
}

func TestInProcessProviderEvents(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e tests in short mode")
	}
	
	runner := NewTestbedRunner(TestbedConfig{
		ResolverType:  integration.InProcess,
		TestbedConfig: "default",
	})
	defer runner.Cleanup()
	
	ctx := context.Background()
	if err := runner.SetupContainer(ctx); err != nil {
		t.Fatalf("Failed to setup container: %v", err)
	}
	
	featurePaths := []string{
		"../../flagd-testbed/gherkin",
	}
	
	// Run event-specific tests for in-process provider
	tags := "@in-process && @events"
	
	if err := runner.RunGherkinTests(featurePaths, tags); err != nil {
		t.Fatalf("Events Gherkin tests failed: %v", err)
	}
}

func TestInProcessProviderTargeting(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e tests in short mode")
	}
	
	runner := NewTestbedRunner(TestbedConfig{
		ResolverType:  integration.InProcess,
		TestbedConfig: "default",
	})
	defer runner.Cleanup()
	
	ctx := context.Background()
	if err := runner.SetupContainer(ctx); err != nil {
		t.Fatalf("Failed to setup container: %v", err)
	}
	
	featurePaths := []string{
		"../../flagd-testbed/gherkin",
	}
	
	// Run targeting-specific tests
	tags := "@in-process && @targeting"
	
	if err := runner.RunGherkinTests(featurePaths, tags); err != nil {
		t.Fatalf("Targeting Gherkin tests failed: %v", err)
	}
}