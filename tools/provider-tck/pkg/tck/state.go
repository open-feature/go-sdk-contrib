package tck

import (
	"context"
	"errors"
	"fmt"
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
	value     any
	variant   string
	reason    openfeature.Reason
	errorCode openfeature.ErrorCode

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

// scenarioState is everything one scenario accumulates. A fresh instance is
// created per scenario and carried through the step definitions in the context.
type scenarioState struct {
	cfg *Config

	client *openfeature.Client

	flag       *flagUnderTest
	last       *evaluation
	remembered any
	hasMemory  bool

	recorders map[openfeature.EventType]*eventRecorder
}

func newScenarioState(cfg *Config) *scenarioState {
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
// is what makes the SDK shut this one down. See Config.domain.
func (s *scenarioState) teardown() {
	if s.client == nil {
		return
	}
	for _, r := range s.recorders {
		r.detach(s.client)
	}
	s.recorders = map[openfeature.EventType]*eventRecorder{}
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
	events    chan openfeature.EventDetails
	callback  openfeature.EventCallback

	// last is the most recently consumed event, which the payload assertions
	// inspect.
	last *openfeature.EventDetails
}

// newEventRecorder attaches a recorder for an event type to a client.
//
// The SDK replays a matching event on registration when the provider is already
// in the corresponding state, so a handler added after the provider became
// ready still observes its PROVIDER_READY. That is what lets the feature files
// register handlers after "Given a stable provider" without racing it.
func newEventRecorder(client *openfeature.Client, eventType openfeature.EventType) *eventRecorder {
	r := &eventRecorder{
		eventType: eventType,
		events:    make(chan openfeature.EventDetails, eventQueueDepth),
	}

	// The SDK calls this synchronously from AddHandler when it replays a state
	// event, so the send must not block.
	fn := func(details openfeature.EventDetails) {
		select {
		case r.events <- details:
		default:
		}
	}
	r.callback = &fn

	client.AddHandler(eventType, r.callback)
	return r
}

// await consumes the next event of this recorder's type, waiting up to timeout.
func (r *eventRecorder) await(ctx context.Context, timeout time.Duration) (openfeature.EventDetails, error) {
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case details := <-r.events:
		r.last = &details
		return details, nil
	case <-timer.C:
		return openfeature.EventDetails{}, fmt.Errorf(
			"timed out after %s waiting for a %s event. If the provider is simply slower than this "+
				"to notice, raise Config.EventTimeout rather than treating it as a failure",
			timeout, r.eventType)
	case <-ctx.Done():
		return openfeature.EventDetails{}, fmt.Errorf("cancelled while waiting for a %s event: %w", r.eventType, ctx.Err())
	}
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
		return nil, errors.New("provider-tck: no scenario state in context; this is a bug in the TCK itself")
	}
	return s, nil
}
