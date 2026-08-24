package tck

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/cucumber/godog"
	"github.com/open-feature/go-sdk/openfeature"
)

// registerProviderSteps binds the steps that put a provider under test, and
// those that address the provider itself rather than a flag: its identity and
// its lifecycle.
func registerProviderSteps(ctx *godog.ScenarioContext) {
	ctx.Step(`^a stable provider$`, aStableProvider)
	ctx.Step(`^an? unavailable provider$`, anUnavailableProvider)
	ctx.Step(`^the provider metadata name should not be empty$`, theProviderMetadataNameShouldNotBeEmpty)
	ctx.Step(`^the provider is shut down$`, theProviderIsShutDown)
	ctx.Step(`^the provider is initialized again$`, theProviderIsInitializedAgain)
	ctx.Step(`^the shutdown should have completed within (\d+)ms$`, theShutdownShouldHaveCompletedWithin)
}

// aStableProvider registers the provider under test against the running,
// seeded backend and waits for it to become ready.
//
// The wait is the point. Every scenario that follows assumes a provider that
// has finished initialising, and a suite that started evaluating before that
// would report races in the TCK as defects in the provider.
func aStableProvider(ctx context.Context) (context.Context, error) {
	state, err := stateFrom(ctx)
	if err != nil {
		return ctx, err
	}

	provider, err := state.cfg.NewProvider(ctx)
	if err != nil {
		return ctx, fmt.Errorf("the provider factory from tck.WithProvider failed: %w", err)
	}
	if provider == nil {
		return ctx, errors.New("the provider factory from tck.WithProvider returned a nil provider")
	}

	panicValue, regErr := registerProvider(ctx, state.cfg.domain(), provider, state.cfg.readyTimeout())
	if panicValue != nil {
		return ctx, fmt.Errorf(
			"the provider panicked while being registered: %v. A provider must never panic out of "+
				"registration — it takes the host application down with it", panicValue)
	}
	if regErr != nil {
		return ctx, fmt.Errorf(
			"the provider did not become ready within %s: %w. The backend is up and seeded at this "+
				"point, so either initialisation is genuinely failing or tck.WithReadyTimeout is too short",
			state.cfg.readyTimeout(), regErr)
	}

	state.provider = provider
	state.providerName = provider.Metadata().Name
	state.client = openfeature.NewClient(state.cfg.domain())
	return ctx, nil
}

// anUnavailableProvider registers a provider pointed at a backend that does not
// exist.
//
// Neither a failed registration nor a returned error is a failure here: what
// the contract requires is that the provider settles into an observable error
// state promptly instead of hanging or panicking, and the scenario asserts that
// through the event and the client state. A panic is fatal, because a provider
// that panics out of registration takes the application with it.
func anUnavailableProvider(ctx context.Context) (context.Context, error) {
	state, err := stateFrom(ctx)
	if err != nil {
		return ctx, err
	}

	if state.cfg.NewUnavailableProvider == nil {
		return ctx, fmt.Errorf(
			"no tck.WithUnavailableProvider was given but an @unavailable scenario ran. This is a "+
				"test-configuration bug rather than a provider defect: the suite declared %s "+
				"without supplying a provider that cannot reach its backend. Remove that "+
				"capability, or supply the factory", UnavailableInit)
	}

	provider, err := state.cfg.NewUnavailableProvider(ctx)
	if err != nil {
		return ctx, fmt.Errorf("the factory from tck.WithUnavailableProvider failed: %w", err)
	}
	if provider == nil {
		return ctx, errors.New("the factory from tck.WithUnavailableProvider returned a nil provider")
	}

	// The registration error is deliberately discarded. What the contract
	// requires is an observable error state, which the scenario checks through
	// the event and the client status; whether registration also returned an
	// error is an SDK detail rather than part of the provider contract.
	panicValue, _ := registerProvider(ctx, state.cfg.domain(), provider, state.cfg.readyTimeout())
	if panicValue != nil {
		return ctx, fmt.Errorf(
			"the provider panicked while being registered against an unreachable backend: %v. "+
				"Failing to connect must be reported as an error state, never as a panic", panicValue)
	}

	state.provider = provider
	state.client = openfeature.NewClient(state.cfg.domain())
	return ctx, nil
}

// theProviderMetadataNameShouldNotBeEmpty asserts that the provider identifies
// itself (requirement 2.1.1).
//
// Too small to test, until a conformance report keyed on this name made an
// empty one into a report nobody can attribute.
func theProviderMetadataNameShouldNotBeEmpty(ctx context.Context) error {
	state, err := stateFrom(ctx)
	if err != nil {
		return err
	}
	provider, err := state.requireProvider()
	if err != nil {
		return err
	}

	if provider.Metadata().Name == "" {
		return errors.New("the provider's metadata name is empty. A provider must identify itself; " +
			"an empty name leaves every report and log line about it unattributable")
	}
	return nil
}

