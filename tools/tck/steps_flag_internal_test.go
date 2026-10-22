package tck

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/cucumber/godog"
	messages "github.com/cucumber/messages/go/v34"
	"github.com/open-feature/go-sdk/openfeature"
)

// stepState puts a fresh scenario state into a context, the way the suite's
// Before hook does, and hands back both so a test can drive a step and then
// read what it left behind.
func stepState(t *testing.T, opts ...Option) (context.Context, *scenarioState) {
	t.Helper()

	state := newScenarioState(newConfig(opts))
	return withState(context.Background(), state), state
}

// stepTable builds the data table a scenario hands a step, header row first.
func stepTable(header []string, rows ...[]string) *godog.Table {
	table := &godog.Table{}
	for _, cells := range append([][]string{header}, rows...) {
		row := &messages.PickleTableRow{}
		for _, cell := range cells {
			row.Cells = append(row.Cells, &messages.PickleTableCell{Value: cell})
		}
		table.Rows = append(table.Rows, row)
	}
	return table
}

// contextTable builds a table with the context step's own header.
func contextTable(rows ...[]string) *godog.Table {
	return stepTable([]string{"key", "type", "value"}, rows...)
}

// metadataTable builds a table with the specification's metadata header.
func metadataTable(rows ...[]string) *godog.Table {
	return stepTable([]string{"key", "metadata_type", "value"}, rows...)
}

// The context table rebuilds the evaluation context, and rebuilding is where an
// earlier statement about it can be lost or a later one ignored.

func TestATargetingKeySetEarlierSurvivesATableThatDoesNotMentionIt(t *testing.T) {
	ctx, state := stepState(t)

	if err := aContextContainingATargetingKey(ctx, "tim"); err != nil {
		t.Fatalf("a context containing a targeting key: %v", err)
	}
	if err := aContextWithTheFollowingKeys(ctx, contextTable([]string{"region", "String", "eu"})); err != nil {
		t.Fatalf("a context with the following keys: %v", err)
	}

	if got := state.evalContext.TargetingKey(); got != "tim" {
		t.Errorf("targeting key is %q, want %q: the table said nothing about it", got, "tim")
	}
	if got := state.evalContext.Attribute("region"); got != "eu" {
		t.Errorf("region is %s, want \"eu\"", describeValue(got))
	}
}

func TestATableRowMaySetTheTargetingKey(t *testing.T) {
	ctx, state := stepState(t)

	if err := aContextContainingATargetingKey(ctx, "tim"); err != nil {
		t.Fatalf("a context containing a targeting key: %v", err)
	}
	if err := aContextWithTheFollowingKeys(ctx, contextTable([]string{openfeature.TargetingKey, "String", "ada"})); err != nil {
		t.Fatalf("a context with the following keys: %v", err)
	}

	// The row is the later statement about it, so it is the one that holds.
	if got := state.evalContext.TargetingKey(); got != "ada" {
		t.Errorf("targeting key is %q, want %q", got, "ada")
	}
	// And it reaches the provider as the targeting key rather than as an
	// attribute that merely shares its name.
	if got, present := state.evalContext.Attributes()[openfeature.TargetingKey]; present {
		t.Errorf("the targeting key was left among the attributes as %s", describeValue(got))
	}
}

func TestARowsTypeDecidesWhatTheContextCarries(t *testing.T) {
	ctx, state := stepState(t)

	err := aContextWithTheFollowingKeys(ctx, contextTable(
		[]string{"region", "String", "eu"},
		[]string{"editions", "StringList", "one, two"},
		[]string{"seats", "Long", "4294967295"},
		[]string{"fresh", "Boolean", "true"},
	))
	if err != nil {
		t.Fatalf("a context with the following keys: %v", err)
	}

	if got := state.evalContext.Attribute("region"); got != "eu" {
		t.Errorf("region is %s, want \"eu\"", describeValue(got))
	}
	if got := state.evalContext.Attribute("editions"); !reflect.DeepEqual(got, []string{"one", "two"}) {
		t.Errorf("editions is %s, want a two-element list rather than one comma-joined string",
			describeValue(got))
	}
	if got := state.evalContext.Attribute("seats"); got != int64(4294967295) {
		t.Errorf("seats is %s, want int64(4294967295)", describeValue(got))
	}
	if got := state.evalContext.Attribute("fresh"); got != true {
		t.Errorf("fresh is %s, want true", describeValue(got))
	}
}

func TestEntriesFromTwoContextTablesAddUp(t *testing.T) {
	ctx, state := stepState(t)

	for _, row := range [][]string{{"region", "String", "eu"}, {"tier", "String", "gold"}} {
		if err := aContextWithTheFollowingKeys(ctx, contextTable(row)); err != nil {
			t.Fatalf("a context with the following keys: %v", err)
		}
	}

	if got := state.evalContext.Attribute("region"); got != "eu" {
		t.Errorf("the first table's entry is gone: region is %s", describeValue(got))
	}
	if got := state.evalContext.Attribute("tier"); got != "gold" {
		t.Errorf("tier is %s, want \"gold\"", describeValue(got))
	}
}

