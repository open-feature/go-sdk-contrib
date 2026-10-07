//go:build tck

package tck

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/open-feature/go-sdk-contrib/providers/ofrep"
	"github.com/open-feature/go-sdk-contrib/tools/tck"
	"github.com/open-feature/go-sdk/openfeature"
)

// The OpenFeature Provider Conformance Suite, run against the OFREP provider.
// The backend is the unmodified flagd-testbed image, which serves the OFREP API
// on container port 8016 alongside its own protocols and the launchpad control
// API.
//
// **This suite is currently non-deterministic**: the launchpad's POST /start
// returns before the flags are evaluable (open-feature/flagd-testbed#394) and
// this provider has no initialisation to block on, so it races that load on
// every scenario. No sleep or retry is being added here to hide it. README.md
// has the numbers and the floor to read a red run against.

const (
	// Shared with the other conformance adoptions in this repository so that
	// the image tag cannot drift between suites whose results are only
	// comparable if both answered the same backend. Resolved relative to this
	// package directory, which is where `go test` runs.
	composeFile = "../../../tests/flagd-testbed/docker-compose.yaml"

	// The container-internal port flagd serves OFREP on, and the only port the
	// provider connects to.
	ofrepPort = 8016
)

// TestOFREPConformance runs the suite against the OFREP provider pointed at
// flagd's OFREP endpoint.
func TestOFREPConformance(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e tests in short mode")
	}

	tck.Run(t,
		tck.WithName("ofrep"),

		tck.WithComposeFile(composeFile),
		tck.WithBackendPorts(ofrepPort),

		tck.WithProviderFromEndpoint(func(_ context.Context, endpoint tck.BackendEndpoint) (openfeature.FeatureProvider, error) {
			// endpoint.Host() rather than a hard-coded "localhost": with a
			// remote Docker daemon, Docker Desktop on some platforms or a
			// rootless setup the host is not localhost.
			//
			// The timeout is well under the TCK's own step deadlines so that a
			// wedged backend surfaces as a resolution error attributable to
			// this provider rather than as a suite-level timeout.
			baseURI := fmt.Sprintf("http://%s:%d", endpoint.Host(), endpoint.Port(ofrepPort))
			return ofrep.NewProvider(baseURI, ofrep.WithTimeout(5*time.Second)), nil
		}),

		// tck.WithUnavailableProvider is deliberately absent: this provider
		// cannot declare tck.UnavailableInit. See below.

		// Every omission below is a property of the provider's code rather than
		// a preference. The OFREP provider is stateless -- Metadata, the five
		// typed *Evaluation methods and Hooks, and neither
		// openfeature.EventHandler nor openfeature.StateHandler -- so:
		//
		//   tck.Events, tck.ConfigurationChange, tck.Stale -- with no
		//     EventChannel the provider can never publish a provider event.
		//     Declaring tck.Events would buy one green scenario that asserts
		//     nothing: "becomes ready" would pass on the READY the SDK
		//     synthesises for a provider with no StateHandler, which is emitted
		//     identically against a backend that does not exist.
		//   tck.Lifecycle, tck.UnavailableInit -- there is no initialisation to
		//     fail, so a provider pointed at a closed port sits in READY rather
		//     than settling into ERROR. The evaluation half of the contract is
		//     honoured -- an unreachable host yields a GENERAL resolution error
		//     and the code default -- but the state half cannot be, so these
		//     are withheld rather than half-met.
		//   tck.LargeIntegers -- large-integer-flag is absent from
		//     flagd-testbed, so nothing about this provider can be established
		//     (open-feature/flagd-testbed#392).
		//
		// Of the declared ones, tck.DisabledFlags is worth naming: each typed
		// resolver in internal/evaluate/flags.go checks the DISABLED reason
		// BEFORE it type-asserts, which is what lets a response carrying no
		// value member resolve to the caller's default rather than to
		// TYPE_MISMATCH. README.md has the rest of the table.
		tck.WithCapabilities(
			tck.Object,
			tck.NumericCoercion,
			tck.Variants,
			tck.Targeting,
			tck.DisabledFlags,
			tck.StandardReasons,
			tck.StringTyping,
			tck.FullyTypedValues,
		),

		// The provider has no initialisation to wait for, so this bounds the
		// SDK's registration round trip and nothing else.
		tck.WithReadyTimeout(30*time.Second),

		// Unused in practice — every event-bearing scenario is gated behind
		// tck.Events — but set explicitly so the value does not silently change
		// meaning if the provider grows eventing.
		tck.WithEventTimeout(15*time.Second),
	)
}
