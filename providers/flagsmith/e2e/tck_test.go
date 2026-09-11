//go:build e2e

package e2e

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	flagsmithClient "github.com/Flagsmith/flagsmith-go-client/v5"
	flagsmith "github.com/open-feature/go-sdk-contrib/providers/flagsmith/pkg"
	"github.com/open-feature/go-sdk-contrib/tools/provider-tck/pkg/tck"
	"github.com/open-feature/go-sdk/openfeature"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// The OpenFeature Provider Conformance Suite, run against the Flagsmith
// provider in both of its evaluation modes.
//
// Flagsmith resolves flags two ways and they are separate suites because they
// are separately conformant. Remote evaluation calls the backend's
// /api/v1/flags/ and the *backend* evaluates; local evaluation fetches the
// environment document and the *SDK* evaluates in-process. Any difference
// between the two results is a difference an application would see when it
// switches mode.
//
// That split is worth more here than it looks. Flagsmith's evaluation engine is
// independently reimplemented per language -- Python inside the backend, Go in
// flagsmith-go-client/flagengine -- so running both modes against a
// byte-identical environment document compares two implementations of the same
// engine directly. It is the same shape as GO Feature Flag's one engine in
// several hosts, except these are separate reimplementations, which makes
// divergence more likely rather than less.
//
// The backend is the Flagsmith Edge Proxy, driven by a launchpad implementing
// the control API. See https://github.com/aepfli/flagsmith-tck-testbed.

const (
	// testbedImage is the backend under test.
	//
	// This is a prototype in a personal namespace, which is why the PR opening
	// this adoption is a draft: a contrib repo's CI should not depend on it
	// until it has a permanent home. Overridable so a reviewer can point at
	// their own build.
	defaultTestbedImage = "ghcr.io/aepfli/flagsmith-tck-testbed:latest"

	// The control API has no way to communicate connection parameters -- /start
	// returns a bare 200 with no body -- so these are fixed by the testbed and
	// hardcoded here, exactly as a flagd adoption hardcodes a port. Every
	// SaaS-shaped backend will need something like this, which makes it a gap
	// in Appendix F rather than a quirk of Flagsmith.
	serverSideKey = "ser.provider-tck-server-key"

	proxyPort   = "8000/tcp"
	controlPort = "8080/tcp"
)

// Deviations, as opposed to capabilities the provider simply does not
// implement.
//
// Narrowing the capability list says a scenario did not run; it cannot say
// whether the provider declines the capability or fails at it. Both appear as
// the same skip carrying the same reason, so without an entry here a consumer
// comparing providers reads a defect as a design choice.
var (
	// Withholding @numeric-coercion here is a defect, not a design choice, and
	// it is not the usual one. The familiar failure is a provider narrowing 0.5
	// to 0 -- that code exists in this provider too, `int64(value)` with no
	// fractional check -- but it is unreachable, because Flagsmith has no float
	// type and the backend cannot produce a fractional JSON number.
	//
	// What actually breaks is cruder. The two numeric accessors disagree about
	// the wire type: IntEvaluation asserts res.Value.(float64) and wants a JSON
	// number, while FloatEvaluation asserts res.Value.(string) and calls
	// ParseFloat, commented "Because We store floats as string". So for a
	// Flagsmith integer feature -- which is what the backend natively stores --
	// GetIntValue succeeds and GetFloatValue returns TYPE_MISMATCH. No seeding
	// satisfies both accessors, so the lossless integer-to-float scenario
	// cannot pass however the flag is seeded.
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
)

// TestFlagsmithRemoteConformance runs the suite against remote evaluation: the
// provider calls /api/v1/flags/ and the backend's engine evaluates.
func TestFlagsmithRemoteConformance(t *testing.T) {
	runConformance(t, conformanceSuite{
		name:  "flagsmith-remote",
		local: false,
	})
}

// TestFlagsmithLocalConformance runs the suite against local evaluation: the
// provider fetches the environment document and evaluates in-process with the
// Go engine.
func TestFlagsmithLocalConformance(t *testing.T) {
	runConformance(t, conformanceSuite{
		name:  "flagsmith-local",
		local: true,
	})
}

type conformanceSuite struct {
	name  string
	local bool
}

