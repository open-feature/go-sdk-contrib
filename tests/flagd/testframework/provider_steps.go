package testframework

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cucumber/godog"
	"github.com/open-feature/go-sdk/openfeature"
)

// Provider supplier types moved to types.go

// SetProviderSuppliers sets the provider creation functions
func SetProviderSuppliers(rpc, inProcess, file ProviderSupplier) {
	RPCProviderSupplier = rpc
	InProcessProviderSupplier = inProcess
	FileProviderSupplier = file
}

// InitializeProviderSteps registers provider lifecycle step definitions
func InitializeProviderSteps(ctx *godog.ScenarioContext) {
	ctx.Step(`^the connection is lost for (\d+)s$`,
		withStateIntArg((*TestState).simulateConnectionLoss))

	// Generic provider step definition - accepts any provider type including "stable"
	ctx.Step(`^a (\w+) flagd provider$`,
		withState1Arg((*TestState).createSpecializedFlagdProvider))

	// TODO: deprecate 'is' variant after flagd-testbed/pull/#311 is merged
	ctx.Step(`^the client (?:is|should be) in (\w+) state$`,
		withState1Arg((*TestState).assertClientState))
}

// State methods - these now expect context as first parameter after state

// createProviderInstance creates and initializes a flagd provider (formerly createStableFlagdProvider)
func (s *TestState) createProviderInstance() error {
	// Apply defaults if not set
	s.applyDefaults()

	// Create the appropriate provider based on type
	var provider openfeature.FeatureProvider
	var err error

	s.ProviderOptions = append(s.ProviderOptions, ProviderOption{
		Option:    "RetryGracePeriod",
		ValueType: "Integer",
		Value:     "1",
	})
	switch s.ProviderType {
	case RPC:
		if RPCProviderSupplier == nil {
			return fmt.Errorf("RPC provider supplier not set")
		}
		provider, err = RPCProviderSupplier(*s)
		break
	case InProcess:
		if InProcessProviderSupplier == nil {
			return fmt.Errorf("In-process provider supplier not set")
		}
		provider, err = InProcessProviderSupplier(*s)
		break
	case File:
		if FileProviderSupplier == nil {
			return fmt.Errorf("File provider supplier not set")
		}
		provider, err = FileProviderSupplier(*s)
		break
	default:
		return fmt.Errorf("unknown provider type: %v", s.ProviderType)
	}

	if err != nil {
		return fmt.Errorf("failed to create provider: %w", err)
	}

	s.Provider = provider

	// Set the provider in OpenFeature
	domain := fmt.Sprintf("flagd-e2e-tests-%d", time.Now().UnixNano())
	err = openfeature.SetNamedProvider(domain, provider)
	if err != nil {
		return fmt.Errorf("failed to set provider: %w", err)
	}

	// Create client
	s.Client = openfeature.NewClient(domain)
	return nil
}

// waitForProviderReady waits for the provider to be in READY state
func (s *TestState) waitForProviderReady(ctx context.Context, timeout time.Duration) error {
	if s.Client == nil {
		return fmt.Errorf("no client available to wait for provider ready")
	}

	// Use generic event handler infrastructure
	if err := s.addGenericEventHandler(ctx, "ready"); err != nil {
		return fmt.Errorf("failed to add ready event handler: %w", err)
	}

	// Wait for READY event to be recorded
	return s.waitForEvents("READY", timeout)
}

// simulateConnectionLoss simulates connection loss for specified duration
func (s *TestState) simulateConnectionLoss(ctx context.Context, seconds int) error {
	if s.Container == nil {
		return fmt.Errorf("no container available to simulate connection loss")
	}

	// Use testbed launchpad to restart flagd after delay
	return s.Container.Restart(seconds)
}

func (s *TestState) assertClientState(ctx context.Context, state string) error {
	if string(s.Client.State()) == strings.ToUpper(state) {
		return nil
	}
	return fmt.Errorf("expected client state %s but got %s", state, s.Client.State())
}

