package tck

import (
	"context"
	"testing"

	"github.com/open-feature/go-sdk/openfeature"
)

// TestControllableProvider runs the conformance suite against the TCK's own
// updatable in-memory provider.
//
// It exists because the SDK's in-memory provider cannot update a flag set or
// emit events, so TestInMemoryProvider necessarily skips the
// configuration-change scenarios. Without this suite the change-event step
// definitions would ship with no coverage, and a break in them would first
// surface in a containerised provider suite where it looks like a provider
// defect.
//
// It is also the reference for an in-process control path where the provider
// does support updates.
func TestControllableProvider(t *testing.T) {
	control := NewInProcessControl()

	Run(t,
		WithName("controllable-in-memory"),
		WithControl(control),
		WithProvider(func(context.Context) (openfeature.FeatureProvider, error) {
			return control.NewProvider(), nil
		}),
		// ConfigurationChange is what this suite adds over
		// TestInMemoryProvider, and it is the whole point of it. Stale and
		// UnavailableInit stay undeclared: there is still no connection to
		// lose, and InProcessControl does not implement
		// ConnectionControl.
		//
		// Lifecycle is declared here and nowhere else among the self-tests, for
		// a mechanical reason: controllableProvider implements
		// openfeature.StateHandler, so the client's READY is the observable
		// outcome of its own Init. This is therefore the only suite covering
		// the @lifecycle steps without Docker. Reinitialization comes with it
		// because Shutdown closes the event channel and nothing else, Init has
		// nothing to reconnect, and the flag snapshot outlives both.
		//
		// Every resolution decision is still memprovider's, so the
		// resolution-shaped capabilities match TestInMemoryProvider's exactly:
		// NumericCoercion and Targeting are omitted for the reasons given
		// there, and LargeIntegers, Variants, StandardReasons, StringTyping,
		// FullyTypedValues and DisabledFlags are declared for the same reasons
		// in reverse. Wrapping changes nothing: this provider adds
		// update-and-emit and delegates the rest.
		WithCapabilities(
			Events,
			Lifecycle,
			Reinitialization,
			ConfigurationChange,
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
