package integration

import (
	"fmt"
	"time"
	
	"github.com/cucumber/godog"
	"github.com/open-feature/go-sdk/openfeature"
)

// initializeEventSteps registers event handling step definitions
func initializeEventSteps(ctx *godog.ScenarioContext, state *TestState) {
	ctx.Step(`^a change event handler$`, state.addChangeEventHandler)
	ctx.Step(`^a change event was fired$`, state.waitForChangeEvent)
	ctx.Step(`^the flag should be part of the event payload$`, state.assertFlagInChangeEvent)
}

// addChangeEventHandler adds a handler for flag configuration change events
func (s *TestState) addChangeEventHandler() error {
	if s.Client == nil {
		return fmt.Errorf("no client available to add event handler")
	}
	
	handler := func(details openfeature.EventDetails) {
		s.addEvent("CONFIGURATION_CHANGE", details)
	}
	
	s.EventHandlers["CONFIGURATION_CHANGE"] = handler
	s.Client.AddHandler(openfeature.ProviderConfigChange, &handler)
	return nil
}

// waitForChangeEvent waits for a configuration change event to occur
func (s *TestState) waitForChangeEvent() error {
	// Wait for the change event with a reasonable timeout
	return s.waitForEvents("CONFIGURATION_CHANGE", 5*time.Second)
}

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
			if event.Details.FlagNames != nil {
				for _, flagName := range event.Details.FlagNames {
					if flagName == s.FlagKey {
						return nil
					}
				}
			}
			
			// If no specific flags are listed, assume all flags are affected
			if event.Details.FlagNames == nil || len(event.Details.FlagNames) == 0 {
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