package tck

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/cucumber/godog"
	"github.com/cucumber/godog/formatters"
	messages "github.com/cucumber/messages/go/v21"
)

// WHY THIS FILE EXISTS
//
// The conformance report carries its results as Cucumber Messages -- the ndjson
// protocol at https://github.com/cucumber/messages -- rather than as a format
// this project defines. Messages already carries every fact the report needs
// per scenario: the outcome, the tags, the executed feature source, and the
// exact Scenario Outline row a result came from.
//
// godog 0.15.1 has no Messages formatter. It registers "cucumber" (the legacy
// relishapp JSON), "events", "junit", "pretty" and "progress", and none of them
// is Messages. Its JUnit output is also unusable here: a capability-gated skip
// comes out as skipped="0" on the suite, in a non-standard <error
// type="skipped"> element, with the step text where the skip reason should be.
//
// So this file is a Messages formatter, registered with godog through the
// public godog.Format plugin interface. It consumes the formatter event stream
// and emits messages.Envelope values as ndjson. Nothing here implements the
// Messages specification: the types come from github.com/cucumber/messages,
// which this module already depends on, so this is wiring.
//
// TWO THINGS THE FORMATTER INTERFACE DOES NOT GIVE US
//
// A scenario-finished event. The interface has TestRunStarted, Feature, Pickle,
// the per-step results and Summary, and nothing that fires when a scenario
// ends. So the stream is accumulated and written at Summary rather than
// streamed. That also lets the envelopes be written in the conventional order
// -- meta, then source/gherkinDocument/pickle, then the test-case results --
// instead of the order godog happens to deliver them in.
//
// A reason for a skip. Skipped(pickle, step, definition) carries no error, so
// the capability that gated a scenario cannot be recovered from the event
// stream. The reason therefore comes from the gate itself, through the
// skipReason lookup below, and lands in TestStepResult.Message. This is the one
// part of this file that is specific to this suite rather than general.
//
// WHAT GODOG'S EVENTS DO CARRY, AND WHY THAT IS ENOUGH
//
// Feature is called once per feature with the raw file bytes godog parsed, so
// Source carries exactly the text that was executed.
//
// Pickle is called for every scenario that entered the run, including one the
// capability gate stops: godog runs the Before hooks inside the first step, so
// the pickle has already been announced by the time the gate refuses it.
//
// A gated scenario's steps all arrive as Skipped. godog's suite returns the
// hook's error wrapping godog.ErrSkip, which makes the first step SKIPPED, and
// every later step is skipped because the scenario already carries an error.
// A consumer taking the most severe of a test case's step results therefore
// reads such a scenario as SKIPPED, never as passed -- which is what Appendix F
// requires and what godog's own summary does not do, since it counts
// capability-gated skips in its passed tally.
//
// That last point is worth being precise about, because godog's storage is
// wrong where its events are right. For the first step of a gated scenario
// godog inserts a FAILED step result into its internal storage and then also
// calls Skipped on the formatter. A formatter built on the events sees one
// SKIPPED; the built-in formatters, which read storage, see the FAILED as well,
// which is where the malformed JUnit output comes from.

// messagesFormatterName is the name this formatter registers with godog under.
//
// It is prefixed rather than called "message" so that adding a Messages
// formatter to godog upstream -- which is where this belongs -- cannot collide
// with it.
const messagesFormatterName = "openfeature-tck-messages"

const messagesModule = "github.com/cucumber/messages/go/v21"

// messagesProtocolVersionFallback is used when the build carries no module
// version, which happens for a test binary built from the module itself.
const messagesProtocolVersionFallback = "21.0.1"

// resultsFormatCucumberMessages is the results format the report envelope
// names. It is the schema's expected value.
const resultsFormatCucumberMessages = "cucumber-messages"

