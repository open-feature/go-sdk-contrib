//go:build e2e

package e2e

import (
	"context"
	"fmt"
	"testing"
	"time"

	flagsmithClient "github.com/Flagsmith/flagsmith-go-client/v5"
	flagsmith "github.com/open-feature/go-sdk-contrib/providers/flagsmith/pkg"
	"github.com/open-feature/go-sdk-contrib/tools/tck"
	"github.com/open-feature/go-sdk/openfeature"
)

// The OpenFeature Provider Conformance Suite, run against the Flagsmith provider in both of its
// evaluation modes.
//
// Flagsmith resolves flags two ways and they are separate suites because they are separately
// conformant. Remote evaluation calls the backend's /api/v1/flags/ and the *backend* evaluates;
// local evaluation fetches the environment document and the *SDK* evaluates in-process.
//
// That split is worth more here than it looks. Flagsmith's evaluation engine is independently
// reimplemented per language -- Python inside the backend, Go in flagsmith-go-client/flagengine --
// so running both modes against a byte-identical environment document compares two implementations
// of the same engine directly. It is the same shape as GO Feature Flag's one engine in several
// hosts, except these are separate reimplementations, which makes divergence more likely rather
// than less. They do not diverge.
//
// The backend is the Flagsmith Edge Proxy driven by a launchpad implementing the control API:
// https://github.com/aepfli/flagsmith-tck-testbed

const (
	// Fixed by the testbed. The control API has no way to communicate connection parameters --
	// POST /start returns a bare 200 with no body -- so every adoption hardcodes these, exactly as
	// a flagd adoption hardcodes a port. Every SaaS-shaped backend needs something like it, which
	// makes it a gap in Appendix F rather than a quirk of Flagsmith.
	serverSideKey = "ser.provider-tck-server-key"

	composeFile = "testbed-compose.yaml"
	proxyPort   = 8000
)

// Deviations, as opposed to capabilities the provider simply does not implement.
//
// Narrowing the capability list says a scenario did not run; it cannot say whether the provider
// declines the capability or fails at it. Both appear as the same skip carrying the same reason, so
// without an entry here a consumer comparing providers reads a defect as a design choice.
var (
	// Withholding @numeric-coercion is a defect, not a design choice, and it is not the usual one.
	// The familiar failure is a provider narrowing 0.5 to 0 -- that code exists here too,
	// int64(value) with no fractional check -- but it is unreachable, because Flagsmith has no
	// float type and the backend cannot produce a fractional JSON number.
	//
	// What actually breaks is cruder. The two numeric accessors disagree about the wire type:
	// IntEvaluation asserts res.Value.(float64) and wants a JSON number, while FloatEvaluation
	// asserts res.Value.(string) and calls ParseFloat, commented "Because We store floats as
	// string". So for a Flagsmith integer feature -- which is what the backend natively stores --
	// GetIntValue succeeds and GetFloatValue returns TYPE_MISMATCH. No seeding of the canonical set
	// satisfies both accessors, so the lossless integer-to-float scenario cannot pass however the
	// flag is seeded.
	numericCoercionDeviation = tck.UntrackedDeviation(
		tck.NumericCoercion,
		"The Int and Float accessors disagree about the wire type: IntEvaluation asserts "+
			"float64 (a JSON number) while FloatEvaluation asserts string and calls ParseFloat, "+
			"so a Flagsmith integer feature resolves through GetIntValue and returns "+
			"TYPE_MISMATCH through GetFloatValue. No seeding of the canonical set satisfies both. "+
			"Separately and less importantly, IntEvaluation ends in int64(value) with no "+
			"fractional check -- the usual narrowing defect -- but Flagsmith has no float type so "+
			"that branch is unreachable through this backend. Both modes are affected: the "+
			"accessors are in the shared provider layer, above the transport.")

	// @object IS declared and mostly holds -- structured values resolve, and three of the four
	// structured-as-scalar rows report TYPE_MISMATCH correctly. This entry exists for the fourth.
	//
	// A deviation against a *declared* capability is the case the docs call out: the capability
	// holds, but one of the scenarios it gates does not. Without it the suite reports a bare
	// failure and a reader cannot tell an unimplemented feature from a backend that cannot express
	// the distinction.
	//
	// Withholding @object instead would be worse. Object resolution genuinely works, and dropping
	// the capability would skip five scenarios to hide one failure -- trading a visible, explained
	// defect for four silent non-results.
	objectAsStringDeviation = tck.UntrackedDeviation(
		tck.Object,
		"object-flag requested as a String resolves to the raw JSON text rather than reporting "+
			"TYPE_MISMATCH. Flagsmith stores an object as a string -- feature_state_value is "+
			"natively boolean, integer or string only -- so on this backend the request is not a "+
			"type mismatch at all and correctly succeeds. The other three rows of the outline "+
			"(Boolean, Integer, Float) pass, and structured resolution itself passes. The same "+
			"cause fails one untagged row, float-flag requested as a String, which has no "+
			"capability to hang a deviation on: whether the type-mismatch matrix is satisfiable "+
			"against a backend with a coarser type system is an open question for the suite, not "+
			"a defect this provider can fix.")
)

