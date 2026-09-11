package tck_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	messages "github.com/cucumber/messages/go/v21"
	"github.com/open-feature/go-sdk-contrib/tools/provider-tck/pkg/tck"
	"github.com/open-feature/go-sdk/openfeature"
	"github.com/open-feature/go-sdk/openfeature/memprovider"
)

// The report is an envelope plus a Cucumber Messages payload, so these tests
// read both. The envelope's job is to identify what was tested and what the
// provider declared; the payload's job is to account for every scenario, and
// never to report a scenario the capability gate stopped as passed.

// TestResultsNeverCallASkippedScenarioPassed is the reason the report exists.
//
// Appendix F requires that a scenario skipped for an undeclared capability is
// reported as skipped with the reason, never as passed. Go's runner does not
// honour that in its own summary: godog counts capability-gated skips in its
// passed tally, so the headline number the suite prints says something false and
// only a separate log line reveals it.
//
// The Messages stream is what makes the rule checkable rather than aspirational,
// so this test asserts the property directly over the stream: every scenario
// carrying a tag the suite did not declare is SKIPPED with a reason, and none of
// them is PASSED.
func TestResultsNeverCallASkippedScenarioPassed(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(tck.ReportDirEnv, dir)

	// Deliberately narrow: declaring only Object leaves every event, lifecycle,
	// stale, unavailable and numeric-coercion scenario ungated and skipped,
	// which is precisely the situation the rule governs.
	tck.Run(t, tck.Config{
		Name:    "report-selftest",
		Control: plainMemoryControl{},
		NewProvider: func(context.Context) (openfeature.FeatureProvider, error) {
			return memprovider.NewInMemoryProvider(tck.CanonicalFlagSet()), nil
		},
		Capabilities: []tck.Capability{tck.Object},
	})

	run := readRun(t, dir, "report-selftest")
	declared := map[string]bool{tck.Object.Tag(): true}

	var skipped, passed int
	for _, tc := range run.cases {
		needsUndeclared := false
		for _, tag := range tc.tags {
			// Only tags that gate a capability matter; the feature files are
			// free to carry organisational tags that gate nothing.
			if _, gates := tck.CapabilityForTag(tag); gates && !declared[tag] {
				needsUndeclared = true
			}
		}

		switch {
		case needsUndeclared:
			skipped++
			if tc.status != messages.TestStepResultStatus_SKIPPED {
				t.Errorf("scenario %q needs an undeclared capability but the stream reports it as %s; "+
					"Appendix F requires it be reported as SKIPPED and never as passed",
					tc.name, tc.status)
			}
			if tc.message == "" {
				t.Errorf("scenario %q was skipped without a reason; the reason is what makes a "+
					"skip readable to someone comparing providers", tc.name)
			}
		case tc.status == messages.TestStepResultStatus_PASSED:
			passed++
		default:
			t.Errorf("scenario %q needs no undeclared capability but the stream reports it as %s: %s",
				tc.name, tc.status, tc.message)
		}
	}

	if skipped == 0 {
		t.Fatal("no scenario was skipped, so this test asserted nothing; either the capability " +
			"gate stopped working or the feature files no longer carry capability tags")
	}
	if passed == 0 {
		t.Fatal("no scenario passed, so the suite did not really run")
	}

	// The count is the other half of the property. A stream that simply omitted
	// the scenarios it did not run would satisfy every assertion above while
	// still misleading a consumer, who has no way to know how many questions
	// went unasked.
	if total := len(run.cases); total != skipped+passed {
		t.Errorf("the stream accounts for %d scenarios but %d passed and %d were skipped; "+
			"every scenario in the suite must appear exactly once", total, passed, skipped)
	}

	t.Logf("outcome counts for %q: %d scenarios, %d passed, %d skipped",
		"report-selftest", len(run.cases), passed, skipped)
}