// statusSeverity orders step results so that the most severe of a scenario's
// steps is the scenario's outcome.
//
// This is Cucumber's own ordering, and it is the rule a consumer of the stream
// has to apply too, since testCaseFinished carries no status. It matters that
// SKIPPED outranks PASSED: a capability-gated scenario has a skipped first step
// and would otherwise read as passed, which is exactly what Appendix F forbids.
var statusSeverity = map[messages.TestStepResultStatus]int{
	messages.TestStepResultStatus_UNKNOWN:   0,
	messages.TestStepResultStatus_PASSED:    1,
	messages.TestStepResultStatus_SKIPPED:   2,
	messages.TestStepResultStatus_PENDING:   3,
	messages.TestStepResultStatus_UNDEFINED: 4,
	messages.TestStepResultStatus_AMBIGUOUS: 5,
	messages.TestStepResultStatus_FAILED:    6,
}

// messagesSink is where one run's stream goes, and how it recovers a skip
// reason.
//
// godog builds a formatter from a name and an io.Writer, and gives every
// formatter in a multi-format run the same writer -- stdout, here, since the
// suite also runs "pretty". Its alternative is the "name:path" syntax, which
// splits the format string on the first colon and on commas, and so cannot
// carry an arbitrary filesystem path on Windows. Hence a sink resolved by suite
// name: the run owns the buffer and writes the file itself, which also keeps
// the digest in the same place as the write.
type messagesSink struct {
	out io.Writer
	// skipReason returns the reason a scenario was skipped, or "" if it was
	// not, keyed by pickle id.
	skipReason func(pickleID string) string
	// executed reports one scenario's feature, name and derived outcome as the
	// stream is assembled.
	//
	// The formatter is where this can be observed at all. godog's hooks do not
	// see a scenario the run announced and then never executed — a subtest
	// filtered out by `go test -run` — but the formatter has already been told
	// about its pickle, so a test case with no step results is exactly what such
	// a scenario looks like here. See checkCanonicalCoverage.
	executed func(executedScenario)
}

var (
	messagesSinkMu     sync.Mutex
	messagesSinks      = map[string]*messagesSink{}
	messagesFormatOnce sync.Once
)

// registerMessagesFormatter makes the formatter available to godog.
//
// Registration is global and append-only in godog, so it happens exactly once
// per process however many suites run.
func registerMessagesFormatter() {
	messagesFormatOnce.Do(func() {
		godog.Format(messagesFormatterName,
			"Cucumber Messages (ndjson), as defined by github.com/cucumber/messages",
			func(suite string, out io.Writer) formatters.Formatter {
				sink := lookupMessagesSink(suite)
				if sink == nil {
					// Not a run of this suite. Writing to the writer godog
					// supplied is the right behaviour for a formatter used by
					// name from the command line.
					sink = &messagesSink{out: out}
				}
				return newMessagesFormatter(sink)
			})
	})
}

// installMessagesSink claims the sink for a suite name, reporting whether it
// was free.
//
// A name already in use means two suites with the same name are running at
// once, and their streams would be interleaved into one buffer. That is a
// configuration error worth naming rather than a report to emit quietly.
func installMessagesSink(suite string, sink *messagesSink) bool {
	messagesSinkMu.Lock()
	defer messagesSinkMu.Unlock()
	if _, taken := messagesSinks[suite]; taken {
		return false
	}
	messagesSinks[suite] = sink
	return true
}

func removeMessagesSink(suite string) {
	messagesSinkMu.Lock()
	defer messagesSinkMu.Unlock()
	delete(messagesSinks, suite)
}

func lookupMessagesSink(suite string) *messagesSink {
	messagesSinkMu.Lock()
	defer messagesSinkMu.Unlock()
	return messagesSinks[suite]
}

// messagesFormatter turns godog's formatter events into a Cucumber Messages
// stream.
type messagesFormatter struct {
	sink *messagesSink

	mu sync.Mutex

	nextID       int
	runStartedAt time.Time

	// static holds the meta-independent messages -- source, gherkinDocument and
	// pickle -- in the order godog delivered them, which groups each feature's
	// pickles after that feature's source and document.
	static      []*messages.Envelope
	seenFeature map[string]bool
	seenPickle  map[string]bool

	// cases is keyed by pickle id; order preserves execution order.
	cases map[string]*messagesTestCase
	order []string
}

// messagesTestCase is one scenario's execution.
type messagesTestCase struct {
	pickle       *messages.Pickle
	id           string
	startedID    string
	startedAt    time.Time
	finishedAt   time.Time
	steps        []*messagesTestStep
	byPickleStep map[string]*messagesTestStep
}

