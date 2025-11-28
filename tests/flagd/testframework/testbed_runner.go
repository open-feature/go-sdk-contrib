package testframework

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/cucumber/godog"
	flagd "github.com/open-feature/go-sdk-contrib/providers/flagd/pkg"
	"github.com/open-feature/go-sdk/openfeature"
)

// TestbedRunner manages testbed-based e2e testing
type TestbedRunner struct {
	container     *FlagdTestContainer
	flagsDir      string
	testbedDir    string
	testbedConfig string
	resolverType  ProviderType
	options       []flagd.ProviderOption
	debugHelper   *DebugHelper
	Tag           string
	Image         string
}

// TestbedConfig holds configuration for testbed runner
type TestbedConfig struct {
	ResolverType  ProviderType
	TestbedDir    string
	FlagsDir      string
	TestbedConfig string
	ExtraOptions  []flagd.ProviderOption
	Tag           string
	Image         string
}

// NewTestbedRunner creates a new testbed-based test runner
func NewTestbedRunner(config TestbedConfig) *TestbedRunner {

	testbedDir := config.TestbedDir
	if testbedDir == "" {
		testbedDir = "../flagd-testbed"
	}

	runner := &TestbedRunner{
		resolverType:  config.ResolverType,
		flagsDir:      config.FlagsDir,
		testbedConfig: config.TestbedConfig,
		options:       config.ExtraOptions,
		testbedDir:    testbedDir,
		Tag:           config.Tag,
		Image:         config.Image,
	}

	// Initialize debugging helper (will be set after container setup)
	// runner.debugHelper will be initialized in SetupContainer

	// Initialize container immediately
	ctx := context.Background()
	if err := runner.SetupContainer(ctx); err != nil {
		// Enhanced error reporting with debugging info
		fmt.Printf("❌ Failed to setup container during runner creation: %v\n", err)
		if DebugMode {
			if runner.debugHelper != nil {
				runner.debugHelper.FullDiagnostics()
			}
		}
	} else if DebugMode {
		fmt.Printf("✅ Container setup successful\n")
		if runner.debugHelper != nil {
			runner.debugHelper.FullDiagnostics()
		}
	}

	return runner
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
	containerConfig := FlagdContainerConfig{
		FlagsDir:      flagsDir,
		TestbedDir:    tr.testbedDir,
		ExtraWaitTime: 2 * time.Second,
		Tag:           tr.Tag,
		Image:         tr.Image,
	}

	// Create and start container
	container, err := NewFlagdContainer(ctx, containerConfig)
	if err != nil {
		return fmt.Errorf("failed to create flagd container: %w", err)
	}

	tr.container = container

	// Initialize debug helper now that we have a container
	tr.debugHelper = NewDebugHelper(container, flagsDir)

	// Configure flagd with specific testbed configuration
	if tr.testbedConfig != "" {
		// If flagd is already running and we want the default config, skip the API call
		if tr.testbedConfig == "default" && container.IsHealthy() {
			// flagd is already running with the default config, no need to restart it
		} else {
			if err := container.StartFlagdWithConfig(tr.testbedConfig); err != nil {
				return fmt.Errorf("failed to start flagd with config %s: %w", tr.testbedConfig, err)
			}
		}
	}

	// Note: The launchpad automatically generates flag files like allFlags.json
	// when the testbed starts, so no additional API calls are needed

	// For file provider, wait a moment for launchpad to generate the files
	if tr.resolverType == File && tr.flagsDir != "" {
		// Give launchpad some time to generate allFlags.json
		time.Sleep(2 * time.Second)

		flagFile := filepath.Join(tr.flagsDir, "allFlags.json")
		if _, err := os.Stat(flagFile); os.IsNotExist(err) {
			// File might still be generating, this is for debugging purposes
			fmt.Printf("Warning: allFlags.json not yet available at %s\n", flagFile)
		}
	}

	return nil
}

