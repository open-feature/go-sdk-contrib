package tck_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/open-feature/go-sdk-contrib/tools/provider-tck/pkg/tck"
	"github.com/open-feature/go-sdk/openfeature"
	"github.com/open-feature/go-sdk/openfeature/memprovider"
)

// TestReportNeverCallsASkippedScenarioPassed is the reason the report exists.
//
// Appendix F requires that a scenario skipped for an undeclared capability is
// reported as skipped with the reason, never as passed. Go's runner does not
// honour that in its own summary: godog counts capability-gated skips in its
// passed tally, so the headline number the suite prints says something false and
// only a separate log line reveals it.
//
// The report is what makes the rule checkable rather than aspirational, so this
// test asserts the property directly: every scenario carrying a tag the suite
// did not declare appears in the report as not-declared, with a reason, and
// none of them appears as passed.
func TestReportNeverCallsASkippedScenarioPassed(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(tck.ReportDirEnv, dir)

	// Deliberately narrow: declaring only Object leaves every event, lifecycle,
	// stale, unavailable and strict-numeric-typing scenario ungated and skipped,
	// which is precisely the situation the rule governs.
	tck.Run(t, tck.Config{
		Name:    "report-selftest",
		Control: plainMemoryControl{},
		NewProvider: func(context.Context) (openfeature.FeatureProvider, error) {
			return memprovider.NewInMemoryProvider(tck.CanonicalFlagSet()), nil
		},
		Capabilities: []tck.Capability{tck.Object},
	})

	report := readReport(t, filepath.Join(dir, "report-selftest.json"))

	declared := map[string]bool{tck.Object.Tag(): true}

	var skipped, passed int
	for _, scenario := range report.Scenarios {
		needsUndeclared := false
		for _, tag := range scenario.Tags {
			// Only tags that gate a capability matter; the feature files are
			// free to carry organisational tags that gate nothing.
			if _, gates := tck.CapabilityForTag(tag); gates && !declared[tag] {
				needsUndeclared = true
			}
		}

		switch {
		case needsUndeclared:
			skipped++
			if scenario.Outcome != tck.OutcomeNotDeclared {
				t.Errorf("scenario %q needs an undeclared capability but was reported as %q; "+
					"Appendix F requires it be reported as %q and never as passed",
					scenario.Name, scenario.Outcome, tck.OutcomeNotDeclared)
			}
			if scenario.Reason == "" {
				t.Errorf("scenario %q was skipped without a reason; the reason is what makes a "+
					"skip readable to someone comparing providers", scenario.Name)
			}
		case scenario.Outcome == tck.OutcomePassed:
			passed++
		default:
			t.Errorf("scenario %q needs no undeclared capability but was reported as %q: %s",
				scenario.Name, scenario.Outcome, scenario.Reason)
		}
	}

	if skipped == 0 {
		t.Fatal("no scenario was skipped, so this test asserted nothing; either the capability " +
			"gate stopped working or the feature files no longer carry capability tags")
	}
	if passed == 0 {
		t.Fatal("no scenario passed, so the suite did not really run")
	}

	// The count is the other half of the property. A report that simply omitted
	// the scenarios it did not run would satisfy every assertion above while
	// still misleading a consumer, who has no way to know how many questions
	// went unasked.
	if total := len(report.Scenarios); total != skipped+passed {
		t.Errorf("report accounts for %d scenarios but %d passed and %d were skipped; "+
			"every scenario in the suite must appear exactly once", total, passed, skipped)
	}
}

// TestReportRecordsUndeclaredCapabilities checks the capability summary agrees
// with the per-scenario detail, since a consumer may read either.
func TestReportRecordsUndeclaredCapabilities(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(tck.ReportDirEnv, dir)

	tck.Run(t, tck.Config{
		Name:    "capability-selftest",
		Control: plainMemoryControl{},
		NewProvider: func(context.Context) (openfeature.FeatureProvider, error) {
			return memprovider.NewInMemoryProvider(tck.CanonicalFlagSet()), nil
		},
		Capabilities: []tck.Capability{tck.Object},
	})

	report := readReport(t, filepath.Join(dir, "capability-selftest.json"))

	for _, capability := range tck.AllCapabilities() {
		result, ok := report.Capabilities[capability.Tag()]
		if !ok {
			t.Errorf("capability %s is missing from the report; a capability is omitted only when "+
				"it is declared and no scenario exercises it, which is not the case here",
				capability.Tag())
			continue
		}

		want := tck.OutcomeNotDeclared
		if capability == tck.Object {
			want = tck.OutcomePassed
		}
		if result.State != want {
			t.Errorf("capability %s reported as %q, want %q", capability.Tag(), result.State, want)
		}
		if want == tck.OutcomeNotDeclared && result.Reason == "" {
			t.Errorf("capability %s is not declared but carries no reason", capability.Tag())
		}
	}
}