type messagesTestStep struct {
	id           string
	pickleStepID string
	startedAt    time.Time
	finishedAt   time.Time
	status       messages.TestStepResultStatus
	message      string
	reported     bool
}

func newMessagesFormatter(sink *messagesSink) *messagesFormatter {
	return &messagesFormatter{
		sink:        sink,
		seenFeature: map[string]bool{},
		seenPickle:  map[string]bool{},
		cases:       map[string]*messagesTestCase{},
	}
}

// id issues an identifier for a message this formatter invents.
//
// The prefix keeps these ids out of the space godog's parser numbers from,
// which starts at "0" and covers every AST node, pickle and pickle step. A
// consumer that mixed up a testCaseId and a pickleId would find a plausible
// match rather than nothing, so the two are made impossible to confuse.
func (f *messagesFormatter) id(prefix string) string {
	f.nextID++
	return fmt.Sprintf("%s-%d", prefix, f.nextID)
}

func (f *messagesFormatter) TestRunStarted() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.runStartedAt = time.Now()
}

// Feature records the executed source and its parsed document.
//
// The content is the bytes godog parsed, captured by its parser as it read the
// file, so Source is the text that produced these results and not a re-read of
// a file that may have changed.
func (f *messagesFormatter) Feature(document *messages.GherkinDocument, uri string, content []byte) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.seenFeature[uri] {
		return
	}
	f.seenFeature[uri] = true

	f.static = append(f.static, &messages.Envelope{
		Source: &messages.Source{
			Uri:       uri,
			Data:      string(content),
			MediaType: messages.SourceMediaType_TEXT_X_CUCUMBER_GHERKIN_PLAIN,
		},
	})

	if document == nil {
		return
	}
	// Comments has no omitempty and the schema requires an array, so a document
	// parsed without comments would otherwise emit null. Copied rather than
	// fixed in place: the document is shared with godog's other formatters.
	doc := *document
	if doc.Comments == nil {
		doc.Comments = []*messages.Comment{}
	}
	f.static = append(f.static, &messages.Envelope{GherkinDocument: &doc})
}

// Pickle opens a test case for a scenario.
//
// Called for every scenario that entered the run, including one the capability
// gate goes on to refuse, because godog announces the pickle before running the
// Before hooks.
func (f *messagesFormatter) Pickle(pickle *messages.Pickle) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if pickle == nil || f.seenPickle[pickle.Id] {
		return
	}
	f.seenPickle[pickle.Id] = true

	f.static = append(f.static, &messages.Envelope{Pickle: pickle})

	now := time.Now()
	tc := &messagesTestCase{
		pickle:       pickle,
		id:           f.id("test-case"),
		startedID:    f.id("test-case-started"),
		startedAt:    now,
		finishedAt:   now,
		byPickleStep: map[string]*messagesTestStep{},
	}
	for _, step := range pickle.Steps {
		ts := &messagesTestStep{
			id:           f.id("test-step"),
			pickleStepID: step.Id,
			startedAt:    now,
			finishedAt:   now,
			// A step godog never reports on stays UNKNOWN rather than being
			// invented as passed.
			status: messages.TestStepResultStatus_UNKNOWN,
		}
		tc.steps = append(tc.steps, ts)
		tc.byPickleStep[step.Id] = ts
	}

	f.cases[pickle.Id] = tc
	f.order = append(f.order, pickle.Id)
}

// Defined fires once per step, immediately before it runs, for skipped steps as
// well as executed ones. It is the only step-started signal the interface has.
func (f *messagesFormatter) Defined(pickle *messages.Pickle, step *messages.PickleStep, _ *formatters.StepDefinition) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if ts := f.step(pickle, step); ts != nil && !ts.reported {
		ts.startedAt = time.Now()
	}
}

func (f *messagesFormatter) Passed(pickle *messages.Pickle, step *messages.PickleStep, _ *formatters.StepDefinition) {
	f.finish(pickle, step, messages.TestStepResultStatus_PASSED, "")
}

