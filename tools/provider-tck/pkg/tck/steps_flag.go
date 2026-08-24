package tck

import (
	"context"
	"errors"
	"fmt"

	"github.com/cucumber/godog"
	"github.com/open-feature/go-sdk/openfeature"
)

// registerFlagSteps binds the steps that declare, evaluate and assert flags.
func registerFlagSteps(ctx *godog.ScenarioContext) {
	ctx.Step(`^an? ([A-Za-z]+)-flag with key "([^"]*)" and a default value "([^"]*)"$`, aFlagWithKeyAndDefault)
	ctx.Step(`^the flag was evaluated with details$`, theFlagWasEvaluatedWithDetails)
	ctx.Step(`^the resolved details value should be "([^"]*)"$`, theResolvedValueShouldBe)
	ctx.Step(`^the variant should be "([^"]*)"$`, theVariantShouldBe)
	ctx.Step(`^the reason should be "([^"]*)"$`, theReasonShouldBe)
	ctx.Step(`^the error-code should be "([^"]*)"$`, theErrorCodeShouldBe)
	ctx.Step(`^no exception should have been thrown$`, noExceptionShouldHaveBeenThrown)
	ctx.Step(`^the resolved object value should contain$`, theResolvedObjectValueShouldContain)
	ctx.Step(`^the resolved value is remembered$`, theResolvedValueIsRemembered)
	ctx.Step(`^the resolved details value should have changed$`, theResolvedValueShouldHaveChanged)
	ctx.Step(`^the flag was modified$`, theFlagWasModified)
}

// aFlagWithKeyAndDefault declares the flag the scenario is about, along with
// the type it is requested as. The two are independent on purpose: most of
// errors.feature asks for a flag as a type it is not.
func aFlagWithKeyAndDefault(ctx context.Context, rawType, key, rawDefault string) error {
	state, err := stateFrom(ctx)
	if err != nil {
		return err
	}

	typ, err := parseFlagType(rawType)
	if err != nil {
		return err
	}

	defaultValue, err := typ.parseValue(rawDefault)
	if err != nil {
		return fmt.Errorf("default value for flag %q: %w", key, err)
	}

	state.flag = &flagUnderTest{key: key, typ: typ, defaultValue: defaultValue}
	return nil
}

// theFlagWasEvaluatedWithDetails resolves the declared flag through the typed
// client method matching its declared type.
func theFlagWasEvaluatedWithDetails(ctx context.Context) error {
	state, err := stateFrom(ctx)
	if err != nil {
		return err
	}
	client, err := state.requireClient()
	if err != nil {
		return err
	}
	flag, err := state.requireFlag()
	if err != nil {
		return err
	}

	state.last = evaluate(ctx, client, flag)
	return nil
}

// evaluate performs one typed evaluation, recording a panic rather than letting
// it unwind.
//
// A panic is the Go analogue of the thrown exception the feature files forbid:
// an unhandled failure escaping a flag evaluation takes the host application
// down. A returned error is not that — in Go an errored evaluation returns both
// the code default and a non-nil error, which is the normal, correct shape.
func evaluate(ctx context.Context, client *openfeature.Client, flag *flagUnderTest) (result *evaluation) {
	result = &evaluation{}

	defer func() {
		if v := recover(); v != nil {
			result.panicked = true
			result.panicValue = v
		}
	}()

	empty := openfeature.EvaluationContext{}

	switch flag.typ {
	case typeBoolean:
		details, err := client.BooleanValueDetails(ctx, flag.key, flag.defaultValue.(bool), empty)
		fill(result, details.Value, details.ResolutionDetail, err)
	case typeString:
		details, err := client.StringValueDetails(ctx, flag.key, flag.defaultValue.(string), empty)
		fill(result, details.Value, details.ResolutionDetail, err)
	case typeInteger:
		details, err := client.IntValueDetails(ctx, flag.key, flag.defaultValue.(int64), empty)
		fill(result, details.Value, details.ResolutionDetail, err)
	case typeFloat:
		details, err := client.FloatValueDetails(ctx, flag.key, flag.defaultValue.(float64), empty)
		fill(result, details.Value, details.ResolutionDetail, err)
	case typeObject:
		details, err := client.ObjectValueDetails(ctx, flag.key, flag.defaultValue, empty)
		fill(result, details.Value, details.ResolutionDetail, err)
	default:
		result.err = fmt.Errorf("unknown flag type %q", flag.typ)
	}

	return result
}

