//go:build e2e

package e2e

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/open-feature/go-sdk-contrib/providers/ofrep"
	"github.com/open-feature/go-sdk-contrib/tools/provider-tck/pkg/tck"
	"github.com/open-feature/go-sdk/openfeature"
	"github.com/testcontainers/testcontainers-go/modules/compose"
	"github.com/testcontainers/testcontainers-go/wait"
)

// The OpenFeature Provider Conformance Suite, run against the OFREP provider.
//
// The backend is the existing flagd-testbed, unmodified. flagd serves the OFREP
// API on container port 8016 alongside its own protocols, and the testbed's
// compose file already publishes that port, so the OFREP provider gets a
// conformant backend seeded with the canonical flag set and the standardised
// launchpad control API without a new image, a new compose file, or any change
// to the flagd suites.
//
// THIS SUITE IS CURRENTLY NON-DETERMINISTIC, and the cause is worth reading
// before trusting a run of it.
//
// Three failures are stable, and all three are gaps in the fixture rather than
// in the provider: integral-float-flag and large-integer-flag are absent from
// flagd-testbed (grep the testbed's flags/ directory -- number-zero-flag and
// huge-integer-flag are missing too), so the scenarios that ask for them fail
// with FLAG_NOT_FOUND against any provider. The third is the last row of the
// @variants outline, which asks large-integer-flag for its max-int32 variant
// and gets "" -- a flag that is not there has no variant to name, so it is the
// same absence counted twice rather than a new defect.
// open-feature/flagd-testbed#392 adds them.
//
// Every other failure moves between runs. Two consecutive runs of this file
// produced 13 failures and then 12, with almost disjoint failing sets, and
// every one of them was FLAG_NOT_FOUND on a flag the testbed definitely has --
// boolean-flag, string-zero-flag, object-flag. The cause is that this provider
// has no initialisation: the TCK's per-scenario reset calls POST /start on the
// launchpad, which stops flagd, deletes the combined flag file, regenerates it
// and restarts flagd, polling :8014/readyz until flagd answers. flagd answers
// before its file source has loaded the flags. A provider with a lifecycle does
// not notice, because its own Init blocks until the RPC stream is up or the
// in-process sync completes, and by then the flags are there -- which is why
// both flagd suites are stable against the same backend at the same revision.
// A stateless provider fires its first evaluation the instant POST /start
// returns, and races the load.
//
// So the defect is in the control-API contract rather than here: POST /start
// returning before the backend serves flags makes the reset unusable by exactly
// the providers that have no way to wait for it. Fixing it by adding a sleep or
// a retry to this file would hide it from every other language's adoption, so
// it is written down instead. Until then, read a red result here against the
// list above before attributing anything to the provider.

const (
	// The flagd testbed is a git submodule of the flagd provider. Reused, not
	// copied: two testbeds that drift apart would make a cross-provider
	// disagreement look like a provider defect.
	testbedComposeFile = "../../flagd/flagd-testbed/docker-compose.yaml"

	// Container ports as published by the testbed's compose file. Host ports
	// are assigned dynamically and read back once the stack is up.
	ofrepContainerPort     = "8016"
	launchpadContainerPort = "8080"

	// The compose service that runs both flagd and its launchpad.
	composeService = "flagd"
)

