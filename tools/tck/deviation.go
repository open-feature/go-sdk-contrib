package tck

import (
	"errors"
	"fmt"
)

// KnownDeviation says one thing: this provider fails to do something it is
// required to do.
//
// [Appendix F's rules] are normative and this type does not restate them; the
// three an adopter trips over are that the requirement must be a numbered MUST
// or a rule the implementation bound itself to elsewhere, that a declared
// capability with a visibly failing scenario is the shape to prefer over a
// withheld one, and that a scenario failing because the backend cannot serve
// its fixture is not a provider defect at all. Check the requirement before
// writing one: a failed scenario is not yet a deviation, and a capability you
// cannot satisfy is not yet a defect.
//
// Distinct from an undeclared capability, which on its own is a choice. The Go
// SDK supplies the clearest illustration, one capability withheld twice for
// different reasons: a provider with no streaming transport does not declare
// ConfigurationChange and is not pretending otherwise, while the SDK's
// memprovider does not declare it because it cannot update its flag set at all
// — which Appendix A requires an SDK's in-memory provider to support, so that
// absence is a defect. Both look identical in the results, so the difference
// has to be stated or a consumer cannot tell a design decision from a defect.
//
// Declared by the provider author through tck.WithKnownDeviations, which is the
// only place that knows the difference. The TCK cannot infer it: from the
// outside, a capability the provider chose to withhold and one it withheld
// because it is broken are the same absence.
//
// It lives on the base rather than with the reporting machinery because it is
// something an adopter writes, alongside tck.WithCapabilities. Whatever reads
// the declaration — a machine-readable conformance report, a build check, a
// human — is downstream of it and does not widen it. The JSON field names are
// here for the same reason, and they are the names the Java TCK's report emits:
// a cross-language consumer should not have to know which language produced a
// report to read it.
//
// [Appendix F's rules]: https://github.com/open-feature/spec/blob/main/specification/appendix-f-provider-conformance.md#rules-for-declaring
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
	// anywhere yet. Optional.
	//
	// Use TrackedDeviation and UntrackedDeviation rather than setting this
	// directly, so that which of the two a deviation is stays a decision
	// someone made rather than a field someone forgot.
	Issue string `json:"issue,omitempty"`

	// Summary is what the gap is, in a form someone comparing providers can
	// use. Required.
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
// Worth declaring even so: naming the defect is what separates it from a
// capability the provider chose to withhold. Prefer TrackedDeviation as soon as
// there is an issue to point at.
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
//
// A capability the Go SDK cannot express is rejected too, and for a different
// reason worth keeping separate: its scenarios do exist and are skipped, but
// they are skipped for every provider in this language regardless of what any
// of them does. A deviation there would attribute a property of the SDK to the
// provider, which is the opposite of what the field is for. See
// inexpressibleCapabilities.
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

		if reason, inexpressible := d.Capability.IsInexpressible(); inexpressible {
			problems = append(problems, fmt.Errorf(
				"tck.WithKnownDeviations deviation %d names %q, which the Go SDK cannot express: %s. Its "+
					"scenarios are skipped for every provider written against this SDK whatever the "+
					"provider does, so a deviation here would record a property of the SDK as a defect "+
					"of your provider. Remove it",
				i, d.Capability, reason))
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
