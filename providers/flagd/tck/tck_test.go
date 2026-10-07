//go:build tck

package tck

import (
	"context"
	"testing"
	"time"

	flagd "github.com/open-feature/go-sdk-contrib/providers/flagd/pkg"
	"github.com/open-feature/go-sdk-contrib/tools/tck"
	"github.com/open-feature/go-sdk/openfeature"
)

// The OpenFeature Provider Conformance Suite, run against the flagd provider in
// both of its resolver modes: RPC evaluates remotely over gRPC, in-process syncs
// the ruleset and evaluates locally. They are separate suites because they are
// separately conformant, and any difference between the two results is one an
// application would see when it switches resolver.
//
// README.md has what each resolver declares, the known deviation and the tally
// to read a red run against.

const (
	// Shared with the other conformance adoptions in this repository so that
	// the image tag cannot drift between suites whose results are only
	// comparable if both answered the same backend. Resolved relative to this
	// package directory, which is where `go test` runs.
	composeFile = "../../../tests/flagd-testbed/docker-compose.yaml"

	// The container-internal ports the two resolvers connect to.
	rpcPort       = 8013
	inProcessPort = 8015

	// A port on localhost that nothing listens on, for the
	// initialisation-failure scenarios. Deliberately not a port on the stack:
	// the stack must stay up for the whole suite, and simulated outages belong
	// to the control API.
	unavailablePort = 9999
)

var (
	// Shared by both suites: the two resolvers coerce identically, so the
	// defect is in the provider layer above either transport.
	numericCoercionDeviation = tck.TrackedDeviation(
		tck.NumericCoercion,
		"https://github.com/open-feature/flagd/issues/1996",
		"The lossy half of the coercion rule is not enforced: evaluating float-flag (0.5) "+
			"through GetIntDetails returns 0 with no error code, rather than TYPE_MISMATCH with "+
			"the code default, so the fractional part is discarded silently. The lossless half "+
			"works and is not the defect: integer-flag (10) requested as a float is widened to 10 "+
			"with reason STATIC and no error code, which is why the capability is declared rather "+
			"than withheld. Both resolvers behave identically in both directions. One further "+
			"@numeric-coercion scenario fails here for a reason that is NOT the provider's: it asks "+
			"for integral-float-flag (10.0) as an integer, and that flag is absent from every "+
			"released flagd-testbed, so it fails with FLAG_NOT_FOUND "+
			"(open-feature/flagd-testbed#392).")
)

// TestFlagdRPCConformance runs the suite against the RPC resolver.
func TestFlagdRPCConformance(t *testing.T) {
	runConformance(t, conformanceSuite{
		name:        "flagd-rpc",
		backendPort: rpcPort,
		resolver:    flagd.WithRPCResolver(),

		// Withheld here, each measured rather than inferred:
		//
		//   tck.Stale -- this resolver sends PROVIDER_ERROR on connection loss
		//     and never emits PROVIDER_STALE, so the @stale scenario times out
		//     waiting for one. The in-process resolver does emit it, which is
		//     the one behavioural difference between the two resolvers.
		//   tck.Reinitialization -- Shutdown clears the provider's initialised
		//     flag so a second Init proceeds, but the event stream never
		//     signals ready again and Init returns "provider initialization
		//     deadline exceeded".
		//   tck.LargeIntegers -- large-integer-flag and huge-integer-flag are
		//     absent from the pinned testbed image, so the capability cannot be
		//     verified against this backend at all (flagd-testbed#392). Nothing
		//     here is the provider's: Go's ResolveIntValue is int64 and has
		//     room for both values.
		//
		// tck.NumericCoercion is declared with a known deviation below.
		// README.md has the rest of the table.
		capabilities: []tck.Capability{
			tck.Events,
			tck.Lifecycle,
			tck.ConfigurationChange,
			tck.Object,
			tck.NumericCoercion,
			tck.Variants,
			tck.DisabledFlags,
			tck.Targeting,
			tck.StandardReasons,
			tck.UnavailableInit,
			tck.StringTyping,
			tck.FullyTypedValues,
		},

		knownDeviations: []tck.KnownDeviation{
			numericCoercionDeviation,
		},

		// The RPC resolver asks flagd to resolve each flag, so it is ready as
		// soon as the stream is up.
		readyTimeout: 30 * time.Second,
		gracePeriod:  10,
	})
}

