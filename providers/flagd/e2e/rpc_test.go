//go:build e2e

package e2e

import (
	"context"
	"path/filepath"
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
	
	// Setup container
	ctx := context.Background()
	if err := runner.SetupContainer(ctx); err != nil {
		t.Fatalf("Failed to setup container: %v", err)
	}
	
	// Define feature paths - using flagd-testbed gherkin files
	featurePaths := []string{
		"../../flagd-testbed/gherkin",
		"../../../tests/flagd/features", // Local feature files if any
	}
	
	// Run tests with RPC-specific tags
	tags := "@rpc && ~@targetURI && ~@unixsocket && ~@sync && ~@metadata && ~@in-process && ~@file"
	
	if err := runner.RunGherkinTests(featurePaths, tags); err != nil {
		t.Fatalf("Gherkin tests failed: %v", err)
	}
}

func TestRPCProviderWithSSL(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e tests in short mode")
	}
	
	runner := NewTestbedRunner(TestbedConfig{
		ResolverType:  integration.RPC,
		TestbedConfig: "ssl", // Use SSL testbed configuration
	})
	defer runner.Cleanup()
	
	ctx := context.Background()
	if err := runner.SetupContainer(ctx); err != nil {
		t.Fatalf("Failed to setup container: %v", err)
	}
	
	featurePaths := []string{
		"../../flagd-testbed/gherkin",
	}
	
	// Run SSL-specific tests
	tags := "@rpc && @customCert"
	
	if err := runner.RunGherkinTests(featurePaths, tags); err != nil {
		t.Fatalf("SSL Gherkin tests failed: %v", err)
	}
}

func TestRPCProviderCaching(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e tests in short mode")
	}
	
	runner := NewTestbedRunner(TestbedConfig{
		ResolverType:  integration.RPC,
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
	
	// Run caching-specific tests
	tags := "@rpc && @caching"
	
	if err := runner.RunGherkinTests(featurePaths, tags); err != nil {
		t.Fatalf("Caching Gherkin tests failed: %v", err)
	}
}

func TestRPCProviderEvents(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e tests in short mode")
	}
	
	runner := NewTestbedRunner(TestbedConfig{
		ResolverType:  integration.RPC,
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
	
	// Run event-specific tests
	tags := "@rpc && @events"
	
	if err := runner.RunGherkinTests(featurePaths, tags); err != nil {
		t.Fatalf("Events Gherkin tests failed: %v", err)
	}
}

// Benchmark tests for RPC provider
func BenchmarkRPCProviderEvaluation(b *testing.B) {
	runner := NewTestbedRunner(TestbedConfig{
		ResolverType:  integration.RPC,
		TestbedConfig: "default",
	})
	defer runner.Cleanup()
	
	ctx := context.Background()
	if err := runner.SetupContainer(ctx); err != nil {
		b.Fatalf("Failed to setup container: %v", err)
	}
	
	// Create provider for benchmarking
	supplier := runner.createRPCProviderSupplier()
	provider := supplier(map[string]interface{}{})
	
	// Run benchmark
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Benchmark flag evaluation here
		// This would require access to the provider's evaluation methods
	}
}