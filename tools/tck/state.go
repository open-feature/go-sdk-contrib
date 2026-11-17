package tck

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/open-feature/go-sdk/openfeature"
)

// eventQueueDepth bounds how many unconsumed events a recorder holds. A
// scenario asserts a handful at most; the buffer exists so that the SDK's event
// goroutine is never blocked by a test that stopped reading.
const eventQueueDepth = 32

// flagUnderTest is the flag a scenario declared with the
// "Given a <type>-flag with key ... and a default value ..." step.
type flagUnderTest struct {
	key          string
	typ          flagType
	defaultValue any
}

// evaluation is the outcome of one flag evaluation, flattened across the five
// typed client methods so the assertion steps do not care which was used.
type evaluation struct {
	value        any
	variant      string
	reason       openfeature.Reason
	errorCode    openfeature.ErrorCode
	errorMessage string

	// metadata is the flag metadata the resolution carried. Kept here rather
	// than read back from the client, because only the details call exposes it.
	metadata openfeature.FlagMetadata

	// err is what the client returned. In Go an errored evaluation returns both
	// the code default and a non-nil error, so this being set is normal and is
	// not what "no exception should have been thrown" asserts — see panicked.
	err error

	// panicked records that the evaluation panicked, which is the Go analogue
	// of the thrown exception the feature files forbid. A panic out of a flag
	// evaluation takes the host application down with it.
	panicked   bool
	panicValue any
}

// The two openfeature.StateHandler methods the lifecycle steps call directly.
const (
	lifecycleShutdown = "Shutdown"
	lifecycleInit     = "Init"
)

// lifecycleCall is the outcome of one direct call into the provider's
// openfeature.StateHandler — a Shutdown or an Init — made by a lifecycle step.
//
// The steps call the provider's own methods rather than going through the SDK.
// Replacing the provider in the SDK would test the SDK's bookkeeping as much as
// the provider, which Appendix B already does; and the SDK compares providers
// with reflect.DeepEqual unless they are pointers, so re-registering the same
// one may be judged no change at all and initialise nothing.
type lifecycleCall struct {
	operation string
	duration  time.Duration

	// err is what Init returned. Shutdown returns nothing, so for it this is
	// always nil. Unlike an evaluation's error this one does count as the
	// thrown exception the feature files forbid: a provider that cannot be
	// initialised again after a shutdown has not reverted to its uninitialised
	// state.
	err error

	// panicked records that the call panicked, which is the Go analogue of the
	// thrown exception — see evaluation.panicked.
	panicked   bool
	panicValue any
}

// scenarioState is everything one scenario accumulates. A fresh instance is
// created per scenario and carried through the step definitions in the context.
type scenarioState struct {
	cfg *config

	// provider is the provider under test, held so that the lifecycle and
	// metadata steps can address it directly. Evaluations go through client.
	provider openfeature.FeatureProvider
	client   *openfeature.Client

	flag *flagUnderTest

	// evalContext is the evaluation context the next evaluation is made with,
	// built by "a context containing a targeting key" and by "a context with
	// the following keys". Its zero value is the no-context case, which every
	// scenario without either step supplies.
	evalContext openfeature.EvaluationContext

	last       *evaluation
	remembered any
	hasMemory  bool

	// lifecycle is every direct Shutdown and Init the scenario made, in order.
	// "no exception should have been thrown" inspects all of them.
	lifecycle []lifecycleCall

	recorders map[openfeature.EventType]*eventRecorder

	// clock stamps the events of every recorder from one counter, which is what
	// lets an assertion about one kind of event be answered against another
	// kind's records. See eventClock.
	clock eventClock
}

func newScenarioState(cfg *config) *scenarioState {
	return &scenarioState{
		cfg:       cfg,
		recorders: map[openfeature.EventType]*eventRecorder{},
	}
}

