//go:build e2e

package e2e

import (
	"context"
	"os"
	"testing"
	"time"

	flagd "github.com/open-feature/go-sdk-contrib/providers/flagd/pkg"
	"github.com/open-feature/go-sdk-contrib/tests/flagd/testframework"
	"github.com/open-feature/go-sdk-contrib/tools/provider-tck/pkg/tck"
	"github.com/open-feature/go-sdk/openfeature"
)

// The OpenFeature Provider Conformance Suite, run against the flagd provider in
// both of its resolver modes.
//
// flagd resolves flags two quite different ways — RPC evaluates remotely over
// gRPC, in-process syncs the ruleset and evaluates locally — and they are
// separate suites because they are separately conformant. Any difference
// between the two results is a difference an application would see when it
// switches resolver, which is exactly the kind of thing the suite exists to
// surface.
//
// The existing e2e suites in this package are untouched, and so is
// flagd-testbed. The TCK drives the testbed's launchpad through the
// standardised control API, which the launchpad already implements.

// TestFlagdRPCConformance runs the suite against the RPC resolver.
func TestFlagdRPCConformance(t *testing.T) {
	runConformance(t, conformanceSuite{
		name:     "flagd-rpc",
		portName: "rpc",
		resolver: flagd.WithRPCResolver(),

		// tck.Stale is NOT declared, and that is a finding rather than a
		// configuration choice.
		//
		// The RPC resolver emits PROVIDER_ERROR, PROVIDER_READY and
		// PROVIDER_CONFIGURATION_CHANGED, and never emits PROVIDER_STALE — see
		// pkg/service/rpc/service.go, where losing the stream sends
		// of.ProviderError directly. The in-process resolver, by contrast,
		// emits PROVIDER_STALE on connection loss and only escalates to
		// PROVIDER_ERROR once the retry grace period expires, which is the
		// behaviour the specification describes.
		//
		// So the two resolvers of the same provider report an outage
		// differently: an application that switches from in-process to RPC
		// silently stops receiving stale events. Withholding the capability
		// here reports the @stale scenario as skipped with its reason rather
		// than failing it, and the gap needs its own issue against the
		// provider. Declare this as soon as the RPC resolver emits
		// PROVIDER_STALE.
		capabilities: []tck.Capability{
			tck.Events,
			tck.ConfigurationChange,
			tck.Object,
			tck.UnavailableInit,
			tck.StrictNumericTyping,
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
		name:     "flagd-in-process",
		portName: "in-process",
		resolver: flagd.WithInProcessResolver(),

		// The full set. The in-process resolver emits PROVIDER_STALE on
		// connection loss, so unlike RPC it can satisfy the @stale scenario.
		capabilities: tck.AllCapabilities(),

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

// conformanceSuite is the per-resolver configuration.
type conformanceSuite struct {
	name         string
	portName     string
	resolver     flagd.ProviderOption
	capabilities []tck.Capability
	readyTimeout time.Duration
	gracePeriod  int
}

func runConformance(t *testing.T, suite conformanceSuite) {
	if testing.Short() {
		t.Skip("skipping e2e tests in short mode")
	}

	ctx := context.Background()

	// The launchpad rewrites flag files into this directory. It is per-suite so
	// the two resolver suites cannot disturb each other.
	flagsDir, err := os.MkdirTemp("", "flagd-tck-*")
	if err != nil {
		t.Fatalf("could not create a flags directory: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(flagsDir) })

	// The stack is started once for the whole suite and never restarted.
	// Scenario isolation comes from the control API instead — see the
	// no-container-restart invariant in the control API specification.
	container, err := testframework.NewFlagdContainer(ctx, testframework.FlagdContainerConfig{
		TestbedDir:    "../flagd-testbed",
		FlagsDir:      flagsDir,
		ExtraWaitTime: 2 * time.Second,
	})
	if err != nil {
		t.Fatalf("could not start the flagd testbed: %v", err)
	}
	t.Cleanup(func() {
		if err := container.Stop(); err != nil {
			t.Logf("could not stop the flagd testbed: %v", err)
		}
	})

	control, err := tck.NewHTTPControl(tck.HTTPControlOptions{
		BaseURL: container.GetLaunchpadURL(),
	})
	if err != nil {
		t.Fatalf("could not build the backend control: %v", err)
	}

	// Read once, after the stack is up: the testbed maps host ports
	// dynamically, so these do not exist until now, and they stay valid for the
	// whole suite because nothing restarts a container.
	host := container.GetHost()
	port := uint16(container.GetPort(suite.portName))
	if port == 0 {
		t.Fatalf("the testbed exposed no %q port", suite.portName)
	}

	tck.Run(t, tck.Config{
		Name:    suite.name,
		Control: control,

		NewProvider: func(context.Context) (openfeature.FeatureProvider, error) {
			provider, err := flagd.NewProvider(
				suite.resolver,
				flagd.WithHost(host),
				flagd.WithPort(port),
				flagd.WithDeadline(1000),
				flagd.WithRetryGracePeriod(suite.gracePeriod),
				flagd.WithRetryBackoffMs(500),
			)
			if err != nil {
				return nil, err
			}
			return provider, nil
		},

		// Pointed at a closed port on localhost, never at the backend under
		// test — that has to stay up, and simulated outages belong to the
		// control API. The deadlines are deliberately short: the scenario
		// asserts that failure is reported promptly, so a provider that took
		// 30 seconds to give up would pass a test about eventual failure and
		// fail the one that matters.
		NewUnavailableProvider: func(context.Context) (openfeature.FeatureProvider, error) {
			provider, err := flagd.NewProvider(
				suite.resolver,
				flagd.WithHost("localhost"),
				flagd.WithPort(9999),
				flagd.WithDeadline(500),
				flagd.WithRetryGracePeriod(1),
				flagd.WithRetryBackoffMs(100),
			)
			if err != nil {
				return nil, err
			}
			return provider, nil
		},

		Capabilities: suite.capabilities,
		ReadyTimeout: suite.readyTimeout,
		EventTimeout: 15 * time.Second,
	})
}
