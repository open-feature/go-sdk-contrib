package tck

import (
	"errors"
	"fmt"
)

// KnownDeviation says one thing: this provider fails to do something it is
// required to do.
//
// The requirement has to be a numbered MUST, or a rule the implementation bound
// itself to elsewhere — flagd measured against its own accepted numeric-coercion
// ADR is the worked example. Where the specification permits the choice,
// withholding the capability *is* the honest report, and a deviation entry would
// assert a defect that does not exist. So check the requirement before writing
// one: a failed scenario is not yet a deviation, and a capability you cannot
// satisfy is not yet a defect.
//
// Distinct from an undeclared capability, which on its own is a choice. A
// provider that does not declare ConfigurationChange has no streaming transport
// and is not pretending otherwise; a provider that does not declare
// NumericCoercion because it narrows 0.5 to 0 with no error code has a bug. Both
// look identical in the results — scenarios skipped, reason recoverable from the
// declaration — so the difference has to be stated, or a consumer cannot tell a
// design decision from a defect.
//
// Declared by the provider author through tck.WithKnownDeviations, which is the
// only place that knows the difference. The TCK cannot infer it: from the
// outside, a capability the provider chose to withhold and one it withheld
// because it is broken are the same absence.
//
// # The two legitimate shapes
//
// A report's results already distinguish them.
//
//  1. The capability is declared, the scenario runs, and it fails. Prefer this.
//     The failure stays visible and the deviation says it is known and why.
//  2. The capability is withheld, and its scenarios skip. Legitimate only when
//     the provider cannot attempt the behaviour at all, so running the scenario
//     would establish nothing. The deviation then explains the absence, so a
//     reader can tell a defect from a design decision.
//
// Withdrawing a capability *in order to* turn a failing scenario into a skip is
// the failure mode this field exists to prevent. If the provider attempts the
// behaviour and gets it wrong, shape 1 is the honest report.
//
// Under shape 1 a deviation names a capability that is declared, which is also
// how to record that the capability holds while one of the scenarios it gates
// does not — worth saying precisely because the mode a provider runs in may
// decide whether that scenario fails or passes for the wrong reason, and a
// passing scenario hides it.
//
// Part of the declaration vocabulary rather than of any one consumer of it.
// This is something an adopter writes, alongside tck.WithCapabilities, so it
// belongs to the suite an adopter adopts. Whatever reads the declaration — a
// machine-readable conformance report, a build check, a human — is downstream
// of it and does not widen it.
//
// The JSON field names live here for the same reason. A deviation is meant to
// be read by whoever compares providers across languages, so the wire form is
// part of what the type means rather than a choice each consumer makes; these
// are the names the Java TCK's report emits, and a cross-language consumer
// should not have to know which language produced a report to read it.
type KnownDeviation struct {
	// Capability is the capability the gap is against: declared, with the
	// scenario failing visibly, which is the preferred shape; or withheld,
	// when the provider cannot attempt the behaviour at all and the scenarios
	// skip.
	//
	// Empty when the gap is against a mandatory, ungated scenario and so
	// belongs to no capability. Never a reserved capability: no scenario
	// carries that tag, so there is nothing to deviate from.
	Capability Capability `json:"capability,omitempty"`

	// Issue is a URI where the gap is tracked, empty when it is not tracked
	// anywhere yet. Optional — there is a tracked and an untracked form, and
	// naming an untracked defect is still what separates it from a choice.
	// Prefer the tracked form as soon as there is an issue.
	//
	// Use TrackedDeviation and UntrackedDeviation rather than setting this
	// directly, so that which of the two a deviation is stays a decision
	// someone made rather than a field someone forgot.
	Issue string `json:"issue,omitempty"`

	// Summary is what the gap is, in a form someone comparing providers can
	// use. Required: a deviation whose summary is empty records that something
	// is wrong without saying what, which is worth less than the skip it is
	// trying to explain.
	Summary string `json:"summary"`
}

// TrackedDeviation records a deviation that is tracked somewhere.
//
// Pass an empty Capability when the gap is against a mandatory scenario and so
// belongs to no capability.
func TrackedDeviation(capability Capability, issue, summary string) KnownDeviation {
	return KnownDeviation{Capability: capability, Issue: issue, Summary: summary}
}

// UntrackedDeviation records a deviation that is not tracked anywhere yet.
//
// Worth declaring even so. Naming the defect is what separates it from a
// capability the provider chose to withhold, and a declaration that merely
// omits the tag cannot say which of the two happened. Prefer TrackedDeviation
// as soon as there is an issue to point at.
//
// Pass an empty Capability when the gap is against a mandatory scenario and so
// belongs to no capability.
func UntrackedDeviation(capability Capability, summary string) KnownDeviation {
	return KnownDeviation{Capability: capability, Summary: summary}
}

// IsTracked reports whether this deviation points at somewhere the gap is
// tracked.
func (d KnownDeviation) IsTracked() bool { return d.Issue != "" }

// validateDeviations reports whether the declared deviations say anything a
// consumer can use, naming what is wrong rather than emitting a report that
// records a defect without describing it.
//
// The rules are deliberately narrow. A deviation is prose written by the
// provider author for a human comparing providers, and the TCK cannot check
// prose; what it can check is that the thing is not empty and that the
// capability it names exists. A reserved capability is rejected for the same
// reason declaring one is: no scenario carries the tag, so there is no skip for
// the deviation to explain and nothing it could be about.
func validateDeviations(deviations []KnownDeviation) error {
	var problems []error

	for i, d := range deviations {
		if d.Summary == "" {
			problems = append(problems, fmt.Errorf(
				"tck.WithKnownDeviations deviation %d has no Summary: a deviation exists to say what the gap is, and "+
					"one that does not say it leaves a consumer no better off than the bare skip it "+
					"accompanies", i))
		}

		if d.Capability == "" {
			// Legitimate: the gap is against a mandatory scenario, which
			// belongs to no capability.
			continue
		}

		if _, known := CapabilityForTag(string(d.Capability)); !known {
			problems = append(problems, fmt.Errorf(
				"tck.WithKnownDeviations deviation %d names unknown capability %q: capabilities are the constants "+
					"declared in this package, one of %s",
				i, d.Capability, formatCapabilities(AllCapabilities())))
			continue
		}

		if d.Capability.IsReserved() {
			problems = append(problems, fmt.Errorf(
				"tck.WithKnownDeviations deviation %d names the reserved capability %q: no scenario carries that tag, "+
					"so nothing was skipped for this deviation to explain and no result could show "+
					"the gap. Remove it, or name the capability whose scenarios the gap actually "+
					"affects",
				i, d.Capability))
		}
	}

	return errors.Join(problems...)
}
