package tck

import (
	"fmt"
	"sort"
	"strings"
)

// Capability is an optional part of the OpenFeature provider contract that a
// provider may or may not support.
//
// Each corresponds to exactly one Gherkin tag. An adopter declares the ones its
// provider supports through tck.WithCapabilities; a scenario carrying an
// undeclared tag is skipped before its first step runs and reported as skipped
// with the reason, never as passed. Scenarios with no capability tag are
// mandatory and always run.
//
// The vocabulary, what each tag means and the rules for deciding whether to
// declare one belong to [Appendix F]; the comments below add only what is
// specific to Go. Two kinds of capability cannot be declared, and this package
// refuses both rather than leaving them to adopters: see [Capability.IsReserved]
// and [Capability.IsInexpressible].
//
// [Appendix F]: https://github.com/open-feature/spec/blob/main/specification/appendix-f-provider-conformance.md#capabilities-how-a-provider-says-what-it-cannot-do
type Capability string

const (
	// Events means the provider emits lifecycle events at all — at minimum
	// PROVIDER_READY once it has reached its backend.
	Events Capability = "@events"

	// Lifecycle means the provider performs an initialisation that reaches its
	// backend, with an observable outcome: it becomes READY when the backend
	// answers and settles into ERROR when it does not.
	//
	// In Go, declare it when the provider implements openfeature.StateHandler
	// and Init can fail. A provider that does not implement StateHandler cannot
	// have it however promptly its client reports READY, because that readiness
	// was synthesised by the SDK rather than produced by the provider — a
	// NoopProvider passes the scenario identically. Appendix F has why this is
	// deliberately separate from Events.
	//
	// Whether such a provider can be started again after being shut down is a
	// separate question with its own capability: see Reinitialization.
	Lifecycle Capability = "@lifecycle"

	// Stale means the provider enters STALE and emits PROVIDER_STALE when it
	// loses its backend, then returns to READY when it regains it.
	Stale Capability = "@stale"

	// ConfigurationChange means the provider detects flag configuration changes
	// and emits PROVIDER_CONFIGURATION_CHANGED naming the changed flags.
	ConfigurationChange Capability = "@configuration-change"

	// Object means the provider supports structured (object) flag values.
	Object Capability = "@object"

	// Variants means the provider names the variant it resolved, which
	// Requirement 2.2.4 makes a SHOULD and types.md types as optional.
	//
	// Declare it when the backend names its variants and the provider passes the
	// name through. Withholding it skips the variant scenarios with that reason
	// and leaves the value assertions untouched, 2.2.3 making the value a MUST.
	// The reason field is modelled the same way, for the same reason and one
	// more: see StandardReasons.
	Variants Capability = "@variants"

	// DisabledFlags means the provider resolves a flag that is disabled in the
	// flag management system to the caller's default value, with no error.
	//
	// It is gated because the answer turns on where the caller's default is
	// substituted rather than on provider quality, and it deliberately does not
	// compose with Variants. Appendix F has both arguments, and why the
	// scenarios assert the value and the absence of an error rather than the
	// reason.
	//
	// Declare it when a disabled flag comes back as the code default with no
	// error code. The Go SDK's memprovider.InMemoryProvider does not manage that
	// — it returns the default value but attaches a GENERAL resolution error
	// alongside reason DISABLED, so the error-code assertion fails — which is
	// why none of the in-memory self-tests declares this. See
	// TestCanonicalFlagSetDisabledFlagsCarryAnError, which is the evidence for
	// that and will fail if the SDK stops doing it.
	DisabledFlags Capability = "@disabled-flags"

	// UnavailableInit means the provider reports an error state promptly,
	// rather than hanging or panicking, when initialised against a backend it
	// cannot reach.
	UnavailableInit Capability = "@unavailable"

	// NumericCoercion means the provider coerces between the integer and float
	// types only when the coercion is lossless, and reports TYPE_MISMATCH when
	// it would lose information. An integral float such as 10.0 requested as an
	// integer must succeed and 10 requested as a float must widen; 0.5 requested
	// as an integer must not.
	//
	// The rule is borrowed from flagd's numeric coercion ADR rather than
	// normative: the specification does not define numeric coercion, and a
	// provider that behaves differently is not violating it. Appendix F carries
	// that argument, and the one an adopter actually needs — a provider that
	// coerces and gets one direction wrong declares the capability and records a
	// KnownDeviation beside the failing scenario, rather than withholding.
	//
	// The Go SDK's memprovider.InMemoryProvider type-asserts and never converts
	// between int64 and float64, which is why the in-memory self-tests do not
	// declare this.
	//
	// It is declarable in Go because the integer and float accessors are
	// genuinely distinct types — measured by
	// TestTheIntegerAndFloatAccessorsAreDistinctTypes rather than asserted. A
	// language with a single numeric type cannot put the question at all; see
	// inexpressibleCapabilities. Accessor width is a separate property with its
	// own capability: see LargeIntegers.
	NumericCoercion Capability = "@numeric-coercion"

	// LargeIntegers means the provider resolves integers up to 2^53-1 exactly.
	//
	// Whether such a value can be asked for at all is a property of the
	// language's SDK rather than of the provider. Go's ResolveIntValue is int64
	// — measured by TestTheIntegerAccessorIsWideEnoughToAskForALargeInteger
	// rather than assumed — so every Go provider can be asked, and what it
	// declares here is that the value survives the trip, which anything routed
	// through a 32-bit integer, or through a float and back with rounding, does
	// not. Where the accessor is narrower the capability is inexpressible; see
	// inexpressibleCapabilities.
	LargeIntegers Capability = "@large-integers"

	// StringTyping means the provider reports TYPE_MISMATCH when a non-string
	// flag is requested through the string accessor, rather than returning the
	// value's string representation.
	//
	// It is gated because every value has a string representation, so a backend
	// that stores flag values as strings satisfies the string accessor for
	// every flag and has no mismatch to report: Requirement 2.2.3 asks it for
	// the resolved flag value, and a string is what it holds. The only
	// normative statement nearby is Requirement 1.3.4, a SHOULD on the *client*
	// rather than on the provider, so a provider over an untyped backend
	// withholds this tag and is not thereby non-conformant. Appendix F carries
	// the argument; it is optional for the same reason NumericCoercion is.
	//
	// Declare it when the backend distinguishes a string from a boolean, a
	// number and a structure, and the provider type-asserts rather than
	// formats. Withhold it when the backend stores everything as text --
	// Flagsmith's feature_state_value is the worked example, natively boolean,
	// integer or string only, which is why the Go Flagsmith adoption withholds
	// it and the flagd and OFREP adoptions do not.
	//
	// The three scenarios it gates ask for boolean-flag, integer-flag and
	// float-flag as strings. A fourth asks for object-flag and carries @object
	// too, a provider with no structured values having no way to be asked the
	// question at all; declaring StringTyping without Object skips that one and
	// runs the other three.
	//
	// It is expressible in Go: Client.StringValueDetails takes and returns a
	// string, distinct from every other accessor, so the request can be made.
	StringTyping Capability = "@string-typing"

	// Reinitialization means the provider can be initialised again after it has
	// been shut down, and serves flags afterwards.
	//
	// Requirement 2.5.2 permits reuse rather than requiring it, so a provider
	// that releases its client on shutdown and declines to start again is taking
	// an option the specification offers it. Leave the tag undeclared; it needs
	// no known-deviation entry. Appendix F records why the scenario is gated
	// rather than mandatory, and what the tag buys for a provider that does
	// offer reuse.
	//
	// It is separate from Lifecycle because the scenario's feature carries
	// @lifecycle too: a provider with no observable initialisation is not being
	// asked this question at all, and one that has it may still decline reuse.
	Reinitialization Capability = "@reinitialization"

	// Targeting means the provider resolves a flag differently for a matching
	// evaluation context: the context reaches the backend and the rule there is
	// evaluated against it.
	//
	// What is under test is the provider's passthrough, not the backend's rule
	// language — the canonical set's one targeted flag is specified by behaviour
	// rather than syntax, so a backend expresses it however it expresses
	// targeting.
	//
	// Declare it when the backend evaluates targeting rules at all. A provider
	// whose backend has no notion of them — an in-memory flag set, an
	// environment-variable provider — leaves it undeclared, and the three
	// scenarios are skipped with that reason rather than failed.
	Targeting Capability = "@targeting"

	// StandardReasons means the provider reports the standard resolution
	// reasons, with the meanings Appendix F gives them.
	//
	// It is a claim, not an exemption. Requirement 2.2.5 is a SHOULD that lets a
	// provider populate reason with "some other string indicating the semantic
	// reason for the returned flag value", so a provider whose backend reports
	// vendor-specific reasons is conformant and simply does not declare this.
	// It loses nothing by that: its values, variants and error codes are
	// asserted elsewhere, on MUST requirements.
	//
	// Appendix F carries the reason-by-reason mapping the claim is checked
	// against, which reasons are deliberately not asserted, and why STATIC for a
	// rule-less flag is the call worth knowing about before declaring.
	StandardReasons Capability = "@standard-reasons"

	// Caching is reserved and must not be declared — see IsReserved. No
	// scenario carries this tag yet, and it is the only reserved tag left.
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
	StringTyping,
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
// that prevents it. Appendix F requires the refusal to exist whether or not a
// language has an instance today, and says why it must stay distinguishable
// from a reserved capability.
//
// **It is empty, and that is a measurement rather than an assumption.** Go can
// express both of the capabilities that are inexpressible somewhere else, and
// each is checked by a test rather than argued from the SDK's source:
//
//   - @large-integers, which a 32-bit integer accessor has no room for. Go's is
//     int64 -- see TestTheIntegerAccessorIsWideEnoughToAskForALargeInteger, and
//     the scenario runs and passes in all three self-test suites.
//   - @numeric-coercion, which a single numeric type cannot express, "a float
//     requested as an integer" not being a question its API can ask. Go has
//     genuinely distinct accessors -- see
//     TestTheIntegerAndFloatAccessorsAreDistinctTypes -- and the OFREP provider
//     declares the capability and satisfies all three scenarios.
//
// This map is where a future one goes: adding a line here is the whole change,
// and the message, the default set, the deviation check and the skip reason all
// follow from it. See newCapabilitySet and runner.beforeScenario for the two
// refusals being kept apart deliberately.
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
// It is not a judgement about the provider and it is not a reservation: the
// scenarios exist, are carried by the canonical Gherkin, and pass for providers
// in other languages; what is missing is an API through which any provider here
// could be asked.
//
// Go has none: the second return is always false. See
// inexpressibleCapabilities for the evidence and for what to do when that stops
// being true.
func (c Capability) IsInexpressible() (string, bool) {
	reason, inexpressible := inexpressibleCapabilities[c]
	return reason, inexpressible
}

// IsReserved reports whether this capability is reserved: part of the
// vocabulary, carried by no scenario, and therefore not declarable.
//
// Appendix F states that a reserved capability must not be declared and must
// not appear in a conformance report's declaration. So AllCapabilities does not
// return one, and naming one in tck.WithCapabilities is a configuration error
// rather than a conformance result.
func (c Capability) IsReserved() bool {
	for _, reserved := range reservedCapabilities {
		if c == reserved {
			return true
		}
	}
	return false
}

// AllCapabilities returns every capability the TCK recognises **except the
// reserved ones**, which no scenario carries, **and the ones the Go SDK cannot
// express**, which no provider here could satisfy however it is written. It is
// the default when tck.WithCapabilities is unset.
//
// It is a reasonable starting point for a new adoption: declare everything, run
// the suite, and remove only what your provider genuinely cannot do. Narrowing
// from the full set surfaces gaps; widening towards it hides them until
// something fails for an apparently unrelated reason.
//
// Both exclusions are Appendix F's, and the reserved one exists because this is
// the convenience through which a reserved tag gets claimed by accident: an
// adoption that starts from everything and removes what it cannot do picks the
// tag up on the way past, and the published report then asserts a capability
// nothing examined. Go has no inexpressible capability today, so the second
// exclusion changes nothing here; it is in place so that adding one is a single
// line in one file rather than a fact every adopter has to know.
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
