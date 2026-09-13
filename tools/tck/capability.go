package tck

import (
	"fmt"
	"sort"
	"strings"
)

// Capability is an optional part of the OpenFeature provider contract that a
// provider may or may not support.
//
// Not every provider implements every part of the specification. A provider
// backed by a static file has no meaningful notion of going stale; a provider
// with no streaming transport cannot emit configuration-change events. Rather
// than forcing such providers to fail scenarios they were never going to
// satisfy, each one declares what it supports through tck.WithCapabilities.
//
// Every capability corresponds to exactly one Gherkin tag. A scenario carrying
// a tag whose capability was not declared is skipped before its first step runs
// and is reported as skipped with the reason printed — never as passed. A
// conformance suite that quietly goes green on scenarios it did not run is
// worse than no suite at all.
//
// Scenarios with no capability tag are mandatory and always run.
//
// Two kinds of capability cannot be declared, and the suite refuses both rather
// than leaving them to adopters. One capability is reserved: it is part of the
// vocabulary but no scenario carries its tag. See IsReserved. A capability the
// language's SDK cannot express is refused separately and for a different
// reason -- its scenarios exist and pass elsewhere, but no provider here can be
// asked. Go has none of those; see IsInexpressible.
type Capability string

const (
	// Events means the provider emits lifecycle events at all — at minimum
	// PROVIDER_READY once it has reached its backend.
	Events Capability = "@events"

	// Lifecycle means the provider performs an initialisation that reaches its
	// backend, with an observable outcome: it becomes READY when the backend
	// answers and settles into ERROR when it does not.
	//
	// It is deliberately separate from Events, and conflating the two is the
	// mistake this vocabulary exists to prevent — the tag was wrong in both
	// directions before it existed.
	//
	// Too lax, because declaring Events was enough to run the readiness
	// scenario and the Go SDK synthesises PROVIDER_READY for any provider that
	// does not implement openfeature.StateHandler, on the reasoning that "a
	// provider without state handling capability can be assumed to be ready
	// immediately". A provider with no initialisation therefore passed the
	// scenario without demonstrating anything: a NoopProvider passes it
	// identically. That is a green result for a claim never tested, which is
	// exactly the failure mode this suite is built to make impossible.
	//
	// Too strict, because a stateless HTTP provider such as OFREP resolves
	// every flag over the wire, emits no events of its own and so cannot
	// declare Events — yet the readiness scenario is not really about events,
	// and withholding Events skipped it for the wrong reason. Such a provider
	// declares neither, and the lifecycle scenarios are reported as skipped
	// rather than passing vacuously.
	//
	// Declare it when the provider has an initialisation whose outcome the
	// client observes — in Go, when it implements openfeature.StateHandler and
	// Init can fail. A provider that does not implement StateHandler cannot
	// have it, however promptly its client reports READY, because the readiness
	// it reports was manufactured by the SDK rather than by the provider.
	//
	// Whether such a provider can be started again after being shut down is a
	// separate question the specification leaves to it, with its own
	// capability: see Reinitialization.
	Lifecycle Capability = "@lifecycle"

	// Stale means the provider enters STALE and emits PROVIDER_STALE when it
	// loses its backend, then returns to READY when it regains it.
	Stale Capability = "@stale"

	// ConfigurationChange means the provider detects flag configuration changes
	// and emits PROVIDER_CONFIGURATION_CHANGED naming the changed flags.
	ConfigurationChange Capability = "@configuration-change"

	// Object means the provider supports structured (object) flag values.
	Object Capability = "@object"

	// Variants means the provider names the variant it resolved.
	//
	// Requirement 2.2.4 is a SHOULD — in normal execution a provider "SHOULD
	// populate the resolution details structure's variant field" — and types.md
	// types the field "variant (string, optional)". The same section goes
	// further and says the value "might only be meaningful in the context of
	// the flag management system associated with the provider".
	//
	// So a backend with no variant concept for a plain flag is not defective.
	// Its evaluation response carries no such key, the provider never receives
	// one, and no amount of seeding can produce one. Until this capability
	// existed every evaluation scenario asserted a variant, which failed such a
	// provider ten times over for something its author could not fix — and left
	// nothing to record as a known deviation, because there was no capability
	// to hang one on.
	//
	// Declare it when the backend names its variants and the provider passes
	// the name through. Withholding it skips the variant scenarios with that
	// reason; the value and reason assertions are untagged and unaffected,
	// because 2.2.3 makes the value a MUST.
	//
	// The reason field is modelled the same way, for the same reason and one
	// more: 2.2.5 is also a SHOULD, and it goes further than 2.2.4 by letting a
	// provider populate reason with one of the listed values "or some other
	// string". See StandardReasons.
	Variants Capability = "@variants"

	// DisabledFlags means the provider resolves a flag that is disabled in the
	// flag management system to the caller's default value, with no error.
	//
	// Gated, because the answer depends on where the substitution happens
	// rather than on provider quality. A provider that evaluates locally — the
	// flagd in-process resolver, an in-memory flag set — holds the flag's state
	// and can hand back the value the caller passed in. A provider whose
	// backend decides, and OFREP is the clear case, cannot: the caller's
	// default never leaves the process, the server has never seen it, and no
	// response it could send would carry it. The same flag cannot behave the
	// same way across those two architectures and neither of them is wrong,
	// which is precisely what a capability is for.
	//
	// Nothing in the specification says what a provider owes a disabled flag.
	// Requirement 1.4.7 is about the SDK propagating whatever reason arrived,
	// and 2.2.5 only lists DISABLED among the reason strings a provider may
	// use. So the behaviour is stated by Appendix F, the way @numeric-coercion
	// is, and gated rather than made mandatory.
	//
	// The four scenarios assert the value and the absence of an error, and
	// deliberately not the reason: each row's caller default differs from the
	// flag's configured value, so a provider that ignores the state returns the
	// configured value and is caught on the value alone, which rests on 2.2.3,
	// a MUST. Pinning reason "DISABLED" there would rest on 2.2.5, a SHOULD
	// that permits "some other string"; it is pinned in reason.feature instead,
	// which composes this capability with StandardReasons so that a provider
	// opts into the narrowing rather than inheriting it.
	//
	// It does not compose with Variants, and that is not an omission: a
	// disabled flag has resolved no variant, so there is no name for the
	// resolution to carry and nothing for a variant assertion to be about.
	//
	// Declare it when a disabled flag comes back as the code default with no
	// error code. The Go SDK's memprovider.InMemoryProvider does not manage
	// that — it returns the default value but attaches a GENERAL resolution
	// error alongside reason DISABLED, so the error-code assertion fails —
	// which is why none of the in-memory self-tests declares this. See
	// TestCanonicalFlagSetDisabledFlagsCarryAnError, which is the evidence for
	// that and will fail if the SDK stops doing it.
	DisabledFlags Capability = "@disabled-flags"

	// UnavailableInit means the provider reports an error state promptly,
	// rather than hanging or panicking, when initialised against a backend it
	// cannot reach.
	UnavailableInit Capability = "@unavailable"

	// NumericCoercion means the provider coerces between the integer and float
	// types only when the coercion is lossless, and reports TYPE_MISMATCH when
	// it would lose information.
	//
	// An integral float such as 10.0 requested as an integer must succeed and
	// 10 requested as a float must widen; 0.5 requested as an integer must not.
	// The rule is not "never coerce", which is what this capability was
	// originally named for — see flagd's numeric coercion ADR,
	// https://github.com/open-feature/flagd/issues/1996.
	//
	// The specification does not define this rule. OpenFeature has a single
	// numeric type, and whether a language tells integers from floats is an
	// idiom, so the rule is borrowed from the ADR rather than normative, and a
	// provider that behaves differently is not violating the specification
	// (open-feature/spec#430). What is true regardless is that narrowing 0.5 to
	// 0 with no error code is the worst failure mode a feature flag has: the
	// application sees a plausible value and no signal. A provider withholding
	// this capability should say whether that is a choice or a tracked defect.
	//
	// All three directions have a scenario, and a provider declaring the tag
	// has to satisfy every one of them: rejecting every float passes the lossy
	// scenario alone, and the two lossless ones are what stop that shortcut.
	// The SDK's memprovider.InMemoryProvider is exactly such a provider — it
	// type-asserts and never converts between int64 and float64 — which is why
	// the in-memory self-tests do not declare this.
	//
	// Accessor width is a separate property, of the SDK rather than of the
	// provider, with its own capability: see LargeIntegers.
	//
	// Whether the question can be asked at all is a property of the language:
	// JavaScript has a single numeric type, so "this float, as an integer?" is
	// not expressible there and its suites refuse the declaration. Go has two
	// accessors over int64 and float64, which is why it is declarable here --
	// measured by TestTheIntegerAndFloatAccessorsAreDistinctTypes rather than
	// asserted. See inexpressibleCapabilities.
	NumericCoercion Capability = "@numeric-coercion"

	// LargeIntegers means the provider resolves integers up to 2^53-1 exactly.
	//
	// Whether such a value can be asked for at all is a property of the
	// language's SDK rather than of the provider: Java's integer accessor is a
	// 32-bit Integer and has no room for it, so a provider there cannot declare
	// this and its suite refuses the declaration rather than leaving the fact to
	// adopters -- see inexpressibleCapabilities. Go's ResolveIntValue is int64,
	// measured by TestTheIntegerAccessorIsWideEnoughToAskForALargeInteger rather
	// than assumed, so every Go provider can ask;
	// what it declares here is that the value survives the trip, which anything
	// routed through a 32-bit integer, or through a float and back with
	// rounding, does not. Nothing above 2^53-1 is asked for, JavaScript being
	// unable to represent it. The 32-bit maximum has an untagged scenario of
	// its own, because every language can ask for that.
	LargeIntegers Capability = "@large-integers"

	// Reinitialization means the provider can be initialised again after it has
	// been shut down, and serves flags afterwards.
	//
	// Requirement 2.5.2 says a provider SHOULD revert to its uninitialized
	// state after shutdown, and its supporting text says "some providers MAY
	// allow reinitialization from this state". Reuse is therefore permitted,
	// not required: a provider that releases its client on shutdown and
	// declines to start again is taking an option the specification offers it,
	// not exhibiting a defect, and it needs no known-deviation entry to say so.
	// Simply leave the tag undeclared.
	//
	// The scenario was mandatory until spec fc99d5ac, on the reading that
	// reverting to the uninitialized state "is observable as exactly one thing
	// -- it can be initialized again and then serves flags". That inference
	// does not hold, and the cost was concrete: flagd's RPC resolver failed the
	// scenario and the failure was one step from being filed as a defect
	// against a provider doing nothing wrong. A false failure is the mirror
	// image of a vacuous pass.
	//
	// What the tag buys is the other direction. A provider that does offer
	// reuse has somewhere to be held to it, because "Shutdown releases the
	// client and Init returns early because an initialised flag was never
	// cleared" is easy to write and leaves the provider evaluating against a
	// closed connection rather than failing outright. Reverting the state is
	// not separately observable -- a provider that reverts but refuses reuse
	// presents exactly as one that did neither -- so a gated reuse scenario is
	// the only assertion the requirement admits.
	//
	// It is separate from Lifecycle because the scenario's feature carries
	// @lifecycle too: a provider with no observable initialisation is not being
	// asked this question at all, and one that has it may still decline reuse.
	Reinitialization Capability = "@reinitialization"

	// Targeting means the provider resolves a flag differently for a matching
	// evaluation context: the context reaches the backend and the rule there is
	// evaluated against it.
	//
	// It was reserved and undeclarable until spec 26362f85, on the reasoning
	// that asserting anything about targeting needed an echo endpoint on the
	// control API. It does not. The canonical set now carries exactly one flag
	// with a rule, targeting-key-flag, whose matching context resolves to a
	// different value — so a provider that drops the context is caught by the
	// resolved value itself.
	//
	// What is under test is the provider's passthrough, not the backend's rule
	// language. The rule is specified by behaviour rather than syntax — resolve
	// "hit" when the targeting key is exactly the canonical uuid, "miss"
	// otherwise — so a backend expresses it however it expresses targeting.
	// Three scenarios: a match, a non-match, and no context at all. The
	// non-match is what stops a provider passing by always serving the targeted
	// variant, and the third pins down that an unmatchable rule must not error.
	//
	// Declare it when the backend evaluates targeting rules at all. A provider
	// whose backend has no notion of them — an in-memory flag set, an
	// environment-variable provider — leaves it undeclared, and the three
	// scenarios are skipped with that reason rather than failed. Whether the
	// *whole* context arrives intact, attributes included, is still unverified
	// in every language; see Appendix F's known gaps.
	Targeting Capability = "@targeting"

	// StandardReasons means the provider reports the standard resolution
	// reasons, with the meanings Appendix F gives them.
	//
	// It is a claim, not an exemption. Requirement 2.2.5 is a SHOULD, and it
	// goes further than 2.2.4 does: it lets a provider populate reason with one
	// of the listed values "or some other string indicating the semantic reason
	// for the returned flag value". A provider whose backend reports
	// vendor-specific reasons is therefore conformant, and asserting an exact
	// reason against it would fail it for something the specification permits.
	//
	// An earlier revision of the suite did exactly that, in thirteen places
	// across three feature files. It bought very little: every canonical flag
	// resolves to a value distinct from the caller's default, so a provider
	// that silently falls back is already caught by the value assertion, and
	// the reason only said why it failed. The reasons now live in
	// reason.feature, gated as a whole -- declaring this capability is a
	// provider saying "I use the standard vocabulary with the standard
	// meanings", and that file is what checks the claim.
	//
	// A provider that does not declare it loses nothing: its values, variants
	// and error codes are asserted everywhere else, on MUST requirements. What
	// the declaration adds is something a report's reader can act on -- anyone
	// building telemetry, dashboards or debugging on reason can see that the
	// vocabulary was verified rather than assumed.
	//
	// The meanings are the content of the claim, and constrain nobody who does
	// not make it:
	//
	//	resolved from configuration, no targeting rule   STATIC
	//	a targeting rule matched the evaluation context  TARGETING_MATCH
	//	a targeting rule exists and did not match        DEFAULT
	//	the flag is disabled in the management system    DISABLED
	//	the evaluation failed, with an error code        ERROR
	//
	// STATIC for the first row is the call worth flagging. types.md types
	// DEFAULT as "no dynamic evaluation occurred or dynamic evaluation yielded
	// no result", which a rule-less flag satisfies as readily as STATIC does --
	// two providers can disagree here and both conform. A provider that answers
	// DEFAULT for a rule-less flag is not defective; it does not use the
	// standard meanings, and should not declare the tag.
	//
	// ERROR is the row where the suite's subject is blurred, and it is asserted
	// anyway. The other four rest on Requirement 1.4.7, which makes the SDK
	// propagate the provider's reason -- but only "in cases of normal
	// execution". Abnormal execution is 1.4.9, a SHOULD on the SDK, and nothing
	// requires the provider's reason to survive. So a passing ERROR scenario
	// establishes that the value reaching the application is coherent, not that
	// the provider produced it. It is still worth asserting: the error code
	// alone is already covered for every provider by errors.feature on a MUST,
	// the reason alone could have been written by the SDK, and an evaluation
	// reporting FLAG_NOT_FOUND with reason STATIC is incoherent whoever wrote
	// it.
	//
	// SPLIT, UNKNOWN, CACHED and STALE are not asserted. The first two have no
	// scenario that produces them; CACHED needs a repeat evaluation, which
	// nothing here performs without a configuration change in between (see
	// Caching); and STALE needs a scenario asserting what a provider serves
	// during an outage, which is the same gap.
	//
	// Tags compose, and here that is load-bearing. TARGETING_MATCH cannot be
	// observed without targeting and DISABLED cannot be observed unless the
	// backend distinguishes a disabled flag, so those scenarios carry
	// @targeting and @disabled-flags as well. A provider declaring this alone
	// runs the rest and skips those two with their reason.
	StandardReasons Capability = "@standard-reasons"

	// Caching is reserved and must not be declared — see IsReserved. No
	// scenario carries this tag yet, and it is the only reserved tag left.
	// Whether a stale provider keeps serving last-known values during an outage
	// depends on whether it holds a local copy of the ruleset.
	Caching Capability = "@caching"
)

