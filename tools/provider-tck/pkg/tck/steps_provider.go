package tck

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/cucumber/godog"
	"github.com/open-feature/go-sdk/openfeature"
)

// registerProviderSteps binds the steps that put a provider under test.
func registerProviderSteps(ctx *godog.ScenarioContext) {
	ctx.Step(`^a stable provider$`, aStableProvider)
	ctx.Step(`^an? unavailable provider$`, anUnavailableProvider)
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
		return ctx, fmt.Errorf("Config.NewProvider failed: %w", err)
	}
	if provider == nil {
		return ctx, errors.New("Config.NewProvider returned a nil provider")
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
				"point, so either initialisation is genuinely failing or Config.ReadyTimeout is too short",
			state.cfg.readyTimeout(), regErr)
	}

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
			"Config.NewUnavailableProvider is nil but an @unavailable scenario ran. This is a "+
				"test-configuration bug rather than a provider defect: the suite declared %s "+
				"without supplying a provider that cannot reach its backend. Remove that "+
				"capability, or supply the factory", UnavailableInit)
	}

	provider, err := state.cfg.NewUnavailableProvider(ctx)
	if err != nil {
		return ctx, fmt.Errorf("Config.NewUnavailableProvider failed: %w", err)
	}
	if provider == nil {
		return ctx, errors.New("Config.NewUnavailableProvider returned a nil provider")
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

	state.client = openfeature.NewClient(state.cfg.domain())
	return ctx, nil
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
