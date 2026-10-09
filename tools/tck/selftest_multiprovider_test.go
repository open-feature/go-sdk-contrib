package tck_test

import (
	"context"
	"testing"

	"github.com/open-feature/go-sdk-contrib/tools/tck"
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
// TestControllableProvider already asserts about the child on its own, so any
// difference between the two suites is attributable to the multi-provider and
// nothing else.
//
// It is not a test of aggregation across several backends, it is a test that
// delegation is transparent. It costs one file and needs no Docker.
func TestMultiProvider(t *testing.T) {
	control := tck.NewInProcessControl()

	tck.Run(t,
		tck.WithName("multi-provider"),
		tck.WithControl(control),
		tck.WithProvider(func(context.Context) (openfeature.FeatureProvider, error) {
			provider, err := multi.NewProvider(
				multi.StrategyFirstMatch,
				multi.WithProvider("in-memory", control.NewProvider()),
			)
			if err != nil {
				return nil, err
			}
			return provider, nil
		}),
		// Lifecycle is omitted although TestControllableProvider declares it
		// for the very same child, and the asymmetry is intentional: what makes
		// the child's readiness worth asserting is that it comes out of its own
		// Init, which is a claim about the child that its own suite already
		// makes. Asserting it again through the wrapper would say nothing about
		// delegation.
		//
		// A wrapper cannot pass a scenario its child fails, so NumericCoercion,
		// Targeting and DisabledFlags are omitted for the reasons
		// TestInMemoryProvider gives. DisabledFlags is worth revisiting:
		// dropping an error code on the hop is one of the failure modes this
		// suite exists to see, so declare it here as soon as the child can pass
		// it rather than leaving it withheld.
		//
		// The declared ones are each a delegation question. LargeIntegers: does
		// the int64 survive the hop. Variants: a dropped variant name is the
		// first thing a delegating provider loses. StandardReasons: a wrapper
		// that rewrote STATIC to DEFAULT, or lost ERROR on a type mismatch,
		// would be caught by reason.feature and by nothing else. StringTyping
		// and FullyTypedValues: the child's refusal to format a value as text
		// has to reach the caller as TYPE_MISMATCH rather than being swallowed,
		// and a float or a map is what a wrapper is most likely to flatten.
		// ConfigurationChange: Go's multi-provider forwards
		// PROVIDER_CONFIGURATION_CHANGED straight through.
		tck.WithCapabilities(
			tck.Events,
			tck.ConfigurationChange,
			tck.Object,
			tck.Variants,
			tck.LargeIntegers,
			tck.StandardReasons,
			tck.StringTyping,
			tck.FullyTypedValues,
		),
	)
}