// requireFlag returns the flag under test, or an error naming the missing step.
func (s *scenarioState) requireFlag() (*flagUnderTest, error) {
	if s.flag == nil {
		return nil, errors.New("no flag has been declared in this scenario: " +
			"a \"Given a <type>-flag with key ... and a default value ...\" step must come first")
	}
	return s.flag, nil
}

// requireEvaluation returns the most recent evaluation, or an error naming the
// missing step.
func (s *scenarioState) requireEvaluation() (*evaluation, error) {
	if s.last == nil {
		return nil, errors.New("no flag has been evaluated in this scenario: " +
			"a \"When the flag was evaluated with details\" step must come first")
	}
	return s.last, nil
}

// requireClient returns the client for the provider under test, or an error
// naming the missing step.
func (s *scenarioState) requireClient() (*openfeature.Client, error) {
	if s.client == nil {
		return nil, errors.New("no provider has been registered in this scenario: " +
			"a \"Given a stable provider\" or \"Given a unavailable provider\" step must come first")
	}
	return s.client, nil
}

// requireProvider returns the provider under test itself, or an error naming
// the missing step.
func (s *scenarioState) requireProvider() (openfeature.FeatureProvider, error) {
	if s.provider == nil {
		return nil, errors.New("no provider has been registered in this scenario: " +
			"a \"Given a stable provider\" or \"Given a unavailable provider\" step must come first")
	}
	return s.provider, nil
}

// lastShutdown returns the most recent direct Shutdown call, or an error naming
// the missing step.
func (s *scenarioState) lastShutdown() (*lifecycleCall, error) {
	for i := len(s.lifecycle) - 1; i >= 0; i-- {
		if s.lifecycle[i].operation == lifecycleShutdown {
			return &s.lifecycle[i], nil
		}
	}
	return nil, errors.New("the provider has not been shut down in this scenario: " +
		"a \"When the provider is shut down\" step must come first")
}

// recorder returns the recorder for an event type, or an error naming the
// missing step.
func (s *scenarioState) recorder(eventType openfeature.EventType) (*eventRecorder, error) {
	r, ok := s.recorders[eventType]
	if !ok {
		return nil, fmt.Errorf("no handler was registered for %s in this scenario: "+
			"a \"Given a <kind> event handler\" step must come first", eventType)
	}
	return r, nil
}

// teardown detaches every handler this scenario registered.
//
// The provider itself is left registered: the next scenario replaces it, which
// is what makes the SDK shut this one down. See config.domain. That holds for a
// provider a lifecycle step already shut down directly, whose Shutdown on
// replacement is then a second call, and for one that was shut down and
// initialised again.
func (s *scenarioState) teardown() {
	if s.client == nil {
		return
	}
	for _, r := range s.recorders {
		r.detach(s.client)
	}
	s.recorders = map[openfeature.EventType]*eventRecorder{}
}

// eventClock numbers the events of every recorder in one scenario from a single
// counter.
//
// The recorders hold a channel per event kind, so an await already claims only
// its own record and leaves another kind's alone — an error recorded before an
// awaited ready event is still there afterwards for whatever asks next. What
// per-kind channels cannot answer on their own is which of two kinds arrived
// first, and that is the whole of "with no error event before it". The order
// therefore comes from a counter the recorders share rather than from the
// channels, and no clock time is involved in reading it back.
type eventClock struct {
	next atomic.Uint64
}

// tick returns the sequence number for the event being recorded now.
func (c *eventClock) tick() uint64 {
	return c.next.Add(1)
}

// recordedEvent is one delivered event together with its place in the
// scenario's event order.
type recordedEvent struct {
	details openfeature.EventDetails
	seq     uint64
}