func (f *messagesFormatter) Failed(pickle *messages.Pickle, step *messages.PickleStep, _ *formatters.StepDefinition, err error) {
	message := ""
	if err != nil {
		message = err.Error()
	}
	f.finish(pickle, step, messages.TestStepResultStatus_FAILED, message)
}

// Skipped covers both a step after an earlier failure and every step of a
// scenario the capability gate refused.
//
// The event carries no reason, so the gate's own is looked up. Without it the
// stream would say a scenario did not run but not why, and the why is the whole
// point of reporting a capability skip.
func (f *messagesFormatter) Skipped(pickle *messages.Pickle, step *messages.PickleStep, _ *formatters.StepDefinition) {
	reason := ""
	if pickle != nil && f.sink != nil && f.sink.skipReason != nil {
		reason = f.sink.skipReason(pickle.Id)
	}
	f.finish(pickle, step, messages.TestStepResultStatus_SKIPPED, reason)
}

func (f *messagesFormatter) Undefined(pickle *messages.Pickle, step *messages.PickleStep, _ *formatters.StepDefinition) {
	f.finish(pickle, step, messages.TestStepResultStatus_UNDEFINED,
		"no step definition matches this step")
}

func (f *messagesFormatter) Pending(pickle *messages.Pickle, step *messages.PickleStep, _ *formatters.StepDefinition) {
	f.finish(pickle, step, messages.TestStepResultStatus_PENDING, "")
}

func (f *messagesFormatter) Ambiguous(pickle *messages.Pickle, step *messages.PickleStep, _ *formatters.StepDefinition, err error) {
	message := ""
	if err != nil {
		message = err.Error()
	}
	f.finish(pickle, step, messages.TestStepResultStatus_AMBIGUOUS, message)
}

func (f *messagesFormatter) finish(
	pickle *messages.Pickle,
	step *messages.PickleStep,
	status messages.TestStepResultStatus,
	message string,
) {
	f.mu.Lock()
	defer f.mu.Unlock()

	ts := f.step(pickle, step)
	if ts == nil || ts.reported {
		return
	}

	now := time.Now()
	ts.status = status
	ts.message = message
	ts.finishedAt = now
	ts.reported = true

	if tc := f.cases[pickle.Id]; tc != nil {
		tc.finishedAt = now
	}
}

// step resolves the accumulator for one step of one scenario. Callers hold the
// lock.
func (f *messagesFormatter) step(pickle *messages.Pickle, step *messages.PickleStep) *messagesTestStep {
	if pickle == nil || step == nil {
		return nil
	}
	tc := f.cases[pickle.Id]
	if tc == nil {
		return nil
	}
	return tc.byPickleStep[step.Id]
}

