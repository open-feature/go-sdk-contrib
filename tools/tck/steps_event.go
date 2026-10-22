package tck

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/cucumber/godog"
	"github.com/open-feature/go-sdk/openfeature"
)

// registerEventSteps binds the steps covering provider events, connection loss
// and client state.
func registerEventSteps(ctx *godog.ScenarioContext) {
	ctx.Step(`^an? (ready|stale|error|change) event handler$`, aEventHandler)
	ctx.Step(`^a (ready|stale|error|change) event was fired$`, anEventWasFired)
	ctx.Step(`^the (ready|stale|error|change) event handler should have been executed$`, theEventHandlerShouldHaveBeenExecuted)
	ctx.Step(`^the (ready|stale|error|change) event handler should have been executed within (\d+)ms$`, theEventHandlerShouldHaveBeenExecutedWithin)
	ctx.Step(`^the (ready|stale|error|change) event handler should have been executed with no (ready|stale|error|change) event before it$`,
		theEventHandlerShouldHaveBeenExecutedWithNoneBefore)
	ctx.Step(`^the flag should be part of the event payload$`, theFlagShouldBePartOfTheEventPayload)
	ctx.Step(`^the connection is lost$`, theConnectionIsLost)
	ctx.Step(`^the connection is restored$`, theConnectionIsRestored)
	ctx.Step(`^the client should be in (ready|stale|error) state$`, theClientShouldBeInState)
}

// eventTypeByName maps the words the feature files use onto SDK event types.
var eventTypeByName = map[string]openfeature.EventType{
	"ready":  openfeature.ProviderReady,
	"stale":  openfeature.ProviderStale,
	"error":  openfeature.ProviderError,
	"change": openfeature.ProviderConfigChange,
}

// stateByName maps the words the feature files use onto SDK provider states.
var stateByName = map[string]openfeature.State{
	"ready": openfeature.ReadyState,
	"stale": openfeature.StaleState,
	"error": openfeature.ErrorState,
}

// aEventHandler attaches a recorder for one event type.
//
// Handlers are attached after the provider is registered, which the SDK handles
// by replaying a matching event on registration when the provider is already in
// the corresponding state. That is why "Given a stable provider" followed by
// "And a ready event handler" is not a race.
func aEventHandler(ctx context.Context, name string) error {
	state, err := stateFrom(ctx)
	if err != nil {
		return err
	}
	client, err := state.requireClient()
	if err != nil {
		return err
	}

	eventType, ok := eventTypeByName[name]
	if !ok {
		return fmt.Errorf("unknown event kind %q", name)
	}
	if _, exists := state.recorders[eventType]; exists {
		return nil
	}

	recorder := newEventRecorder(eventType, &state.clock)
	recorder.attach(client)
	state.recorders[eventType] = recorder
	return nil
}

// anEventWasFired consumes an event, so that a later assertion in the same
// scenario observes the next one rather than this one.
//
// The stale scenario depends on it: it consumes the initial PROVIDER_READY here
// and then asserts a second, distinct PROVIDER_READY once the backend is back.
func anEventWasFired(ctx context.Context, name string) error {
	return awaitEvent(ctx, name, 0)
}

func theEventHandlerShouldHaveBeenExecuted(ctx context.Context, name string) error {
	return awaitEvent(ctx, name, 0)
}

// theEventHandlerShouldHaveBeenExecutedWithin bounds the wait explicitly.
//
// The scenarios that use it are asserting promptness rather than eventual
// arrival, so this bound overrides tck.WithEventTimeout rather than being
// clamped by it.
func theEventHandlerShouldHaveBeenExecutedWithin(ctx context.Context, name string, millis int) error {
	return awaitEvent(ctx, name, time.Duration(millis)*time.Millisecond)
}

