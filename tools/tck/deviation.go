package tck

import (
	"errors"
	"fmt"
	"strings"
)

// KnownDeviation says one thing: this provider fails to do something it is
// required to do.
//
// Check the requirement before writing one — a failed scenario is not yet a
// deviation, and a capability you cannot satisfy is not yet a defect.
// [Appendix F's rules] are normative and this type does not restate them.
//
// It is distinct from an undeclared capability, which on its own is a choice. A
// provider with no streaming transport does not declare ConfigurationChange and
// is not pretending otherwise; one that withholds the same capability because
// it is broken has a defect. Both look identical in the results, so the
// difference has to be stated by the provider author through
// tck.WithKnownDeviations — the TCK cannot infer it.
//
// It lives on the base rather than with the reporting machinery because it is
// something an adopter writes, alongside tck.WithCapabilities. The JSON field
// names are here for the same reason, and match the other languages' reports so
// that a cross-language consumer need not know which language produced one.
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
	//
	// A summary of the form "<scenario name>: <explanation>" additionally
	// disables the named scenario: it is skipped rather than run to
	// failure, and reported as disabled. The name is the exact godog
	// scenario name. Outline rows share one scenario name, so naming an
	// outline disables every row it has — re-assert the rows that do pass
	// as vendor scenarios under WithFeatures when that matters. A summary
	// without a colon disables nothing.
	Summary string `json:"summary"`
}

// DisabledScenario reports the scenario this deviation disables, if any.
//
// See Summary for the "<scenario name>: <explanation>" convention.
func (d KnownDeviation) DisabledScenario() (string, bool) {
	idx := strings.Index(d.Summary, ":")
	if idx <= 0 {
		return "", false
	}
	return d.Summary[:idx], true
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
// The rules are deliberately narrow: a deviation is prose written for a human
// comparing providers, and all the TCK can check is that it is not empty and
// that the capability it names exists.
//
// A capability the Go SDK cannot express is rejected, for a reason worth
// keeping separate from the reserved case: its scenarios do exist and are
// skipped, but they are skipped for every provider in this language regardless
// of what any of them does, so a deviation there would attribute a property of
// the SDK to the provider. See inexpressibleCapabilities.
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
