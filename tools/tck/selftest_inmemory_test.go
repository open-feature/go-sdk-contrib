package tck

import (
	"context"
	"errors"
	"testing"

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
// way — see BackendControl.
func TestInMemoryProvider(t *testing.T) {
	Run(t,
		WithName("in-memory"),
		WithControl(plainMemoryControl{}),
		WithProvider(func(context.Context) (openfeature.FeatureProvider, error) {
			flags, err := CanonicalFlagSet()
			if err != nil {
				return nil, err
			}
			return memprovider.NewInMemoryProvider(flags), nil
		}),
		// Every omission is a fact about memprovider rather than a
		// convenience:
		//
		//   - ConfigurationChange: it cannot update its flag set or emit an
		//     event at all — see plainMemoryControl.
		//   - Stale and UnavailableInit: there is no connection to lose, which
		//     is also why plainMemoryControl does not implement
		//     ConnectionControl. The two omissions keep each other honest.
		//   - Lifecycle: it does not implement openfeature.StateHandler, so the
		//     SDK synthesises PROVIDER_READY for it and the readiness scenario
		//     would pass without exercising anything.
		//   - Targeting: an in-memory flag set evaluates no rules.
		//     CanonicalFlagSet deliberately ignores the targeting member of
		//     targeting-key-flag rather than translating JsonLogic into a
		//     ContextEvaluator. The untagged scenario that supplies a context
		//     to an untargeted flag still runs: it asserts only that supplying
		//     one is harmless.
		//   - Caching: reserved, so declaring it is a configuration error.
		//   - NumericCoercion: it type-asserts rather than coercing, so it
		//     refuses integral-float-flag (10.0) as an integer and
		//     integer-flag (10) as a float as well as the lossy
		//     float-flag (0.5) — and the lossless direction is required too.
		//
		// LargeIntegers, Variants, StandardReasons, StringTyping,
		// FullyTypedValues and Object are declared: it hands back the int64 it
		// was seeded with, reports the variant name, reports STATIC for a
		// rule-less flag, and holds Go values, so there is no text
		// representation for a float or a structure to be mistaken for.
		//
		// DisabledFlags is declared too, which it was not until go-sdk v1.19.0:
		// memprovider used to return the caller's default for a disabled flag
		// and attach a GENERAL resolution error to it, so all four rows failed
		// on the empty error-code assertion. go-sdk#574 dropped the error.
		WithCapabilities(
			Events,
			Object,
			Variants,
			LargeIntegers,
			StandardReasons,
			StringTyping,
			FullyTypedValues,
			DisabledFlags,
		),
	)
}

// plainMemoryControl is the backend control for the SDK's in-memory provider.
//
// PrepareScenario is a no-op because the provider is rebuilt from
// CanonicalFlagSet for every scenario, so each one already starts from an
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
			"controllableProvider for what the SDK's provider is missing")
}

func (plainMemoryControl) Description() string {
	return "the Go SDK's memprovider.InMemoryProvider, rebuilt per scenario"
}

// BackendType is in-process because this control manipulates a provider in this
// process rather than a backend over HTTP. It is required rather than optional
// precisely so that a custom control like this one has to say.
func (plainMemoryControl) BackendType() BackendType { return BackendTypeInProcess }