// allCapabilities is every capability the TCK knows about, reserved ones
// included. A Gherkin tag that is not in this list gates nothing and is
// ignored, which is what lets the canonical feature files carry organisational
// tags freely.
var allCapabilities = []Capability{
	Events,
	Lifecycle,
	Stale,
	ConfigurationChange,
	Object,
	Variants,
	DisabledFlags,
	UnavailableInit,
	NumericCoercion,
	LargeIntegers,
	Reinitialization,
	Targeting,
	StandardReasons,
	Caching,
}

// reservedCapabilities is every capability no scenario carries.
//
// It is a single list rather than a property repeated at each use, because the
// rule and the set it applies to have to move together: adding the first
// scenario for one of these means deleting one line here and nothing else.
// Targeting left it exactly that way when spec 26362f85 gave it scenarios.
var reservedCapabilities = []Capability{
	Caching,
}

// inexpressibleCapabilities names every capability whose question this
// language's SDK cannot put to a provider, mapped to the property of the SDK
// that prevents it.
//
// **It is empty, and that is a measurement rather than an assumption.** Go can
// express both of the capabilities that are inexpressible somewhere else, and
// each is checked by a test rather than argued from the SDK's source:
//
//   - @large-integers, which is inexpressible in Java because its integer
//     accessor is a 32-bit Integer with no room for 2^53-1. Go's is int64 --
//     see TestTheIntegerAccessorIsWideEnoughToAskForALargeInteger, and the
//     scenario runs and passes in all three self-test suites.
//   - @numeric-coercion, which is inexpressible in JavaScript because the
//     language has one numeric type, so "a float requested as an integer" is
//     not a question its API can ask. Go has genuinely distinct accessors --
//     see TestTheIntegerAndFloatAccessorsAreDistinctTypes -- and the OFREP
//     provider declares the capability and satisfies all three scenarios.
//
// Appendix F requires the refusal below to exist whether or not a language has
// an instance today, and this map is where a future one goes: adding a line
// here is the whole change, and the message, the default set, the deviation
// check and the skip reason all follow from it.
//
// A capability listed here is refused for a different reason than a reserved
// one, and the two must stay distinguishable because they say different things
// to a reader of a report:
//
//	               | reserved                      | inexpressible
//	why            | no scenario carries the tag   | scenarios exist and pass elsewhere
//	scope          | every language                | this language only
//	lifetime       | until the specification adds  | until the SDK changes
//	               | scenarios                     |
//	what it says   | nothing about any provider    | nothing about any provider *here*
//
// A capability absent from a report has to be readable as either "this provider
// declined" or "no provider in this language can be asked", because only the
// first says anything about the provider. Collapsing the two refusals into one
// predicate or one message destroys that distinction, so they are kept apart
// deliberately -- see newCapabilitySet and runner.beforeScenario.
//
// It is a var rather than a const map so that the tests can install an entry
// and exercise a path Go itself never takes. That is the point of testing it:
// an unused refusal is discovered by the first adopter who needs it, and being
// discovered that way means somebody already published a claim no scenario
// could have verified.
var inexpressibleCapabilities = map[Capability]string{}