// theProviderIsShutDown calls the provider's own Shutdown, directly.
//
// Directly, and not by replacing it in the SDK: replacing a provider tests the
// SDK's bookkeeping as much as the provider's shutdown, and Appendix B already
// covers the SDK. It also leaves the provider registered, so that a following
// "the provider is initialized again" step brings the very same instance back
// and the client still reaches it.
//
// A provider that does not implement openfeature.StateHandler has no shutdown
// to call, and the step records an instantaneous no-op: there is nothing that
// could hang, panic or be repeated. The @lifecycle gate keeps such a provider
// out of these scenarios in the ordinary course of things.
//
// The duration and any panic are recorded rather than asserted here, because
// the assertions belong to later steps: "the shutdown should have completed
// within" reads the duration, "no exception should have been thrown" the
// panic. Only a shutdown that never returns fails this step itself.
func theProviderIsShutDown(ctx context.Context) error {
	state, err := stateFrom(ctx)
	if err != nil {
		return err
	}
	provider, err := state.requireProvider()
	if err != nil {
		return err
	}

	handler, ok := provider.(openfeature.StateHandler)
	if !ok {
		state.lifecycle = append(state.lifecycle, lifecycleCall{operation: lifecycleShutdown})
		return nil
	}

	call, err := callStateHandler(lifecycleShutdown, state.cfg.readyTimeout(), func() error {
		handler.Shutdown()
		return nil
	})
	if err != nil {
		return err
	}
	state.lifecycle = append(state.lifecycle, call)
	return nil
}

// theProviderIsInitializedAgain calls the provider's own Init, directly, on the
// instance the scenario shut down.
//
// The same instance is the point. Requirement 2.5.2 says a shut-down provider
// reverts to its uninitialised state, and the only observable proof is that it
// can be initialised again and then serves flags, which the evaluation step
// that follows checks through the client. The client still routes to this
// instance because it was never replaced in the SDK, and the SDK still holds
// it as READY because the SDK was never told about the shutdown; both are what
// let that evaluation reach the provider at all.
//
// Registering it with the SDK again would not do. The SDK compares providers
// with reflect.DeepEqual unless they are pointers, so it can conclude that the
// provider has not changed and initialise nothing — and even where it does
// re-initialise, the outcome would be the SDK's rather than the provider's.
//
// An error from Init is recorded like a panic and fails "no exception should
// have been thrown". Unlike an evaluation error, which is the normal shape of
// a code default, an initialisation error is the provider refusing to come
// back, which is what the other languages' initialize() throws to say.
func theProviderIsInitializedAgain(ctx context.Context) error {
	state, err := stateFrom(ctx)
	if err != nil {
		return err
	}
	provider, err := state.requireProvider()
	if err != nil {
		return err
	}

	handler, ok := provider.(openfeature.StateHandler)
	if !ok {
		state.lifecycle = append(state.lifecycle, lifecycleCall{operation: lifecycleInit})
		return nil
	}

	call, err := callStateHandler(lifecycleInit, state.cfg.readyTimeout(), func() error {
		return handler.Init(openfeature.EvaluationContext{})
	})
	if err != nil {
		return err
	}
	state.lifecycle = append(state.lifecycle, call)
	return nil
}

// theShutdownShouldHaveCompletedWithin bounds the most recent shutdown.
//
// What is being asserted is that shutdown returns at all rather than waiting
// on a backend that will never answer; the bound the feature file gives is
// generous. A shutdown that panicked is reported by "no exception should have
// been thrown" rather than here — its duration is still real.
func theShutdownShouldHaveCompletedWithin(ctx context.Context, millis int) error {
	state, err := stateFrom(ctx)
	if err != nil {
		return err
	}
	call, err := state.lastShutdown()
	if err != nil {
		return err
	}

	bound := time.Duration(millis) * time.Millisecond
	if call.duration > bound {
		return fmt.Errorf("shutdown took %s, over the %s bound. A shutdown that waits for a graceful "+
			"close of a connection that will never answer hangs the host application's own shutdown",
			call.duration, bound)
	}
	return nil
}

// callStateHandler runs one openfeature.StateHandler method on its own
// goroutine, timing it and converting a panic into a value the assertions can
// report rather than letting it unwind through godog.
//
// Its own goroutine, so that the wait can be bounded. A shutdown that never
// returns is the very defect the prompt-shutdown scenario exists to catch, and
// calling it inline would hang the test binary until go test killed it, with
// no scenario named; giving up after timeout fails the step instead. The
// goroutine is abandoned in that case, which is the price of not hanging.
func callStateHandler(operation string, timeout time.Duration, fn func() error) (lifecycleCall, error) {
	done := make(chan lifecycleCall, 1)
	started := time.Now()

	go func() {
		call := lifecycleCall{operation: operation}
		defer func() {
			call.duration = time.Since(started)
			if v := recover(); v != nil {
				call.panicked = true
				call.panicValue = v
			}
			done <- call
		}()
		call.err = fn()
	}()

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case call := <-done:
		return call, nil
	case <-timer.C:
		return lifecycleCall{}, fmt.Errorf(
			"the provider's %s did not return within %s. A %s that blocks on its backend hangs the "+
				"host application; if the provider is merely slow, raise tck.WithReadyTimeout",
			operation, timeout, operation)
	}
}

// registerProvider hands a provider to the OpenFeature API and waits for
// initialisation to settle, converting a panic into a value the caller can
// report rather than letting it unwind through godog.
func registerProvider(
	ctx context.Context,
	domain string,
	provider openfeature.FeatureProvider,
	timeout time.Duration,
) (panicValue any, regErr error) {
	defer func() {
		if v := recover(); v != nil {
			panicValue = v
		}
	}()

	tctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	regErr = openfeature.SetNamedProviderWithContextAndWait(tctx, domain, provider)
	return nil, regErr
}
