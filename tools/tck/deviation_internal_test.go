package tck

import (
	"context"
	"strings"
	"testing"

	"github.com/open-feature/go-sdk/openfeature"
)

// A KnownDeviation is the only thing that distinguishes a capability withheld
// because the provider does not have it from one withheld because it is
// broken. The results cannot: both are a skip carrying the same reason. So
// what these tests pin is that a deviation says something — an entry that
// records a defect without describing it is worth less than the skip it
// accompanies — and that it cannot be about a tag no scenario carries.

// deviationConfig is a configuration that validates, so that a test naming one bad
// deviation sees that deviation's error and not a pile of unrelated ones.
func deviationConfig(deviations ...KnownDeviation) *config {
	return newConfig([]Option{
		WithName("deviation"),
		WithControl(stubControl{}),
		WithProvider(func(context.Context) (openfeature.FeatureProvider, error) {
			return openfeature.NoopProvider{}, nil
		}),
		WithCapabilities(Object),
		WithKnownDeviations(deviations...),
	})
}

func TestTheDeviationConstructorsDifferOnlyInWhetherTheGapIsTracked(t *testing.T) {
	const issue = "https://github.com/open-feature/flagd/issues/1996"

	tracked := TrackedDeviation(NumericCoercion, issue, "narrows 0.5 to 0 with no error code")
	if tracked.Capability != NumericCoercion {
		t.Errorf("TrackedDeviation kept capability %q, want %q", tracked.Capability, NumericCoercion)
	}
	if tracked.Issue != issue {
		t.Errorf("TrackedDeviation kept issue %q, want %q", tracked.Issue, issue)
	}
	if tracked.Summary != "narrows 0.5 to 0 with no error code" {
		t.Errorf("TrackedDeviation kept summary %q", tracked.Summary)
	}
	if !tracked.IsTracked() {
		t.Error("a deviation carrying an issue reports IsTracked false")
	}

	untracked := UntrackedDeviation(Lifecycle, "cannot be initialised again after shutdown")
	if untracked.Capability != Lifecycle {
		t.Errorf("UntrackedDeviation kept capability %q, want %q", untracked.Capability, Lifecycle)
	}
	if untracked.Issue != "" {
		t.Errorf("UntrackedDeviation invented the issue %q", untracked.Issue)
	}
	if untracked.IsTracked() {
		t.Error("a deviation carrying no issue reports IsTracked true")
	}
}

func TestValidateRejectsADeviationThatDoesNotSayWhatTheGapIs(t *testing.T) {
	cfg := deviationConfig(UntrackedDeviation(NumericCoercion, ""))

	err := cfg.validate()
	if err == nil {
		t.Fatal("validate accepted a deviation with no summary; the entry would record that " +
			"something is broken without saying what")
	}
	if !strings.Contains(err.Error(), "Summary") {
		t.Errorf("the error does not mention Summary: %v", err)
	}
}

// A reserved capability is carried by no scenario, so nothing was skipped for a
// deviation about it to explain. Declaring one is already rejected; a deviation
// naming one is the same mistake arriving through the other half of the
// declaration vocabulary, and it has to be caught in both.
func TestValidateRejectsADeviationNamingAReservedCapability(t *testing.T) {
	for _, reserved := range reservedCapabilities {
		cfg := deviationConfig(UntrackedDeviation(reserved, "something is wrong here"))

		err := cfg.validate()
		if err == nil {
			t.Errorf("validate accepted a deviation naming the reserved capability %s", reserved)
			continue
		}
		if !strings.Contains(err.Error(), string(reserved)) {
			t.Errorf("the error for %s does not name it: %v", reserved, err)
		}
	}
}

func TestValidateRejectsADeviationNamingAnUnknownCapability(t *testing.T) {
	cfg := deviationConfig(UntrackedDeviation("@invented", "something is wrong here"))

	err := cfg.validate()
	if err == nil {
		t.Fatal("validate accepted a deviation naming a capability the vocabulary does not have")
	}
	if !strings.Contains(err.Error(), "@invented") {
		t.Errorf("the error does not name the unknown capability: %v", err)
	}
}

// The two shapes that are legitimate and must not be rejected.
//
// A deviation with no capability is a gap against a mandatory scenario, which
// belongs to no capability and so can name none. A deviation naming a declared
// capability is the case where the capability holds but one of the scenarios it
// gates does not — which is worth recording precisely because the scenario may
// pass for the wrong reason and hide it.
func TestValidateAcceptsTheLegitimateDeviationShapes(t *testing.T) {
	for _, tc := range []struct {
		name      string
		deviation KnownDeviation
	}{
		{
			name:      "against a mandatory scenario, so no capability",
			deviation: UntrackedDeviation("", "resolves large-integer-flag through a float and rounds it"),
		},
		{
			name:      "against a scenario under a capability that is declared",
			deviation: UntrackedDeviation(Object, "returns object values with their members reordered"),
		},
		{
			name:      "tracked, against a capability that is withheld",
			deviation: TrackedDeviation(NumericCoercion, "https://example.invalid/1", "narrows 0.5 to 0"),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := deviationConfig(tc.deviation).validate(); err != nil {
				t.Errorf("validate rejected a legitimate deviation: %v", err)
			}
		})
	}
}

// Silence is not a claim, and must not be an error either: a provider with no
// known gaps declares nothing.
func TestValidateAcceptsNoDeviationsAtAll(t *testing.T) {
	if err := deviationConfig().validate(); err != nil {
		t.Errorf("validate rejected a config declaring no deviations: %v", err)
	}
}
