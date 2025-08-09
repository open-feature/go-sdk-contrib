package integration

import (
	"fmt"
	"time"

	"github.com/cucumber/godog"
	"github.com/open-feature/go-sdk/openfeature"
)

// ProviderSupplier is a function type that creates providers
type ProviderSupplier func(state TestState) (openfeature.FeatureProvider, error)

// Global provider suppliers for different resolver types
var (
	RPCProviderSupplier       ProviderSupplier
	InProcessProviderSupplier ProviderSupplier
	FileProviderSupplier      ProviderSupplier
)

// SetProviderSuppliers sets the provider creation functions
func SetProviderSuppliers(rpc, inProcess, file ProviderSupplier) {
	RPCProviderSupplier = rpc
	InProcessProviderSupplier = inProcess
	FileProviderSupplier = file
}

// initializeProviderSteps registers provider lifecycle step definitions
func initializeProviderSteps(ctx *godog.ScenarioContext, state *TestState) {
	ctx.Step(`^a stable flagd provider$`, state.createStableFlagdProvider)
	ctx.Step(`^a ready event handler$`, state.addReadyEventHandler)
	ctx.Step(`^a error event handler$`, state.addErrorEventHandler)
	ctx.Step(`^a stale event handler$`, state.addStaleEventHandler)
	ctx.Step(`^the ready event handler should have been executed$`, state.assertReadyEventExecuted)
	ctx.Step(`^the error event handler should have been executed$`, state.assertErrorEventExecuted)
	ctx.Step(`^the stale event handler should have been executed$`, state.assertStaleEventExecuted)
	ctx.Step(`^the connection is lost for (\d+)s$`, state.simulateConnectionLoss)
}

// createStableFlagdProvider creates and initializes a flagd provider
func (s *TestState) createStableFlagdProvider() error {
	// Apply defaults if not set
	s.applyDefaults()

	// Create the appropriate provider based on type
	var provider openfeature.FeatureProvider
	var err error

	switch s.ProviderType {
	case RPC:
		if RPCProviderSupplier == nil {
			return fmt.Errorf("RPC provider supplier not set")
		}
		provider, err = RPCProviderSupplier(*s)
	case InProcess:
		if InProcessProviderSupplier == nil {
			return fmt.Errorf("In-process provider supplier not set")
		}
		provider, err = InProcessProviderSupplier(*s)
	case File:
		if FileProviderSupplier == nil {
			return fmt.Errorf("File provider supplier not set")
		}
		provider, err = FileProviderSupplier(*s)
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

	// Wait for provider to be ready
	return s.waitForProviderReady(15 * time.Second)
}

// waitForProviderReady waits for the provider to be in READY state
func (s *TestState) waitForProviderReady(timeout time.Duration) error {
	// Check if we already have a READY event
	for _, event := range s.Events {
		if event.Type == "READY" {
			return nil
		}
	}
	
	// Use existing event handler infrastructure
	if err := s.addReadyEventHandler(); err != nil {
		return fmt.Errorf("failed to add ready event handler: %w", err)
	}
	
	// Wait for READY event to be recorded
	return s.waitForEvents("READY", timeout)
}

// addReadyEventHandler adds a handler for provider ready events
func (s *TestState) addReadyEventHandler() error {
	handler := func(details openfeature.EventDetails) {
		s.addEvent("READY", details)
	}

	s.EventHandlers["READY"] = handler
	s.Client.AddHandler(openfeature.ProviderReady, &handler)
	return nil
}

// addErrorEventHandler adds a handler for provider error events
func (s *TestState) addErrorEventHandler() error {
	handler := func(details openfeature.EventDetails) {
		s.addEvent("ERROR", details)
	}

	s.EventHandlers["ERROR"] = handler
	s.Client.AddHandler(openfeature.ProviderError, &handler)
	return nil
}

// addStaleEventHandler adds a handler for provider stale events
func (s *TestState) addStaleEventHandler() error {
	handler := func(details openfeature.EventDetails) {
		s.addEvent("STALE", details)
	}

	s.EventHandlers["STALE"] = handler
	s.Client.AddHandler(openfeature.ProviderStale, &handler)
	return nil
}

// assertReadyEventExecuted verifies that a READY event was received
func (s *TestState) assertReadyEventExecuted() error {
	return s.assertEventOccurred("READY")
}

// assertErrorEventExecuted verifies that an ERROR event was received
func (s *TestState) assertErrorEventExecuted() error {
	return s.assertEventOccurred("ERROR")
}

// assertStaleEventExecuted verifies that a STALE event was received
func (s *TestState) assertStaleEventExecuted() error {
	return s.assertEventOccurred("STALE")
}

// simulateConnectionLoss simulates connection loss for specified duration
func (s *TestState) simulateConnectionLoss(seconds int) error {
	if s.Container == nil {
		return fmt.Errorf("no container available to simulate connection loss")
	}

	// Use testbed launchpad to restart flagd after delay
	return s.Container.Restart(seconds)
}

// Event handler helpers for provider state changes
func (s *TestState) handleProviderStateChange(eventType string) func(openfeature.EventDetails) {
	return func(details openfeature.EventDetails) {
		s.addEvent(eventType, details)
	}
}

// Cleanup removes all event handlers
func (s *TestState) cleanupEventHandlers() {
	if s.Client == nil {
		return
	}

	for eventType, handler := range s.EventHandlers {
		switch eventType {
		case "READY":
			s.Client.RemoveHandler(openfeature.ProviderReady, &handler)
		case "ERROR":
			s.Client.RemoveHandler(openfeature.ProviderError, &handler)
		case "STALE":
			s.Client.RemoveHandler(openfeature.ProviderStale, &handler)
		case "CONFIGURATION_CHANGE":
			s.Client.RemoveHandler(openfeature.ProviderConfigChange, &handler)
		}
	}
}
