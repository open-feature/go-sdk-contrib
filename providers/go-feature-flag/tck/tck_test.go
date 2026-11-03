//go:build tck

package tck

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"testing"
	"time"

	gofeatureflag "github.com/open-feature/go-sdk-contrib/providers/go-feature-flag/pkg"
	"github.com/open-feature/go-sdk-contrib/tools/tck"
	"github.com/open-feature/go-sdk/openfeature"
)

// The OpenFeature Provider Conformance Suite against the GO Feature Flag provider, in both of its
// evaluation types: in-process pulls the flag configuration from the relay proxy and evaluates it
// with the GO Feature Flag core module, remote asks the relay proxy over OFREP. They are separate
// suites because they are separately conformant, and any difference between the two results is one
// an application would see when it switches evaluation type.
//
// Backend and findings: https://github.com/aepfli/goff-tck-testbed

const (
	composeFile = "testdata/tck/docker-compose.yaml"

	// The relay proxy's container-internal port, serving both evaluation types.
	relayProxyPort = 1031

	// A port on localhost that nothing listens on, for the initialisation-failure scenarios.
	// Deliberately not a port on the stack: the stack must stay up for the whole suite, and
	// simulated outages belong to the control API.
	unavailablePort = 9999

	// How often the provider polls the relay proxy for a changed configuration. Lowered from the
	// two-minute default, against which no event scenario can complete.
	flagChangePollingInterval = 500 * time.Millisecond
)

var (
	// The lossy half of @numeric-coercion. The lossless half passes, which is why the capability is
	// declared rather than withheld.
	floatNarrowingDeviation = tck.UntrackedDeviation(
		tck.NumericCoercion,
		"A float requested as an integer is narrowed silently: float-flag (0.5) through "+
			"IntEvaluation resolves to 0 with reason STATIC, variant \"half\" and no error code, "+
			"rather than TYPE_MISMATCH with the code default. The narrowing is the GO Feature Flag "+
			"core module's (evaluation.Evaluate converts any float64 with int(value) when the "+
			"expected type is \"int\"), and this provider's in-process evaluator calls it with "+
			"\"int\" and passes the result through. The Java provider evaluates the same flag "+
			"through the WASM build of the same engine untyped and type-checks the result itself, "+
			"so it reports TYPE_MISMATCH and passes. Remote mode passes too: the OFREP provider "+
			"round-trips the float through int64 and refuses the lossy case.")

	// @configuration-change is attempted and the change is applied; only the payload is missing.
	changedFlagsDeviation = tck.UntrackedDeviation(
		tck.ConfigurationChange,
		"PROVIDER_CONFIGURATION_CHANGED names no changed flags: the in-process evaluator emits it "+
			"with only a message (\"Configuration has changed\") and an empty FlagChanges, although "+
			"loadConfiguration has the old and the new flag map in hand when it swaps them. The "+
			"event arrives promptly and the new value is served, so the rest of the scenario "+
			"passes. The Java provider names the changed flags and passes.")
)

// TestGoFeatureFlagInProcessConformance runs the suite against in-process evaluation.
func TestGoFeatureFlagInProcessConformance(t *testing.T) {
	runConformance(t, conformanceSuite{
		name:           "go-feature-flag-in-process",
		evaluationType: gofeatureflag.EvaluationTypeInProcess,

		// Everything declarable, and every capability is attempted: Init fetches the configuration
		// and fails when it cannot, the poll emits PROVIDER_STALE on a failed refresh and
		// PROVIDER_READY when it recovers, and a changed ETag emits PROVIDER_CONFIGURATION_CHANGED.
		// @large-integers passes because large-integer-flag (2^53-1) survives float64.
		//
		// Two are declared and fail, see the deviations above.
		//
		// Not asserted by any scenario, measured separately: Init pushes a
		// PROVIDER_CONFIGURATION_CHANGED onto the event channel for the first load, before Init
		// returns. The SDK forwards the provider's channel on a different goroutine from the one
		// that emits READY, so a handler registered before the provider sees that event before
		// READY in some runs and after it in others.
		capabilities: tck.AllCapabilities(),

		knownDeviations: []tck.KnownDeviation{
			floatNarrowingDeviation,
			changedFlagsDeviation,
		},
	})
}

