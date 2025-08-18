package testframework

import (
	"fmt"
	"strings"
	"time"

	"github.com/cucumber/godog"
	"github.com/open-feature/go-sdk/openfeature"
)

// initializeEventSteps registers event handling step definitions
func initializeEventSteps(ctx *godog.ScenarioContext, state *TestState) {
	// Specific event handlers that have custom logic
	ctx.Step(`^the flag should be part of the event payload$`, state.assertFlagInChangeEvent)

	// Generic wildcard event handler patterns - future-proof
	ctx.Step(`^a (\w+) event handler$`, state.addGenericEventHandler)
	ctx.Step(`^a (\w+) event was fired$`, state.waitForGenericEvent)
	ctx.Step(`^the (\w+) event handler should have been executed$`, state.assertGenericEventExecuted)

	// Missing step definition - added as stub
	ctx.Step(`^the (\w+) event handler should have been executed within (\d+)ms$`, state.assertGenericEventExecutedWithin)
}

// Specific event handlers consolidated into generic handlers above

// assertFlagInChangeEvent verifies that the current flag is in the change event payload
func (s *TestState) assertFlagInChangeEvent() error {
	if s.FlagKey == "" {
		return fmt.Errorf("no flag key set for verification")
	}

	// Find the most recent CONFIGURATION_CHANGE event
	for i := len(s.Events) - 1; i >= 0; i-- {
		event := s.Events[i]
		if event.Type == "CONFIGURATION_CHANGE" {
			// Check if the flag key is in the event details
			if event.Details.FlagChanges != nil {
				for _, flagName := range event.Details.FlagChanges {
					if flagName == s.FlagKey {
						return nil
					}
				}
			}

			// If no specific flags are listed, assume all flags are affected
			if event.Details.FlagChanges == nil || len(event.Details.FlagChanges) == 0 {
				return nil
			}

			return fmt.Errorf("flag %s not found in change event payload", s.FlagKey)
		}
	}

	return fmt.Errorf("no configuration change event found")
}

// Event verification helpers

// getEventsOfType returns all events of a specific type
func (s *TestState) getEventsOfType(eventType string) []EventRecord {
	var events []EventRecord
	for _, event := range s.Events {
		if event.Type == eventType {
			events = append(events, event)
		}
	}
	return events
}

// getLastEventOfType returns the most recent event of a specific type
func (s *TestState) getLastEventOfType(eventType string) (*EventRecord, error) {
	for i := len(s.Events) - 1; i >= 0; i-- {
		event := s.Events[i]
		if event.Type == eventType {
			return &event, nil
		}
	}
	return nil, fmt.Errorf("no event of type %s found", eventType)
}

// clearEvents removes all recorded events (useful for test isolation)
func (s *TestState) clearEvents() {
	s.Events = []EventRecord{}
}

// assertEventSequence verifies that events occurred in a specific order
func (s *TestState) assertEventSequence(expectedSequence []string) error {
	if len(s.Events) < len(expectedSequence) {
		return fmt.Errorf("expected at least %d events, got %d", len(expectedSequence), len(s.Events))
	}

	// Check if the events match the expected sequence (allowing for additional events)
	sequenceIndex := 0
	for _, event := range s.Events {
		if sequenceIndex < len(expectedSequence) && event.Type == expectedSequence[sequenceIndex] {
			sequenceIndex++
		}
	}

	if sequenceIndex != len(expectedSequence) {
		return fmt.Errorf("event sequence incomplete: expected %v, found %d matching events", expectedSequence, sequenceIndex)
	}

	return nil
}

// assertEventWithinTimeframe verifies that an event occurred within a specific timeframe
func (s *TestState) assertEventWithinTimeframe(eventType string, maxAge time.Duration) error {
	event, err := s.getLastEventOfType(eventType)
	if err != nil {
		return err
	}

	age := time.Since(event.Timestamp)
	if age > maxAge {
		return fmt.Errorf("event %s occurred %v ago, which exceeds maximum age of %v", eventType, age, maxAge)
	}

	return nil
}

// Helper method for debugging events
func (s *TestState) debugEventHistory() string {
	result := "Event History:\n"
	for i, event := range s.Events {
		result += fmt.Sprintf("[%d] %s at %s\n", i, event.Type, event.Timestamp.Format("15:04:05.000"))
	}
	return result
}

// Generic event handler functions - consolidated and future-proof

// addGenericEventHandler adds a handler for any event type
func (s *TestState) addGenericEventHandler(eventType string) error {
	if s.Client == nil {
		return fmt.Errorf("no client available to add %s event handler", eventType)
	}

	handler := func(details openfeature.EventDetails) {
		s.addEvent(strings.ToUpper(eventType), details)
	}

	eventTypeUpper := strings.ToUpper(eventType)
	s.EventHandlers[eventTypeUpper] = handler

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
func (s *TestState) waitForGenericEvent(eventType string) error {
	timeout := 5 * time.Second
	return s.waitForEvents(strings.ToUpper(eventType), timeout)
}

// assertGenericEventExecuted verifies that any event type was received
func (s *TestState) assertGenericEventExecuted(eventType string) error {
	return s.assertEventOccurred(strings.ToUpper(eventType))
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

// Missing step definition implementation - added as stub that throws error

// assertGenericEventExecutedWithin checks if any event was executed within specified time
func (s *TestState) assertGenericEventExecutedWithin(eventType string, timeoutMs int) error {
	timeout := time.Duration(timeoutMs) * time.Millisecond
	return s.waitForEvents(strings.ToUpper(eventType), timeout)
}
