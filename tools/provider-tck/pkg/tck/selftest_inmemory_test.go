package tck_test

import (
	"context"
	"errors"
	"testing"

	"github.com/open-feature/go-sdk-contrib/tools/provider-tck/pkg/tck"
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
// one struct literal.
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
	tck.Run(t, tck.Config{
		Name:    "in-memory",
		Control: plainMemoryControl{},
		NewProvider: func(context.Context) (openfeature.FeatureProvider, error) {
			return memprovider.NewInMemoryProvider(tck.CanonicalFlagSet()), nil
		},
		// Three capabilities, and every omission is a fact about the provider
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
		//   - Targeting and Caching are omitted because no scenario carries
		//     their tags yet, so leaving them out skips nothing.
		//
		// StrictNumericTyping is declared, and that is worth stating plainly:
		// memprovider refuses to narrow float-flag (0.5) to an integer and
		// reports TYPE_MISMATCH instead. It is the reference behaviour that
		// capability describes.
		Capabilities: []tck.Capability{
			tck.Events,
			tck.Object,
			tck.StrictNumericTyping,
		},
	})
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

func (plainMemoryControl) Description() string {
	return "the Go SDK's memprovider.InMemoryProvider, rebuilt per scenario"
}