// Summary writes the stream.
//
// It is the last event godog delivers, and the interface has nothing that fires
// when a scenario ends, so this is where the accumulated run becomes ndjson.
func (f *messagesFormatter) Summary() {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.sink == nil || f.sink.out == nil {
		return
	}

	envelopes := make([]*messages.Envelope, 0, len(f.static)+4*len(f.order)+3)

	envelopes = append(envelopes, &messages.Envelope{Meta: messagesMeta()})
	envelopes = append(envelopes, f.static...)

	started := f.runStartedAt
	if started.IsZero() {
		started = time.Now()
	}
	envelopes = append(envelopes, &messages.Envelope{
		TestRunStarted: &messages.TestRunStarted{Timestamp: timestamp(started)},
	})

	success := true
	finishedAt := started

	for _, pickleID := range f.order {
		tc := f.cases[pickleID]
		if tc == nil {
			continue
		}

		testSteps := make([]*messages.TestStep, 0, len(tc.steps))
		for _, ts := range tc.steps {
			testSteps = append(testSteps, &messages.TestStep{
				Id:           ts.id,
				PickleStepId: ts.pickleStepID,
			})
		}

		envelopes = append(envelopes,
			&messages.Envelope{TestCase: &messages.TestCase{
				Id:        tc.id,
				PickleId:  pickleID,
				TestSteps: testSteps,
			}},
			&messages.Envelope{TestCaseStarted: &messages.TestCaseStarted{
				Id:         tc.startedID,
				TestCaseId: tc.id,
				Attempt:    0,
				Timestamp:  timestamp(tc.startedAt),
			}},
		)

		// A test case's outcome is the most severe of its step results, which is
		// Cucumber's own rule; Messages has no per-test-case status field.
		outcome := messages.TestStepResultStatus_UNKNOWN

		for _, ts := range tc.steps {
			if statusSeverity[ts.status] > statusSeverity[outcome] {
				outcome = ts.status
			}
			envelopes = append(envelopes,
				&messages.Envelope{TestStepStarted: &messages.TestStepStarted{
					TestCaseStartedId: tc.startedID,
					TestStepId:        ts.id,
					Timestamp:         timestamp(ts.startedAt),
				}},
				&messages.Envelope{TestStepFinished: &messages.TestStepFinished{
					TestCaseStartedId: tc.startedID,
					TestStepId:        ts.id,
					TestStepResult: &messages.TestStepResult{
						Status:   ts.status,
						Message:  ts.message,
						Duration: duration(ts.finishedAt.Sub(ts.startedAt)),
					},
					Timestamp: timestamp(ts.finishedAt),
				}},
			)
			if ts.status == messages.TestStepResultStatus_FAILED ||
				ts.status == messages.TestStepResultStatus_AMBIGUOUS ||
				ts.status == messages.TestStepResultStatus_UNDEFINED {
				success = false
			}
		}

		envelopes = append(envelopes, &messages.Envelope{
			TestCaseFinished: &messages.TestCaseFinished{
				TestCaseStartedId: tc.startedID,
				Timestamp:         timestamp(tc.finishedAt),
				WillBeRetried:     false,
			},
		})

		if f.sink.executed != nil && tc.pickle != nil {
			f.sink.executed(executedScenario{
				uri:    tc.pickle.Uri,
				name:   tc.pickle.Name,
				status: outcome,
			})
		}

		if tc.finishedAt.After(finishedAt) {
			finishedAt = tc.finishedAt
		}
	}

	envelopes = append(envelopes, &messages.Envelope{
		TestRunFinished: &messages.TestRunFinished{
			Success:   success,
			Timestamp: timestamp(finishedAt),
		},
	})

	encoder := json.NewEncoder(f.sink.out)
	for _, envelope := range envelopes {
		// json.Encoder writes one compact object per line, which is exactly the
		// ndjson framing the protocol asks for.
		if err := encoder.Encode(envelope); err != nil {
			// The sink is an in-memory buffer, so this cannot fail in practice;
			// swallowing it silently would still be the wrong shape of code.
			fmt.Fprintf(f.sink.out, "\n")
			return
		}
	}
}

// messagesMeta describes what produced the stream.
//
// Messages records the runner, the runtime and the machine. It has no slot for
// the subject under test, which is why the report envelope still has to name
// the provider itself.
func messagesMeta() *messages.Meta {
	return &messages.Meta{
		ProtocolVersion: messagesProtocolVersion(),
		Implementation: &messages.Product{
			Name:    tckImplementation,
			Version: tckVersion(),
		},
		Runtime: &messages.Product{Name: "go", Version: runtime.Version()},
		Os:      &messages.Product{Name: runtime.GOOS},
		Cpu:     &messages.Product{Name: runtime.GOARCH},
	}
}

// messagesProtocolVersion is the version of the Messages protocol the stream
// conforms to, which is the version of the module whose types produced it.
func messagesProtocolVersion() string {
	version := strings.TrimPrefix(moduleVersion(messagesModule), "v")
	if version == "" || version == "unknown" {
		return messagesProtocolVersionFallback
	}
	return version
}

// lockedBuffer is the sink a run collects its stream into.
//
// The formatter writes it while godog runs and the report reads it afterwards,
// which is already ordered by Run returning. The lock says so rather than
// leaving a reader to work it out from godog's concurrency settings.
type lockedBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *lockedBuffer) bytes() []byte {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]byte, b.buf.Len())
	copy(out, b.buf.Bytes())
	return out
}

func timestamp(t time.Time) *messages.Timestamp {
	ts := messages.GoTimeToTimestamp(t)
	return &ts
}

func duration(d time.Duration) *messages.Duration {
	if d < 0 {
		d = 0
	}
	converted := messages.GoDurationToDuration(d)
	return &converted
}
