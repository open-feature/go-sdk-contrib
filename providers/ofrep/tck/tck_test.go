//go:build tck

package tck

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/open-feature/go-sdk-contrib/providers/ofrep"
	"github.com/open-feature/go-sdk-contrib/tools/tck"
	"github.com/open-feature/go-sdk/openfeature"
)

// The OpenFeature Provider Conformance Suite, run against the OFREP provider.
//
// There is no container code here: the suite owns the stack. The backend is the
// unmodified flagd-testbed image, which serves the OFREP API on container port
// 8016 alongside its own protocols and the launchpad control API on 8080 — so
// the provider gets a conformant backend seeded with the canonical flag set
// without a new image and without any change to the flagd suites.
//
// **This suite is currently non-deterministic**, and README.md has the numbers
// and the floor to read a red run against. The cause is not here and no sleep or
// retry is being added to hide it: the launchpad's POST /start returns before
// the flags are evaluable, and a provider with no initialisation to block on
// races that load on every scenario. It is measured and explained in
// open-feature/flagd-testbed#394.

const (
	// composeFile describes the backend stack. It is shared with the other
	// conformance adoptions in this repository so that the image tag cannot
	// drift between suites whose results are only comparable if both answered
	// the same backend. Resolved relative to this package directory, which is
	// where `go test` runs.
	composeFile = "../../../tests/flagd-testbed/docker-compose.yaml"

	// ofrepPort is the container-internal port flagd serves OFREP on, and the
	// only port the provider connects to. The launchpad's control port is
	// exposed by the harness and is not named here.
	ofrepPort = 8016
)

