//go:build e2e

package e2e

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/cucumber/godog"
	flagd "github.com/open-feature/go-sdk-contrib/providers/flagd/pkg"
	"github.com/open-feature/go-sdk-contrib/tests/flagd/pkg/integration"
	"github.com/open-feature/go-sdk/openfeature"
)

// TestbedRunner manages testbed-based e2e testing
type TestbedRunner struct {
	container     *integration.FlagdTestContainer
	flagsDir      string
	testbedConfig string
	resolverType  integration.ProviderType
	options       []flagd.ProviderOption
}

// TestbedConfig holds configuration for testbed runner
type TestbedConfig struct {
	ResolverType  integration.ProviderType
	FlagsDir      string
	TestbedConfig string
	ExtraOptions  []flagd.ProviderOption
}

// NewTestbedRunner creates a new testbed-based test runner
func NewTestbedRunner(config TestbedConfig) *TestbedRunner {
	return &TestbedRunner{
		resolverType:  config.ResolverType,
		flagsDir:      config.FlagsDir,
		testbedConfig: config.TestbedConfig,
		options:       config.ExtraOptions,
	}
}

// SetupContainer starts and configures the flagd testbed container
func (tr *TestbedRunner) SetupContainer(ctx context.Context) error {
	// Determine flags directory - use testbed's built-in flags if none specified
	flagsDir := tr.flagsDir
	if flagsDir == "" {
		// Create temporary directory and copy testbed flags
		tempDir, err := os.MkdirTemp("", "flagd-e2e-*")
		if err != nil {
			return fmt.Errorf("failed to create temp directory: %w", err)
		}
		flagsDir = tempDir
	}

	// Create container configuration
	containerConfig := integration.FlagdContainerConfig{
		Image:         "ghcr.io/open-feature/flagd-testbed",
		Tag:           tr.getTestbedVersion(),
		FlagsDir:      flagsDir,
		ExtraWaitTime: 2 * time.Second,
	}

	// Create and start container
	container, err := integration.NewFlagdContainer(ctx, containerConfig)
	if err != nil {
		return fmt.Errorf("failed to create flagd container: %w", err)
	}

	tr.container = container

	// Configure flagd with specific testbed configuration
	if tr.testbedConfig != "" {
		if err := container.StartFlagdWithConfig(tr.testbedConfig); err != nil {
			return fmt.Errorf("failed to start flagd with config %s: %w", tr.testbedConfig, err)
		}
	}

	return nil
}

// RunGherkinTests executes gherkin tests against the testbed
func (tr *TestbedRunner) RunGherkinTests(featurePaths []string, tags string) error {
	if tr.container == nil {
		return fmt.Errorf("container not initialized")
	}

	// Setup provider suppliers for the integration package
	integration.SetProviderSuppliers(
		tr.createRPCProviderSupplier(),
		tr.createInProcessProviderSupplier(),
		tr.createFileProviderSupplier(),
	)

	// Configure godog
	opts := godog.Options{
		Format:      "pretty",
		Paths:       featurePaths,
		Tags:        tags,
		Concurrency: 1,
	}

	// Create test suite
	suite := godog.TestSuite{
		Name:                "flagd-e2e",
		ScenarioInitializer: tr.initializeScenario,
		Options:             &opts,
	}

	// Run tests
	status := suite.Run()
	if status != 0 {
		return fmt.Errorf("tests failed with status: %d", status)
	}

	return nil
}

// initializeScenario initializes the scenario with our testbed-specific setup
func (tr *TestbedRunner) initializeScenario(ctx *godog.ScenarioContext) {
	// Initialize the base integration steps
	integration.InitializeScenario(ctx)

	// Add testbed-specific steps
	tr.initializeTestbedSteps(ctx)
}

// initializeTestbedSteps adds testbed-specific step definitions
func (tr *TestbedRunner) initializeTestbedSteps(ctx *godog.ScenarioContext) {
	ctx.Step(`^the flag was modified$`, tr.modifyFlag)
	ctx.Step(`^a change event was fired$`, tr.fireChangeEvent)
	ctx.Step(`^the connection is lost for (\d+)s$`, tr.simulateConnectionLoss)
}

// Provider supplier functions

func (tr *TestbedRunner) createRPCProviderSupplier() integration.ProviderSupplier {
	return func(state integration.TestState) (openfeature.FeatureProvider, error) {
		opts := tr.buildProviderOptions(state, integration.RPC)
		return flagd.NewProvider(opts...)
	}
}

func (tr *TestbedRunner) createInProcessProviderSupplier() integration.ProviderSupplier {
	return func(state integration.TestState) (openfeature.FeatureProvider, error) {
		opts := tr.buildProviderOptions(state, integration.InProcess)
		return flagd.NewProvider(opts...)
	}
}

func (tr *TestbedRunner) createFileProviderSupplier() integration.ProviderSupplier {
	return func(state integration.TestState) (openfeature.FeatureProvider, error) {
		opts := tr.buildProviderOptions(state, integration.File)
		return flagd.NewProvider(opts...)
	}
}

// buildProviderOptions creates flagd provider options from config and container info
func (tr *TestbedRunner) buildProviderOptions(state integration.TestState, resolverType integration.ProviderType) []flagd.ProviderOption {
	var opts []flagd.ProviderOption

	// Add resolver type
	switch resolverType {
	case integration.RPC:
		opts = append(opts, flagd.WithRPCResolver())
		opts = append(opts, flagd.WithHost(tr.container.GetHost()))
		opts = append(opts, flagd.WithPort(uint16(tr.container.GetPort("rpc"))))
	case integration.InProcess:
		opts = append(opts, flagd.WithInProcessResolver())
		opts = append(opts, flagd.WithHost(tr.container.GetHost()))
		opts = append(opts, flagd.WithPort(uint16(tr.container.GetPort("in-process"))))
	case integration.File:
		opts = append(opts, flagd.WithInProcessResolver())
		if tr.flagsDir != "" {
			// Use the flags directory directly
			flagFile := filepath.Join(tr.flagsDir, "testing-flags.json")
			opts = append(opts, flagd.WithOfflineFilePath(flagFile))
		}
	}

	opts = append(opts, state.GenerateOpts()...)

	// Add extra options
	opts = append(opts, tr.options...)

	return opts
}

// Testbed interaction methods

func (tr *TestbedRunner) modifyFlag() error {
	if tr.container == nil {
		return fmt.Errorf("container not initialized")
	}

	return tr.container.TriggerFlagChange()
}

func (tr *TestbedRunner) fireChangeEvent() error {
	// The flag modification should trigger the change event
	return tr.modifyFlag()
}

func (tr *TestbedRunner) simulateConnectionLoss(seconds int) error {
	if tr.container == nil {
		return fmt.Errorf("container not initialized")
	}

	return tr.container.Restart(seconds)
}

// Utility methods

func (tr *TestbedRunner) getTestbedVersion() string {
	// Read version from test-harness/version.txt or use default
	versionFile := "../../flagd-testbed/version.txt"
	if data, err := os.ReadFile(versionFile); err == nil {
		return string(data)
	}
	return "latest"
}

// Cleanup releases resources
func (tr *TestbedRunner) Cleanup() error {
	if tr.container != nil {
		return tr.container.Terminate()
	}
	return nil
}