// TestOFREPConformance runs the suite against the OFREP provider pointed at
// flagd's OFREP endpoint.
func TestOFREPConformance(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e tests in short mode")
	}

	ctx := context.Background()

	baseURI, launchpadURL := startTestbed(ctx, t)

	control, err := tck.NewHTTPControl(tck.HTTPControlOptions{
		BaseURL: launchpadURL,
	})
	if err != nil {
		t.Fatalf("could not build the backend control: %v", err)
	}

	tck.Run(t, tck.Config{
		Name:    "ofrep",
		Control: control,

		NewProvider: func(context.Context) (openfeature.FeatureProvider, error) {
			// NewProvider never fails: it only builds an http.Client and a
			// base URI, and does not contact the backend. See
			// providers/ofrep/provider.go:25-40.
			//
			// The timeout is well under the TCK's own step deadlines so that a
			// wedged backend surfaces as a resolution error attributable to
			// this provider rather than as a suite-level timeout.
			return ofrep.NewProvider(baseURI, ofrep.WithTimeout(5*time.Second)), nil
		},

		// Config.NewUnavailableProvider is deliberately absent, which is the
		// configuration tck.Config documents for a provider that cannot
		// declare tck.UnavailableInit. See the capability notes below.

		// The declared set is Object and NumericCoercion, and every
		// omission is a property of the provider's code rather than a
		// preference.
		//
		// tck.Events is NOT declared. The OFREP provider is stateless: its
		// entire method set is Metadata, the five typed *Evaluation methods and
		// Hooks (providers/ofrep/provider.go:42-70). It implements neither
		// openfeature.EventHandler (no EventChannel method) nor
		// openfeature.StateHandler (no Init, no Shutdown), so it can never
		// publish a provider event — the SDK only starts a listener goroutine
		// for providers that implement EventHandler.
		//
		// It would be easy to declare tck.Events anyway and watch the
		// "reaching its backend becomes ready" scenario pass, because the SDK
		// synthesises PROVIDER_READY for a provider with no StateHandler:
		// initializerWithContext in go-sdk v1.18.0's openfeature_api.go returns
		// a ProviderReady event with the comment "a provider without state
		// handling capability can be assumed to be ready immediately". That
		// READY says nothing about the backend — it is emitted identically
		// against a backend that does not exist — so declaring the capability
		// would buy one green scenario that asserts nothing. It stays
		// undeclared until the provider emits events of its own.
		//
		// tck.ConfigurationChange follows from the same absence: with no event
		// channel there is nothing that could emit
		// PROVIDER_CONFIGURATION_CHANGED. The provider also holds no local
		// ruleset — every evaluation is a fresh POST to
		// /ofrep/v1/evaluate/flags/{key}
		// (providers/ofrep/internal/outbound/http.go:15,54-72) — so it would
		// pass the second half of that scenario (values do change) while
		// failing the first (nothing signals the change).
		//
		// tck.Stale follows too: STALE is a provider state transition
		// communicated by PROVIDER_STALE, and a provider with no state handling
		// has no state to transition.
		//
		// tck.UnavailableInit is NOT declared because there is no
		// initialisation to fail. With no StateHandler the SDK reports READY
		// unconditionally, so a provider pointed at a closed port would sit in
		// READY rather than settling into ERROR, and the @unavailable scenarios
		// would fail on "the client should be in error state". The evaluation
		// half of the contract is in fact honoured — an unreachable host yields
		// a GENERAL resolution error and the code default
		// (providers/ofrep/internal/evaluate/resolver.go:38-42) — but the state
		// half cannot be, so the capability is withheld rather than half-met.
		//
		// tck.Object IS declared. ObjectEvaluation passes the decoded value
		// straight through
		// (providers/ofrep/internal/evaluate/flags.go:253-260), which for a
		// JSON object is a map[string]any, and requesting object-flag as any
		// scalar lands in a failed type assertion or a default switch branch
		// and reports TYPE_MISMATCH (flags.go:49-59, 94-104, 141-155, 209-217).
		//
		// tck.NumericCoercion IS declared, which is worth stating plainly
		// because OFREP is JSON and JSON has exactly one number type: 0.5 and
		// 10 both arrive from encoding/json as float64, so the provider cannot
		// learn integer-ness from the wire. It does not have to. ResolveInt
		// round-trips the float64 through int64 and reports TYPE_MISMATCH when
		// the round trip is lossy (flags.go:197-208), so float-flag requested
		// as an Integer is a mismatch rather than a silent narrowing to 0,
		// which is exactly what the @numeric-coercion scenario asserts.
		//
		// That lossy-round-trip check is, independently, the rule flagd's
		// numeric coercion ADR settles on (open-feature/flagd#1996): coercion
		// is permitted when lossless and must fail when it would lose
		// information. This provider got there from the constraints of JSON
		// rather than from the ADR, which is some evidence the rule is the
		// natural one rather than a flagd preference. Worth knowing that the
		// specification does not require it either way -- OpenFeature has one
		// numeric type, of "unspecified type or size" -- so the capability is
		// tested against a borrowed rule; see open-feature/spec#430.
		//
		// The converse — integer-flag requested as a Float — is accepted and
		// returns 10.0, because ResolveFloat takes any float64
		// (flags.go:141-155) and that is what a JSON 10 decodes to. That is the
		// lossless direction, and it now HAS scenarios: the canonical flag set
		// gained integral-float-flag, so "An integral float requested as an
		// integer is coerced without loss" and "An integer requested as a float
		// is widened without loss" both run. The capability is therefore a
		// stronger claim than it was when this comment was first written, and
		// the declaration is kept deliberately rather than by inertia.
		//
		// One of those two cannot be verified against this backend, and it is
		// the fixture's fault: flagd-testbed has no integral-float-flag, so the
		// scenario fails with FLAG_NOT_FOUND no matter what the provider does.
		// The capability is still declared, because the two scenarios that the
		// backend CAN answer -- the lossy half, and the widening half through
		// integer-flag -- both pass, and withholding the tag would skip the
		// lossy scenario too. That is the one worth keeping: silently narrowing
		// 0.5 to 0 is the failure mode flagd has and this provider does not.
		// The failure gets no deviation entry, because the gap is in the
		// fixture and an entry there would attribute it to the provider.
		// open-feature/flagd-testbed#392.
		//
		// tck.Variants IS declared. OFREP's evaluation response carries a
		// variant field and this provider passes it straight into
		// ResolutionDetail, so seven of the eight rows pass -- booleans,
		// strings, integers, floats and all three falsy flags. The eighth is
		// large-integer-flag, which the testbed does not serve, so it is the
		// fixture gap above rather than a variant defect; the same reasoning
		// that keeps tck.NumericCoercion declared keeps this one declared.
		//
		// tck.Targeting IS declared, and for a JSON-over-HTTP provider it is
		// the cheapest capability here to get right: the evaluation context IS
		// the request body, so there is no separate passthrough path to get
		// wrong. All three scenarios pass -- targeting-key-flag resolves to
		// "hit" for the matching key and "miss" for a non-matching one or none
		// at all -- and so does the new untagged scenario that supplies a
		// context to an untargeted flag. Worth having: until this revision no
		// scenario supplied a context at all, so a provider that serialised it
		// into a malformed body passed the whole suite, and for this provider
		// that body is the entire request.
		//
		// tck.DisabledFlags IS declared, and it is the one capability here
		// that was expected to be impossible. It is gated because a disabled
		// flag's resolution depends on where the caller's default is
		// substituted: a provider that evaluates locally holds it, one whose
		// backend decides does not. OFREP is the clearest case of the second
		// kind -- the request body carries the context and the flag key and
		// nothing else -- so the plan was to leave the tag undeclared and
		// write the architecture down beside it.
		//
		// It passes, all four rows, over three consecutive runs. The reasoning
		// was right about the server and wrong about what the capability
		// needs. flagd's OFREP endpoint answers 200 with
		// {"key":"disabled-string-flag","reason":"DISABLED","metadata":{}} --
		// no value member and no variant, verified by curl against the testbed
		// rather than inferred -- so the response does not have to carry the
		// caller's default. It only has to distinguish a disabled flag from a
		// resolved one, and OFREP's reason field does.
		//
		// This provider then acts on it: each of the five typed resolvers in
		// internal/evaluate/flags.go checks the DISABLED reason BEFORE it
		// type-asserts the value, and returns defaultValue with reason
		// DISABLED and no resolution error. That ordering is what earns the
		// tag. Without the branch an absent value would fail the type
		// assertion and come back as TYPE_MISMATCH, which is exactly what the
		// error-code step would have caught.
		//
		// The capability stays gated for the reason it always was -- a backend
		// whose response says nothing about state leaves a provider no way to
		// answer -- but this is not a property OFREP providers lack, and there
		// is nothing here to record as a deviation.
		Capabilities: []tck.Capability{
			tck.Object,
			tck.NumericCoercion,
			tck.Variants,
			tck.Targeting,
			tck.DisabledFlags,
		},

		// The provider has no initialisation to wait for, so this bounds the
		// SDK's registration round trip and nothing else.
		ReadyTimeout: 30 * time.Second,

		// Unused in practice — every event-bearing scenario is gated behind
		// tck.Events — but set explicitly so the value does not silently change
		// meaning if the provider grows eventing.
		EventTimeout: 15 * time.Second,
	})
}

