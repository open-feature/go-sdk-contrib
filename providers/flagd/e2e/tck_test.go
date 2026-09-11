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

// The gaps this provider is known to have, as opposed to the capabilities it
// simply does not implement.
//
// Narrowing the capability list says a scenario did not run; it cannot say
// whether that is because the provider declines the capability or because it
// fails at it. In the results the two are the same skip carrying the same
// reason, so a consumer comparing providers reads a defect as a design choice
// unless something says otherwise. This entry is that something.
//
// WHAT DOES NOT BELONG HERE is the more useful half of the lesson, because two
// entries were written and then deleted before this file was committed. Before
// recording a deviation, find the numbered requirement the scenario maps to and
// read what it actually says:
//
//   - Re-initialisation after shutdown. The scenario fails in both resolvers,
//     and it is not a deviation. Requirement 2.5.2 says a provider SHOULD
//     revert to its uninitialized state and its supporting text says "some
//     providers MAY allow reinitialization from this state" -- permitted, not
//     required. So the scenario is gated on @reinitialization, which this
//     adoption simply does not declare. See the RPC suite below.
//
//   - PROVIDER_STALE on the RPC resolver. Requirement 5.1.1's supporting text
//     says a provider "can" signal an outage with PROVIDER_ERROR, and that one
//     which caches rule-sets or evaluations "can" signal PROVIDER_STALE. Both
//     are permitted; neither is required. The RPC resolver takes the first
//     option, which is a design choice and not a defect. See the RPC suite
//     below.
//
// A false failure is the mirror image of a vacuous pass, and a deviation
// recorded against a permitted choice is how the first one gets published.
var (
	// Tracked against flagd's numeric coercion ADR, which is where the rule
	// this deviates from is settled: coercion is permitted when it is lossless
	// and must fail with TYPE_MISMATCH only when information would be lost. The
	// summary says which half is broken, because "flagd coerces numbers" on its
	// own reads as a description of intended behaviour rather than a defect.
	//
	// This is the one deviation here whose rule survives the check described
	// above, and it survives it in an unusual way: the rule is not in the
	// specification at all, but flagd accepted it for itself and has an open
	// issue to implement it, so the provider is measured against its own
	// commitment. The summary says so, because a consumer should not read this
	// as a specification violation.
	numericCoercionDeviation = tck.TrackedDeviation(
		tck.NumericCoercion,
		"https://github.com/open-feature/flagd/issues/1996",
		"The lossy half of the coercion rule is not enforced: evaluating float-flag (0.5) "+
			"through GetIntDetails returns 0 with no error code, rather than TYPE_MISMATCH with "+
			"the code default, so the fractional part is discarded silently. Lossless coercion is "+
			"permitted and is not the defect. Both resolvers behave identically, which places it "+
			"in the shared provider layer rather than in either transport. The rule is flagd's "+
			"own accepted numeric-coercion ADR rather than a specification requirement -- the "+
			"specification does not define numeric coercion at all (open-feature/spec#430) -- so "+
			"this is a deviation from a commitment flagd made, not from the provider contract.")
)