// errorSuffix appends the error the client returned, when there was one, to a
// value mismatch. It is usually the shortest route to the cause: a code default
// where a resolved value was expected almost always arrives with an error
// explaining why.
func errorSuffix(err error) string {
	if err == nil {
		return ""
	}
	return fmt.Sprintf(" (the client also returned: %v)", err)
}

func fill(result *evaluation, value any, detail openfeature.ResolutionDetail, err error) {
	result.value = value
	result.variant = detail.Variant
	result.reason = detail.Reason
	result.errorCode = detail.ErrorCode
	result.err = err
}

func theResolvedValueShouldBe(ctx context.Context, raw string) error {
	state, err := stateFrom(ctx)
	if err != nil {
		return err
	}
	flag, err := state.requireFlag()
	if err != nil {
		return err
	}
	result, err := state.requireEvaluation()
	if err != nil {
		return err
	}

	expected, err := flag.typ.parseValue(raw)
	if err != nil {
		return fmt.Errorf("expected value for flag %q: %w", flag.key, err)
	}

	if !valuesEqual(expected, result.value) {
		return fmt.Errorf("flag %q resolved to %s, expected %s%s",
			flag.key, describeValue(result.value), describeValue(expected), errorSuffix(result.err))
	}
	return nil
}

func theVariantShouldBe(ctx context.Context, expected string) error {
	state, err := stateFrom(ctx)
	if err != nil {
		return err
	}
	result, err := state.requireEvaluation()
	if err != nil {
		return err
	}

	if result.variant != expected {
		return fmt.Errorf("variant was %q, expected %q. A variant that does not survive the trip "+
			"from the backend is one of the easiest parts of the contract to drop",
			result.variant, expected)
	}
	return nil
}

func theReasonShouldBe(ctx context.Context, expected string) error {
	state, err := stateFrom(ctx)
	if err != nil {
		return err
	}
	result, err := state.requireEvaluation()
	if err != nil {
		return err
	}

	if string(result.reason) != expected {
		return fmt.Errorf("reason was %q, expected %q", result.reason, expected)
	}
	return nil
}

// theErrorCodeShouldBe asserts the reported error code, where the empty string
// means no error code at all.
//
// The empty case matters as much as the populated ones. A provider that reports
// a plausible value with no error code is the failure mode the suite is most
// concerned with, because the application has no way to notice.
func theErrorCodeShouldBe(ctx context.Context, expected string) error {
	state, err := stateFrom(ctx)
	if err != nil {
		return err
	}
	result, err := state.requireEvaluation()
	if err != nil {
		return err
	}

	if string(result.errorCode) != expected {
		if expected == "" {
			return fmt.Errorf("error-code was %q, expected none", result.errorCode)
		}
		if result.errorCode == "" {
			return fmt.Errorf("no error-code was reported, expected %q. Returning the code default "+
				"without an error code leaves the application unable to tell that anything went wrong",
				expected)
		}
		return fmt.Errorf("error-code was %q, expected %q", result.errorCode, expected)
	}
	return nil
}

// noExceptionShouldHaveBeenThrown asserts that the evaluation did not panic.
//
// Go has no exceptions, and the returned error is not one: an errored
// evaluation correctly returns the code default alongside a non-nil error. The
// behaviour the feature files forbid — an unhandled failure escaping a flag
// evaluation and taking the application down — is a panic here.
func noExceptionShouldHaveBeenThrown(ctx context.Context) error {
	state, err := stateFrom(ctx)
	if err != nil {
		return err
	}
	result, err := state.requireEvaluation()
	if err != nil {
		return err
	}

	if result.panicked {
		return fmt.Errorf("the evaluation panicked: %v. A flag evaluation must always return a "+
			"value and an error code, never panic", result.panicValue)
	}
	return nil
}

