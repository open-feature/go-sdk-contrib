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

		// The declared set is Object and StrictNumericTyping, and every
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
		// tck.StrictNumericTyping IS declared, which is worth stating plainly
		// because OFREP is JSON and JSON has exactly one number type: 0.5 and
		// 10 both arrive from encoding/json as float64, so the provider cannot
		// learn integer-ness from the wire. It does not have to. ResolveInt
		// round-trips the float64 through int64 and reports TYPE_MISMATCH when
		// the round trip is lossy (flags.go:197-208), so float-flag requested
		// as an Integer is a mismatch rather than a silent narrowing to 0,
		// which is exactly what the @strict-numeric-typing scenario asserts.
		//
		// The converse — integer-flag requested as a Float — is accepted and
		// returns 10.0, because ResolveFloat takes any float64
		// (flags.go:141-155) and that is what a JSON 10 decodes to. No scenario
		// covers that direction today, and for a JSON protocol it is arguably
		// the correct answer, but the asymmetry is worth knowing about.
		Capabilities: []tck.Capability{
			tck.Object,
			tck.StrictNumericTyping,
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
