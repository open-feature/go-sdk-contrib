package tck_test

import (
	"context"
	"errors"
	"testing"

	"github.com/open-feature/go-sdk-contrib/tools/tck"
	"github.com/open-feature/go-sdk/openfeature"
	"github.com/open-feature/go-sdk/openfeature/memprovider"
)

// TestInMemoryProvider runs the conformance suite against the Go SDK's own
// memprovider.InMemoryProvider.
//
// It is the reference adoption for a provider with no backend, and it is the
// Docker-free canary: needing no container, no compose stack and no network, it
// runs in a fraction of a second anywhere, so a broken step definition, a
// mis-wired capability gate or a regression in the shared harness shows up here
// first and points at the TCK rather than at a provider.
//
// It does not license providers that have a backend to test themselves this
// way — see tck.BackendControl.
func TestInMemoryProvider(t *testing.T) {
	tck.Run(t,
		tck.WithName("in-memory"),
		tck.WithControl(plainMemoryControl{}),
		tck.WithProvider(func(context.Context) (openfeature.FeatureProvider, error) {
			return memprovider.NewInMemoryProvider(tck.CanonicalFlagSet()), nil
		}),
		// Every omission is a fact about memprovider rather than a
		// convenience:
		//
		//   - ConfigurationChange: it cannot update its flag set or emit an
		//     event at all — see plainMemoryControl.
		//   - Stale and UnavailableInit: there is no connection to lose, which
		//     is also why plainMemoryControl does not implement
		//     tck.ConnectionControl. The two omissions keep each other honest.
		//   - Lifecycle: it does not implement openfeature.StateHandler, so the
		//     SDK synthesises PROVIDER_READY for it and the readiness scenario
		//     would pass without exercising anything.
		//   - Targeting: an in-memory flag set evaluates no rules.
		//     tck.CanonicalFlagSet deliberately ignores the targeting member of
		//     targeting-key-flag rather than translating JsonLogic into a
		//     ContextEvaluator. The untagged scenario that supplies a context
		//     to an untargeted flag still runs: it asserts only that supplying
		//     one is harmless.
		//   - Caching: reserved, so declaring it is a configuration error.
		//   - NumericCoercion: it type-asserts rather than coercing, so it
		//     refuses integral-float-flag (10.0) as an integer and
		//     integer-flag (10) as a float as well as the lossy
		//     float-flag (0.5) — and the lossless direction is required too.
		//   - DisabledFlags: the one omission here that records a defect rather
		//     than an absence. memprovider returns the caller's default but
		//     attaches a GENERAL resolution error while reporting reason
		//     DISABLED, so all four rows fail on the empty error-code —
		//     measured by TestCanonicalFlagSetDisabledFlagsCarryAnError, which
		//     also fails when the SDK stops doing it so the declaration can be
		//     added. Withholding for an identified defect is allowed only under
		//     Appendix F's self-test carve-out, whose condition is exactly that
		//     pinning test. An adoption has no such licence.
		//
		// LargeIntegers, Variants, StandardReasons, StringTyping,
		// FullyTypedValues and Object are declared: it hands back the int64 it
		// was seeded with, reports the variant name, reports STATIC for a
		// rule-less flag, and holds Go values, so there is no text
		// representation for a float or a structure to be mistaken for.
		tck.WithCapabilities(
			tck.Events,
			tck.Object,
			tck.Variants,
			tck.LargeIntegers,
			tck.StandardReasons,
			tck.StringTyping,
			tck.FullyTypedValues,
		),
	)
}

// plainMemoryControl is the backend control for the SDK's in-memory provider.
//
// PrepareScenario is a no-op because the provider is rebuilt from
// tck.CanonicalFlagSet for every scenario, so each one already starts from an
// untouched baseline.
//
// ChangeFlag cannot be implemented at all, and the error says why. The suite
// above therefore leaves ConfigurationChange undeclared and the scenario is
// reported as skipped with its reason; reaching this error would mean the
// capability had been declared anyway.
type plainMemoryControl struct{}

func (plainMemoryControl) PrepareScenario(context.Context) error { return nil }

func (plainMemoryControl) ChangeFlag(context.Context) error {
	return errors.New(
		"memprovider.InMemoryProvider cannot change its flag set: it exposes no update method and " +
			"does not implement openfeature.EventHandler, so a configuration change can be neither " +
			"applied nor signalled. Appendix A of the specification requires both. See " +
			"tck.ControllableProvider for what the SDK's provider is missing")
}

// ControlAPI reports that this control manipulates a provider in this process
// rather than driving a backend over HTTP, which is what the in-memory provider
// is: there is no backend to drive.
func (plainMemoryControl) ControlAPI() string { return "in-process" }

func (plainMemoryControl) Description() string {
	return "the Go SDK's memprovider.InMemoryProvider, rebuilt per scenario"
}

// ControlAPI is in-process because this control manipulates a provider in this
// process rather than a backend over HTTP. It is required rather than optional
// precisely so that a custom control like this one has to say.
func (plainMemoryControl) ControlAPI() tck.ControlAPI { return tck.ControlAPIInProcess }
