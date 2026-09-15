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
// Two suites because the modes are separately conformant: remote evaluation calls the backend's
// /api/v1/flags/ and the backend evaluates, while local evaluation fetches the environment document
// and this SDK evaluates in-process. Why that comparison is worth making, and what it found, is in
// the testbed's FINDINGS.md -- the backend and its findings are documented there rather than
// restated in each of the four language adoptions:
// https://github.com/aepfli/flagsmith-tck-testbed

const (
	// Fixed by the testbed, because the control API has no way to hand connection parameters to a
	// provider -- hardcoded here exactly as a flagd adoption hardcodes a port. FINDINGS #1.
	serverSideKey = "ser.provider-tck-server-key"

	composeFile = "testdata/tck/docker-compose.yaml"
	proxyPort   = 8000
)

// Known deviations. What the field means and when to use which shape is tck.KnownDeviation's own
// documentation; what follows is only which gaps this provider has. The summaries are deliberately
// self-contained, because they travel into a conformance report read across languages.
var (
	// Declared and failing: this provider attempts the coercion and gets it wrong.
	//
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

	// Two entries stood here until spec d47a66eb, objectAsStringDeviation and
	// floatAsStringDeviation, and both are gone rather than rewritten.
	//
	// They recorded the same fact twice: object-flag and float-flag requested as Strings come back
	// as their text rather than as TYPE_MISMATCH, because Flagsmith stores both as strings. Each
	// said in its own summary that this was not a defect the provider could fix and that whether
	// the matrix was satisfiable against a coarser type system was an open question for the suite.
	// The suite has now answered it: those rows live behind @string-typing, which this adoption
	// withholds, so they are skipped with a reason instead of failed with an excuse. A deviation
	// asserts the provider fails something it is *required* to do, and after d47a66eb nothing
	// requires this -- so keeping either entry would be the misattribution the field exists to
	// prevent. See the withholding note beside tck.WithCapabilities.
	//
	// Retiring them also cleans @object up: its remaining three rows (Boolean, Integer, Float)
	// pass, structured resolution passes, and the capability now has no failing scenario at all.

	// The defect the testbed recorded as FINDINGS #4 long before anything could catch it:
	// @standard-reasons is what closes the blind spot, and this is the scenario that finds it.
	targetingMatchDeviation = tck.UntrackedDeviation(
		tck.StandardReasons,
		"A targeting rule that does not match still reports TARGETING_MATCH rather than DEFAULT. "+
			"resolveFlag sets the reason from the mere presence of a targeting key in the "+
			"evaluation context, before evaluating anything, and never revises it -- so the "+
			"provider tells the caller a rule matched when none did. The other scenarios this "+
			"capability gates pass: STATIC for an untargeted resolution, DISABLED for a disabled "+
			"flag, and TARGETING_MATCH for a genuine match. The capability is declared rather than "+
			"withheld so that those keep running and this one failure stays visible with its "+
			"reason attached.")
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
		// @standard-reasons IS declared, and one of its scenarios fails -- see
		// targetingMatchDeviation. Declaring it is still right: this provider reports STATIC for an
		// untargeted resolution, DISABLED for a disabled flag and TARGETING_MATCH for a genuine
		// match, so withholding would skip several scenarios it satisfies in order to hide one it
		// does not. The Java and JavaScript Flagsmith providers are in a different position
		// entirely -- Java leaves the reason null and JavaScript reports TARGETING_MATCH for every
		// enabled flag -- which is the difference this capability exists to make visible.
		//
		// @string-typing is WITHHELD, and this adoption is the reason the capability exists.
		// Flagsmith's feature_state_value is natively boolean, integer or string -- no float type
		// and no object type -- so a float and a structure are both stored as text, and asking for
		// either through the string accessor is a correct request that correctly succeeds. There is
		// no mismatch to report and no code here that could report one.
		//
		// Measured both ways rather than argued. Declared: 16 capability-skips and 4 failures, the
		// two extra being float-flag resolving to "0.5" and object-flag to its raw JSON text.
		// Withheld: 20 skips and 2 failures. Identical in both evaluation modes.
		//
		// **The provider answers two of the four scenarios and the tag cannot say so.** boolean-flag
		// and integer-flag requested as Strings both report TYPE_MISMATCH, because those two types
		// really are native to feature_state_value -- so Flagsmith is partially typed, not untyped,
		// and withholding gives up two passes it has honestly earned. That is a granularity cost in
		// the capability rather than a fact about this provider: the three scalar rows are one
		// Examples table, so there is no way to claim the two that hold. It is withheld anyway,
		// because the alternative is worse -- declaring means two permanently failing scenarios
		// carrying known-deviation entries for behaviour no numbered requirement asks for, which is
		// exactly the misattribution Appendix F's rules for declaring forbid. The old
		// objectAsStringDeviation and floatAsStringDeviation were that shape, and they are deleted
		// above.
		//
		// Worth revisiting if the capability is ever split so that the scalar rows can be answered
		// separately from the ones a text-valued backend cannot answer.
		tck.WithCapabilities(
			tck.Object,
			tck.LargeIntegers,
			tck.Targeting,
			tck.DisabledFlags,
			tck.StandardReasons,
			tck.NumericCoercion,
		),

		tck.WithKnownDeviations(
			numericCoercionDeviation,
			targetingMatchDeviation,
		),

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