// startTestbed brings up the flagd testbed for the whole suite and returns the
// OFREP base URI and the launchpad URL, both built from dynamically mapped host
// ports.
//
// tests/flagd/testframework.NewFlagdContainer is deliberately not used. It maps
// only container ports 8013, 8014, 8015 and 8080 (testcontainer.go:80-95), its
// GetPort accessor knows only "rpc", "in-process", "launchpad" and "health"
// (testcontainer.go:141-154), and it exposes neither the compose stack nor the
// service container, so there is no way to reach the OFREP port through it.
// Widening that helper would change a published module every other flagd suite
// depends on; driving compose directly here costs about thirty lines and
// changes nothing outside this file.
//
// The stack is started once for the whole suite and never restarted. Scenario
// isolation comes from the control API instead — see the no-container-restart
// invariant in the control API specification.
func startTestbed(ctx context.Context, t *testing.T) (baseURI, launchpadURL string) {
	t.Helper()

	stack, err := compose.NewDockerCompose(testbedComposeFile)
	if err != nil {
		t.Fatalf("could not read the flagd testbed compose file %s: %v. It is a git submodule of "+
			"the flagd provider, so an empty directory here means the submodule was never checked "+
			"out: git submodule update --init --recursive", testbedComposeFile, err)
	}

	// The launchpad writes flag files into this directory, which the compose
	// file bind-mounts. It is per-suite so that nothing else can disturb it.
	flagsDir, err := os.MkdirTemp("", "ofrep-tck-*")
	if err != nil {
		t.Fatalf("could not create a flags directory: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(flagsDir) })

	stack.WithEnv(map[string]string{"FLAGS_DIR": flagsDir})

	// Wait on the launchpad rather than on flagd: flagd is not running yet at
	// this point, because the launchpad is what starts it, and the TCK's first
	// control call is what asks for that.
	stack.WaitForService(composeService,
		wait.ForListeningPort(launchpadContainerPort+"/tcp").WithStartupTimeout(60*time.Second))

	if err := stack.Up(ctx); err != nil {
		t.Fatalf("could not start the flagd testbed: %v", err)
	}
	t.Cleanup(func() {
		if err := stack.Down(context.Background()); err != nil {
			t.Logf("could not stop the flagd testbed: %v", err)
		}
	})

	service, err := stack.ServiceContainer(ctx, composeService)
	if err != nil {
		t.Fatalf("the testbed has no %q service: %v", composeService, err)
	}

	// Read once, after the stack is up: compose assigns host ports dynamically,
	// so these do not exist until now, and they stay valid for the whole suite
	// because nothing restarts a container.
	mapped := func(containerPort string) int {
		port, err := service.MappedPort(ctx, containerPort)
		if err != nil {
			t.Fatalf("the testbed published no host port for container port %s: %v. flagd serves "+
				"OFREP on 8016 and its launchpad on 8080; both must be published by %s",
				containerPort, err, testbedComposeFile)
		}
		return int(port.Num())
	}

	baseURI = fmt.Sprintf("http://localhost:%d", mapped(ofrepContainerPort))
	launchpadURL = fmt.Sprintf("http://localhost:%d", mapped(launchpadContainerPort))

	// The launchpad's listener accepts connections slightly before it is ready
	// to act on one. The flagd suite allows the same grace through
	// FlagdContainerConfig.ExtraWaitTime.
	time.Sleep(2 * time.Second)

	return baseURI, launchpadURL
}