// IsInexpressible reports whether this language's SDK can put the question this
// capability is about to a provider at all, returning the property of the SDK
// that prevents it when it cannot.
//
// It is not a judgement about the provider and it is not a reservation. The
// scenarios exist, they are carried by the canonical Gherkin, and they pass for
// providers in other languages; what is missing is an API through which any
// provider here could be asked. Java's integer accessor is a 32-bit Integer, so
// 2^53-1 cannot be asked for; JavaScript has a single numeric type, so "a float
// requested as an integer" cannot be expressed. Nothing a provider does changes
// either.
//
// Go has none: the second return is always false. See inexpressibleCapabilities
// for the evidence and for what to do when that stops being true.
func (c Capability) IsInexpressible() (string, bool) {
	reason, inexpressible := inexpressibleCapabilities[c]
	return reason, inexpressible
}

// IsReserved reports whether this capability is reserved: part of the
// vocabulary, carried by no scenario, and therefore not declarable.
//
// A reserved capability exists so the vocabulary has a place for it once
// scenarios do, but Appendix F states that it must not be declared and must not
// appear in a conformance report's declaration. Nothing carries the tag, so
// declaring it cannot be verified, cannot produce a skip, and tells a reader of
// the report only that something was claimed and nothing examined — the vacuous
// conformance claim this whole vocabulary exists to prevent.
//
// So AllCapabilities does not return one, and naming one in tck.WithCapabilities
// is a configuration error rather than a conformance result.
func (c Capability) IsReserved() bool {
	for _, reserved := range reservedCapabilities {
		if c == reserved {
			return true
		}
	}
	return false
}

