package tck

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/open-feature/go-sdk/openfeature"
)

// shortEventTimeout keeps the awaits in these tests from costing the default
// twelve seconds each. Every case here either has its record waiting already or
// is about the absence of one.
const shortEventTimeout = 150 * time.Millisecond

// eventState puts a scenario state in a context with recorders for the named
// event kinds, attached to no client: a client with a ready provider behind it
// has an event replayed into it the moment a handler goes on, and a test about
// event order cannot have a record it did not put there.
func eventState(t *testing.T, kinds ...string) (context.Context, map[string]*eventRecorder) {
	t.Helper()

	ctx, state := stepState(t, WithEventTimeout(shortEventTimeout))

	recorders := map[string]*eventRecorder{}
	for _, kind := range kinds {
		eventType, ok := eventTypeByName[kind]
		if !ok {
			t.Fatalf("no such event kind %q", kind)
		}
		recorder := newEventRecorder(eventType, &state.clock)
		state.recorders[eventType] = recorder
		recorders[kind] = recorder
	}
	return ctx, recorders
}

func TestAForbiddenEventRecordedFirstFailsTheOrdering(t *testing.T) {
	ctx, recorders := eventState(t, "ready", "error")
	recorders["error"].record(openfeature.EventDetails{})
	recorders["ready"].record(openfeature.EventDetails{})

	err := theEventHandlerShouldHaveBeenExecutedWithNoneBefore(ctx, "ready", "error")
	if err == nil {
		t.Fatal("an error recorded before the ready event satisfied the ordering")
	}
	if !strings.Contains(err.Error(), "error event was recorded before the ready event") {
		t.Errorf("the failure does not say which order was seen: %v", err)
	}
}

func TestTheSameEventsInTheOtherOrderPassAndLeaveTheForbiddenRecordBehind(t *testing.T) {
	ctx, recorders := eventState(t, "ready", "error")
	recorders["ready"].record(openfeature.EventDetails{})
	recorders["error"].record(openfeature.EventDetails{})

	if err := theEventHandlerShouldHaveBeenExecutedWithNoneBefore(ctx, "ready", "error"); err != nil {
		t.Fatalf("an error recorded after the ready event failed the ordering: %v", err)
	}

	// The ready was claimed and the error was not: an assertion that comes
	// next is entitled to the record this one had no quarrel with.
	if err := theEventHandlerShouldHaveBeenExecuted(ctx, "error"); err != nil {
		t.Errorf("the error record was consumed by an assertion about the ready event: %v", err)
	}
}

func TestAnOrderingQuestionWithNothingRecordedTimesOutRatherThanPassing(t *testing.T) {
	ctx, _ := eventState(t, "ready", "error")

	err := theEventHandlerShouldHaveBeenExecutedWithNoneBefore(ctx, "ready", "error")
	if err == nil {
		t.Fatal("the ordering passed with no ready event recorded at all")
	}
	// It awaits the event it is about; absence of that event is a failure and
	// not a silent pass, which is the trap an absence step falls into.
	if !strings.Contains(err.Error(), "waiting for a PROVIDER_READY event") {
		t.Errorf("the failure does not say which event never arrived: %v", err)
	}
}

func TestAwaitingOneKindOfEventDoesNotThrowAwayAnotherKindsRecord(t *testing.T) {
	ctx, recorders := eventState(t, "ready", "error", "change")
	recorders["error"].record(openfeature.EventDetails{})
	recorders["change"].record(openfeature.EventDetails{})
	recorders["ready"].record(openfeature.EventDetails{})

	// The positive assertion in between is what Java's TCK had to be fixed
	// for: an await that drained the records through its match would discard
	// the error, and the ordering question below would have nothing left to
	// object to.
	if err := theEventHandlerShouldHaveBeenExecuted(ctx, "change"); err != nil {
		t.Fatalf("the change event handler should have been executed: %v", err)
	}

	if err := theEventHandlerShouldHaveBeenExecutedWithNoneBefore(ctx, "ready", "error"); err == nil {
		t.Fatal("claiming the change event made the earlier error event unobservable")
	}
}

func TestAnAwaitClaimsItsOwnRecordAndNoMore(t *testing.T) {
	ctx, recorders := eventState(t, "ready")
	recorders["ready"].record(openfeature.EventDetails{})

	if err := theEventHandlerShouldHaveBeenExecuted(ctx, "ready"); err != nil {
		t.Fatalf("the ready event handler should have been executed: %v", err)
	}

	// One record cannot satisfy two awaits: the stale chain asserts a ready at
	// the start and a second, different ready once the backend is back.
	if err := theEventHandlerShouldHaveBeenExecutedWithin(ctx, "ready", 50); err == nil {
		t.Error("a second await was satisfied by the record the first one claimed")
	}
}

func TestTheOrderingStepRequiresAHandlerForTheForbiddenKind(t *testing.T) {
	ctx, recorders := eventState(t, "ready")
	recorders["ready"].record(openfeature.EventDetails{})

	// Treating a missing handler as "none were seen" would make the assertion
	// hold for every provider, including one that fires the forbidden event
	// first every time.
	err := theEventHandlerShouldHaveBeenExecutedWithNoneBefore(ctx, "ready", "error")
	if err == nil {
		t.Fatal("the ordering passed with nothing recording the forbidden event")
	}
	if !strings.Contains(err.Error(), "no handler was registered for PROVIDER_ERROR") {
		t.Errorf("the failure does not name the missing handler: %v", err)
	}
}

func TestAnEventCannotBeForbiddenBeforeItself(t *testing.T) {
	ctx, recorders := eventState(t, "ready")
	recorders["ready"].record(openfeature.EventDetails{})

	err := theEventHandlerShouldHaveBeenExecutedWithNoneBefore(ctx, "ready", "ready")
	if err == nil {
		t.Fatal("a ready event was accepted as forbidden before itself")
	}
	if !strings.Contains(err.Error(), "cannot be forbidden before itself") {
		t.Errorf("the failure does not say why the step is unanswerable: %v", err)
	}
}