// TestGoFeatureFlagRemoteConformance runs the suite against remote evaluation.
func TestGoFeatureFlagRemoteConformance(t *testing.T) {
	runConformance(t, conformanceSuite{
		name:           "go-feature-flag-remote",
		evaluationType: gofeatureflag.EvaluationTypeRemote,

		// Remote evaluation is the OFREP provider underneath: each typed method calls
		// ofrep.Provider directly, and with the cache off nothing else ever reaches the backend.
		// Withheld, each because this mode does not attempt it, and each measured as failing when
		// declared:
		//
		//   tck.ConfigurationChange, tck.Stale -- the evaluator never writes to the event channel.
		//     The only poll belongs to the cache, and all it does on a changed ETag is purge it.
		//   tck.UnavailableInit -- Init returns nil at once when the cache is off, and with the
		//     cache on it ignores the result of its one request, so a provider pointed at a closed
		//     port reaches READY in under a millisecond and never settles into ERROR. Evaluations
		//     then report GENERAL with the code default, which is the evaluation half of the
		//     contract.
		//
		// @events and @lifecycle stay declared and pass, as they do for the Java remote mode: the
		// provider implements the state and event handlers, and the scenarios that remain hold.
		// Whether a remote mode should emit events and fail init at all is
		// open-feature/java-sdk-contrib#1948.
		//
		// Worth stating because Java differs: @disabled-flags passes (the Go OFREP provider checks
		// for reason DISABLED before it type-asserts, so the value-less answer resolves to the code
		// default), and so does @numeric-coercion (the OFREP provider widens 10 to a float and
		// refuses 0.5 as an integer with TYPE_MISMATCH).
		capabilities: allCapabilitiesExcept(
			tck.ConfigurationChange,
			tck.Stale,
			tck.UnavailableInit,
		),
	})
}

// allCapabilitiesExcept is tck.AllCapabilities with the withheld ones removed.
func allCapabilitiesExcept(withheld ...tck.Capability) []tck.Capability {
	var out []tck.Capability
	for _, c := range tck.AllCapabilities() {
		if !slices.Contains(withheld, c) {
			out = append(out, c)
		}
	}
	return out
}

// conformanceSuite is the per-evaluation-type configuration: everything the two suites differ in,
// and nothing else.
type conformanceSuite struct {
	name            string
	evaluationType  gofeatureflag.EvaluationType
	capabilities    []tck.Capability
	knownDeviations []tck.KnownDeviation
}

func runConformance(t *testing.T, suite conformanceSuite) {
	if testing.Short() {
		t.Skip("skipping e2e tests in short mode")
	}

	newProvider := func(endpoint string) (openfeature.FeatureProvider, error) {
		return gofeatureflag.NewProvider(gofeatureflag.ProviderOptions{
			Endpoint:       endpoint,
			EvaluationType: suite.evaluationType,
			// Well under the TCK's own step deadlines, so that a wedged backend surfaces as a
			// resolution error attributable to this provider rather than as a suite timeout.
			HTTPClient:                &http.Client{Timeout: 5 * time.Second},
			FlagChangePollingInterval: flagChangePollingInterval,
			// Remote only. Every OFREP response from the relay proxy says
			// gofeatureflag_cacheable:true, and a cache hit rewrites the reason to CACHED, so with
			// the cache on the suite would be asserting this client's cache rather than the
			// resolution. @caching is reserved; nothing here can be said about the cache.
			DisableCache: true,
			// The data collector is not part of the provider contract, and its background flushes
			// would only add traffic to the backend under test.
			DataCollectorDisabled: true,
		})
	}

	tck.Run(t,
		tck.WithName(suite.name),

		tck.WithComposeFile(composeFile),
		tck.WithBackendPorts(relayProxyPort),

		tck.WithProviderFromEndpoint(func(_ context.Context, e tck.BackendEndpoint) (openfeature.FeatureProvider, error) {
			// e.Host() rather than a hard-coded "localhost": with a remote Docker daemon, Docker
			// Desktop on some platforms or a rootless setup the host is not localhost.
			return newProvider(fmt.Sprintf("http://%s:%d", e.Host(), e.Port(relayProxyPort)))
		}),

		tck.WithUnavailableProvider(func(context.Context) (openfeature.FeatureProvider, error) {
			return newProvider(fmt.Sprintf("http://localhost:%d", unavailablePort))
		}),

		tck.WithCapabilities(suite.capabilities...),
		tck.WithKnownDeviations(suite.knownDeviations...),

		tck.WithReadyTimeout(30*time.Second),
		tck.WithEventTimeout(15*time.Second),
	)
}
