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
// satisfy, each one declares what it supports through Config.Capabilities.
//
// Every capability corresponds to exactly one Gherkin tag. A scenario carrying
// a tag whose capability was not declared is skipped before its first step runs
// and is reported as skipped with the reason printed — never as passed. A
// conformance suite that quietly goes green on scenarios it did not run is
// worse than no suite at all.
//
// Scenarios with no capability tag are mandatory and always run.
//
// A few capabilities are reserved: they are part of the vocabulary but no
// scenario carries their tag, so they must not be declared. See IsReserved.
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
	Lifecycle Capability = "@lifecycle"

	// Stale means the provider enters STALE and emits PROVIDER_STALE when it
	// loses its backend, then returns to READY when it regains it.
	Stale Capability = "@stale"

	// ConfigurationChange means the provider detects flag configuration changes
	// and emits PROVIDER_CONFIGURATION_CHANGED naming the changed flags.
	ConfigurationChange Capability = "@configuration-change"

	// Object means the provider supports structured (object) flag values.
	Object Capability = "@object"

	// UnavailableInit means the provider reports an error state promptly,
	// rather than hanging or panicking, when initialised against a backend it
	// cannot reach.
	UnavailableInit Capability = "@unavailable"

	// NumericCoercion means the provider coerces between the integer and float
	// types only when the coercion is lossless, and reports TYPE_MISMATCH when
	// it would lose information.
	//
	// An integral float such as 10.0 requested as an integer must succeed; 0.5
	// requested as an integer must not. The rule is not "never coerce", which is
	// what this capability was originally named for — see flagd's numeric
	// coercion ADR, https://github.com/open-feature/flagd/issues/1996.
	//
	// Unlike every other entry here this is not an optional feature. The
	// specification requires a provider to report TYPE_MISMATCH when the
	// requested type cannot be satisfied, and narrowing 0.5 to 0 to satisfy an
	// integer request loses information silently — the worst failure mode a
	// feature flag has, because the application sees a plausible value and no
	// error at all.
	//
	// It is a capability only so that a provider with this defect can adopt the
	// suite today and see the gap reported as an explicit skip, rather than
	// being unable to adopt at all. Not declaring it is an admission of a known
	// bug, not a design choice. Declare it as soon as the provider is fixed.
	//
	// Only the lossy half of the rule has a scenario. The canonical flag set
	// contains no integral float to ask the lossless half of, so a provider that
	// wrongly rejects 10.0 as an integer still passes; Appendix F records that as
	// an open gap, because closing it changes the flag set for every language at
	// once. Accessor width is not modelled either — Go's ResolveIntValue is
	// int64, as is the canonical Long, and a 32-bit accessor would need its own
	// scenarios.
	NumericCoercion Capability = "@numeric-coercion"

	// Targeting is reserved and must not be declared — see IsReserved. No
	// scenario carries this tag: targeting is backend evaluation logic, which
	// the TCK deliberately does not test. It exists so the vocabulary stays
	// aligned with the flagd test harness, and so that context-passthrough
	// scenarios have a home once the control API grows an echo endpoint.
	Targeting Capability = "@targeting"

	// Caching is reserved and must not be declared — see IsReserved. No
	// scenario carries this tag yet. Whether a stale provider keeps serving
	// last-known values during an outage depends on whether it holds a local
	// copy of the ruleset.
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
	UnavailableInit,
	NumericCoercion,
	Targeting,
	Caching,
}

// reservedCapabilities is every capability no scenario carries.
//
// It is a single list rather than a property repeated at each use, because the
// rule and the set it applies to have to move together: adding the first
// scenario for one of these means deleting one line here and nothing else.
var reservedCapabilities = []Capability{
	Targeting,
	Caching,
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
// So AllCapabilities does not return one, and naming one in Config.Capabilities
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
// declared. It is the default when Config.Capabilities is unset.
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
// hand out tags nothing tests.
func AllCapabilities() []Capability {
	out := make([]Capability, 0, len(allCapabilities))
	for _, c := range allCapabilities {
		if c.IsReserved() {
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
// The reserved check lives here rather than in Config.validate so that the set
// the conformance report's declaration is built from cannot be constructed with
// a reserved capability in it at all. Config.validate calls this, so an adopter
// still sees the problem reported as a configuration error before any scenario
// runs; the point of putting it here is that there is no second path to a
// declaration that could drift from the rule.
func newCapabilitySet(caps []Capability) (capabilitySet, error) {
	set := make(capabilitySet, len(caps))
	for _, c := range caps {
		if _, known := CapabilityForTag(string(c)); !known {
			return nil, fmt.Errorf(
				"unknown capability %q: capabilities are the constants declared in this package, one of %s",
				c, formatCapabilities(AllCapabilities()))
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
