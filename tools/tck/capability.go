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
// [Appendix F] owns the vocabulary, what each tag means and the rules for
// deciding whether to declare one; the comments below add what is specific to
// Go. Two kinds of capability cannot be declared: see [Capability.IsReserved]
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
	// was synthesised by the SDK — a NoopProvider passes the scenario
	// identically.
	//
	// Whether such a provider can be started again after being shut down is a
	// separate question: see Reinitialization.
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
	// Declare it when the backend names its variants and the provider passes
	// the name through. Withholding it skips the variant scenarios with that
	// reason and leaves the value assertions untouched. Optional; Appendix F
	// explains why.
	Variants Capability = "@variants"

	// DisabledFlags means the provider resolves a flag that is disabled in the
	// flag management system to the caller's default value, with no error.
	//
	// Declare it when a disabled flag comes back as the code default with no
	// error code. The Go SDK's memprovider.InMemoryProvider attaches a GENERAL
	// resolution error alongside reason DISABLED, so the error-code assertion
	// fails and none of the in-memory self-tests declares this — measured by
	// TestCanonicalFlagSetDisabledFlagsCarryAnError, which will fail if the SDK
	// stops doing it. Optional, and deliberately not composed with Variants;
	// Appendix F explains why.
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
	// The Go SDK's memprovider.InMemoryProvider type-asserts and never converts
	// between int64 and float64, which is why the in-memory self-tests do not
	// declare this. Declarable in Go because the integer and float accessors
	// are genuinely distinct types — measured by
	// TestTheIntegerAndFloatAccessorsAreDistinctTypes rather than assumed.
	// Optional; Appendix F explains why, and what a provider that coerces and
	// gets one direction wrong should do instead of withholding.
	//
	// Accessor width is a separate property: see LargeIntegers.
	NumericCoercion Capability = "@numeric-coercion"

	// LargeIntegers means the provider resolves integers up to 2^53-1 exactly.
	//
	// Whether such a value can be asked for at all is a property of the
	// language's SDK rather than of the provider. Go's ResolveIntValue is int64
	// — measured by TestTheIntegerAccessorIsWideEnoughToAskForALargeInteger
	// rather than assumed — so every Go provider can be asked, and what it
	// declares here is that the value survives the trip, which anything routed
	// through a 32-bit integer, or through a float and back with rounding, does
	// not.
	LargeIntegers Capability = "@large-integers"

	// StringTyping means the provider reports TYPE_MISMATCH when a boolean or
	// integer flag is requested through the string accessor, rather than
	// returning the value's string representation.
	//
	// Declare it when the backend records a boolean as a boolean and an integer
	// as an integer, and the provider type-asserts rather than formats.
	// Withhold it when the backend stores everything as text. Optional;
	// Appendix F explains why a provider over an untyped backend is not thereby
	// non-conformant.
	//
	// The float and structured cases carry FullyTypedValues as well.
	StringTyping Capability = "@string-typing"

	// FullyTypedValues means the backend records a native type for float and
	// structured values too, so the StringTyping question can be asked of them
	// as well.
	//
	// It is never declared alone: the scenarios carrying it carry @string-typing
	// too, so it widens StringTyping's question rather than asking a new one.
	// Declare it when the backend has a float type and a structure type and the
	// provider type-asserts on both. Appendix F explains why it is separate
	// from StringTyping.
	FullyTypedValues Capability = "@fully-typed-values"

	// Reinitialization means the provider can be initialised again after it has
	// been shut down, and serves flags afterwards.
	//
	// A provider that releases its client on shutdown and declines to start
	// again leaves the tag undeclared and needs no known-deviation entry.
	// Optional; Appendix F explains why.
	//
	// Separate from Lifecycle because the scenario's feature carries @lifecycle
	// too: a provider with no observable initialisation is not being asked this
	// question at all, and one that has it may still decline reuse.
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
	// Declare it when the backend evaluates targeting rules at all.
	Targeting Capability = "@targeting"

	// StandardReasons means the provider reports the standard resolution
	// reasons, with the meanings Appendix F gives them.
	//
	// It is a claim, not an exemption: a provider whose backend reports
	// vendor-specific reasons is conformant and simply does not declare this.
	// Appendix F carries the reason-by-reason mapping the claim is checked
	// against, and which reasons are deliberately not asserted.
	StandardReasons Capability = "@standard-reasons"

	// Caching is reserved and must not be declared — see IsReserved.
	Caching Capability = "@caching"
)