// The flag-metadata assertions. Each case is written so that the opposite
// implementation would fail it: a value that matches and one that does not,
// metadata that is empty and metadata that is not.

// stateWithMetadata puts an evaluation carrying metadata into the state, as the
// evaluation step would.
func stateWithMetadata(t *testing.T, metadata openfeature.FlagMetadata) context.Context {
	t.Helper()

	ctx, state := stepState(t)
	state.last = &evaluation{value: true, metadata: metadata}
	return ctx
}

func TestEveryMetadataTypeIsComparedByThatType(t *testing.T) {
	ctx := stateWithMetadata(t, openfeature.FlagMetadata{
		"string":  "1.0.2",
		"integer": int64(2),
		"float":   0.1,
		"boolean": true,
	})

	err := theResolvedMetadataShouldContain(ctx, metadataTable(
		[]string{"string", "String", "1.0.2"},
		[]string{"integer", "Integer", "2"},
		[]string{"float", "Float", "0.1"},
		[]string{"boolean", "Boolean", "true"},
	))
	if err != nil {
		t.Fatalf("the resolved metadata should contain: %v", err)
	}
}

func TestAMetadataValueThatDiffersFailsNamingTheKey(t *testing.T) {
	ctx := stateWithMetadata(t, openfeature.FlagMetadata{"integer": int64(2)})

	err := theResolvedMetadataShouldContain(ctx, metadataTable([]string{"integer", "Integer", "3"}))
	if err == nil {
		t.Fatal("a metadata value of 2 satisfied an assertion asking for 3")
	}
	for _, want := range []string{`"integer"`, "Integer"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the failure does not mention %s: %v", want, err)
		}
	}
}

func TestAMetadataEntryTheTableDoesNotListIsNotChecked(t *testing.T) {
	ctx := stateWithMetadata(t, openfeature.FlagMetadata{"listed": "yes", "unlisted": "whatever"})

	if err := theResolvedMetadataShouldContain(ctx, metadataTable([]string{"listed", "String", "yes"})); err != nil {
		t.Fatalf("an unlisted entry was treated as part of the assertion: %v", err)
	}
}

func TestAMetadataEntryOfTheWrongTypeIsNotSilentlyAccepted(t *testing.T) {
	// A provider that put its version number in as a float cannot be read back
	// with GetInt, which is the whole of what the typed row asks.
	ctx := stateWithMetadata(t, openfeature.FlagMetadata{"integer": 2.0})

	err := theResolvedMetadataShouldContain(ctx, metadataTable([]string{"integer", "Integer", "2"}))
	if err == nil {
		t.Fatal("a float64 entry satisfied a row declaring Integer")
	}
	if !strings.Contains(err.Error(), "is not readable as Integer") {
		t.Errorf("the failure does not say the entry is unreadable as that type: %v", err)
	}
}

func TestAMetadataKeyThatIsAbsentFails(t *testing.T) {
	ctx := stateWithMetadata(t, openfeature.FlagMetadata{})

	err := theResolvedMetadataShouldContain(ctx, metadataTable([]string{"missing", "String", "x"}))
	if err == nil {
		t.Fatal("an absent metadata key satisfied the assertion")
	}
}

func TestEmptyMetadataSatisfiesTheEmptyAssertionAndAnythingElseDoesNot(t *testing.T) {
	if err := theResolvedMetadataIsEmpty(stateWithMetadata(t, openfeature.FlagMetadata{})); err != nil {
		t.Fatalf("empty metadata failed the empty assertion: %v", err)
	}

	err := theResolvedMetadataIsEmpty(stateWithMetadata(t, openfeature.FlagMetadata{"a": "b"}))
	if err == nil {
		t.Fatal("metadata carrying an entry satisfied the empty assertion")
	}
}

func TestAMetadataTypeFlagMetadataCannotCarryIsRefused(t *testing.T) {
	ctx := stateWithMetadata(t, openfeature.FlagMetadata{"a": "b"})

	err := theResolvedMetadataShouldContain(ctx, metadataTable([]string{"a", "Object", "{}"}))
	if err == nil {
		t.Fatal("a structured metadata type was accepted, which no provider can store and no application read back")
	}
	if !strings.Contains(err.Error(), "flag metadata is String, Boolean, Integer, Float") {
		t.Errorf("the failure does not name the four types: %v", err)
	}
}

func TestAssertingMetadataBeforeAnyEvaluationNamesTheMissingStep(t *testing.T) {
	ctx, _ := stepState(t)

	err := theResolvedMetadataIsEmpty(ctx)
	if err == nil {
		t.Fatal("the metadata assertion passed with nothing evaluated")
	}
	if !strings.Contains(err.Error(), "the flag was evaluated with details") {
		t.Errorf("the failure does not name the step that is missing: %v", err)
	}
}
