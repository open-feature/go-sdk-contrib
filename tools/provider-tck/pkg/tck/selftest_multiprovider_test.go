package tck_test

import (
	"context"
	"testing"

	"github.com/open-feature/go-sdk-contrib/tools/provider-tck/pkg/tck"
	"github.com/open-feature/go-sdk/openfeature"
	"github.com/open-feature/go-sdk/openfeature/multi"
)

// TestMultiProvider runs the conformance suite against the SDK's multi-provider
// wrapping exactly one child.
//
// A provider that delegates is still a provider, and delegation is where the
// contract is easiest to drop on the floor: a variant that does not survive the
// hop, a reason rewritten to DEFAULT, an error code flattened to GENERAL, an
// event that never reaches the client. Wrapping exactly one child makes each of
// those observable, because the correct answer is precisely what
// TestControllableProvider already asserts about the child on its own. Any
// difference between the two suites is attributable to the multi-provider and
// nothing else.
//
// That framing is why this belongs here rather than in the SDK: it is not a
// test of aggregation across several backends, it is a test that delegation is
// transparent. It costs one file and needs no Docker.
//
// The equivalent Java suite has to leave ConfigurationChange undeclared,
// because Java's MultiProvider never subscribes to its children and swallows
// their events (open-feature/java-sdk#1882). Go's forwards
// PROVIDER_CONFIGURATION_CHANGED straight through, explicitly matching the JS
// reference behaviour, so the capability is declared here and the scenario is
// expected to pass. If it does not, the two SDKs disagree and this suite is
// where that shows up.
func TestMultiProvider(t *testing.T) {
	control := tck.NewInProcessControl()

	tck.Run(t, tck.Config{
		Name:    "multi-provider",
		Control: control,
		NewProvider: func(context.Context) (openfeature.FeatureProvider, error) {
			provider, err := multi.NewProvider(
				multi.StrategyFirstMatch,
				multi.WithProvider("in-memory", control.NewProvider()),
			)
			if err != nil {
				return nil, err
			}
			return provider, nil
		},
		// Lifecycle is omitted although TestControllableProvider declares it for
		// the very same child, and the asymmetry is intentional. There is no
		// backend to reach on either side; what makes the child's readiness
		// worth asserting is that it comes out of its own Init, and that is a
		// claim about the child, which its own suite already makes. Asserting
		// it again through the wrapper would say nothing about delegation —
		// which is the only thing this suite is for — while spending a green
		// scenario on it. Before @lifecycle existed the readiness scenario ran
		// here on the strength of Events and passed vacuously.
		Capabilities: []tck.Capability{
			tck.Events,
			tck.ConfigurationChange,
			tck.Object,
			tck.StrictNumericTyping,
		},
	})
}