// TestReportNotWrittenByDefault keeps report emission opt-in.
//
// A suite that wrote files into the working directory of every developer who
// ran it would be a nuisance, and worse, a report written by accident is a
// report nobody checked.
func TestReportNotWrittenByDefault(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(tck.ReportDirEnv, "")

	tck.Run(t, tck.Config{
		Name:    "no-report",
		Control: plainMemoryControl{},
		NewProvider: func(context.Context) (openfeature.FeatureProvider, error) {
			return memprovider.NewInMemoryProvider(tck.CanonicalFlagSet()), nil
		},
		Capabilities: []tck.Capability{tck.Object},
	})

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("reading %s: %v", dir, err)
	}
	if len(entries) != 0 {
		t.Errorf("no report directory was configured but %d file(s) were written", len(entries))
	}
}

// TestSpecRevisionIsRecorded guards the generated constants.
//
// They are what lets a consumer know which questions a report answers, and they
// are generated rather than written, so the failure mode is silence: a
// regenerate that stopped emitting them would leave the report structurally
// valid and semantically useless.
func TestSpecRevisionIsRecorded(t *testing.T) {
	if len(tck.SpecRevision) != 40 {
		t.Errorf("SpecRevision = %q, want a 40-character commit SHA; run `make provider-tck-assets`",
			tck.SpecRevision)
	}
	if len(tck.AssetsTree) != 40 {
		t.Errorf("AssetsTree = %q, want a 40-character tree SHA; run `make provider-tck-assets`",
			tck.AssetsTree)
	}
}

// TestOutlineRowsAreDistinguishable is the property the example field exists to
// establish.
//
// Every row of a Scenario Outline shares one feature and one name. The
// type-mismatch matrix in errors.feature is eleven rows, so before the field
// existed the report held eleven entries differing only in how long each took;
// if one row failed and ten passed, nothing in the report said which. The
// identity of a row is its parameters, so this asserts that feature, name and
// example together are unique across the whole report -- which is exactly the
// key a consumer needs to be able to use.
func TestOutlineRowsAreDistinguishable(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(tck.ReportDirEnv, dir)

	tck.Run(t, tck.Config{
		Name:    "outline-identity",
		Control: plainMemoryControl{},
		NewProvider: func(context.Context) (openfeature.FeatureProvider, error) {
			return memprovider.NewInMemoryProvider(tck.CanonicalFlagSet()), nil
		},
		Capabilities: []tck.Capability{tck.Object},
	})

	report := readReport(t, filepath.Join(dir, "outline-identity.json"))

	seen := map[string]bool{}
	for _, scenario := range report.Scenarios {
		key := scenarioKey(scenario)
		if seen[key] {
			t.Errorf("two entries share the identity %s; a Scenario Outline row is identified by "+
				"its parameters, so a consumer keying on feature, name and example would keep "+
				"only one of them", key)
		}
		seen[key] = true
	}

	// A report where nothing came from an outline would satisfy the loop above
	// while asserting nothing, so the matrix itself is pinned: eleven rows,
	// eleven different parameter sets.
	const matrix = "Requesting the wrong type returns the code default"
	examples := map[string]bool{}
	rows := 0
	for _, scenario := range report.Scenarios {
		if scenario.Name != matrix {
			continue
		}
		rows++
		if len(scenario.Example) == 0 {
			t.Errorf("row %d of %q carries no example, so it is indistinguishable from the others",
				rows, matrix)
			continue
		}
		examples[canonicalExample(scenario.Example)] = true
	}
	if rows == 0 {
		t.Fatalf("no entry for %q; either the feature files changed or the suite did not run", matrix)
	}
	if len(examples) != rows {
		t.Errorf("%d rows of %q produced %d distinct examples", rows, matrix, len(examples))
	}

	// The field is present only for outline rows. Emitting an empty object for an
	// ordinary scenario would be a second thing for four implementations to agree
	// on, and the schema asks for omission instead.
	const ordinary = "An unknown flag key returns the code default"
	found := false
	for _, scenario := range report.Scenarios {
		if scenario.Name != ordinary {
			continue
		}
		found = true
		if scenario.Example != nil {
			t.Errorf("%q is not a Scenario Outline but carries example %v", ordinary, scenario.Example)
		}
	}
	if !found {
		t.Errorf("no entry for %q, so the omission of example was not checked", ordinary)
	}
}