// AllCapabilities returns every capability the TCK recognises **except the
// reserved ones**, which no scenario carries and which therefore must not be
// declared, **and the ones the Go SDK cannot express**, which no provider here
// could satisfy however it is written. It is the default when
// tck.WithCapabilities is unset.
//
// It is a reasonable starting point for a new adoption: declare everything, run
// the suite, and remove only what your provider genuinely cannot do. Narrowing
// from the full set surfaces gaps; widening towards it hides them until
// something fails for an apparently unrelated reason.
//
// Reserved capabilities are excluded because this is the convenience through
// which they get claimed by accident. An adoption that starts here and removes
// what it cannot do picks up every reserved tag on the way past, and a
// published conformance report then asserts capabilities that were never
// examined — which is how a Java report came to declare @targeting and
// @caching, not by anyone's decision. A declare-everything shortcut must not
// hand out tags nothing tests. That report named @targeting and @caching, back
// when both were reserved; only @caching still is.
//
// Inexpressible capabilities are excluded for the same reason and by a separate
// rule -- see inexpressibleCapabilities. Go has none, so today this returns
// exactly the non-reserved set; the exclusion is here so that adding one is a
// single line in one place rather than a fact every adopter has to know.
func AllCapabilities() []Capability {
	out := make([]Capability, 0, len(allCapabilities))
	for _, c := range allCapabilities {
		if c.IsReserved() {
			continue
		}
		if _, inexpressible := c.IsInexpressible(); inexpressible {
			continue
		}
		out = append(out, c)
	}
	return out
}