// eventRecorder captures the events of one type delivered to the client, in
// order, so that a scenario can consume them one at a time.
//
// Consuming rather than merely observing is what makes the stale scenario work:
// it awaits a PROVIDER_READY at the start and a second, different
// PROVIDER_READY after the backend comes back, and a recorder that only
// remembered "ready has fired at some point" would report the second assertion
// as satisfied by the first event.
type eventRecorder struct {
	eventType openfeature.EventType
	events    chan recordedEvent
	callback  openfeature.EventCallback
	clock     *eventClock

	// last is the most recently consumed event, which the payload assertions
	// inspect.
	last *openfeature.EventDetails

	mu sync.Mutex
	// observed holds the sequence number of every event of this kind that was
	// delivered, consumed or not. Unlike events it is never claimed: an
	// ordering assertion about a different kind is entitled to see a record
	// this recorder has already handed over, and a record the queue had no room
	// for still happened.
	observed []uint64
}

// newEventRecorder returns a recorder for one event type, not yet attached to a
// client.
//
// Building and attaching are separate, the way attach and detach are, so that
// what a recorder does with the events it is given can be exercised without the
// global SDK state a client carries — a client attached to a provider that is
// already ready has an event replayed into it the moment a handler goes on, and
// a test about event order cannot have a record it did not put there.
func newEventRecorder(eventType openfeature.EventType, clock *eventClock) *eventRecorder {
	return &eventRecorder{
		eventType: eventType,
		events:    make(chan recordedEvent, eventQueueDepth),
		clock:     clock,
	}
}

// attach registers this recorder's handler with a client.
//
// The SDK replays a matching event on registration when the provider is already
// in the corresponding state, so a handler added after the provider became
// ready still observes its PROVIDER_READY. That is what lets the feature files
// register handlers after "Given a stable provider" without racing it.
func (r *eventRecorder) attach(client *openfeature.Client) {
	fn := func(details openfeature.EventDetails) { r.record(details) }
	r.callback = &fn

	client.AddHandler(r.eventType, r.callback)
}

// record stamps one delivered event with its place in the scenario's event
// order and queues it.
//
// The SDK calls this synchronously from AddHandler when it replays a state
// event, so it must never block. A full queue therefore drops the event from
// the channel while observed still remembers that it happened, which is the
// half an ordering assertion reads.
func (r *eventRecorder) record(details openfeature.EventDetails) {
	event := recordedEvent{details: details, seq: r.clock.tick()}

	r.mu.Lock()
	r.observed = append(r.observed, event.seq)
	r.mu.Unlock()

	select {
	case r.events <- event:
	default:
	}
}

// await consumes the next event of this recorder's type, waiting up to timeout.
func (r *eventRecorder) await(ctx context.Context, timeout time.Duration) (recordedEvent, error) {
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case event := <-r.events:
		r.last = &event.details
		return event, nil
	case <-timer.C:
		return recordedEvent{}, fmt.Errorf(
			"timed out after %s waiting for a %s event. If the provider is simply slower than this "+
				"to notice, raise tck.WithEventTimeout rather than treating it as a failure",
			timeout, r.eventType)
	case <-ctx.Done():
		return recordedEvent{}, fmt.Errorf("cancelled while waiting for a %s event: %w", r.eventType, ctx.Err())
	}
}

// observedBefore reports whether an event of this recorder's kind was recorded
// before the given point in the scenario's event order.
func (r *eventRecorder) observedBefore(seq uint64) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, observed := range r.observed {
		if observed < seq {
			return true
		}
	}
	return false
}

// detach removes the handler from the client.
func (r *eventRecorder) detach(client *openfeature.Client) {
	client.RemoveHandler(r.eventType, r.callback)
}

// stateKey is the context key under which the scenario state travels between
// step definitions. An unexported struct type cannot collide with a key from
// another package.
type stateKey struct{}

func withState(ctx context.Context, s *scenarioState) context.Context {
	return context.WithValue(ctx, stateKey{}, s)
}

// stateFrom retrieves the scenario state. A missing state means the Before hook
// did not run, which is a bug in this package rather than in a step.
func stateFrom(ctx context.Context) (*scenarioState, error) {
	s, ok := ctx.Value(stateKey{}).(*scenarioState)
	if !ok || s == nil {
		return nil, errors.New("tck: no scenario state in context; this is a bug in the TCK itself")
	}
	return s, nil
}
