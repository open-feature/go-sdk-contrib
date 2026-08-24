package tck_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
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
			t.Errorf("capability %s is missing from the report; every capability is reported, "+
				"because an absent one is indistinguishable from one that was forgotten",
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