// allCapabilities is every capability the TCK knows about, reserved ones
// included. It is the vocabulary, and being absent from it means different
// things in the two places a tag can come from.
//
// On an extension scenario an unknown tag gates nothing and is ignored, which
// is what lets an adopter's own feature files carry organisational tags freely.
//
// On a canonical scenario it fails the run: ignoring it is not the harmless
// default it looks like, because the tag then gates nothing, its scenarios stay
// mandatory for every adopter, and a provider that legitimately withholds the
// new capability shows unexplained failures. See unknownCapabilityTag in run.go.
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
	FullyTypedValues,
	Reinitialization,
	Targeting,
	StandardReasons,
	Caching,
}

// reservedCapabilities is every capability no scenario carries.
//
// It is a single list rather than a property repeated at each use, so that
// adding the first scenario for one of these means deleting one line here and
// nothing else.
var reservedCapabilities = []Capability{
	Caching,
}

// inexpressibleCapabilities names every capability whose question this
// language's SDK cannot put to a provider, mapped to the property of the SDK
// that prevents it.
//
// It is empty, and that is a measurement rather than an assumption: Go can
// express both of the capabilities that are inexpressible somewhere else, each
// checked by a test rather than argued from the SDK's source —
// TestTheIntegerAccessorIsWideEnoughToAskForALargeInteger for @large-integers,
// TestTheIntegerAndFloatAccessorsAreDistinctTypes for @numeric-coercion.
//
// Adding a line here is the whole change for a future one: the message, the
// default set, the deviation check and the skip reason all follow from it. See
// newCapabilitySet and runner.beforeScenario for the two refusals being kept
// apart deliberately.
//
// It is a var rather than a const map so that the tests can install an entry
// and exercise a path Go itself never takes; an untested refusal is first
// discovered by the adopter who needs it, by which point a claim no scenario
// could verify has already been published.
var inexpressibleCapabilities = map[Capability]string{}

// IsInexpressible reports whether this language's SDK can put the question this
// capability is about to a provider at all, returning the property of the SDK
// that prevents it when it cannot.
//
// It is not a judgement about the provider and it is not a reservation: the
// scenarios exist and pass for providers in other languages; what is missing is
// an API through which any provider here could be asked. Go has none, so the
// second return is always false.
func (c Capability) IsInexpressible() (string, bool) {
	reason, inexpressible := inexpressibleCapabilities[c]
	return reason, inexpressible
}

// IsReserved reports whether this capability is reserved: part of the
// vocabulary, carried by no scenario, and therefore not declarable.
//
// A reserved capability is not returned by AllCapabilities, and naming one in
// tck.WithCapabilities is a configuration error rather than a conformance
// result.
func (c Capability) IsReserved() bool {
	for _, reserved := range reservedCapabilities {
		if c == reserved {
			return true
		}
	}
	return false
}

// AllCapabilities returns every capability the TCK recognises except the
// reserved ones, which no scenario carries, and the ones the Go SDK cannot
// express, which no provider here could satisfy however it is written. It is
// the default when tck.WithCapabilities is unset.
//
// It is a reasonable starting point for a new adoption: declare everything, run
// the suite, and remove only what your provider genuinely cannot do.
//
// The reserved exclusion matters because this is the convenience through which
// a reserved tag gets claimed by accident: an adoption that starts from
// everything and removes what it cannot do picks the tag up on the way past,
// and the published report then asserts a capability nothing examined.
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
// Exported because a caller outside this package has to tell a
// capability-gating tag from a merely organisational one — deciding whether a
// scenario was skipped legitimately is exactly that question.
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
// The checks live here rather than in config.validate so that the set the
// conformance report's declaration is built from cannot be constructed with an
// undeclarable capability in it at all; there is no second path to a
// declaration that could drift from the rule. config.validate calls this, so an
// adopter still sees the problem reported as a configuration error before any
// scenario runs.
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