// theResolvedObjectValueShouldContain asserts members of a structured value,
// each with its own expected type.
func theResolvedObjectValueShouldContain(ctx context.Context, table *godog.Table) error {
	state, err := stateFrom(ctx)
	if err != nil {
		return err
	}
	result, err := state.requireEvaluation()
	if err != nil {
		return err
	}

	rows, err := dataRows(table, "key", "type", "value")
	if err != nil {
		return err
	}

	for _, row := range rows {
		key, rawType, rawValue := row[0], row[1], row[2]

		typ, err := parseFlagType(rawType)
		if err != nil {
			return fmt.Errorf("member %q: %w", key, err)
		}
		expected, err := typ.parseValue(rawValue)
		if err != nil {
			return fmt.Errorf("member %q: %w", key, err)
		}

		actual, present, err := objectMember(result.value, key)
		if err != nil {
			return err
		}
		if !present {
			return fmt.Errorf("resolved object value has no member %q", key)
		}
		if !valuesEqual(expected, actual) {
			return fmt.Errorf("object member %q was %s, expected %s",
				key, describeValue(actual), describeValue(expected))
		}
	}

	return nil
}

// theResolvedValueIsRemembered stores the current value so that a later step
// can assert it changed.
func theResolvedValueIsRemembered(ctx context.Context) error {
	state, err := stateFrom(ctx)
	if err != nil {
		return err
	}
	result, err := state.requireEvaluation()
	if err != nil {
		return err
	}

	state.remembered = result.value
	state.hasMemory = true
	return nil
}

// theResolvedValueShouldHaveChanged asserts that re-evaluation produced a
// different value.
//
// This is the half of the configuration-change contract that providers actually
// get wrong. Emitting PROVIDER_CONFIGURATION_CHANGED and then continuing to
// resolve the old value is worse than emitting nothing, because the application
// acted on a signal that was not true.
func theResolvedValueShouldHaveChanged(ctx context.Context) error {
	state, err := stateFrom(ctx)
	if err != nil {
		return err
	}
	result, err := state.requireEvaluation()
	if err != nil {
		return err
	}
	if !state.hasMemory {
		return errors.New("no value was remembered in this scenario: " +
			"a \"the resolved value is remembered\" step must come first")
	}

	if valuesEqual(state.remembered, result.value) {
		return fmt.Errorf("the resolved value is still %s after the configuration changed. "+
			"The change was signalled but not applied, so the event told the application something untrue",
			describeValue(result.value))
	}
	return nil
}

// theFlagWasModified changes flag configuration on the backend.
func theFlagWasModified(ctx context.Context) error {
	state, err := stateFrom(ctx)
	if err != nil {
		return err
	}

	if err := state.cfg.Control.ChangeFlag(ctx); err != nil {
		return fmt.Errorf("could not change flag configuration on %s: %w",
			state.cfg.Control.Description(), err)
	}
	return nil
}

// dataRows validates a Gherkin table's header and returns its body rows.
func dataRows(table *godog.Table, header ...string) ([][]string, error) {
	if table == nil || len(table.Rows) == 0 {
		return nil, fmt.Errorf("expected a data table with columns %v", header)
	}

	head := table.Rows[0]
	if len(head.Cells) != len(header) {
		return nil, fmt.Errorf("expected a data table with columns %v, got %d columns", header, len(head.Cells))
	}
	for i, want := range header {
		if head.Cells[i].Value != want {
			return nil, fmt.Errorf("expected column %d to be %q, got %q", i+1, want, head.Cells[i].Value)
		}
	}

	rows := make([][]string, 0, len(table.Rows)-1)
	for _, row := range table.Rows[1:] {
		values := make([]string, len(row.Cells))
		for i, cell := range row.Cells {
			values[i] = cell.Value
		}
		rows = append(rows, values)
	}
	return rows, nil
}
