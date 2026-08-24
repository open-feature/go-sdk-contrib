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

	// StrictNumericTyping means the provider keeps the integer and float types
	// distinct instead of coercing between them.
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
	StrictNumericTyping Capability = "@strict-numeric-typing"

	// Targeting is reserved. No scenario carries this tag: targeting is backend
	// evaluation logic, which the TCK deliberately does not test. It exists so
	// the vocabulary stays aligned with the flagd test harness, and so that
	// context-passthrough scenarios have a home once the control API grows an
	// echo endpoint.
	Targeting Capability = "@targeting"

	// Caching is reserved; no scenario carries this tag yet. Whether a stale
	// provider keeps serving last-known values during an outage depends on
	// whether it holds a local copy of the ruleset.
	Caching Capability = "@caching"
)

// allCapabilities is every capability the TCK knows about. A Gherkin tag that
// is not in this list gates nothing and is ignored, which is what lets the
// canonical feature files carry organisational tags freely.
var allCapabilities = []Capability{
	Events,
	Lifecycle,
	Stale,
	ConfigurationChange,
	Object,
	UnavailableInit,
	StrictNumericTyping,
	Targeting,
	Caching,
}

// AllCapabilities returns every capability the TCK recognises.
//
// It is a reasonable starting point for a new adoption: declare everything, run
// the suite, and remove only what your provider genuinely cannot do. Narrowing
// from the full set surfaces gaps; widening towards it hides them until
// something fails for an apparently unrelated reason.
func AllCapabilities() []Capability {
	out := make([]Capability, len(allCapabilities))
	copy(out, allCapabilities)
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
// Exported because a conformance report is read by things outside this package:
// deciding whether a scenario was skipped legitimately means knowing which of
// its tags gate a capability and which are merely organisational.
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

func newCapabilitySet(caps []Capability) (capabilitySet, error) {
	set := make(capabilitySet, len(caps))
	for _, c := range caps {
		if _, known := CapabilityForTag(string(c)); !known {
			return nil, fmt.Errorf(
				"unknown capability %q: capabilities are the constants declared in this package, one of %s",
				c, formatCapabilities(allCapabilities))
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
