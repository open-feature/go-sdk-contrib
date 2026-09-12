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
		tck.WithCapabilities(inMemoryCapabilities()...),
	)
}

// inMemoryCapabilities is everything memprovider.InMemoryProvider can honestly
// claim, and the one place that judgement is recorded.
//
// Every self-test that drives this backend shares it rather than restating it,
// including the extension self-tests in selftest_extensions_test.go, which are
// this same suite plus two fields. A capability list copied per test drifts
// silently: when the canonical assets gained the lossless coercion scenarios,
// the copy that still declared NumericCoercion began failing them while this
// list had already been corrected. Sharing it means a capability the provider
// turns out not to have is withdrawn once.
//
// Four capabilities, and every omission is a fact about the provider rather
// than a convenience:
//
//   - ConfigurationChange is omitted because the SDK's in-memory
//     provider cannot update its flag set or emit an event. That is a
//     finding, not a configuration choice — see plainMemoryControl.
//   - Stale and UnavailableInit are omitted because there is no
//     connection to lose. plainMemoryControl does not implement
//     tck.ConnectionControl for the same reason, and the two omissions
//     keep each other honest: the scenarios are skipped before any step
//     can reach an operation the control cannot perform.
//   - Lifecycle is omitted because there is no backend to reach and no
//     initialisation that reaches it: memprovider.InMemoryProvider does
//     not implement openfeature.StateHandler, so the SDK synthesises
//     PROVIDER_READY for it. Until @lifecycle existed the readiness
//     scenario ran here on the strength of Events alone and passed
//     without exercising anything — a NoopProvider would have passed it
//     the same way. Skipping it is the honest outcome, and it is the
//     reference answer for every backend-less provider adopting this
//     suite.
//   - Targeting is omitted because an in-memory flag set evaluates no
//     rules. tck.CanonicalFlagSet deliberately ignores the targeting
//     member of targeting-key-flag rather than translating flagd's
//     JsonLogic into a ContextEvaluator, so the flag resolves to its
//     miss variant whatever the context and the matching scenario
//     would fail. Undeclared is the accurate report — see
//     tck.CanonicalFlagSet. The untagged scenario that supplies a
//     context to an untargeted flag is unaffected and runs: it asserts
//     only that supplying one is harmless, which is a property of the
//     provider rather than of the flag set.
//   - Caching is omitted because it is reserved: no scenario carries
//     the tag, so declaring it is rejected as a configuration error
//     rather than reported as a result.
//   - NumericCoercion is omitted because memprovider does not coerce:
//     it type-asserts, so it refuses to narrow float-flag (0.5) to an
//     integer — the lossy half of the rule, which it gets right — but
//     it equally refuses integral-float-flag (10.0) as an integer and
//     integer-flag (10) as a float, and reports TYPE_MISMATCH for both.
//     Those are the lossless half, which the capability requires too;
//     rejecting every float is precisely the shortcut those scenarios
//     exist to stop. Until the flag set contained an integral float
//     this suite declared the capability on the strength of the lossy
//     scenario alone, which is the gap the spec closed.
//   - DisabledFlags is omitted, and this is the one omission here that
//     records a defect rather than an absence. An in-memory provider is
//     the architecture that can satisfy the capability — the caller's
//     default never leaves the process, so there is nothing to ask a
//     server for — and memprovider does return that default. It just
//     attaches a GENERAL resolution error to it while reporting reason
//     DISABLED, and those contradict each other: 2.2.5 lists DISABLED
//     among the reasons a resolution that worked may carry. So all four
//     rows fail on "the error-code should be """, measured rather than
//     inferred, and the tag is withheld.
//
//     Withholding for an identified defect is what Appendix F's
//     known-deviation rule otherwise forbids, and it is allowed here by
//     the self-test carve-out the appendix added in spec 045950ca: a
//     TCK implementation's own suites are a fixture for the harness
//     rather than a report about a third party, and they run in the
//     ordinary build, where a permanently failing scenario is a broken
//     build rather than a finding. The carve-out has one condition --
//     that the defect be pinned by a test of its own -- and that is the
//     condition met here, by
//     TestCanonicalFlagSetDisabledFlagsCarryAnError, which asserts the
//     behaviour directly and fails when the SDK stops doing it, so that
//     the declaration can be added. An adoption has no such licence.
//
// LargeIntegers is declared: the accessor is int64 and memprovider hands the
// int64 it was seeded with straight back, so 2^53-1 survives the trip
// untouched.
//
// Variants is declared: memprovider resolves a named variant and reports its
// name, so every row of the variant outline resolves the name the canonical set
// gives it.
//
// StandardReasons is declared: memprovider reports STATIC for a rule-less flag
// and the SDK reports ERROR for the unknown-flag and type-mismatch cases, which
// is the whole of reason.feature that is reachable here. Its two @targeting
// rows and its one @disabled-flags row compose with capabilities this suite
// withholds, so they are skipped for those rather than for this one.
//
// StringTyping is declared, and it is the same property of memprovider that
// costs it NumericCoercion paying off: it type-asserts, so a boolean, an
// integer, a float or a structure requested through the string accessor is
// refused rather than formatted. All four scenarios pass, the object one
// included, @object being declared here too.
func inMemoryCapabilities() []tck.Capability {
	return []tck.Capability{
		tck.Events,
		tck.Object,
		tck.Variants,
		tck.LargeIntegers,
		tck.StandardReasons,
		tck.StringTyping,
		tck.FullyTypedValues,
	}
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

func (plainMemoryControl) Description() string {
	return "the Go SDK's memprovider.InMemoryProvider, rebuilt per scenario"
}

// ControlAPI is in-process because this control manipulates a provider in this
// process rather than a backend over HTTP. It is required rather than optional
// precisely so that a custom control like this one has to say.
func (plainMemoryControl) ControlAPI() tck.ControlAPI { return tck.ControlAPIInProcess }