// TestFlagdRPCConformance runs the suite against the RPC resolver.
func TestFlagdRPCConformance(t *testing.T) {
	runConformance(t, conformanceSuite{
		name:     "flagd-rpc",
		portName: "rpc",
		resolver: flagd.WithRPCResolver(),

		// tck.Lifecycle IS declared, and legitimately so. The RPC resolver
		// reaches flagd during initialisation -- Init builds the client, starts
		// the event stream and blocks until the stream is up or the deadline
		// expires -- so the lifecycle scenarios assert something real here.
		// What would make them vacuous is the Go SDK synthesising
		// PROVIDER_READY for a provider that does not implement
		// openfeature.StateHandler; this provider implements Init, Status and
		// Shutdown, and Init can and does fail.
		//
		// Go withheld this capability while Java declared it, which is why Java
		// ran 36 of the 40 scenarios and Go ran 29. Withholding it is the
		// expensive mistake, not declaring it: it made this adoption blind to
		// six scenarios another language was running.
		//
		// tck.Reinitialization is NOT declared, and that is a choice the
		// specification offers rather than a gap. Requirement 2.5.2 says a
		// provider SHOULD revert to its uninitialized state after shutdown and
		// its supporting text says "some providers MAY allow reinitialization
		// from this state", so reuse is permitted and not required. This
		// provider does not offer it: Shutdown clears the provider's own
		// initialised flag, so a second Init proceeds, but what it then waits
		// for never arrives -- the RPC service's event stream never signals
		// ready again -- and Init returns "provider initialization deadline
		// exceeded". Leaving the tag undeclared reports "A provider that was
		// shut down can be initialized again" as skipped with its reason, which
		// is the accurate result. It gets no knownDeviations entry, because
		// nothing is deviating.
		//
		// That scenario was mandatory until spec fc99d5ac, failed here, and was
		// written down as a known deviation against this provider before anyone
		// read 2.5.2. Gating it is the fix.
		//
		// tck.Stale is NOT declared either. That is a real difference between
		// the two resolvers, confirmed by running rather than inferred from the
		// code: declaring it fails "Losing the backend makes the provider
		// stale, regaining it makes it ready again" with "timed out after 15s
		// waiting for a PROVIDER_STALE event".
		//
		// The RPC resolver emits PROVIDER_ERROR, PROVIDER_READY and
		// PROVIDER_CONFIGURATION_CHANGED, and never emits PROVIDER_STALE — see
		// pkg/service/rpc/service.go, where losing the stream sends
		// of.ProviderError directly. The in-process resolver, by contrast,
		// emits PROVIDER_STALE on connection loss and only escalates to
		// PROVIDER_ERROR once the retry grace period expires.
		//
		// So the two resolvers of the same provider report an outage
		// differently: an application that switches from in-process to RPC
		// stops receiving stale events. That is worth knowing and is why it is
		// written down here -- but it is not a conformance defect, and it gets
		// no knownDeviations entry. Requirement 5.1.1's supporting text offers
		// both behaviours in the same breath: a provider unable to evaluate
		// flags "can" signal that with PROVIDER_ERROR, and a provider that
		// caches rule-sets or evaluations "can" signal PROVIDER_STALE. "Can",
		// twice. The RPC resolver takes the first option and goes to ERROR,
		// which also means it is not quietly serving cached values while
		// disconnected -- the SDK short-circuits to the code default instead.
		// Declare this if and when the RPC resolver emits PROVIDER_STALE.
		//
		// Worth stating plainly, because the pull is the other way: the JS and
		// Java adoptions declare @stale for both resolvers. On this evidence
		// that is a vacuous declaration for RPC -- the event never arrives --
		// which is a reason to leave it withheld here, not a reason to copy
		// them.
		//
		// tck.NumericCoercion is NOT declared either, and this one was found
		// by running the suite rather than by reading the code. Evaluating
		// float-flag (0.5) through GetIntDetails returns 0 with no error code
		// at all -- not TYPE_MISMATCH with the code default. The application
		// sees a plausible value and no indication anything went wrong, which
		// is the worst failure mode a feature flag has.
		//
		// Both resolvers do it identically, so the defect is in this provider's
		// shared layer rather than in either transport. The Java flagd provider
		// does it too -- but the Python one does not, in either resolver: its
		// RPC path asks flagd for an Int and gets INVALID_ARGUMENT for a
		// float-valued flag, and its in-process path admits only int for an
		// integer request. So this is a Go and Java provider issue, not a
		// flagd-wide one, and the server is not the thing getting it wrong.
		//
		// flagd's own fix is open-feature/flagd#1996, which implements flagd's
		// numeric coercion ADR: coercion is permitted when lossless, so
		// 10.0 -> 10 keeps working, and must return TYPE_MISMATCH when it
		// would lose information, which 0.5 does.
		//
		// Worth knowing when reading this: the specification does not actually
		// require that. OpenFeature has one numeric type, of "unspecified type
		// or size", and differentiating integers from floats is an optional
		// language idiom -- so this capability is tested against a rule
		// borrowed from flagd rather than a requirement, and the gap in the
		// provider contract is open-feature/spec#430. Declare this once
		// flagd#1996 lands.
		//
		// tck.LargeIntegers is NOT declared, and this absence is neither a
		// choice nor a provider defect: huge-integer-flag is absent from
		// flagd-testbed, so the capability cannot be verified against this
		// backend at all. The untagged large-integer-flag scenario fails for
		// the same reason, and it is the one failure this suite carries that
		// says nothing whatever about the provider -- Go's ResolveIntValue is
		// int64 and has room for both values.
		// open-feature/flagd-testbed#392 adds the flags; declare this and the
		// failure goes away together once it lands. It gets no
		// knownDeviations entry on purpose, because the gap is in the fixture
		// and an entry there would attribute it to the provider.
		capabilities: []tck.Capability{
			tck.Events,
			tck.Lifecycle,
			tck.ConfigurationChange,
			tck.Object,
			tck.UnavailableInit,
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
		name:     "flagd-in-process",
		portName: "in-process",
		resolver: flagd.WithInProcessResolver(),

		// Everything except tck.NumericCoercion, tck.LargeIntegers and
		// tck.Reinitialization. Unlike RPC, the in-process resolver emits
		// PROVIDER_STALE on connection loss, so it can satisfy the @stale
		// scenario -- and does: the scenario passes here and would fail on RPC,
		// which is the difference an application would see if it switched
		// resolver.
		//
		// It narrows float-flag (0.5) to 0 on an integer request exactly as
		// the RPC resolver does -- observed, not inferred -- which places that
		// defect in the shared provider layer. See the RPC suite above, and
		// the same for tck.LargeIntegers, which the testbed cannot exercise.
		//
		// tck.Lifecycle holds here for the same reason it does on RPC, and more
		// visibly: the in-process resolver syncs the whole ruleset before
		// reporting ready, so initialisation is unambiguously doing work.
		//
		// tck.Reinitialization is NOT declared here either, for the reason the
		// RPC suite gives: 2.5.2 permits reuse rather than requiring it, and
		// this resolver does not offer it -- a second Init waits for a sync
		// that never completes and times out. Undeclared, the scenario is
		// skipped, which is what it should be.
		//
		// Worth knowing when comparing languages: Java's in-process resolver
		// PASSES that scenario, and would pass it even if it declared the tag,
		// because its evaluator keeps serving the last-synced ruleset -- so the
		// assertion is satisfied without a re-initialisation having happened.
		// A skip that says "not offered" is more honest than a pass that says
		// nothing.
		capabilities: []tck.Capability{
			tck.Events,
			tck.Lifecycle,
			tck.Stale,
			tck.ConfigurationChange,
			tck.Object,
			tck.UnavailableInit,
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

// conformanceSuite is the per-resolver configuration.
type conformanceSuite struct {
	name            string
	portName        string
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

		// Without this the per-suite knownDeviations would be collected and
		// then dropped on the floor, which is the quietest possible way for a
		// conformance report to lose the one field that tells a defect apart
		// from a design choice. It read as wired because the struct field was
		// populated.
		KnownDeviations: suite.knownDeviations,

		ReadyTimeout: suite.readyTimeout,
		EventTimeout: 15 * time.Second,
	})
}