// Tag returns the Gherkin tag, including the leading at-sign, that gates this
// capability.
func (c Capability) Tag() string { return string(c) }

// String implements fmt.Stringer.
func (c Capability) String() string { return string(c) }

// CapabilityForTag maps a Gherkin tag onto the capability it gates, reporting
// whether the tag gates anything at all.
//
// Exported because a caller outside this package has to tell a capability-gating
// tag from a merely organisational one -- deciding whether a scenario was skipped
// legitimately is exactly that question. Conformance reporting is the case that
// needs it, and it is exported here rather than there so that adding reporting
// widens no API: the suite works without it, and the branch that adds it should
// only add.
func CapabilityForTag(tag string) (Capability, bool) {
	for _, c := range allCapabilities {
		if string(c) == tag {
			return c, true
		}
	}
	return "", false
}

// capabilitySet is a declared capability set, in lookup form.
type capabilitySet map[Capability]struct{}

// newCapabilitySet turns a declared capability list into lookup form, rejecting
// anything that cannot legitimately be declared.
//
// The reserved check lives here rather than in config.validate so that the set
// the conformance report's declaration is built from cannot be constructed with
// a reserved capability in it at all. config.validate calls this, so an adopter
// still sees the problem reported as a configuration error before any scenario
// runs; the point of putting it here is that there is no second path to a
// declaration that could drift from the rule. The inexpressibility check is
// here for the same reason and says a different thing -- see
// inexpressibleCapabilities.
func newCapabilitySet(caps []Capability) (capabilitySet, error) {
	set := make(capabilitySet, len(caps))
	for _, c := range caps {
		if _, known := CapabilityForTag(string(c)); !known {
			return nil, fmt.Errorf(
				"unknown capability %q: capabilities are the constants declared in this package, one of %s",
				c, formatCapabilities(AllCapabilities()))
		}
		if reason, inexpressible := c.IsInexpressible(); inexpressible {
			return nil, fmt.Errorf(
				"capability %q cannot be expressed by the Go SDK and so cannot be declared by any "+
					"provider written against it: %s. This is not a judgement about your provider "+
					"and it is not the reserved-capability rule -- the scenarios exist and pass in "+
					"other languages, and no provider here can be asked, so declaring it would put "+
					"a claim in a conformance report that no scenario could verify. Remove it; the "+
					"declarable capabilities are %s",
				c, reason, formatCapabilities(AllCapabilities()))
		}
		if c.IsReserved() {
			return nil, fmt.Errorf(
				"capability %q is reserved and cannot be declared: no scenario carries that tag, so "+
					"declaring it cannot be verified and cannot even produce a skip — a conformance "+
					"report saying it was declared would claim something nothing examined. Remove it; "+
					"the declarable capabilities are %s",
				c, formatCapabilities(AllCapabilities()))
		}
		set[c] = struct{}{}
	}
	return set, nil
}

func (s capabilitySet) has(c Capability) bool {
	_, ok := s[c]
	return ok
}

// sorted returns the declared capabilities in a stable order, for messages a
// human reads.
func (s capabilitySet) sorted() []Capability {
	out := make([]Capability, 0, len(s))
	for c := range s {
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func formatCapabilities(caps []Capability) string {
	parts := make([]string, len(caps))
	for i, c := range caps {
		parts[i] = string(c)
	}
	return "[" + strings.Join(parts, " ") + "]"
}