// RunGherkinTestsWithSubtests executes gherkin tests with individual Go subtests for each scenario
// This makes each Gherkin scenario appear as a separate test in IntelliJ
func (tr *TestbedRunner) RunGherkinTestsWithSubtests(t *testing.T, featurePaths []string, tags string) error {
	if tr.container == nil {
		return fmt.Errorf("container not initialized")
	}
	ctx := context.Background()
	ctx = context.WithValue(ctx, "resolver", tr.resolverType)
	ctx = context.WithValue(ctx, "flagDir", tr.flagsDir)

	// Setup provider suppliers for the integration package
	SetProviderSuppliers(
		tr.createRPCProviderSupplier(),
		tr.createInProcessProviderSupplier(),
		tr.createFileProviderSupplier(),
	)

	for i, path := range featurePaths {
		featurePaths[i] = filepath.Join(tr.testbedDir, path)
	}

	// Configure godog with TestingT to create individual subtests
	opts := godog.Options{
		Format:         "pretty",
		Paths:          featurePaths,
		Tags:           tags,
		TestingT:       t, // This is the key! Creates individual Go subtests for each scenario
		Concurrency:    1,
		DefaultContext: ctx,
	}

	// Create test suite
	suite := godog.TestSuite{
		Name:                "flagd-e2e",
		ScenarioInitializer: tr.initializeScenario,
		Options:             &opts,
	}

	// Run tests - each scenario will appear as a separate subtest in IntelliJ
	if status := suite.Run(); status != 0 {
		return fmt.Errorf("tests failed with status: %d", status)
	}

	return nil
}

// initializeScenario initializes the scenario with our testbed-specific setup
func (tr *TestbedRunner) initializeScenario(ctx *godog.ScenarioContext) {
	// Initialize the base integration steps
	InitializeScenario(ctx)

	// Add a before hook to set the container in TestState
	ctx.Before(tr.setupScenario)
}

// setupScenario sets up the testbed container in the TestState before each scenario
func (tr *TestbedRunner) setupScenario(ctx context.Context, sc *godog.Scenario) (context.Context, error) {
	// Get the TestState from context
	if state := ctx.Value(TestStateKey{}); state != nil {
		if testState, ok := state.(*TestState); ok {
			// Set the container in the TestState so integration steps can use it
			testState.Container = tr.container
		}
	}
	return ctx, nil
}

// Provider supplier functions

func (tr *TestbedRunner) createRPCProviderSupplier() ProviderSupplier {
	return func(state TestState) (openfeature.FeatureProvider, error) {
		opts := tr.buildProviderOptions(state, RPC)
		return flagd.NewProvider(opts...)
	}
}

func (tr *TestbedRunner) createInProcessProviderSupplier() ProviderSupplier {
	return func(state TestState) (openfeature.FeatureProvider, error) {
		opts := tr.buildProviderOptions(state, InProcess)
		return flagd.NewProvider(opts...)
	}
}

func (tr *TestbedRunner) createFileProviderSupplier() ProviderSupplier {
	return func(state TestState) (openfeature.FeatureProvider, error) {
		opts := tr.buildProviderOptions(state, File)
		return flagd.NewProvider(opts...)
	}
}

// buildProviderOptions creates flagd provider options from config and container info
func (tr *TestbedRunner) buildProviderOptions(state TestState, resolverType ProviderType) []flagd.ProviderOption {
	var opts []flagd.ProviderOption

	host := tr.container.host
	var port int
	// Add resolver type
	switch resolverType {
	case RPC:
		port = tr.container.rpcPort
		opts = append(opts, flagd.WithRPCResolver())
	case InProcess:
		port = tr.container.inProcessPort
		opts = append(opts, flagd.WithInProcessResolver())
	case File:
		opts = append(opts, flagd.WithInProcessResolver())
		if tr.flagsDir != "" {
			// Use the local path to the launchpad-generated allFlags.json file
			// The container mounts tr.flagsDir to /flags, so launchpad will generate
			// allFlags.json in the local tr.flagsDir, which we can access directly
			flagFile := filepath.Join(tr.flagsDir, "allFlags.json")
			opts = append(opts, flagd.WithOfflineFilePath(flagFile))
		} else {
			panic(fmt.Errorf("flagsDir must be specified for file provider testing"))
		}
	}

	opts = append(opts, flagd.WithHost(host))
	opts = append(opts, flagd.WithPort(uint16(port)))
	for i, option := range state.ProviderOptions {
		if option.Option == "targetUri" {
			if option.Value != "" {
				option.Value = strings.ReplaceAll(
					option.Value,
					"<port>",
					strconv.Itoa(tr.container.envoyPort),
				)
				state.ProviderOptions[i] = option
				state.ProviderOptions = append(state.ProviderOptions, ProviderOption{
					Option:    "port",
					Value:     "99999",
					ValueType: "Integer",
				})
				break
			}
		}
		if option.Option == "port" {
			if option.Value == "9212" {
				option.Value = strconv.Itoa(tr.container.forbiddenPort)
				state.ProviderOptions[i] = option
			}
		}
	}
	opts = append(opts, state.GenerateOpts()...)

	// Add extra options
	opts = append(opts, tr.options...)

	return opts
}

// Testbed interaction methods

// Cleanup releases resources
func (tr *TestbedRunner) Cleanup() error {
	if tr.container != nil {
		return tr.container.Terminate()
	}
	return nil
}