// createSpecializedFlagdProvider creates specialized flagd providers based on type
func (s *TestState) createSpecializedFlagdProvider(ctx context.Context, providerType string) error {
	// Apply specialized configuration based on provider type
	if err := s.applySpecializedConfig(providerType); err != nil {
		return fmt.Errorf("failed to apply specialized config for %s provider: %w", providerType, err)
	}

	// Trigger testbed configuration for specialized provider types if needed
	if err := s.triggerTestbedConfiguration(providerType); err != nil {
		return fmt.Errorf("failed to configure testbed for %s provider: %w", providerType, err)
	}

	// Create the actual provider instance using the stable provider logic
	if err := s.createProviderInstance(); err != nil {
		return fmt.Errorf("failed to create instance for %s provider: %w", providerType, err)
	}

	if providerType != "unavailable" && providerType != "forbidden" {
		if s.ProviderType == RPC {
			// Small delay to allow flagd server to fully load flags after connection
			time.Sleep(50 * time.Millisecond)
		}

		// Wait for provider to be ready
		return s.waitForProviderReady(ctx, 15*time.Second)
	}
	return nil
}

// applySpecializedConfig applies provider-type specific configuration
func (s *TestState) applySpecializedConfig(providerType string) error {
	// Apply defaults first
	s.applyDefaults()

	switch strings.ToLower(providerType) {
	case "stable":
		return nil
	case "unavailable":
		return s.configureUnavailableProvider()
	case "forbidden": return s.configureForbiddenProvider()
	case "socket":
		return s.configureSocketProvider()
	case "ssl", "tls":
		return s.configureSslProvider()
	case "metadata":
		return s.configureMetadataProvider()
	case "syncpayload":
		return nil
	default:
		// For unknown provider types, just use default configuration
		return nil
	}
}

func (s *TestState) configureUnavailableProvider() error {
	// Set an unreachable port to simulate unavailable provider
	s.addProviderOption("port", "Integer", "9999")
	s.addProviderOption("host", "String", "127.0.0.1")
	s.addProviderOption("deadlineMs", "Integer", "1000") // Short timeout for faster failure
	s.addProviderOption("offlineFlagSourcePath", "String", "not-existing")
	return nil
}

func (s *TestState) configureForbiddenProvider() error {
	// Set an Envoy port which always responds with forbidden
	s.addProviderOption("port", "Integer", "9212")
	return nil
}

func (s *TestState) configureSocketProvider() error {
	// Configure for unix socket connection
	s.addProviderOption("socketPath", "String", "/tmp/flagd.sock")
	s.addProviderOption("port", "Integer", "0") // Disable port when using socket
	return nil
}

func (s *TestState) configureSslProvider() error {
	// Configure SSL/TLS connection
	s.addProviderOption("tls", "Boolean", "true")
	s.addProviderOption("certPath", "String", "../flagd-testbed/ssl/custom-root-cert.crt")
	return nil
}

func (s *TestState) configureMetadataProvider() error {
	// Configure provider for metadata testing
	// Check if selector is already configured from previous steps
	if selector := s.findExistingProviderOption("selector"); selector != "" {
		// Selector already configured, keep it
		return nil
	}
	return nil
}

// Helper methods for ProviderOption management

// addProviderOption adds a provider option to the current test state
func (s *TestState) addProviderOption(option, valueType, value string) {
	providerOption := ProviderOption{
		Option:    option,
		ValueType: valueType,
		Value:     value,
	}
	s.ProviderOptions = append(s.ProviderOptions, providerOption)
}

// findExistingProviderOption finds an existing provider option value by name
func (s *TestState) findExistingProviderOption(optionName string) string {
	for _, opt := range s.ProviderOptions {
		if opt.Option == optionName {
			return opt.Value
		}
	}
	return ""
}

// triggerTestbedConfiguration configures the testbed container for specialized provider types
func (s *TestState) triggerTestbedConfiguration(providerType string) error {
	if s.Container == nil {
		// No container available - this is fine for some test scenarios
		return nil
	}

	if container, ok := s.Container.(*FlagdTestContainer); ok {
		switch strings.ToLower(providerType) {
		case "stable":
			// Stable provider doesn't need testbed - uses offline file source
			// No need to start flagd in testbed for stable provider
			return container.StartFlagdWithConfig("default")
		case "socket":
			// Use socket configuration - testbed will expose unix socket
			return container.StartFlagdWithConfig("socket")
		case "ssl", "tls":
			// Use SSL configuration - testbed will enable TLS
			return container.StartFlagdWithConfig("ssl")
		default:
			// Most providers use default testbed configuration
			// This includes: unavailable, syncpayload, metadata, target, etc.
			return container.StartFlagdWithConfig("default")
		}
	}

	return nil
}