// TestEveryScenarioIsAccountedForExactlyOnce pins the accounting property on
// its own, independently of any outcome.
//
// A test case per pickle, a pickle per scenario, and no scenario twice. The
// stream's own pickle list is the denominator, so this also catches a formatter
// that announced a scenario and then failed to open a test case for it — which
// is how a gated scenario would disappear rather than be reported.
func TestEveryScenarioIsAccountedForExactlyOnce(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(tck.ReportDirEnv, dir)

	tck.Run(t, tck.Config{
		Name:    "accounting",
		Control: plainMemoryControl{},
		NewProvider: func(context.Context) (openfeature.FeatureProvider, error) {
			return memprovider.NewInMemoryProvider(tck.CanonicalFlagSet()), nil
		},
		Capabilities: []tck.Capability{tck.Object},
	})

	run := readRun(t, dir, "accounting")

	if len(run.pickles) == 0 {
		t.Fatal("the stream carries no pickles, so the suite parsed nothing")
	}
	if len(run.cases) != len(run.pickles) {
		t.Errorf("the stream carries %d scenarios but %d results", len(run.pickles), len(run.cases))
	}

	seen := map[string]int{}
	for _, tc := range run.cases {
		seen[tc.pickleID]++
	}
	for id, count := range seen {
		if count != 1 {
			t.Errorf("scenario %s has %d results; every scenario must be reported exactly once",
				id, count)
		}
	}
	for id := range run.pickles {
		if seen[id] == 0 {
			t.Errorf("scenario %s appears in the stream but has no result at all", id)
		}
	}

	// Every result must resolve to a status. UNKNOWN means the formatter opened
	// a test case and never heard what happened to it, which would report a
	// scenario as neither run nor skipped.
	for _, tc := range run.cases {
		if tc.status == messages.TestStepResultStatus_UNKNOWN {
			t.Errorf("scenario %q has no outcome in the stream", tc.name)
		}
	}
}

