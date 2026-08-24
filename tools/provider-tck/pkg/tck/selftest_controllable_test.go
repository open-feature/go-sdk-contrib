package tck_test

import (
	"context"
	"testing"

	"github.com/open-feature/go-sdk-contrib/tools/provider-tck/pkg/tck"
	"github.com/open-feature/go-sdk/openfeature"
)

// TestControllableProvider runs the conformance suite against the TCK's own
// updatable in-memory provider.
//
// This is the suite that exercises the configuration-change path, and it exists
// because the SDK's in-memory provider cannot: it has no way to update a flag
// set and emits no events, so TestInMemoryProvider necessarily skips those
// scenarios. Without this suite the change-event step definitions would ship
// with no coverage at all, and a break in them would first surface in a
// containerised provider suite where it looks like a provider defect.
//
// It is also the reference for what an in-process control path looks like when
// the provider does support updates, which is what a file-based or
// environment-variable provider should be able to do.
func TestControllableProvider(t *testing.T) {
	control := tck.NewInProcessControl()

	tck.Run(t, tck.Config{
		Name:    "controllable-in-memory",
		Control: control,
		NewProvider: func(context.Context) (openfeature.FeatureProvider, error) {
			return control.NewProvider(), nil
		},
		// Stale and UnavailableInit stay undeclared: there is still no
		// connection to lose, and tck.InProcessControl does not implement
		// tck.ConnectionControl. ConfigurationChange is the one this suite adds
		// over TestInMemoryProvider, and it is the whole point of it.
		Capabilities: []tck.Capability{
			tck.Events,
			tck.ConfigurationChange,
			tck.Object,
			tck.StrictNumericTyping,
		},
	})
}