func runConformance(t *testing.T, suite conformanceSuite) {
	if testing.Short() {
		t.Skip("skipping e2e tests in short mode")
	}

	ctx := context.Background()

	image := os.Getenv("FLAGSMITH_TESTBED_IMAGE")
	if image == "" {
		image = defaultTestbedImage
	}

	// Started once for the whole suite and never restarted. Scenario isolation
	// comes from the control API instead -- the no-container-restart invariant
	// exists because dynamically mapped host ports do not survive a restart.
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		Started: true,
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        image,
			ExposedPorts: []string{proxyPort, controlPort},
			// /healthz reports readiness of the CONTROL API, and nothing about
			// the backend -- the two deliberately differ, because the control
			// API has to stay reachable while the backend is down during an
			// outage scenario. So this gate does NOT mean flags are being
			// served, and an adoption that evaluated straight after it would be
			// racing the backend.
			//
			// It is still the right wait strategy: the TCK calls POST /start
			// before each scenario, and /start is the operation that must not
			// return until the seeded state is actually served. Readiness of
			// the control API is exactly the precondition for that call.
			WaitingFor: wait.ForHTTP("/healthz").
				WithPort(controlPort).
				WithStartupTimeout(90 * time.Second),
		},
	})
	if err != nil {
		t.Fatalf("could not start the flagsmith testbed: %v", err)
	}
	t.Cleanup(func() {
		if err := testcontainers.TerminateContainer(container); err != nil {
			t.Logf("could not stop the flagsmith testbed: %v", err)
		}
	})

	// Read once, after the stack is up: the testbed maps host ports
	// dynamically, so these do not exist until now, and they stay valid for the
	// whole suite because nothing restarts a container.
	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("could not read the testbed host: %v", err)
	}
	evaluationPort, err := container.MappedPort(ctx, proxyPort)
	if err != nil {
		t.Fatalf("could not read the mapped evaluation port: %v", err)
	}
	launchpadPort, err := container.MappedPort(ctx, controlPort)
	if err != nil {
		t.Fatalf("could not read the mapped control port: %v", err)
	}

	// The Flagsmith SDK appends its own path segments, so the base URL is the
	// API root with a trailing slash -- remote evaluation requests "flags/"
	// under it, local evaluation "environment-document/".
	baseURL := fmt.Sprintf("http://%s:%s/api/v1/", host, evaluationPort.Port())

	control, err := tck.NewHTTPControl(tck.HTTPControlOptions{
		BaseURL: fmt.Sprintf("http://%s:%s", host, launchpadPort.Port()),
	})
	if err != nil {
		t.Fatalf("could not build the backend control: %v", err)
	}

	tck.Run(t, tck.Config{
		Name:    suite.name,
		Control: control,

		NewProvider: func(scenarioCtx context.Context) (openfeature.FeatureProvider, error) {
			options := []flagsmithClient.Option{
				flagsmithClient.WithBaseURL(baseURL),
			}
			if suite.local {
				// Local evaluation polls the environment document. The default
				// refresh interval is 60 seconds, which would make every
				// scenario following a POST /change time out; the testbed's own
				// upstream poll is already 1s, so this is the second of two
				// polling hops.
				//
				// The context here is the SUITE's, deliberately, not the one
				// this factory is handed. WithLocalEvaluation starts a polling
				// goroutine bound to the context it is given, and the factory's
				// context is scoped to the scenario -- passing it cancels the
				// poll the moment the scenario that created the provider ends,
				// after which every flag resolves to its code default with
				// error code GENERAL. That reads exactly like a broken backend.
				options = append(options,
					flagsmithClient.WithLocalEvaluation(ctx),
					flagsmithClient.WithEnvironmentRefreshInterval(time.Second),
				)
			}
			_ = scenarioCtx

			provider := flagsmith.NewProvider(flagsmithClient.NewClient(serverSideKey, options...))

			if suite.local {
				// Wait for the first environment sync before handing the
				// provider back.
				//
				// This compensates for a real defect rather than for a quirk of
				// the harness, and it is worth being explicit about which.
				// The provider implements no openfeature.StateHandler, so it
				// has no Init for the TCK to call and no way to report that it
				// is not ready yet. The SDK therefore synthesises
				// PROVIDER_READY on registration while the client's first poll
				// is still in flight, and evaluations in that window return the
				// code default with GENERAL and the message "local environment
				// has not yet been updated".
				//
				// That is the flagd#2047 shape moved into the provider:
				// something reports ready before it can serve a flag. A
				// conformant provider would block in Init. This adoption cannot
				// fix that, so it waits here, and FINDINGS records it. Without
				// the wait the suite fails non-deterministically, which would
				// look like flakiness rather than like the defect it is.
				if err := waitForLocalSync(provider); err != nil {
					return nil, err
				}
			}
			return provider, nil
		},

		// What is NOT declared here, and why. All of it comes down to one fact:
		// the provider implements none of Init, Shutdown, Status or
		// EventChannel, so it is neither an openfeature.StateHandler nor an
		// openfeature.EventHandler.
		//
		//   - @lifecycle, @events, @stale, @configuration-change. A provider
		//     with no observable initialisation has no lifecycle to assert
		//     against: the SDK synthesises PROVIDER_READY on registration, so
		//     declaring @lifecycle would make those scenarios pass without the
		//     provider having done anything. That is a vacuous pass, which is
		//     worse than a skip.
		//   - @unavailable for the same reason. The scenario asserts that a
		//     provider unable to reach its backend settles into ERROR and emits
		//     PROVIDER_ERROR. This provider cannot fail initialisation because
		//     it has no initialisation, so it reports READY against a dead
		//     backend. Config.NewUnavailableProvider is therefore not set, and
		//     the scenarios skip with their reason.
		//   - @reinitialization follows from the same absence, and 2.5.2 only
		//     says a provider SHOULD revert to its uninitialized state anyway.
		//
		// None of these get a known-deviation entry. The specification does not
		// require a provider to implement StateHandler or EventHandler, so
		// declining them is an option the contract offers rather than a defect.
		// What is worth saying plainly is that it makes POST /change, /restart
		// and /reset dead weight in this adoption: the testbed implements them
		// and nothing here observes them yet.
		//
		// @object holds -- the provider json.Unmarshals the JSON string
		// Flagsmith stores an object as.
		//
		// @large-integers holds: Go's ResolveIntValue is int64, and 2^53-1
		// survives the JSON number -> float64 -> int64 trip exactly.
		Capabilities: []tck.Capability{
			tck.Object,
			tck.LargeIntegers,
		},

		KnownDeviations: []tck.KnownDeviation{
			numericCoercionDeviation,
		},

		// Both modes poll. Remote evaluation is a single hop and could be
		// quicker, but local evaluation waits on two -- the testbed's upstream
		// poll and the SDK's environment refresh, each 1s -- and the scenarios
		// are shared.
		EventTimeout: 15 * time.Second,
		ReadyTimeout: 30 * time.Second,
	})
}

// waitForLocalSync blocks until the provider can resolve a flag from the
// canonical set, or gives up.
//
// It resolves boolean-flag directly through the provider, before the TCK has
// registered it with the OpenFeature API, so it observes the client's own
// readiness rather than anything the SDK synthesises.
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