// TestFlagdInProcessConformance runs the suite against the in-process resolver.
func TestFlagdInProcessConformance(t *testing.T) {
	runConformance(t, conformanceSuite{
		name:        "flagd-in-process",
		backendPort: inProcessPort,
		resolver:    flagd.WithInProcessResolver(),

		// As RPC, except that tck.Stale IS declared: this resolver emits
		// PROVIDER_STALE on connection loss and only escalates to
		// PROVIDER_ERROR once the retry grace period expires.
		//
		// tck.Reinitialization and tck.LargeIntegers are withheld for the
		// reasons the RPC suite gives, and tck.NumericCoercion coerces
		// identically in both directions, which is why one deviation covers
		// both suites.
		capabilities: []tck.Capability{
			tck.Events,
			tck.Lifecycle,
			tck.Stale,
			tck.ConfigurationChange,
			tck.Object,
			tck.NumericCoercion,
			tck.Variants,
			tck.DisabledFlags,
			tck.Targeting,
			tck.StandardReasons,
			tck.UnavailableInit,
			tck.StringTyping,
			tck.FullyTypedValues,
		},

		knownDeviations: []tck.KnownDeviation{
			numericCoercionDeviation,
		},

		// In-process syncs the whole ruleset before reporting ready, so it
		// needs longer than RPC to initialise.
		readyTimeout: 60 * time.Second,

		// The grace period has to outlast the outage in the @stale scenario.
		// The resolver goes STALE immediately on connection loss and escalates
		// to ERROR when this expires, so too short a value would turn a
		// scenario about staleness into one about failure.
		gracePeriod: 30,
	})
}

// conformanceSuite is the per-resolver configuration: everything the two
// suites differ in, and nothing else.
type conformanceSuite struct {
	name            string
	backendPort     int
	resolver        flagd.ProviderOption
	capabilities    []tck.Capability
	knownDeviations []tck.KnownDeviation
	readyTimeout    time.Duration
	gracePeriod     int
}

func runConformance(t *testing.T, suite conformanceSuite) {
	if testing.Short() {
		t.Skip("skipping e2e tests in short mode")
	}

	tck.Run(t,
		tck.WithName(suite.name),

		tck.WithComposeFile(composeFile),
		tck.WithBackendPorts(suite.backendPort),

		tck.WithProviderFromEndpoint(func(_ context.Context, endpoint tck.BackendEndpoint) (openfeature.FeatureProvider, error) {
			provider, err := flagd.NewProvider(
				suite.resolver,
				flagd.WithHost(endpoint.Host()),
				flagd.WithPort(uint16(endpoint.Port(suite.backendPort))),
				flagd.WithDeadline(1000),
				flagd.WithRetryGracePeriod(suite.gracePeriod),
				flagd.WithRetryBackoffMs(500),
			)
			if err != nil {
				return nil, err
			}
			return provider, nil
		}),

		// Pointed at a closed port on localhost, never at the backend under
		// test — that has to stay up, and simulated outages belong to the
		// control API. The deadlines are deliberately short: the scenario
		// asserts that failure is reported promptly, so a provider that took
		// 30 seconds to give up would pass a test about eventual failure and
		// fail the one that matters.
		tck.WithUnavailableProvider(func(context.Context) (openfeature.FeatureProvider, error) {
			provider, err := flagd.NewProvider(
				suite.resolver,
				flagd.WithHost("localhost"),
				flagd.WithPort(unavailablePort),
				flagd.WithDeadline(500),
				flagd.WithRetryGracePeriod(1),
				flagd.WithRetryBackoffMs(100),
			)
			if err != nil {
				return nil, err
			}
			return provider, nil
		}),

		tck.WithCapabilities(suite.capabilities...),
		tck.WithKnownDeviations(suite.knownDeviations...),

		tck.WithReadyTimeout(suite.readyTimeout),
		tck.WithEventTimeout(15*time.Second),
	)
}