// theEventHandlerShouldHaveBeenExecutedWithNoneBefore awaits an event of one
// kind and asserts that no event of another kind was recorded before it.
//
// This is how a scenario says "and no error": as an order, not as a window. "No
// error event fired" can only be answered over some stretch of time, and at any
// one instant an event that never fires and one that fires a moment later are
// the same observation — so a step asking it that way either slows every run
// down or passes because nothing has arrived yet. "No error before the provider
// came back ready" is a question about what was recorded, and the recording
// answers it with no waiting at all.
func theEventHandlerShouldHaveBeenExecutedWithNoneBefore(ctx context.Context, name, forbidden string) error {
	state, err := stateFrom(ctx)
	if err != nil {
		return err
	}

	eventType, ok := eventTypeByName[name]
	if !ok {
		return fmt.Errorf("unknown event kind %q", name)
	}
	forbiddenType, ok := eventTypeByName[forbidden]
	if !ok {
		return fmt.Errorf("unknown event kind %q", forbidden)
	}
	if eventType == forbiddenType {
		return fmt.Errorf("a %s event cannot be forbidden before itself: this step asks for an "+
			"order between two kinds of event", name)
	}

	recorder, err := state.recorder(eventType)
	if err != nil {
		return err
	}
	// The forbidden kind needs a handler too, and a missing one is an error
	// rather than "none were seen": with nothing recording them the assertion
	// would hold for every provider, including one that fires the forbidden
	// event first every time.
	forbiddenRecorder, err := state.recorder(forbiddenType)
	if err != nil {
		return err
	}

	event, err := recorder.await(ctx, state.cfg.eventTimeout())
	if err != nil {
		return err
	}

	if forbiddenRecorder.observedBefore(event.seq) {
		return fmt.Errorf("a %s event was recorded before the %s event. The scenario asks for the "+
			"%s event to be reached without one, so an intervening %s event means the provider "+
			"passed through a state it forbids", forbidden, name, name, forbidden)
	}
	return nil
}

// awaitEvent consumes the next event of the named type. A timeout of zero means
// tck.WithEventTimeout.
func awaitEvent(ctx context.Context, name string, timeout time.Duration) error {
	state, err := stateFrom(ctx)
	if err != nil {
		return err
	}

	eventType, ok := eventTypeByName[name]
	if !ok {
		return fmt.Errorf("unknown event kind %q", name)
	}

	recorder, err := state.recorder(eventType)
	if err != nil {
		return err
	}

	if timeout <= 0 {
		timeout = state.cfg.eventTimeout()
	}

	if _, err := recorder.await(ctx, timeout); err != nil {
		return err
	}
	return nil
}

// theFlagShouldBePartOfTheEventPayload asserts that the configuration-change
// event named the flag that changed.
func theFlagShouldBePartOfTheEventPayload(ctx context.Context) error {
	state, err := stateFrom(ctx)
	if err != nil {
		return err
	}
	flag, err := state.requireFlag()
	if err != nil {
		return err
	}

	recorder, err := state.recorder(openfeature.ProviderConfigChange)
	if err != nil {
		return err
	}
	if recorder.last == nil {
		return errors.New("no configuration-change event has been consumed in this scenario: " +
			"a \"the change event handler should have been executed\" step must come first")
	}

	changed := recorder.last.FlagChanges
	if slices.Contains(changed, flag.key) {
		return nil
	}

	if len(changed) == 0 {
		return fmt.Errorf("the configuration-change event carried no changed flags, expected it to "+
			"name %q", flag.key)
	}
	return fmt.Errorf("the configuration-change event named [%s], expected it to include %q",
		strings.Join(changed, " "), flag.key)
}

func theConnectionIsLost(ctx context.Context) error {
	state, err := stateFrom(ctx)
	if err != nil {
		return err
	}

	control, err := connectionControl(state.cfg.Control, "disconnect")
	if err != nil {
		return err
	}
	if err := control.Disconnect(ctx); err != nil {
		return fmt.Errorf("could not make the backend unreachable: %w", err)
	}
	return nil
}

func theConnectionIsRestored(ctx context.Context) error {
	state, err := stateFrom(ctx)
	if err != nil {
		return err
	}

	control, err := connectionControl(state.cfg.Control, "reconnect")
	if err != nil {
		return err
	}
	if err := control.Reconnect(ctx); err != nil {
		return fmt.Errorf("could not make the backend reachable again: %w", err)
	}
	return nil
}

// theClientShouldBeInState asserts the provider status the client reports.
//
// It is checked after the corresponding event has been consumed, and the SDK
// writes provider status before running handlers, so no polling is needed here:
// if the event arrived, the status is already current.
func theClientShouldBeInState(ctx context.Context, name string) error {
	state, err := stateFrom(ctx)
	if err != nil {
		return err
	}
	client, err := state.requireClient()
	if err != nil {
		return err
	}

	expected, ok := stateByName[name]
	if !ok {
		return fmt.Errorf("unknown provider state %q", name)
	}

	if actual := client.State(); actual != expected {
		return fmt.Errorf("client reports state %q, expected %q", actual, expected)
	}
	return nil
}
