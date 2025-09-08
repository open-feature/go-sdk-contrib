package testframework

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cucumber/godog"
	"github.com/open-feature/go-sdk/openfeature"
)

// InitializeEventSteps registers event handling step definitions
func InitializeEventSteps(ctx *godog.ScenarioContext) {
	// Specific event handlers that have custom logic
	ctx.Step(`^the flag should be part of the event payload$`,
		withStateNoArgs((*TestState).assertFlagInChangeEvent))

	// Generic wildcard event handler patterns - future-proof
	ctx.Step(`^a (\w+) event handler$`,
		withState1Arg((*TestState).addGenericEventHandler))
	ctx.Step(`^a (\w+) event was fired$`,
		withState1Arg((*TestState).waitForGenericEvent))
	ctx.Step(`^the (\w+) event handler should have been executed$`,
		withState1Arg((*TestState).assertGenericEventExecuted))

	// Missing step definition - added as stub
	ctx.Step(`^the (\w+) event handler should have been executed within (\d+)ms$`,
		withStateStringAndInt((*TestState).assertGenericEventExecutedWithin))
}

// Additional helper for string + int arguments
func withStateStringAndInt(fn func(*TestState, context.Context, string, int) error) func(context.Context, string, int) error {
	return func(ctx context.Context, arg1 string, arg2 int) error {
		state := GetStateFromContext(ctx)
		if state == nil {
			return fmt.Errorf("test state not found in context")
		}
		return fn(state, ctx, arg1, arg2)
	}
}

// State methods - these now expect context as first parameter after state

// assertFlagInChangeEvent verifies that the current flag is in the change event payload
func (s *TestState) assertFlagInChangeEvent(ctx context.Context) error {
	if s.FlagKey == "" {
		return fmt.Errorf("no flag key set for verification")
	}

	// Check the last event instead of waiting for a new one
	if s.LastEvent == nil {
		return fmt.Errorf("no event available to check")
	}

	if s.LastEvent.Type != "CONFIGURATION_CHANGE" {
		return fmt.Errorf("last event was %s, not CONFIGURATION_CHANGE", s.LastEvent.Type)
	}

	// Check if the flag key is in the event details
	if s.LastEvent.Details.FlagChanges != nil {
		for _, flagName := range s.LastEvent.Details.FlagChanges {
			if flagName == s.FlagKey {
				return nil
			}
		}
		return fmt.Errorf("flag %s not found in change event payload", s.FlagKey)
	}

	// If no specific flags are listed, assume all flags are affected
	return nil
}

// Event verification helpers

// clearEvents removes all recorded events (useful for test isolation)
func (s *TestState) clearEvents() {
	// Clear the last event
	s.LastEvent = nil

	// Drain the channel
	for {
		select {
		case <-s.EventChannel:
			// Continue draining
		default:
			// Channel is empty
			return
		}
	}
}

// Generic event handler functions - consolidated and future-proof

// addGenericEventHandler adds a handler for any event type
func (s *TestState) addGenericEventHandler(ctx context.Context, eventType string) error {
	if s.Client == nil {
		return fmt.Errorf("no client available to add %s event handler", eventType)
	}

	handler := func(details openfeature.EventDetails) {
		s.addEvent(strings.ToUpper(eventType), details)
	}

	eventTypeUpper := strings.ToUpper(eventType)

	// Map event types to OpenFeature event constants
	switch eventTypeUpper {
	case "READY":
		s.Client.AddHandler(openfeature.ProviderReady, &handler)
	case "ERROR":
		s.Client.AddHandler(openfeature.ProviderError, &handler)
	case "STALE":
		s.Client.AddHandler(openfeature.ProviderStale, &handler)
	case "CHANGE", "CONFIGURATION_CHANGE":
		s.Client.AddHandler(openfeature.ProviderConfigChange, &handler)
	default:
		return fmt.Errorf("unsupported event type: %s", eventType)
	}

	return nil
}

// waitForGenericEvent waits for any event type to be fired
func (s *TestState) waitForGenericEvent(ctx context.Context, eventType string) error {
	timeout := 5 * time.Second
	return s.waitForEvents(strings.ToUpper(eventType), timeout)
}

// assertGenericEventExecuted verifies that any event type was received
func (s *TestState) assertGenericEventExecuted(ctx context.Context, eventType string) error {
	return s.assertEventOccurred(strings.ToUpper(eventType))
}

// Event handler helpers for provider state changes
func (s *TestState) handleProviderStateChange(eventType string) func(openfeature.EventDetails) {
	return func(details openfeature.EventDetails) {
		s.addEvent(eventType, details)
	}
}

// assertGenericEventExecutedWithin checks if any event was executed within specified time
func (s *TestState) assertGenericEventExecutedWithin(ctx context.Context, eventType string, timeoutMs int) error {
	timeout := time.Duration(timeoutMs) * time.Millisecond
	return s.waitForEvents(strings.ToUpper(eventType), timeout)
}

// Event utility functions moved from step_definitions.go

// addEvent adds an event to the event channel with proper handling
func (s *TestState) addEvent(eventType string, details openfeature.EventDetails) {
	event := EventRecord{
		Type:      eventType,
		Timestamp: time.Now(),
		Details:   details,
	}

	// Send to channel for immediate notification (non-blocking)
	select {
	case s.EventChannel <- event:
		// Event sent successfully
	default:
		// Channel is full, skip to prevent blocking
		// This shouldn't happen with a buffered channel, but safety first
	}
}

// waitForEvents waits for specific events to occur using channels
func (s *TestState) waitForEvents(eventType string, maxWait time.Duration) error {
	timeout := time.After(maxWait)
	for {
		select {
		case <-timeout:
			return fmt.Errorf("timeout waiting for %s event", eventType)
		case event := <-s.EventChannel:
			// Store the last event regardless of type
			s.LastEvent = &event
			if event.Type == eventType {
				return nil
			}
			// Event was not the type we're looking for, continue waiting
		}
	}
}

// assertEventOccurred checks if a specific event occurred (with immediate timeout)
func (s *TestState) assertEventOccurred(eventType string) error {
	return s.waitForEvents(eventType, 10*time.Second)
}

// waitForEventWithPayload waits for a specific event type and validates its payload
func (s *TestState) waitForEventWithPayload(eventType string, maxWait time.Duration, validator func(openfeature.EventDetails) bool) (*EventRecord, error) {
	timeout := time.After(maxWait)
	for {
		select {
		case <-timeout:
			return nil, fmt.Errorf("timeout waiting for %s event with valid payload", eventType)
		case event := <-s.EventChannel:
			// Store the last event regardless of type or validation
			s.LastEvent = &event
			if event.Type == eventType && validator(event.Details) {
				return &event, nil
			}
			// Event was not the type we're looking for or payload didn't match, continue waiting
		}
	}
}