// TestEnvelopeReferencesTheResults covers the split between the two files.
func TestEnvelopeReferencesTheResults(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(tck.ReportDirEnv, dir)

	tck.Run(t, tck.Config{
		Name:    "envelope",
		Control: plainMemoryControl{},
		NewProvider: func(context.Context) (openfeature.FeatureProvider, error) {
			return memprovider.NewInMemoryProvider(tck.CanonicalFlagSet()), nil
		},
		Capabilities: []tck.Capability{tck.Object},
	})

	report := readReport(t, filepath.Join(dir, "envelope.json"))

	if report.Results.Format != "cucumber-messages" {
		t.Errorf("results.format = %q, want %q", report.Results.Format, "cucumber-messages")
	}

	// The location has to be resolvable relative to the envelope on any
	// platform, so it must be a bare name rather than anything built with a
	// filesystem separator.
	if strings.ContainsAny(report.Results.Location, `/\`) {
		t.Errorf("results.location = %q, which is not relative to the envelope",
			report.Results.Location)
	}
	if report.Results.Location != "envelope.ndjson" {
		t.Errorf("results.location = %q, want %q", report.Results.Location, "envelope.ndjson")
	}

	payload, err := os.ReadFile(filepath.Join(dir, report.Results.Location))
	if err != nil {
		t.Fatalf("results.location does not resolve to a file: %v", err)
	}
	if want := "sha256:" + sha256Hex(payload); report.Results.Digest != want {
		t.Errorf("results.digest = %q but the payload hashes to %q; a digest that does not match "+
			"is worse than none, since a consumer would reject a correct payload",
			report.Results.Digest, want)
	}

	// The declaration is an input to reading the results, not a summary of
	// them, so it has to say what the provider claimed rather than what
	// happened.
	if got := report.Declaration.Declared; len(got) != 1 || got[0] != tck.Object.Tag() {
		t.Errorf("declaration.declared = %v, want exactly [%s]", got, tck.Object.Tag())
	}
}

// TestDeclarationOfNothingIsAnEmptyListNotNull keeps the difference between
// stating none and saying nothing.
//
// The schema requires declared to be an array. A provider that declares no
// capability is making a claim; null would be silence, and a consumer cannot
// tell silence from a bug in the emitter.
func TestDeclarationOfNothingIsAnEmptyListNotNull(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(tck.ReportDirEnv, dir)

	tck.Run(t, tck.Config{
		Name:    "declares-nothing",
		Control: plainMemoryControl{},
		NewProvider: func(context.Context) (openfeature.FeatureProvider, error) {
			return memprovider.NewInMemoryProvider(tck.CanonicalFlagSet()), nil
		},
		// Empty rather than nil: nil means "declare everything".
		Capabilities: []tck.Capability{},
	})

	data, err := os.ReadFile(filepath.Join(dir, "declares-nothing.json"))
	if err != nil {
		t.Fatalf("no conformance report: %v", err)
	}
	if !strings.Contains(string(data), `"declared": []`) {
		t.Errorf("a provider declaring no capability did not emit an empty declared list:\n%s", data)
	}
}

// TestEnvelopeCarriesTheKnownDeviations is what makes a withheld capability
// readable as a defect rather than a decision.
//
// The declaration says a capability was not claimed and the results say the
// scenario was skipped; neither says whether the provider declines the
// capability or merely fails at it. Appendix F puts that in knownDeviations, so
// the envelope has to carry what the adopter declared -- verbatim, because it is
// prose written for whoever compares two providers, and a summary the emitter
// paraphrased is no longer the author's statement.
func TestEnvelopeCarriesTheKnownDeviations(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(tck.ReportDirEnv, dir)

	const issue = "https://github.com/open-feature/flagd/issues/1996"

	tck.Run(t, tck.Config{
		Name:    "deviations",
		Control: plainMemoryControl{},
		NewProvider: func(context.Context) (openfeature.FeatureProvider, error) {
			return memprovider.NewInMemoryProvider(tck.CanonicalFlagSet()), nil
		},
		Capabilities: []tck.Capability{tck.Object},
		KnownDeviations: []tck.KnownDeviation{
			// A tracked gap behind a capability that is withheld.
			tck.TrackedDeviation(tck.NumericCoercion, issue, "narrows 0.5 to 0 with no error code"),
			// An untracked gap against a mandatory scenario, which belongs to
			// no capability -- the shape that must survive with an empty
			// capability rather than being dropped or defaulted.
			tck.UntrackedDeviation("", "resolves large-integer-flag through a float and rounds it"),
		},
	})

	report := readReport(t, filepath.Join(dir, "deviations.json"))

	if len(report.KnownDeviations) != 2 {
		t.Fatalf("knownDeviations has %d entries, want 2: %+v",
			len(report.KnownDeviations), report.KnownDeviations)
	}

	tracked := report.KnownDeviations[0]
	if tracked.Capability != tck.NumericCoercion {
		t.Errorf("knownDeviations[0].capability = %q, want %q", tracked.Capability, tck.NumericCoercion)
	}
	if tracked.Issue != issue {
		t.Errorf("knownDeviations[0].issue = %q, want %q", tracked.Issue, issue)
	}
	if tracked.Summary != "narrows 0.5 to 0 with no error code" {
		t.Errorf("knownDeviations[0].summary was not carried verbatim: %q", tracked.Summary)
	}

	untracked := report.KnownDeviations[1]
	if untracked.Capability != "" {
		t.Errorf("knownDeviations[1].capability = %q, want it absent", untracked.Capability)
	}
	if untracked.Issue != "" {
		t.Errorf("knownDeviations[1].issue = %q, want it absent", untracked.Issue)
	}

	// The declaration is unchanged by any of this: a deviation explains an
	// absence, it does not create or remove one.
	if got := report.Declaration.Declared; len(got) != 1 || got[0] != tck.Object.Tag() {
		t.Errorf("declaration.declared = %v, want exactly [%s]", got, tck.Object.Tag())
	}
}

// TestNoKnownDeviationsSaysNothingRatherThanNone is the opposite of the
// declared-capability rule, and deliberately so.
//
// An empty declared list is a claim -- this provider declares no capability --
// so it has to be emitted. Deviations are not a claim: a provider with no entry
// is not asserting it has no defects, only that it has not recorded any. An
// empty array would read as the former, so the field is omitted instead.
func TestNoKnownDeviationsSaysNothingRatherThanNone(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(tck.ReportDirEnv, dir)

	tck.Run(t, tck.Config{
		Name:    "no-deviations",
		Control: plainMemoryControl{},
		NewProvider: func(context.Context) (openfeature.FeatureProvider, error) {
			return memprovider.NewInMemoryProvider(tck.CanonicalFlagSet()), nil
		},
		Capabilities: []tck.Capability{tck.Object},
	})

	data, err := os.ReadFile(filepath.Join(dir, "no-deviations.json"))
	if err != nil {
		t.Fatalf("no conformance report: %v", err)
	}
	if strings.Contains(string(data), "knownDeviations") {
		t.Errorf("a provider recording no deviations still emitted the field:\n%s", data)
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

// TestSpecRevisionIsRecorded pins the constant to the module the assets come from.
//
// SpecRevision is what lets a consumer know which questions a report answers,
// and it is now written by hand: the assets arrive as a Go module, so there is
// no sync step left to generate it, and no build-time source to derive it from
// either (see revision.go). A hand-written constant can go stale in silence,
// which is the one failure mode that leaves a report structurally valid and
// semantically wrong — naming a revision the run did not use.
//
// So compare it against the only other place the revision is recorded, the
// module pin in go.mod. That file is inside the module, present in every clone
// and in every module zip, so moving the pin without updating the constant
// fails here rather than in a consumer's report.
func TestSpecRevisionIsRecorded(t *testing.T) {
	gomod, err := os.ReadFile(filepath.Join("..", "..", "go.mod"))
	if err != nil {
		t.Fatalf("reading go.mod: %v", err)
	}

	var pinned string
	for _, line := range strings.Split(string(gomod), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] == tck.SpecModulePath {
			pinned = fields[1]
			break
		}
	}
	if pinned == "" {
		t.Fatalf("go.mod does not require %s, so the conformance assets have no recorded revision",
			tck.SpecModulePath)
	}

	if tck.SpecRevision != pinned {
		t.Errorf("SpecRevision = %q but go.mod pins %s at %q; move the pin with `go get` and "+
			"update SpecRevision to match", tck.SpecRevision, tck.SpecModulePath, pinned)
	}
}

// TestOutlineRowsAreDistinguishable is the property Messages carries for free,
// and which the report used to carry a bespoke field for.
//
// Every row of a Scenario Outline shares one feature and one name. The
// type-mismatch matrix in errors.feature is eleven rows, so a result keyed on
// feature and name is eleven entries differing in nothing; if one row failed and
// ten passed, nothing would say which. Messages identifies the row exactly: a
// pickle's last AST node id is the id of the Examples TableRow it was expanded
// from, and the stream carries the gherkinDocument those ids belong to, so the
// row's cells are recoverable from the stream alone.
//
// That is strictly better than the field it replaces, which had to reproduce
// godog's node numbering by reparsing the feature files -- a coupling to
// godog's internals that this removes.
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

	run := readRun(t, dir, "outline-identity")

	// No two scenarios share an identity, where the identity is the pickle's
	// AST node ids -- the scenario node, plus the Examples row for an outline.
	seen := map[string]string{}
	for _, tc := range run.cases {
		key := tc.uri + "\x00" + strings.Join(tc.astNodeIDs, ",")
		if previous, clash := seen[key]; clash {
			t.Errorf("scenarios %q and %q share the AST identity %s; a consumer keying on it "+
				"would keep only one of them", previous, tc.name, key)
		}
		seen[key] = tc.name
	}

	// A stream where nothing came from an outline would satisfy the loop above
	// while asserting nothing, so the matrix itself is pinned: eleven rows,
	// eleven different parameter sets, recovered from the stream's own
	// gherkinDocument.
	const matrix = "Requesting the wrong type returns the code default"
	rows := 0
	cells := map[string]bool{}
	for _, tc := range run.cases {
		if tc.name != matrix {
			continue
		}
		rows++
		row, ok := run.exampleRow(tc)
		if !ok {
			t.Errorf("row %d of %q does not resolve to an Examples row, so it is indistinguishable "+
				"from the others", rows, matrix)
			continue
		}
		cells[strings.Join(row, "|")] = true
	}
	if rows != 11 {
		t.Errorf("%q produced %d results, want the 11 rows of the matrix in errors.feature",
			matrix, rows)
	}
	if len(cells) != rows {
		t.Errorf("%d rows of %q resolved to %d distinct Examples rows", rows, matrix, len(cells))
	}

	// An ordinary scenario resolves to no Examples row, which is how a consumer
	// tells the two apart without a separate field saying so.
	const ordinary = "An unknown flag key returns the code default"
	found := false
	for _, tc := range run.cases {
		if tc.name != ordinary {
			continue
		}
		found = true
		if row, ok := run.exampleRow(tc); ok {
			t.Errorf("%q is not a Scenario Outline but resolves to the Examples row %v", ordinary, row)
		}
	}
	if !found {
		t.Errorf("no result for %q, so the ordinary case was not checked", ordinary)
	}
}

// TestSkippedOutlineRowsAreDistinguishable covers the capability gate, which
// stops a scenario before its first step and so is the path most likely to lose
// a row's identity.
//
// A skipped outline row is exactly as ambiguous as a failed one: the four rows
// of the @object outline are four results that differ in nothing unless the row
// is recoverable.
func TestSkippedOutlineRowsAreDistinguishable(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(tck.ReportDirEnv, dir)

	tck.Run(t, tck.Config{
		Name:    "skipped-outline",
		Control: plainMemoryControl{},
		NewProvider: func(context.Context) (openfeature.FeatureProvider, error) {
			return memprovider.NewInMemoryProvider(tck.CanonicalFlagSet()), nil
		},
		// A provider that declares nothing, so the @object outline is gated in
		// the Before hook and never starts.
		Capabilities: []tck.Capability{},
	})

	run := readRun(t, dir, "skipped-outline")

	const outline = "Requesting a structured flag as a scalar returns the code default"
	rows := 0
	cells := map[string]bool{}
	for _, tc := range run.cases {
		if tc.name != outline {
			continue
		}
		rows++
		if tc.status != messages.TestStepResultStatus_SKIPPED {
			t.Errorf("%q is reported as %s; @object was not declared", outline, tc.status)
		}
		row, ok := run.exampleRow(tc)
		if !ok {
			t.Errorf("a skipped row of %q does not resolve to an Examples row, so the results "+
				"cannot say which row was skipped", outline)
			continue
		}
		cells[strings.Join(row, "|")] = true
	}
	if rows == 0 {
		t.Fatalf("no result for %q; the capability gate did not report it at all", outline)
	}
	if len(cells) != rows {
		t.Errorf("%d skipped rows of %q resolved to %d distinct Examples rows", rows, outline, len(cells))
	}
}

// TestTheExecutedSourceIsCarried is why assetsTree was dropped.
//
// The report used to carry a git tree hash over the asset directory so a
// consumer could tell which questions a run asked. The Messages stream carries
// the feature text godog actually parsed, which answers the same question with
// the source rather than with a hash of it -- and unlike the hash it cannot be
// asserted wrongly, because it is the input.
func TestTheExecutedSourceIsCarried(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(tck.ReportDirEnv, dir)

	tck.Run(t, tck.Config{
		Name:    "sources",
		Control: plainMemoryControl{},
		NewProvider: func(context.Context) (openfeature.FeatureProvider, error) {
			return memprovider.NewInMemoryProvider(tck.CanonicalFlagSet()), nil
		},
		Capabilities: []tck.Capability{tck.Object},
	})

	run := readRun(t, dir, "sources")

	if len(run.sources) == 0 {
		t.Fatal("the stream carries no feature source")
	}
	for uri, data := range run.sources {
		if !strings.Contains(data, "Feature:") {
			t.Errorf("the source for %s does not look like a feature file", uri)
		}
	}

	// Every scenario's feature must be present, or a consumer cannot see the
	// question behind a result.
	for _, tc := range run.cases {
		if _, ok := run.sources[tc.uri]; !ok {
			t.Errorf("scenario %q came from %s, whose source is not in the stream", tc.name, tc.uri)
		}
	}
}

// --- reading the two files back ----------------------------------------------

// runResults is a Cucumber Messages stream indexed the way these tests ask
// questions of it.
type runResults struct {
	sources map[string]string
	pickles map[string]*messages.Pickle
	cases   []resultCase
	// tableRows maps an Examples TableRow id to its cells, built from the
	// documents in the stream.
	tableRows map[string][]string
}

// resultCase is one scenario's result, flattened.
type resultCase struct {
	pickleID   string
	name       string
	uri        string
	tags       []string
	astNodeIDs []string
	// status is the most severe of the test case's step results, which is how
	// Cucumber derives a scenario's outcome; Messages has no per-test-case
	// status field.
	status  messages.TestStepResultStatus
	message string
}

// severity orders statuses so that the most severe of a test case's steps is
// the test case's outcome. This is Cucumber's own ordering.
var severity = map[messages.TestStepResultStatus]int{
	messages.TestStepResultStatus_UNKNOWN:   0,
	messages.TestStepResultStatus_PASSED:    1,
	messages.TestStepResultStatus_SKIPPED:   2,
	messages.TestStepResultStatus_PENDING:   3,
	messages.TestStepResultStatus_UNDEFINED: 4,
	messages.TestStepResultStatus_AMBIGUOUS: 5,
	messages.TestStepResultStatus_FAILED:    6,
}

// exampleRow resolves the Examples row a scenario was expanded from, using only
// what the stream carries.
//
// A pickle's AST node ids end with the id of the Examples TableRow for an
// outline row, and with the Scenario node for an ordinary scenario, so a lookup
// that misses is the ordinary case rather than an error.
func (r *runResults) exampleRow(tc resultCase) ([]string, bool) {
	if len(tc.astNodeIDs) == 0 {
		return nil, false
	}
	row, ok := r.tableRows[tc.astNodeIDs[len(tc.astNodeIDs)-1]]
	return row, ok
}

func readRun(t *testing.T, dir, name string) *runResults {
	t.Helper()

	report := readReport(t, filepath.Join(dir, name+".json"))
	path := filepath.Join(dir, report.Results.Location)

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("no results payload at %s: %v", path, err)
	}

	// A null where the protocol requires an array is the failure a Go emitter
	// falls into most easily: a nil slice marshals to null, and most of the
	// required array fields in the Messages types have no omitempty. Checked on
	// every stream these tests read, since the fields at risk depend on what
	// the feature files happen to contain.
	if strings.Contains(string(data), ":null") {
		t.Errorf("%s contains a null where the Messages protocol requires a value", path)
	}

	run := &runResults{
		sources:   map[string]string{},
		pickles:   map[string]*messages.Pickle{},
		tableRows: map[string][]string{},
	}

	// testCaseId -> pickleId, and testCaseStartedId -> testCaseId, so a step
	// result can be attributed to a scenario the way a consumer has to do it.
	casePickle := map[string]string{}
	startedCase := map[string]string{}
	worst := map[string]messages.TestStepResultStatus{}
	reasons := map[string]string{}
	var order []string

	for i, line := range strings.Split(strings.TrimRight(string(data), "\n"), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var envelope messages.Envelope
		if err := json.Unmarshal([]byte(line), &envelope); err != nil {
			t.Fatalf("line %d of %s is not a Cucumber Messages envelope: %v", i+1, path, err)
		}

		switch {
		case envelope.Source != nil:
			run.sources[envelope.Source.Uri] = envelope.Source.Data
		case envelope.GherkinDocument != nil:
			collectTableRows(envelope.GherkinDocument, run.tableRows)
		case envelope.Pickle != nil:
			run.pickles[envelope.Pickle.Id] = envelope.Pickle
		case envelope.TestCase != nil:
			casePickle[envelope.TestCase.Id] = envelope.TestCase.PickleId
		case envelope.TestCaseStarted != nil:
			startedCase[envelope.TestCaseStarted.Id] = envelope.TestCaseStarted.TestCaseId
			order = append(order, envelope.TestCaseStarted.Id)
		case envelope.TestStepFinished != nil:
			finished := envelope.TestStepFinished
			if finished.TestStepResult == nil {
				t.Fatalf("a testStepFinished in %s carries no result", path)
			}
			id := finished.TestCaseStartedId
			if severity[finished.TestStepResult.Status] > severity[worst[id]] {
				worst[id] = finished.TestStepResult.Status
			}
			if reasons[id] == "" && finished.TestStepResult.Message != "" {
				reasons[id] = finished.TestStepResult.Message
			}
		}
	}

	for _, startedID := range order {
		pickleID := casePickle[startedCase[startedID]]
		pickle, ok := run.pickles[pickleID]
		if !ok {
			t.Fatalf("%s reports a result for scenario %s, which the stream does not describe",
				path, pickleID)
		}

		tags := make([]string, 0, len(pickle.Tags))
		for _, tag := range pickle.Tags {
			tags = append(tags, tag.Name)
		}

		run.cases = append(run.cases, resultCase{
			pickleID:   pickleID,
			name:       pickle.Name,
			uri:        pickle.Uri,
			tags:       tags,
			astNodeIDs: pickle.AstNodeIds,
			status:     worst[startedID],
			message:    reasons[startedID],
		})
	}

	if len(run.cases) == 0 {
		t.Fatalf("%s reports no scenario results at all", path)
	}
	return run
}

// collectTableRows indexes every Examples TableRow in a document by its id.
func collectTableRows(document *messages.GherkinDocument, into map[string][]string) {
	if document == nil || document.Feature == nil {
		return
	}

	collect := func(scenario *messages.Scenario) {
		if scenario == nil {
			return
		}
		for _, examples := range scenario.Examples {
			if examples == nil {
				continue
			}
			for _, row := range examples.TableBody {
				if row == nil {
					continue
				}
				cells := make([]string, 0, len(row.Cells))
				for _, cell := range row.Cells {
					cells = append(cells, cell.Value)
				}
				into[row.Id] = cells
			}
		}
	}

	for _, child := range document.Feature.Children {
		if child == nil {
			continue
		}
		collect(child.Scenario)
		if child.Rule == nil {
			continue
		}
		for _, ruleChild := range child.Rule.Children {
			if ruleChild != nil {
				collect(ruleChild.Scenario)
			}
		}
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
	if report.Results.Location == "" {
		t.Fatalf("report at %s does not say where its results are", path)
	}
	return report
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