// TestFlagsmithRemoteConformance runs the suite against remote evaluation: the provider calls
// /api/v1/flags/ and the backend's engine evaluates.
func TestFlagsmithRemoteConformance(t *testing.T) {
	runConformance(t, "flagsmith-remote", false)
}

// TestFlagsmithLocalConformance runs the suite against local evaluation: the provider fetches the
// environment document and evaluates in-process with the Go engine.
func TestFlagsmithLocalConformance(t *testing.T) {
	runConformance(t, "flagsmith-local", true)
}

func runConformance(t *testing.T, name string, local bool) {
	if testing.Short() {
		t.Skip("skipping e2e tests in short mode")
	}

	// The suite's own context, not a scenario's. WithLocalEvaluation binds its polling goroutine to
	// whatever context it is given, and a scenario-scoped one would cancel the poll the moment the
	// scenario that created the provider ended -- after which every flag resolves to its code
	// default with GENERAL, which reads exactly like a broken backend.
	suiteCtx := context.Background()

	tck.Run(t,
		tck.WithName(name),

		// The TCK owns the stack: it brings Compose up, waits for the control API, resolves the
		// mapped host ports and tears down. Nothing here touches testcontainers.
		tck.WithComposeFile(composeFile),
		tck.WithBackendPorts(proxyPort),
		tck.WithStartupTimeout(90*time.Second),

		tck.WithProviderFromEndpoint(func(_ context.Context, e tck.BackendEndpoint) (openfeature.FeatureProvider, error) {
			// The Flagsmith SDK appends its own path segments, so baseURL is the API root with a
			// trailing slash: remote evaluation requests "flags/" beneath it, local evaluation
			// "environment-document/".
			baseURL := fmt.Sprintf("http://%s:%d/api/v1/", e.Host(), e.Port(proxyPort))

			options := []flagsmithClient.Option{flagsmithClient.WithBaseURL(baseURL)}
			if local {
				// Local evaluation polls the environment document. The default refresh interval is
				// 60 seconds, which would make every scenario following a POST /change time out;
				// the testbed's own upstream poll is already 1s, so this is the second of two hops.
				options = append(options,
					flagsmithClient.WithLocalEvaluation(suiteCtx),
					flagsmithClient.WithEnvironmentRefreshInterval(time.Second),
				)
			}

			provider := flagsmith.NewProvider(flagsmithClient.NewClient(serverSideKey, options...))
			if !local {
				return provider, nil
			}

			// Wait for the first environment sync before handing the provider back.
			//
			// This compensates for a real defect rather than a quirk of the harness. The provider
			// implements no openfeature.StateHandler, so it has no Init for the TCK to call and no
			// way to report that it is not ready. The SDK therefore synthesises PROVIDER_READY on
			// registration while the client's first poll is still in flight, and evaluations in
			// that window return the code default with GENERAL and the message "local environment
			// has not yet been updated".
			//
			// That is the flagd#2047 shape moved into the provider: something reports ready before
			// it can serve a flag. A conformant provider would block in Init. This adoption cannot
			// fix it, so it waits here -- without which the suite fails non-deterministically,
			// which would look like flakiness rather than like the defect it is.
			if err := waitForLocalSync(provider); err != nil {
				return nil, err
			}
			return provider, nil
		}),

		// What is NOT declared, and why. Most of it comes down to one fact: the provider implements
		// none of Init, Shutdown, Status or EventChannel, so it is neither an
		// openfeature.StateHandler nor an openfeature.EventHandler.
		//
		//   - @lifecycle, @events, @stale, @configuration-change. A provider with no observable
		//     initialisation has no lifecycle to assert against: the SDK synthesises
		//     PROVIDER_READY on registration, so declaring @lifecycle would make those scenarios
		//     pass without the provider having done anything. A vacuous pass is worse than a skip.
		//   - @unavailable for the same reason. The scenario asserts that a provider unable to
		//     reach its backend settles into ERROR and emits PROVIDER_ERROR. This provider cannot
		//     fail initialisation because it has no initialisation, so it reports READY against a
		//     dead backend. No unavailable-provider factory is supplied, and those scenarios skip.
		//   - @reinitialization follows from the same absence, and 2.5.2 only says a provider
		//     SHOULD revert to its uninitialized state anyway.
		//
		// None of these get a known-deviation entry: the specification does not require a provider
		// to implement StateHandler or EventHandler, so declining them is an option the contract
		// offers rather than a defect. The consequence worth stating is that POST /change,
		// /restart and /reset are implemented by the testbed and observed by nothing here.
		//
		// @variants is withheld for a different reason. Flagsmith has no variant concept for a
		// plain feature: a feature state is `enabled` plus `feature_state_value`, nothing names the
		// value, and the evaluation response carries no variant key at all. The provider never
		// receives one and no seeding can produce one -- permitted rather than defective, since
		// 2.2.4 makes the variant a SHOULD and types.md marks the field optional.
		//
		// @disabled-flags IS declared. Flagsmith's native model is `enabled` plus a value, so the
		// canonical set's four disabled-* flags map directly onto it, and this provider returns the
		// caller's default with reason DISABLED, which is what the tag asserts.
		//
		// @object holds -- the provider json.Unmarshals the JSON string Flagsmith stores an object
		// as. @large-integers holds: ResolveIntValue is int64, and 2^53-1 survives the JSON number
		// -> float64 -> int64 trip exactly. @targeting holds via identity overrides: a targeting
		// key IS a Flagsmith identifier, since the provider calls GetIdentityFlags(targetingKey).
		tck.WithCapabilities(
			tck.Object,
			tck.LargeIntegers,
			tck.Targeting,
			tck.DisabledFlags,
		),

		tck.WithKnownDeviations(numericCoercionDeviation, objectAsStringDeviation),

		// Both modes poll. Remote is a single hop; local waits on two -- the testbed's upstream
		// poll and the SDK's environment refresh, each 1s -- and the scenarios are shared.
		tck.WithEventTimeout(15*time.Second),
		tck.WithReadyTimeout(30*time.Second),
	)
}

// waitForLocalSync blocks until the provider can resolve a flag from the canonical set, or gives up.
//
// It resolves boolean-flag directly through the provider, before the TCK has registered it with the
// OpenFeature API, so it observes the client's own readiness rather than anything the SDK
// synthesises.
func waitForLocalSync(provider openfeature.FeatureProvider) error {
	const (
		timeout  = 20 * time.Second
		interval = 50 * time.Millisecond
	)
	deadline := time.Now().Add(timeout)
	var last openfeature.ResolutionError
	for time.Now().Before(deadline) {
		res := provider.BooleanEvaluation(context.Background(), "boolean-flag", false, nil)
		if res.Error() == nil && res.Value {
			return nil
		}
		last = res.ResolutionError
		time.Sleep(interval)
	}
	return fmt.Errorf("the local environment was not synced within %s: %w", timeout, last)
}