// TestSkippedOutlineRowsCarryTheirExample covers the capability gate, which
// records its outcome before the scenario runs and so is a second place the
// example has to be filled in.
//
// A skipped outline row is exactly as ambiguous as a failed one: the four rows
// of the @object outline are four entries that differ in nothing without it.
func TestSkippedOutlineRowsCarryTheirExample(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(tck.ReportDirEnv, dir)

	tck.Run(t, tck.Config{
		Name:    "skipped-outline",
		Control: plainMemoryControl{},
		NewProvider: func(context.Context) (openfeature.FeatureProvider, error) {
			return memprovider.NewInMemoryProvider(tck.CanonicalFlagSet()), nil
		},
		// Empty rather than nil: a nil Capabilities means "declare everything",
		// and what this test needs is a provider that declares nothing, so the
		// @object outline is gated in the Before hook and never starts.
		Capabilities: []tck.Capability{},
	})

	report := readReport(t, filepath.Join(dir, "skipped-outline.json"))

	const outline = "Requesting a structured flag as a scalar returns the code default"
	examples := map[string]bool{}
	rows := 0
	for _, scenario := range report.Scenarios {
		if scenario.Name != outline {
			continue
		}
		rows++
		if scenario.Outcome != tck.OutcomeNotDeclared {
			t.Errorf("%q was reported as %q; @object was not declared", outline, scenario.Outcome)
		}
		if len(scenario.Example) == 0 {
			t.Errorf("a skipped row of %q carries no example, so the report cannot say which row "+
				"was skipped", outline)
			continue
		}
		examples[canonicalExample(scenario.Example)] = true
	}
	if rows == 0 {
		t.Fatalf("no entry for %q; the capability gate did not record it at all", outline)
	}
	if len(examples) != rows {
		t.Errorf("%d skipped rows of %q produced %d distinct examples", rows, outline, len(examples))
	}
}

// scenarioKey is the identity of a report entry: feature, name and example.
func scenarioKey(scenario tck.ReportScenario) string {
	return scenario.Feature + "/" + scenario.Name + "/" + canonicalExample(scenario.Example)
}

// canonicalExample renders an example so two of them compare equal exactly when
// their parameters do, independently of map iteration order.
func canonicalExample(example map[string]string) string {
	keys := make([]string, 0, len(example))
	for key := range example {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+example[key])
	}
	return strings.Join(parts, "\x00")
}

func readReport(t *testing.T, path string) tck.Report {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("no conformance report at %s: %v", path, err)
	}

	var report tck.Report
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatalf("report at %s is not valid JSON: %v", path, err)
	}
	if report.SchemaVersion == "" {
		t.Fatalf("report at %s has no schemaVersion", path)
	}
	return report
}

// TestReservedCapabilityIsNotReportedAsPassed covers the capability a provider
// declares and the suite never tests.
//
// @targeting is reserved: it exists in the vocabulary but no scenario carries
// it, because asserting that an evaluation context reached the backend needs an
// echo operation the control API does not have yet. Reporting it as passed would
// be a green result for a claim nothing tested -- the same vacuous pass the
// capability vocabulary was introduced to eliminate, arriving through the report
// instead of through the suite.
//
// Omitting it is the honest answer: the suite asked no question, so it has none
// to report. A consumer sees the tag is absent rather than a pass it cannot rely
// on.
func TestReservedCapabilityIsNotReportedAsPassed(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(tck.ReportDirEnv, dir)

	tck.Run(t, tck.Config{
		Name:    "reserved-capability",
		Control: plainMemoryControl{},
		NewProvider: func(context.Context) (openfeature.FeatureProvider, error) {
			return memprovider.NewInMemoryProvider(tck.CanonicalFlagSet()), nil
		},
		// Targeting is declared and no scenario carries it. Object is declared so
		// the suite still does something.
		Capabilities: []tck.Capability{tck.Object, tck.Targeting},
	})

	report := readReport(t, filepath.Join(dir, "reserved-capability.json"))

	if result, present := report.Capabilities[tck.Targeting.Tag()]; present {
		t.Errorf("%s was declared and no scenario exercises it, but the report states %q; "+
			"a capability the suite never tested must not be reported as a result",
			tck.Targeting.Tag(), result.State)
	}

	// The declared capability that is exercised must still be reported, so the
	// omission above is specific rather than a general failure to report.
	if result, present := report.Capabilities[tck.Object.Tag()]; !present {
		t.Errorf("%s was declared and exercised but is missing from the report", tck.Object.Tag())
	} else if result.State != tck.OutcomePassed {
		t.Errorf("%s reported as %q, want %q", tck.Object.Tag(), result.State, tck.OutcomePassed)
	}
}
