package tck

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	messages "github.com/cucumber/messages/go/v21"
)

// These tests drive the Messages formatter through godog's formatter events
// directly, rather than by running the suite.
//
// Two of the statuses cannot be reached end to end. The self-test suites all
// pass, so nothing that runs a real suite produces a FAILED step, and a failed
// scenario is exactly the case where the outcome has to be right. Driving the
// events also pins the assumption the whole design rests on -- that a
// capability-gated scenario reaches a formatter as SKIPPED -- against the shape
// of the call sequence rather than against godog's summary, which counts such a
// scenario as passed.

// syntheticStream builds a formatter over a buffer, with a fixed skip reason.
func syntheticStream(reason string) (*messagesFormatter, *lockedBuffer) {
	buf := &lockedBuffer{}
	return newMessagesFormatter(&messagesSink{
		out:        buf,
		skipReason: func(string) string { return reason },
	}), buf
}

func syntheticPickle(id, name string, tags ...string) *messages.Pickle {
	pickle := &messages.Pickle{
		Id:         id,
		Uri:        "gherkin/synthetic.feature",
		Name:       name,
		AstNodeIds: []string{"ast-" + id},
		Steps: []*messages.PickleStep{
			{Id: id + "-step-1", Text: "a step", AstNodeIds: []string{"ast-" + id + "-1"}},
			{Id: id + "-step-2", Text: "another step", AstNodeIds: []string{"ast-" + id + "-2"}},
		},
		Tags: []*messages.PickleTag{},
	}
	for _, tag := range tags {
		pickle.Tags = append(pickle.Tags, &messages.PickleTag{Name: tag})
	}
	return pickle
}

// TestGatedScenarioIsSkippedNotPassed is the property Appendix F turns on.
//
// The call sequence is the one godog's suite produces for a Before hook that
// returns an error wrapping godog.ErrSkip: the pickle is announced, then every
// step is defined and reported as skipped. Nothing in that sequence is a pass,
// and the assertion is that nothing in the stream says otherwise.
func TestGatedScenarioIsSkippedNotPassed(t *testing.T) {
	formatter, buf := syntheticStream(
		"requires capability @stale, which this provider does not declare")

	pickle := syntheticPickle("1", "an outage is detected", "@events", "@stale")

	formatter.TestRunStarted()
	formatter.Pickle(pickle)
	for _, step := range pickle.Steps {
		formatter.Defined(pickle, step, nil)
		formatter.Skipped(pickle, step, nil)
	}
	formatter.Summary()

	stream := parseStream(t, buf.bytes())

	if got := stream.worst("1"); got != messages.TestStepResultStatus_SKIPPED {
		t.Errorf("a capability-gated scenario is reported as %s; Appendix F requires SKIPPED and "+
			"forbids passed", got)
	}
	if reason := stream.message("1"); !strings.Contains(reason, "@stale") {
		t.Errorf("the skip carries the reason %q, which does not name the capability that caused "+
			"it; the reason is what makes a skip readable", reason)
	}
	if stream.statuses["1"][messages.TestStepResultStatus_PASSED] > 0 {
		t.Error("a step of a gated scenario is reported as PASSED")
	}
}

// TestFailedScenarioIsFailed covers the branch no passing suite reaches.
//
// godog reports the failing step and then skips the rest, so the test case has
// a mixture, and the most severe result has to be the failure rather than the
// skip that followed it.
func TestFailedScenarioIsFailed(t *testing.T) {
	formatter, buf := syntheticStream("")

	pickle := syntheticPickle("2", "a structured flag resolves", "@object")

	formatter.TestRunStarted()
	formatter.Pickle(pickle)
	formatter.Defined(pickle, pickle.Steps[0], nil)
	formatter.Failed(pickle, pickle.Steps[0], nil, errSynthetic{})
	formatter.Defined(pickle, pickle.Steps[1], nil)
	formatter.Skipped(pickle, pickle.Steps[1], nil)
	formatter.Summary()

	stream := parseStream(t, buf.bytes())

	if got := stream.worst("2"); got != messages.TestStepResultStatus_FAILED {
		t.Errorf("a scenario with a failing step is reported as %s, want FAILED", got)
	}
	if msg := stream.message("2"); !strings.Contains(msg, "resolved to nil") {
		t.Errorf("the failure carries the message %q, which does not say what went wrong", msg)
	}
	if stream.success {
		t.Error("testRunFinished reports success for a run with a failing scenario")
	}
}

// TestSkippedScenarioDoesNotFailTheRun keeps a capability skip out of the
// run-level verdict.
//
// A provider that declines a capability has not failed anything, and a stream
// that said otherwise would make declining a capability indistinguishable from
// being broken.
func TestSkippedScenarioDoesNotFailTheRun(t *testing.T) {
	formatter, buf := syntheticStream("requires capability @stale")

	pickle := syntheticPickle("3", "an outage is detected", "@stale")

	formatter.TestRunStarted()
	formatter.Pickle(pickle)
	for _, step := range pickle.Steps {
		formatter.Defined(pickle, step, nil)
		formatter.Skipped(pickle, step, nil)
	}
	formatter.Summary()

	if !parseStream(t, buf.bytes()).success {
		t.Error("testRunFinished reports failure for a run whose only skips were capability skips")
	}
}

