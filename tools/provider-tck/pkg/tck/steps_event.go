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

	state.recorders[eventType] = newEventRecorder(client, eventType)
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
// The scenarios that use it are asserting promptness, not just eventual
// arrival: a provider that cannot reach its backend has to report that fact
// quickly, because an application blocked on provider registration is down.
// This bound therefore overrides Config.EventTimeout rather than being clamped
// by it.
func theEventHandlerShouldHaveBeenExecutedWithin(ctx context.Context, name string, millis int) error {
	return awaitEvent(ctx, name, time.Duration(millis)*time.Millisecond)
}

// awaitEvent consumes the next event of the named type. A timeout of zero means
// Config.EventTimeout.
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
//
// Naming the changed flags is what makes the event actionable: a consumer that
// caches evaluations needs to know what to invalidate, and an event carrying no
// keys forces it to invalidate everything.
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
