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
// It earns its keep twice over.
//
// It is the reference adoption for a provider with no backend: everything a
// file-based or environment-variable provider has to write is here, and it is
// four options.
//
// It is also the Docker-free canary. Needing no container, no compose stack and
// no network, it runs in a fraction of a second on any machine and in any CI
// job, which makes it the fast check that catches a broken step definition, a
// mis-wired capability gate or a regression in the shared harness long before a
// containerised suite would. When a change breaks both this and the flagd
// suite, this one says so in a second and points at the TCK rather than at a
// provider.
//
// What it does not do is license providers that have a backend to test
// themselves this way — see tck.BackendControl for why.
func TestInMemoryProvider(t *testing.T) {
	tck.Run(t,
		tck.WithName("in-memory"),
		tck.WithControl(plainMemoryControl{}),
		tck.WithProvider(func(context.Context) (openfeature.FeatureProvider, error) {
			return memprovider.NewInMemoryProvider(tck.CanonicalFlagSet()), nil
		}),
		// Four capabilities, and every omission is a fact about the provider
		// rather than a convenience:
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
		//     inferred, and the tag is withheld. See
		//     TestCanonicalFlagSetDisabledFlagsCarryAnError, which fails when
		//     the SDK stops doing it so that the declaration can be added.
		//
		// LargeIntegers is declared: the accessor is int64 and memprovider
		// hands the int64 it was seeded with straight back, so 2^53-1
		// survives the trip untouched.
		//
		// Variants is declared: memprovider resolves a named variant and
		// reports its name, so every row of the variant outline resolves the
		// name the canonical set gives it.
		tck.WithCapabilities(
			tck.Events,
			tck.Object,
			tck.Variants,
			tck.LargeIntegers,
		),
	)
}

// plainMemoryControl is the backend control for the SDK's in-memory provider.
//
// PrepareScenario is a no-op because the provider is rebuilt from
// tck.CanonicalFlagSet for every scenario, so each one already starts from an
// untouched baseline.
//
// ChangeFlag cannot be implemented at all, and the error says why. Appendix A
// of the specification requires an SDK's in-memory provider to "support a means
// of updating the flag set, resulting in the emission of
// PROVIDER_CONFIGURATION_CHANGED events"; the Go SDK's does not, so there is
// nothing to call. The suite above therefore leaves ConfigurationChange
// undeclared and the scenario is reported as skipped with its reason, which is
// the honest outcome. Reaching this error would mean the capability had been
// declared anyway.
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
