//go:build tck

package tck

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
// There is no container code here: the suite owns the stack. README.md has what
// each resolver declares, the tally to read a red run against, and why this is a
// module of its own rather than part of providers/flagd/e2e.

const (
	// composeFile describes the backend stack. It is shared with the other
	// conformance adoptions in this repository so that the image tag cannot
	// drift between suites whose results are only comparable if both answered
	// the same backend. Resolved relative to this package directory, which is
	// where `go test` runs.
	composeFile = "../../../tests/flagd-testbed/docker-compose.yaml"

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

var (
	// The one gap this provider is known to have. It is measured against
	// flagd's own accepted numeric-coercion ADR rather than against the
	// specification, which does not define coercion at all — so a consumer
	// should not read it as a specification violation, and the summary says so.
	//
	// It accompanies a capability that IS declared: the scenario runs, it
	// fails, and this entry says the failure is known and why. See the RPC
	// suite below for the measurement that settled which shape is right.
	numericCoercionDeviation = tck.TrackedDeviation(
		tck.NumericCoercion,
		"https://github.com/open-feature/flagd/issues/1996",
		"The lossy half of the coercion rule is not enforced: evaluating float-flag (0.5) "+
			"through GetIntDetails returns 0 with no error code, rather than TYPE_MISMATCH with "+
			"the code default, so the fractional part is discarded silently. The lossless half "+
			"works and is not the defect: integer-flag (10) requested as a float is widened to 10 "+
			"with reason STATIC and no error code, which is why the capability is declared rather "+
			"than withheld -- this provider coerces, and gets one direction wrong. Both resolvers "+
			"behave identically in both directions, which places it in the shared provider layer "+
			"rather than in either transport. One further @numeric-coercion scenario fails here "+
			"for a reason that is NOT the provider's, and is named so that a reader does not count "+
			"it against flagd: the other lossless scenario asks for integral-float-flag (10.0) as "+
			"an integer, and that flag is absent from every released flagd-testbed, so it fails with "+
			"FLAG_NOT_FOUND (open-feature/flagd-testbed#392). That cost is accepted knowingly "+
			"rather than used as a reason to withhold the tag. The rule is flagd's own accepted "+
			"numeric-coercion ADR rather than a specification requirement -- the specification "+
			"does not define numeric coercion at all (open-feature/spec#430) -- so this is a "+
			"deviation from a commitment flagd made, not from the provider contract.")
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
		// tck.Reinitialization is NOT declared, and that is a choice
		// Requirement 2.5.2 offers rather than a gap, so it gets no
		// knownDeviations entry. Measured: Shutdown clears the provider's own
		// initialised flag, so a second Init proceeds, but what it then waits
		// for never arrives -- the RPC service's event stream never signals
		// ready again -- and Init returns "provider initialization deadline
		// exceeded". Undeclared, "A provider that was shut down can be
		// initialized again" is reported as skipped with its reason, which is
		// the accurate result.
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
		// PROVIDER_ERROR once the retry grace period expires. So the two
		// resolvers of the same provider report an outage differently: an
		// application that switches from in-process to RPC stops receiving
		// stale events. That is worth knowing and is why it is written down
		// here.
		//
		// It is not a conformance defect and gets no knownDeviations entry.
		// Requirement 5.1.1's supporting text offers both behaviours in the
		// same breath: a provider unable to evaluate flags "can" signal that
		// with PROVIDER_ERROR, and a provider that caches rule-sets or
		// evaluations "can" signal PROVIDER_STALE. "Can", twice. The RPC
		// resolver takes the first option and goes to ERROR, which also means
		// it is not quietly serving cached values while disconnected -- the SDK
		// short-circuits to the code default instead. Declare this if and when
		// the RPC resolver emits PROVIDER_STALE.
		//
		// tck.NumericCoercion IS declared, and one of its three scenarios
		// fails. Measured over three full runs, both resolvers, identically:
		//
		//   - "An integer requested as a float is widened without loss" PASSES.
		//     integer-flag (10) through GetFloatDetails returns 10 with reason
		//     STATIC and no error code. So this provider does coerce.
		//   - "A float flag is not silently narrowed to an integer" FAILS.
		//     float-flag (0.5) through GetIntDetails returns 0 with no error
		//     code at all -- not TYPE_MISMATCH with the code default. The
		//     application sees a plausible value and no indication anything
		//     went wrong, which is the worst failure mode a feature flag has.
		//     That is the deviation recorded above.
		//   - "An integral float requested as an integer is coerced without
		//     loss" FAILS, and this one is the backend's: integral-float-flag
		//     is absent from the pinned testbed image, so it fails with
		//     FLAG_NOT_FOUND. Same fixture gap as tck.LargeIntegers below, not
		//     a second provider defect, and the deviation summary says so.
		//
		// So two of the three scenarios can be put to this provider, which is
		// what Appendix F's first rule for declaring turns on, and the widening
		// pass is exactly what distinguishes "does not coerce" from "coerces,
		// and loses information one way round". Both resolvers narrow
		// identically, so the defect is in this provider's shared layer rather
		// than in either transport.
		//
		// tck.LargeIntegers is NOT declared, and this absence is neither a
		// choice nor a provider defect: huge-integer-flag is absent from the
		// pinned testbed image, so the capability cannot be verified against
		// this backend at all. Two scenarios fail for the same reason -- the
		// untagged "A large integer resolves without loss of precision", and
		// the last row of the @variants outline below, which asks
		// large-integer-flag for its max-int32 variant and gets "" because the
		// flag is not there to have one. They are the only failures this suite
		// carries that say nothing whatever about the provider: Go's
		// ResolveIntValue is int64 and has room for both values. Declare this
		// and both failures go away together once flagd-testbed#392 lands. It
		// gets no knownDeviations entry on purpose, because the gap is in the
		// fixture and an entry there would attribute it to the provider. Read
		// it as "the backend has no flag to ask about", not as "Go cannot ask":
		// Go can, and this provider would answer.
		//
		// tck.Variants IS declared, on the evidence of the run rather than on
		// the reasoning that flagd obviously has variants. Seven of the eight
		// rows pass in both resolvers: the variant name survives the trip from
		// the ruleset through the wire format into ResolutionDetail for
		// booleans, strings, integers, floats and all three falsy flags. The
		// eighth is the fixture gap described above, so withholding the tag
		// would skip seven working rows to hide one missing flag.
		//
		// tck.DisabledFlags IS declared, on the evidence of the run, and the
		// run is the only thing that could have settled it. The capability is
		// gated because what a disabled flag resolves to depends on where the
		// substitution happens, and the RPC resolver is on the wrong side of
		// that line by construction -- it asks flagd to resolve every flag --
		// so the honest expectation was that it would fail and the in-process
		// resolver would pass.
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
		// value is taken.
		//
		// All four rows pass in both resolvers. Two verification passes were
		// needed to say so: one earlier run failed this outline with
		// FLAG_NOT_FOUND and failed the object scenario with reason ERROR at
		// the same time, and both went away on re-running. That is the
		// launchpad's start race (open-feature/flagd-testbed#394) and not a
		// property of this outline. A single red run here means re-run before
		// concluding anything.
		//
		// tck.Targeting IS declared. All three scenarios pass in both
		// resolvers, and they assert something this suite could not otherwise
		// see: that the evaluation context reaches the backend at all.
		// targeting-key-flag has one JsonLogic rule on the targeting key, so a
		// matching context resolves to a different value than a non-matching
		// one or none -- which means a provider that silently dropped the
		// context would be caught by the resolved value itself, with no echo
		// endpoint needed. The flag has been in flagd-testbed since v0.5.1, so
		// this needs no image bump -- unlike tck.LargeIntegers above, which is
		// waiting on one.
		//
		// tck.StandardReasons IS declared. flagd reports STATIC for a rule-less
		// flag, TARGETING_MATCH for a matching rule, DEFAULT for a rule that
		// exists and did not match, DISABLED for a disabled flag and ERROR for
		// a failed evaluation -- which is Appendix F's mapping exactly.
		// Measured rather than read off the source: all nine executed rows of
		// reason.feature pass in both resolvers, and because tck.Targeting and
		// tck.DisabledFlags are declared too, all six of its scenarios run.
		capabilities: []tck.Capability{
			tck.Events,
			tck.Lifecycle,
			tck.ConfigurationChange,
			tck.Object,
			tck.NumericCoercion,
			tck.Variants,
			tck.DisabledFlags,
			tck.Targeting,
			tck.StandardReasons,
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

		// Everything except tck.LargeIntegers and tck.Reinitialization.
		//
		// tck.Stale IS declared here, and that is the one real difference
		// between the resolvers: this one emits PROVIDER_STALE on connection
		// loss, so the @stale scenario passes here and fails on RPC. See the
		// RPC suite above.
		//
		// tck.NumericCoercion IS declared here too, and the two resolvers agree
		// in both directions -- observed, not inferred. This one widens
		// integer-flag (10) to 10.0 correctly and narrows float-flag (0.5) to 0
		// on an integer request exactly as the RPC resolver does, which places
		// the defect in the shared provider layer and is why one deviation
		// covers both suites. See the RPC suite above, and for
		// tck.LargeIntegers, which the testbed cannot exercise at all.
		//
		// tck.Lifecycle holds here for the same reason it does on RPC, and more
		// visibly: the in-process resolver syncs the whole ruleset before
		// reporting ready, so initialisation is unambiguously doing work.
		//
		// tck.Reinitialization is NOT declared here either, for the reason the
		// RPC suite gives, and measured the same way: a second Init waits for a
		// sync that never completes and times out.
		//
		// tck.Variants, tck.Targeting, tck.DisabledFlags and tck.StandardReasons
		// are declared here as well, and both resolvers produce the identical
		// result: 65 scenarios, 61 passed, 4 failed, the same four in both.
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
			tck.NumericCoercion,
			tck.Variants,
			tck.DisabledFlags,
			tck.Targeting,
			tck.StandardReasons,
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

// runConformance is where both suites above come.
//
// What keeps a Docker stack out of every pull request is the module path and the
// build tag above, doing two different jobs; neither is this function's name,
// and nothing asserts it. See README.md and the harness README.
//
// The short-mode skip below is not the exclusion either. It is the one guard
// left for someone who names this package directly and asks for the tag, and it
// stays for that.
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
		// container.
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
		tck.WithKnownDeviations(suite.knownDeviations...),

		tck.WithReadyTimeout(suite.readyTimeout),
		tck.WithEventTimeout(15*time.Second),
	)
}
