package testframework

import (
	"fmt"

	"github.com/cucumber/godog"
)

// initializeContextSteps registers evaluation context step definitions
func initializeContextSteps(ctx *godog.ScenarioContext, state *TestState) {
	ctx.Step(`^a context containing a key "([^"]*)", with type "([^"]*)" and with value "([^"]*)"$`, state.addContextValue)
	ctx.Step(`^an empty context$`, state.clearContext)
	ctx.Step(`^a context with the following keys:$`, state.addContextFromTable)

	// Missing step definitions - added as stubs
	ctx.Step(`^a context containing a nested property with outer key "([^"]*)" and inner key "([^"]*)", with value "([^"]*)"$`, state.addNestedContextProperty)
	ctx.Step(`^a context containing a targeting key with value "([^"]*)"$`, state.addTargetingKeyToContext)
}

// addContextValue adds a typed value to the evaluation context
func (s *TestState) addContextValue(key, valueType, value string) error {
	convertedValue, err := convertValueForSteps(value, valueType)
	if err != nil {
		return fmt.Errorf("failed to convert context value for key %s: %w", key, err)
	}

	s.EvalContext[key] = convertedValue
	return nil
}

// clearContext removes all values from the evaluation context
func (s *TestState) clearContext() error {
	s.EvalContext = make(map[string]interface{})
	return nil
}

// addContextFromTable adds multiple context values from a Gherkin data table
func (s *TestState) addContextFromTable(table *godog.Table) error {
	if len(table.Rows) < 2 {
		return fmt.Errorf("table must have at least header row and one data row")
	}

	header := table.Rows[0]
	if len(header.Cells) < 3 {
		return fmt.Errorf("table must have at least 3 columns: key, type, value")
	}

	// Find column indices
	var keyCol, typeCol, valueCol int = -1, -1, -1
	for i, cell := range header.Cells {
		switch cell.Value {
		case "key":
			keyCol = i
		case "type":
			typeCol = i
		case "value":
			valueCol = i
		}
	}

	if keyCol == -1 || typeCol == -1 || valueCol == -1 {
		return fmt.Errorf("table must have columns named 'key', 'type', and 'value'")
	}

	// Process data rows
	for _, row := range table.Rows[1:] {
		if len(row.Cells) <= keyCol || len(row.Cells) <= typeCol || len(row.Cells) <= valueCol {
			return fmt.Errorf("table row has insufficient columns")
		}

		key := row.Cells[keyCol].Value
		valueType := row.Cells[typeCol].Value
		value := row.Cells[valueCol].Value

		if err := s.addContextValue(key, valueType, value); err != nil {
			return err
		}
	}

	return nil
}

// Helper methods for context management

// hasContextKey checks if a key exists in the evaluation context
func (s *TestState) hasContextKey(key string) bool {
	_, exists := s.EvalContext[key]
	return exists
}

// getContextValue retrieves a value from the evaluation context
func (s *TestState) getContextValue(key string) (interface{}, bool) {
	value, exists := s.EvalContext[key]
	return value, exists
}

// contextSize returns the number of keys in the evaluation context
func (s *TestState) contextSize() int {
	return len(s.EvalContext)
}

// getContextKeys returns all keys in the evaluation context
func (s *TestState) getContextKeys() []string {
	keys := make([]string, 0, len(s.EvalContext))
	for key := range s.EvalContext {
		keys = append(keys, key)
	}
	return keys
}

// contextContainsKey checks if the evaluation context contains a specific key
func (s *TestState) contextContainsKey(key string) error {
	if !s.hasContextKey(key) {
		return fmt.Errorf("evaluation context does not contain key: %s", key)
	}
	return nil
}

// contextValueEquals checks if a context value equals the expected value
func (s *TestState) contextValueEquals(key string, expectedValue interface{}) error {
	actualValue, exists := s.getContextValue(key)
	if !exists {
		return fmt.Errorf("evaluation context does not contain key: %s", key)
	}

	if actualValue != expectedValue {
		return fmt.Errorf("context value for key %s: expected %v, got %v", key, expectedValue, actualValue)
	}

	return nil
}

// contextIsEmpty checks if the evaluation context is empty
func (s *TestState) contextIsEmpty() error {
	if s.contextSize() != 0 {
		return fmt.Errorf("expected empty context, but context has %d keys", s.contextSize())
	}
	return nil
}

// Additional step definitions for context validation

// registerContextValidationSteps adds validation steps for evaluation context
func (s *TestState) registerContextValidationSteps(ctx *godog.ScenarioContext) {
	ctx.Step(`^the context should contain key "([^"]*)"$`, s.contextContainsKeyStep)
	ctx.Step(`^the context should be empty$`, s.contextIsEmptyStep)
	ctx.Step(`^the context should have (\d+) keys?$`, s.contextShouldHaveKeysStep)
	ctx.Step(`^the context value for "([^"]*)" should be "([^"]*)" of type "([^"]*)"$`, s.contextValueShouldBeStep)
}

// contextContainsKeyStep is a step definition wrapper
func (s *TestState) contextContainsKeyStep(key string) error {
	return s.contextContainsKey(key)
}

// contextIsEmptyStep is a step definition wrapper
func (s *TestState) contextIsEmptyStep() error {
	return s.contextIsEmpty()
}

// contextShouldHaveKeysStep checks the number of keys in context
func (s *TestState) contextShouldHaveKeysStep(expectedCount int) error {
	actualCount := s.contextSize()
	if actualCount != expectedCount {
		return fmt.Errorf("expected context to have %d keys, but it has %d", expectedCount, actualCount)
	}
	return nil
}

// contextValueShouldBeStep checks a specific context value
func (s *TestState) contextValueShouldBeStep(key, expectedValue, valueType string) error {
	convertedExpected, err := convertValueForSteps(expectedValue, valueType)
	if err != nil {
		return fmt.Errorf("failed to convert expected value: %w", err)
	}

	return s.contextValueEquals(key, convertedExpected)
}

// Missing step definition implementations - added as stubs that throw errors

// addNestedContextProperty adds a nested property to evaluation context
func (s *TestState) addNestedContextProperty(outerKey, innerKey, value string) error {
	return s.addContextValue(outerKey, "Object", fmt.Sprintf("{\"%s\": \"%s\"}", innerKey, value))
}

// addTargetingKeyToContext adds a targeting key to evaluation context
func (s *TestState) addTargetingKeyToContext(value string) error {
	s.TargetingKey = value
	return nil
}