// TestStreamCarriesTagsAndSource keeps the two facts the report stopped
// carrying itself.
//
// The declaration in the envelope only says what the provider claimed. Deciding
// whether a skip was legitimate needs the scenario's tags, and checking which
// question was asked needs the source, so both have to be in the stream.
func TestStreamCarriesTagsAndSource(t *testing.T) {
	formatter, buf := syntheticStream("")

	document := &messages.GherkinDocument{
		Uri:      "gherkin/synthetic.feature",
		Comments: []*messages.Comment{},
	}
	pickle := syntheticPickle("4", "a flag resolves", "@object")

	formatter.TestRunStarted()
	formatter.Feature(document, "gherkin/synthetic.feature",
		[]byte("Feature: synthetic\n"))
	formatter.Pickle(pickle)
	for _, step := range pickle.Steps {
		formatter.Defined(pickle, step, nil)
		formatter.Passed(pickle, step, nil)
	}
	formatter.Summary()

	stream := parseStream(t, buf.bytes())

	if source := stream.sources["gherkin/synthetic.feature"]; source != "Feature: synthetic\n" {
		t.Errorf("the stream carries the source %q, not the bytes that were executed", source)
	}
	if tags := stream.tags["4"]; len(tags) != 1 || tags[0] != "@object" {
		t.Errorf("the stream carries the tags %v, want [@object]", tags)
	}
	if got := stream.worst("4"); got != messages.TestStepResultStatus_PASSED {
		t.Errorf("a scenario whose every step passed is reported as %s", got)
	}
}

// TestEveryLineIsOneEnvelope pins the ndjson framing, which is what makes the
// payload readable by anything that consumes Messages.
func TestEveryLineIsOneEnvelope(t *testing.T) {
	formatter, buf := syntheticStream("")

	pickle := syntheticPickle("5", "a flag resolves")

	formatter.TestRunStarted()
	formatter.Pickle(pickle)
	for _, step := range pickle.Steps {
		formatter.Defined(pickle, step, nil)
		formatter.Passed(pickle, step, nil)
	}
	formatter.Summary()

	data := buf.bytes()
	if len(data) == 0 {
		t.Fatal("the formatter wrote nothing")
	}
	if data[len(data)-1] != '\n' {
		t.Error("the stream does not end with a newline, so appending to it would corrupt a line")
	}

	for i, line := range strings.Split(strings.TrimRight(string(data), "\n"), "\n") {
		var fields map[string]json.RawMessage
		if err := json.Unmarshal([]byte(line), &fields); err != nil {
			t.Fatalf("line %d is not a JSON object: %v", i+1, err)
		}
		if len(fields) != 1 {
			t.Errorf("line %d carries %d message types; an envelope carries exactly one",
				i+1, len(fields))
		}
	}

	// A null where the protocol requires an array is the failure a Go emitter
	// falls into most easily, because a nil slice marshals to null and most of
	// these fields have no omitempty.
	if strings.Contains(string(data), ":null") {
		t.Errorf("the stream contains a null where the protocol requires a value:\n%s", data)
	}
}

// TestMetaDescribesTheProducer keeps the stream self-describing.
func TestMetaDescribesTheProducer(t *testing.T) {
	formatter, buf := syntheticStream("")
	formatter.TestRunStarted()
	formatter.Summary()

	stream := parseStream(t, buf.bytes())
	if stream.meta == nil {
		t.Fatal("the stream carries no meta message")
	}
	if stream.meta.ProtocolVersion == "" {
		t.Error("meta does not state which version of the protocol the stream conforms to")
	}
	if stream.meta.Implementation == nil || stream.meta.Implementation.Name != tckImplementation {
		t.Errorf("meta does not name this TCK as the implementation: %+v", stream.meta.Implementation)
	}
	// Messages records the runner and the machine, not the subject under test.
	// That is why the report envelope still has to name the provider.
	if stream.meta.Runtime == nil || stream.meta.Runtime.Name != "go" {
		t.Errorf("meta does not name the runtime: %+v", stream.meta.Runtime)
	}
}

// --- reading a synthetic stream back -----------------------------------------

type parsedStream struct {
	meta     *messages.Meta
	sources  map[string]string
	tags     map[string][]string
	statuses map[string]map[messages.TestStepResultStatus]int
	reasons  map[string]string
	success  bool
}

func (s *parsedStream) worst(pickleID string) messages.TestStepResultStatus {
	worst := messages.TestStepResultStatus_UNKNOWN
	for status, count := range s.statuses[pickleID] {
		if count > 0 && statusSeverity[status] > statusSeverity[worst] {
			worst = status
		}
	}
	return worst
}