// TestOFREPConformance runs the suite against the OFREP provider pointed at
// flagd's OFREP endpoint.
//
// The name is not load-bearing and nothing asserts it. What keeps a Docker stack
// out of every pull request is the module path and the build tag above, doing
// two different jobs; see README.md and the harness README.
//
// The short-mode skip below is not the exclusion either. It is the one guard
// left for someone who names this package directly and asks for the tag, and it
// stays for that.
func TestOFREPConformance(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e tests in short mode")
	}

	tck.Run(t,
		tck.WithName("ofrep"),

		// The suite starts this stack once, discovers the host port Docker
		// mapped to 8016, builds the HTTP control against the launchpad on 8080
		// and waits until it accepts commands. Scenario isolation comes from
		// the control API, never from restarting a container.
		tck.WithComposeFile(composeFile),
		tck.WithBackendPorts(ofrepPort),

		// Called once per scenario, because the mapped port does not exist
		// until the stack is up.
		tck.WithProviderFromEndpoint(func(_ context.Context, endpoint tck.BackendEndpoint) (openfeature.FeatureProvider, error) {
			// NewProvider never fails: it only builds an http.Client and a
			// base URI, and does not contact the backend. See
			// providers/ofrep/provider.go:25-40.
			//
			// endpoint.Host() rather than a hard-coded "localhost": with a
			// remote Docker daemon, Docker Desktop on some platforms or a
			// rootless setup the host is not localhost.
			//
			// The timeout is well under the TCK's own step deadlines so that a
			// wedged backend surfaces as a resolution error attributable to
			// this provider rather than as a suite-level timeout.
			baseURI := fmt.Sprintf("http://%s:%d", endpoint.Host(), endpoint.Port(ofrepPort))
			return ofrep.NewProvider(baseURI, ofrep.WithTimeout(5*time.Second)), nil
		}),

		// tck.WithUnavailableProvider is deliberately absent, which is the
		// configuration the TCK documents for a provider that cannot declare
		// tck.UnavailableInit. See the capability notes below.

		// Every omission below is a property of the provider's code rather than
		// a preference, and every declaration is evidence from a run.
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
		// would buy one green scenario that asserts nothing.
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
		// which is exactly what the @numeric-coercion scenario asserts. The
		// converse is accepted and returns 10.0, because ResolveFloat takes any
		// float64 (flags.go:141-155) and that is what a JSON 10 decodes to.
		//
		// That the provider arrived at the lossy-round-trip rule from the
		// constraints of JSON, rather than from flagd's numeric coercion ADR
		// the tag is tested against, is some evidence the rule is the natural
		// one rather than a flagd preference.
		//
		// One of the three scenarios cannot be verified against this backend
		// and it is the fixture's fault: flagd-testbed has no
		// integral-float-flag, so that one fails with FLAG_NOT_FOUND whatever
		// the provider does (open-feature/flagd-testbed#392). The other two --
		// the lossy half, and the widening half through integer-flag -- are
		// answered and both pass, which is what Appendix F's first rule for
		// declaring turns on. The failure gets no deviation entry, an entry
		// there being an accusation against the provider for the backend's gap.
		// The same rule decides @large-integers the other way: one scenario,
		// and the testbed serves no flag for it.
		//
		// tck.Variants IS declared. OFREP's evaluation response carries a
		// variant field and this provider passes it straight into
		// ResolutionDetail, so seven of the eight rows pass -- booleans,
		// strings, integers, floats and all three falsy flags. The eighth is
		// large-integer-flag, the same fixture gap.
		//
		// tck.Targeting IS declared, and for a JSON-over-HTTP provider it is
		// the cheapest capability here to get right: the evaluation context IS
		// the request body, so there is no separate passthrough path to get
		// wrong. All three scenarios pass -- targeting-key-flag resolves to
		// "hit" for the matching key and "miss" for a non-matching one or none
		// at all -- and so does the untagged scenario that supplies a context
		// to an untargeted flag.
		//
		// tck.DisabledFlags IS declared, and it is the one capability here
		// that was expected to be impossible. It is gated because a disabled
		// flag's resolution depends on where the caller's default is
		// substituted, and OFREP is the clearest case of a backend that
		// decides -- the request body carries the context and the flag key and
		// nothing else -- so the plan was to leave the tag undeclared and write
		// the architecture down beside it.
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
		// tck.StandardReasons IS declared, and for OFREP it is the thinnest
		// claim of the six: the provider passes the server's reason string
		// straight into ResolutionDetail, so what the suite verifies here is a
		// property of flagd's OFREP endpoint plus this provider's refusal to
		// rewrite it. That is still worth verifying -- a provider that mapped
		// reasons onto an enumeration and dropped the ones it did not know
		// would fail exactly here, and python-sdk-contrib#418 is that bug in
		// another language's OFREP provider.
		//
		// Measured over three runs: all six scenarios pass twice, and in the
		// third two rows of the STATIC outline report reason ERROR -- the
		// launchpad race above wearing another field's clothes, not a
		// vocabulary disagreement. Judge this capability on whether its
		// scenarios fail consistently.
		// tck.StringTyping and tck.FullyTypedValues are BOTH declared. Every
		// typed resolver in internal/evaluate/flags.go type-asserts the decoded
		// value before it returns, so ResolveString on a JSON true, a JSON
		// number or a JSON object fails the assertion and reports TYPE_MISMATCH
		// rather than formatting the value. Nothing in this provider calls
		// fmt.Sprint on a resolved value, which is the mistake the capability
		// exists to find.
		//
		// Spec bda599f1 split the question in two: @string-typing keeps the
		// boolean and integer rows, and the float and structured scenarios
		// carry @fully-typed-values, which asks whether the backend records a
		// native type for those as well. For OFREP it does, and the reason is
		// the wire format rather than anything a particular server chose: JSON
		// distinguishes a number and an object from a string, so both arrive
		// typed and the type survives to the assertion. A backend that stored
		// flag values as text would withhold both tags; one that typed only
		// booleans and integers would declare the first and withhold the
		// second. OFREP is neither, so all four scenarios keep running.
		//
		// The four were mandatory until spec d47a66eb, and this suite was
		// already passing them, so both declarations claim what was already
		// measured rather than adding coverage. Verified when the tag was first
		// registered: undeclared, the skip count rose from 9 to 13, and
		// declaring it put it back at 9 with no new failure.
		tck.WithCapabilities(
			tck.Object,
			tck.NumericCoercion,
			tck.Variants,
			tck.Targeting,
			tck.DisabledFlags,
			tck.StandardReasons,
			tck.StringTyping,
			tck.FullyTypedValues,
		),

		// The provider has no initialisation to wait for, so this bounds the
		// SDK's registration round trip and nothing else.
		tck.WithReadyTimeout(30*time.Second),

		// Unused in practice — every event-bearing scenario is gated behind
		// tck.Events — but set explicitly so the value does not silently change
		// meaning if the provider grows eventing.
		tck.WithEventTimeout(15*time.Second),
	)
}
