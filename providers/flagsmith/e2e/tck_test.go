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

// The OpenFeature Provider Conformance Suite against the Flagsmith provider, in both evaluation
// modes: remote, where the backend evaluates, and local, where this SDK does.
//
// Backend and findings: https://github.com/aepfli/flagsmith-tck-testbed

const (
	// Fixed by the testbed; the control API cannot hand connection parameters to a provider.
	serverSideKey = "ser.provider-tck-server-key"

	composeFile = "testdata/tck/docker-compose.yaml"
	proxyPort   = 8000
)

// Deviations. Summaries stay self-contained: they travel into a cross-language report.
var (
	numericCoercionDeviation = tck.UntrackedDeviation(
		tck.NumericCoercion,
		"The Int and Float accessors disagree about the wire type: IntEvaluation asserts "+
			"float64 (a JSON number) while FloatEvaluation asserts string and calls ParseFloat, "+
			"so a Flagsmith integer feature resolves through GetIntValue and returns "+
			"TYPE_MISMATCH through GetFloatValue. No seeding of the canonical set satisfies both. "+
			"Affects both evaluation modes: the accessors sit above the transport.")

	targetingMatchDeviation = tck.UntrackedDeviation(
		tck.StandardReasons,
		"A targeting rule that does not match still reports TARGETING_MATCH rather than DEFAULT. "+
			"resolveFlag sets the reason from the mere presence of a targeting key in the "+
			"evaluation context, before evaluating anything, and never revises it -- so the "+
			"provider tells the caller a rule matched when none did. The capability's other "+
			"scenarios pass: STATIC untargeted, DISABLED when disabled, TARGETING_MATCH on a real "+
			"match.")
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

	// The suite's context, not a scenario's: WithLocalEvaluation binds its poll goroutine to
	// whatever it is given, and a scenario-scoped one dies with the scenario that built the provider.
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
			// Compensates for a provider defect, not a harness quirk: with no StateHandler there is
			// no Init to block in, so the SDK reports READY while the first poll is still in flight
			// and evaluations in that window return the code default with GENERAL.
			if err := waitForLocalSync(provider); err != nil {
				return nil, err
			}
			return provider, nil
		}),

		// Withheld, all permitted rather than defective, so none carries a deviation:
		//   @lifecycle @events @stale @configuration-change @unavailable @reinitialization --
		//     the provider implements no StateHandler or EventHandler, so there is no lifecycle to
		//     assert against. A consequence worth knowing: /change, /restart and /reset are
		//     implemented by the testbed and observed by nothing here.
		//   @variants -- Flagsmith names no value; a feature state is `enabled` plus a value.
		//   @fully-typed-values -- feature_state_value is natively boolean, integer or string, so
		//     float-flag and object-flag are stored as text and the string accessor answers them
		//     correctly. Nothing to mismatch.
		//
		// @string-typing IS declared: boolean and integer are native, and both report TYPE_MISMATCH
		// through the string accessor. Flagsmith is partially typed, which is why the tag is split.
		//
		// @numeric-coercion and @standard-reasons are declared and each fails one scenario; see the
		// deviations above.
		tck.WithCapabilities(
			tck.Object,
			tck.LargeIntegers,
			tck.Targeting,
			tck.DisabledFlags,
			tck.StandardReasons,
			tck.NumericCoercion,
			tck.StringTyping,
		),

		tck.WithKnownDeviations(
			numericCoercionDeviation,
			targetingMatchDeviation,
		),

		// Local waits on two 1s polls: the testbed's upstream and the SDK's refresh.
		tck.WithEventTimeout(15*time.Second),
		tck.WithReadyTimeout(30*time.Second),
	)
}

// waitForLocalSync blocks until the provider can resolve a canonical flag. It goes through the
// provider directly, before registration, so it sees the client's readiness and not the SDK's.
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
