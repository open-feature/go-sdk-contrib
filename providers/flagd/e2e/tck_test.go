//go:build e2e

package e2e

import (
	"context"
	"testing"
	"time"

	flagd "github.com/open-feature/go-sdk-contrib/providers/flagd/pkg"
	"github.com/open-feature/go-sdk-contrib/tools/tck"
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
// There is no container code in this file. The suite owns the stack: it is
// handed the Compose file below, told which container-internal port each
// resolver connects to, and given a factory that builds a provider from the
// host ports it discovered. Everything else — starting the stack once, waiting
// for the launchpad to accept commands, resetting the backend between
// scenarios, tearing down — belongs to tck.Run. The hand-rolled wrapper this
// replaces read the testbed's compose file through
// tests/flagd/testframework.NewFlagdContainer, created a temporary flags
// directory for it to bind-mount, built the HTTP control itself and looked up
// two named ports by string; all of that is now the harness's, and every future
// adopter gets it without writing it.
//
// The existing e2e suites in this package are untouched, and so is
// flagd-testbed. The TCK drives the testbed's launchpad through the
// standardised control API, which the launchpad already implements.

const (
	// composeFile describes the backend stack. Resolved relative to this
	// package directory, which is where `go test` runs.
	//
	// Deliberately not the testbed submodule's own compose file — see the
	// comment at the top of it for why.
	composeFile = "testdata/tck/docker-compose.yaml"

	// rpcPort and inProcessPort are the container-internal ports the two
	// resolvers connect to. The launchpad's control port is exposed by the
	// harness and is not named here.
	rpcPort       = 8013
	inProcessPort = 8015

	// unavailablePort is a port on localhost that nothing listens on, for the
	// initialisation-failure scenarios. Deliberately not a port on the stack:
	// the stack must stay up for the whole suite, and simulated outages belong
	// to the control API.
	unavailablePort = 9999
)

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
		name:        "flagd-rpc",
		backendPort: rpcPort,
		resolver:    flagd.WithRPCResolver(),

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
		// backend at all. Two scenarios fail for the same reason -- the
		// untagged "A large integer resolves without loss of precision", and
		// the last row of the @variants outline below, which asks
		// large-integer-flag for its max-int32 variant and gets "" because the
		// flag is not there to have one. They are the only failures this suite
		// carries that say nothing whatever about the provider: Go's
		// ResolveIntValue is int64 and has room for both values.
		// open-feature/flagd-testbed#392 adds the flags; declare this and both
		// failures go away together once it lands. It gets no knownDeviations
		// entry on purpose, because the gap is in the fixture and an entry
		// there would attribute it to the provider.
		//
		// tck.Variants IS declared, on the evidence of the run rather than on
		// the reasoning that flagd obviously has variants. Seven of the eight
		// rows pass in both resolvers: the variant name survives the trip from
		// the ruleset through the wire format into ResolutionDetail for
		// booleans, strings, integers, floats and all three falsy flags. The
		// eighth is the fixture gap described above and not a variant defect,
		// which is why the capability is declared rather than withheld -- a
		// withheld tag would skip seven working rows to hide one missing flag.
		//
		// tck.DisabledFlags IS declared, on the evidence of the run, and the
		// run is the only thing that could have settled it. The capability is
		// gated because what a disabled flag resolves to depends on where the
		// substitution happens: a provider that evaluates locally can hand
		// back the caller's default, and one whose backend decides cannot,
		// because the default never left the process. The RPC resolver is on
		// the wrong side of that line by construction -- it asks flagd to
		// resolve every flag -- so the honest expectation was that it would
		// fail and the in-process resolver would pass.
		//
		// It passes in both, and running it is what showed why: flagd's
		// evaluation response does not have to carry the caller's default. It
		// reports reason DISABLED with an empty variant and a zero value, and
		// isDefaultOrDisabledFallback in pkg/service/rpc/service.go recognises
		// that pair and keeps defaultValue instead of taking the response's.
		// So the line is not "does the server see the caller's default" but
		// "does the response distinguish a disabled flag from a resolved one".
		// The zero value is not what carries it: only the boolean row's
		// default (false) coincides with its zero, and the other three -- "bye"
		// against "", 1 against 0, 0.1 against 0.0 -- fail if the response
		// value is taken. An OFREP response carries no such distinction, which
		// is what makes the tag worth having.
		//
		// All four rows pass in both resolvers. Two verification passes were
		// needed to say so: one earlier run failed this outline with
		// FLAG_NOT_FOUND and failed the object scenario with reason ERROR at
		// the same time, and both went away on re-running. That is the
		// launchpad reset race the OFREP suite documents at length -- POST
		// /start returns before flagd's file source has loaded the flags -- and
		// not a property of this outline. A single red run here means re-run
		// before concluding anything.
		//
		// tck.Targeting IS declared, and it was reserved rather than declarable
		// until spec 26362f85. All three scenarios pass in both resolvers, and
		// they assert something this suite could not otherwise see: that the
		// evaluation context reaches the backend at all. targeting-key-flag has
		// one JsonLogic rule on the targeting key, so a matching context
		// resolves to a different value than a non-matching one or none --
		// which means a provider that silently dropped the context would be
		// caught by the resolved value itself, with no echo endpoint needed.
		// The flag has been in flagd-testbed since flagd-testbed#103, released
		// in v0.5.1 in February 2024, so this needs no image bump -- unlike
		// tck.LargeIntegers above, which is waiting on one.
		capabilities: []tck.Capability{
			tck.Events,
			tck.Lifecycle,
			tck.ConfigurationChange,
			tck.Object,
			tck.Variants,
			tck.DisabledFlags,
			tck.Targeting,
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
		name:        "flagd-in-process",
		backendPort: inProcessPort,
		resolver:    flagd.WithInProcessResolver(),

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
		//
		// tck.Variants, tck.Targeting and tck.DisabledFlags are declared here
		// as well, and both resolvers produce the identical result: 56
		// scenarios, 54 passed, 2 failed, and the two failures are the same
		// pair of large-integer-flag assertions the fixture cannot serve.
		// Running both mattered rather than being a formality -- in-process
		// evaluates the JsonLogic rule itself while RPC has flagd evaluate it,
		// so the @targeting scenarios exercise genuinely different code, and
		// agreement between them is evidence rather than duplication. The same
		// goes double for @disabled-flags, where the two resolvers were
		// expected to disagree: in-process reads the state out of the ruleset
		// it synced, RPC gets reason DISABLED back over the wire, and both
		// arrive at the caller's default. See the RPC suite above.
		capabilities: []tck.Capability{
			tck.Events,
			tck.Lifecycle,
			tck.Stale,
			tck.ConfigurationChange,
			tck.Object,
			tck.Variants,
			tck.DisabledFlags,
			tck.Targeting,
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

		// The suite starts this stack once, discovers the host port Docker
		// mapped to suite.backendPort, builds the HTTP control against the
		// launchpad on 8080 and waits until it accepts commands. Scenario
		// isolation comes from the control API, never from restarting a
		// container: mapped host ports do not survive a restart, so a restart
		// would invalidate every provider already pointed at the old one.
		tck.WithComposeFile(composeFile),
		tck.WithBackendPorts(suite.backendPort),

		// Called once per scenario, because the mapped port does not exist
		// until the stack is up.
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

		// Without this the per-suite knownDeviations would be collected and
		// then dropped on the floor, which is the quietest possible way for a
		// conformance report to lose the one field that tells a defect apart
		// from a design choice. It read as wired because the struct field was
		// populated.
		tck.WithKnownDeviations(suite.knownDeviations...),

		tck.WithReadyTimeout(suite.readyTimeout),
		tck.WithEventTimeout(15*time.Second),
	)
}
