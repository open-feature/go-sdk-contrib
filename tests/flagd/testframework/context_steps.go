package testframework

import (
	"context"
	"fmt"
	"github.com/cucumber/godog"
)

// initializeContextSteps registers evaluation context step definitions
func InitializeContextSteps(ctx *godog.ScenarioContext) {
	ctx.Step(`^a context containing a key "([^"]*)", with type "([^"]*)" and with value "([^"]*)"$`,
		withState3Args((*TestState).addContextValue))
	ctx.Step(`^an empty context$`,
		withStateNoArgs((*TestState).clearContext))
	ctx.Step(`^a context with the following keys:$`,
		withStateTable((*TestState).addContextFromTable))

	// Missing step definitions - added as stubs
	ctx.Step(`^a context containing a nested property with outer key "([^"]*)" and inner key "([^"]*)", with value "([^"]*)"$`,
		withState3Args((*TestState).addNestedContextProperty))
	ctx.Step(`^a context containing a targeting key with value "([^"]*)"$`,
		withState1Arg((*TestState).addTargetingKeyToContext))

	// Context validation steps
	ctx.Step(`^the context should contain key "([^"]*)"$`,
		withState1Arg((*TestState).contextContainsKeyStep))
	ctx.Step(`^the context should be empty$`,
		withStateNoArgs((*TestState).contextIsEmptyStep))
	ctx.Step(`^the context should have (\d+) keys?$`,
		withStateIntArg((*TestState).contextShouldHaveKeysStep))
	ctx.Step(`^the context value for "([^"]*)" should be "([^"]*)" of type "([^"]*)"$`,
		withState3Args((*TestState).contextValueShouldBeStep))
}

// Additional helper functions for different signatures
func withState1Arg(fn func(*TestState, context.Context, string) error) func(context.Context, string) error {
	return func(ctx context.Context, arg1 string) error {
		state := GetStateFromContext(ctx)
		if state == nil {
			return fmt.Errorf("test state not found in context")
		}
		return fn(state, ctx, arg1)
	}
}

func withStateTable(fn func(*TestState, context.Context, *godog.Table) error) func(context.Context, *godog.Table) error {
	return func(ctx context.Context, table *godog.Table) error {
		state := GetStateFromContext(ctx)
		if state == nil {
			return fmt.Errorf("test state not found in context")
		}
		return fn(state, ctx, table)
	}
}

func withStateIntArg(fn func(*TestState, context.Context, int) error) func(context.Context, int) error {
	return func(ctx context.Context, arg1 int) error {
		state := GetStateFromContext(ctx)
		if state == nil {
			return fmt.Errorf("test state not found in context")
		}
		return fn(state, ctx, arg1)
	}
}

// State methods - these now expect context as first parameter after state

// addContextValue adds a typed value to the evaluation context
func (s *TestState) addContextValue(ctx context.Context, key, valueType, value string) error {
	convertedValue, err := convertValueForSteps(value, valueType)
	if err != nil {
		return fmt.Errorf("failed to convert context value for key %s: %w", key, err)
	}

	s.EvalContext[key] = convertedValue
	return nil
}

// clearContext removes all values from the evaluation context
func (s *TestState) clearContext(ctx context.Context) error {
	s.EvalContext = make(map[string]interface{})
	return nil
}

// addContextFromTable adds multiple context values from a Gherkin data table
func (s *TestState) addContextFromTable(ctx context.Context, table *godog.Table) error {
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

		if err := s.addContextValue(ctx, key, valueType, value); err != nil {
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

// Step definition wrappers

// contextContainsKeyStep is a step definition wrapper
func (s *TestState) contextContainsKeyStep(ctx context.Context, key string) error {
	return s.contextContainsKey(key)
}

// contextIsEmptyStep is a step definition wrapper
func (s *TestState) contextIsEmptyStep(ctx context.Context) error {
	return s.contextIsEmpty()
}

// contextShouldHaveKeysStep checks the number of keys in context
func (s *TestState) contextShouldHaveKeysStep(ctx context.Context, expectedCount int) error {
	actualCount := s.contextSize()
	if actualCount != expectedCount {
		return fmt.Errorf("expected context to have %d keys, but it has %d", expectedCount, actualCount)
	}
	return nil
}

// contextValueShouldBeStep checks a specific context value
func (s *TestState) contextValueShouldBeStep(ctx context.Context, key, expectedValue, valueType string) error {
	convertedExpected, err := convertValueForSteps(expectedValue, valueType)
	if err != nil {
		return fmt.Errorf("failed to convert expected value: %w", err)
	}

	return s.contextValueEquals(key, convertedExpected)
}

// Missing step definition implementations

// addNestedContextProperty adds a nested property to evaluation context
func (s *TestState) addNestedContextProperty(ctx context.Context, outerKey, innerKey, value string) error {
	return s.addContextValue(ctx, outerKey, "Object", fmt.Sprintf("{\"%s\": \"%s\"}", innerKey, value))
}

// addTargetingKeyToContext adds a targeting key to evaluation context
func (s *TestState) addTargetingKeyToContext(ctx context.Context, value string) error {
	s.TargetingKey = value
	return nil
}