func (s *parsedStream) message(pickleID string) string { return s.reasons[pickleID] }

func parseStream(t *testing.T, data []byte) *parsedStream {
	t.Helper()

	stream := &parsedStream{
		sources:  map[string]string{},
		tags:     map[string][]string{},
		statuses: map[string]map[messages.TestStepResultStatus]int{},
		reasons:  map[string]string{},
	}

	casePickle := map[string]string{}
	startedPickle := map[string]string{}

	for i, line := range strings.Split(strings.TrimRight(string(data), "\n"), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var envelope messages.Envelope
		if err := json.Unmarshal([]byte(line), &envelope); err != nil {
			t.Fatalf("line %d is not a Cucumber Messages envelope: %v", i+1, err)
		}

		switch {
		case envelope.Meta != nil:
			stream.meta = envelope.Meta
		case envelope.Source != nil:
			stream.sources[envelope.Source.Uri] = envelope.Source.Data
		case envelope.Pickle != nil:
			tags := make([]string, 0, len(envelope.Pickle.Tags))
			for _, tag := range envelope.Pickle.Tags {
				tags = append(tags, tag.Name)
			}
			stream.tags[envelope.Pickle.Id] = tags
		case envelope.TestCase != nil:
			casePickle[envelope.TestCase.Id] = envelope.TestCase.PickleId
		case envelope.TestCaseStarted != nil:
			startedPickle[envelope.TestCaseStarted.Id] = casePickle[envelope.TestCaseStarted.TestCaseId]
		case envelope.TestStepFinished != nil:
			pickleID := startedPickle[envelope.TestStepFinished.TestCaseStartedId]
			result := envelope.TestStepFinished.TestStepResult
			if result == nil {
				t.Fatalf("line %d has a testStepFinished with no result", i+1)
			}
			if stream.statuses[pickleID] == nil {
				stream.statuses[pickleID] = map[messages.TestStepResultStatus]int{}
			}
			stream.statuses[pickleID][result.Status]++
			if stream.reasons[pickleID] == "" && result.Message != "" {
				stream.reasons[pickleID] = result.Message
			}
		case envelope.TestRunFinished != nil:
			stream.success = envelope.TestRunFinished.Success
		}
	}

	return stream
}

// errSynthetic is a step failure with a message worth asserting on.
type errSynthetic struct{}

func (errSynthetic) Error() string { return "resolved to nil, expected an object" }

// TestTheInexpressibleSkipReasonReachesTheReport is the report-side half of the
// distinction Appendix F requires, and the half a consumer actually reads.
//
// A capability-gated skip carries its reason into the Messages stream as the
// step result's message, so the two refusals have to differ there and not only
// in the log. "This provider does not declare it" describes a choice; a
// capability the Go SDK cannot express was never the provider's to choose, and
// recording the first when the second is true attributes a defect to a provider
// that has none.
//
// Go has no inexpressible capability, so the test installs one. See
// inexpressibleCapabilities for why the mechanism exists regardless.
func TestTheInexpressibleSkipReasonReachesTheReport(t *testing.T) {
	const property = "the integer accessor is 32 bits wide"
	withInexpressible(t, LargeIntegers, property)

	caps, err := newCapabilitySet([]Capability{Object})
	if err != nil {
		t.Fatalf("newCapabilitySet: %v", err)
	}
	r := &runner{caps: caps, cfg: config{Name: "gate", Control: stubControl{}}, t: t}

	unaskable := scenarioWithTags("an integer beyond 32 bits", "@large-integers")
	unaskable.Id = "pickle-unaskable"
	if _, err := r.beforeScenario(context.Background(), unaskable); err == nil {
		t.Fatal("the gate let a scenario needing an inexpressible capability run")
	}

	withheld := scenarioWithTags("an outage is detected", "@stale")
	withheld.Id = "pickle-withheld"
	if _, err := r.beforeScenario(context.Background(), withheld); err == nil {
		t.Fatal("the gate let a scenario needing an undeclared capability run")
	}

	unaskableReason := r.skipReason(unaskable.Id)
	withheldReason := r.skipReason(withheld.Id)

	if !strings.Contains(unaskableReason, property) {
		t.Errorf("the reported reason does not name the property of the SDK: %q", unaskableReason)
	}
	if strings.Contains(unaskableReason, "which this provider does not declare") {
		t.Errorf("the reported reason blames the provider for a property of the SDK: %q", unaskableReason)
	}
	if !strings.Contains(withheldReason, "which this provider does not declare") {
		t.Errorf("an ordinary withheld capability lost its reason: %q", withheldReason)
	}
	if strings.Contains(withheldReason, "cannot express") {
		t.Errorf("a capability the provider withheld was reported as inexpressible: %q", withheldReason)
	}
}
