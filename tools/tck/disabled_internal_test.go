package tck

import (
	"context"
	"errors"
	"testing"

	"github.com/cucumber/godog"
)

// A deviation whose summary opens with "<scenario name>:" disables that
// scenario by exact godog scenario name, unlike a capability skip, which
// follows Gherkin tags.
func TestDisabledScenarioSkipsByExactName(t *testing.T) {
	caps, err := newCapabilitySet([]Capability{Object})
	if err != nil {
		t.Fatalf("newCapabilitySet: %v", err)
	}
	r := &runner{
		caps: caps,
		cfg: config{
			Name:    "gate",
			Control: stubControl{},
			KnownDeviations: []KnownDeviation{
				UntrackedDeviation("", "A falsy value is a value, not an absence: empty keys cannot be seeded"),
			},
		},
	}

	_, hookErr := r.beforeScenario(context.Background(), scenarioWithTags("A falsy value is a value, not an absence"))
	if hookErr == nil {
		t.Fatal("the Before hook ran a scenario a known deviation disables")
	}
	if !errors.Is(hookErr, godog.ErrSkip) {
		t.Fatalf("the Before hook returned %v, which godog would treat as a failure rather than a skip", hookErr)
	}
	if len(r.disabled) != 1 || r.disabled[0].name != "A falsy value is a value, not an absence" {
		t.Fatalf("the disabled scenario was not recorded: %+v", r.disabled)
	}
}

func TestDisabledMatchIsExact(t *testing.T) {
	caps, err := newCapabilitySet([]Capability{Object})
	if err != nil {
		t.Fatalf("newCapabilitySet: %v", err)
	}
	r := &runner{
		caps: caps,
		cfg: config{
			Name:    "gate",
			Control: stubControl{},
			KnownDeviations: []KnownDeviation{
				UntrackedDeviation("", "A falsy value is a value, not an absence: empty keys cannot be seeded"),
			},
		},
	}

	_, hookErr := r.beforeScenario(context.Background(), scenarioWithTags("A falsy value is a value"))
	if hookErr != nil {
		t.Fatalf("the Before hook skipped a scenario that only shares a prefix with a disabled name: %v", hookErr)
	}
	if len(r.disabled) != 0 {
		t.Fatalf("a non-matching scenario was recorded as disabled: %+v", r.disabled)
	}
}

func TestSummaryWithoutColonDisablesNothing(t *testing.T) {
	caps, err := newCapabilitySet([]Capability{Object})
	if err != nil {
		t.Fatalf("newCapabilitySet: %v", err)
	}
	r := &runner{
		caps: caps,
		cfg: config{
			Name:    "gate",
			Control: stubControl{},
			KnownDeviations: []KnownDeviation{
				UntrackedDeviation("", "empty keys cannot be seeded"),
			},
		},
	}

	_, hookErr := r.beforeScenario(context.Background(), scenarioWithTags("A falsy value is a value, not an absence"))
	if hookErr != nil {
		t.Fatalf("the Before hook skipped a scenario for a deviation that names none: %v", hookErr)
	}
	if len(r.disabled) != 0 {
		t.Fatalf("a scenario was recorded as disabled with no name given: %+v", r.disabled)
	}
}

func TestDisabledWinsOverCapabilityGate(t *testing.T) {
	caps, err := newCapabilitySet([]Capability{Object})
	if err != nil {
		t.Fatalf("newCapabilitySet: %v", err)
	}
	r := &runner{
		caps: caps,
		cfg: config{
			Name:    "gate",
			Control: stubControl{},
			KnownDeviations: []KnownDeviation{
				UntrackedDeviation(Stale, "outage: stale is broken here"),
			},
		},
	}

	_, hookErr := r.beforeScenario(context.Background(), scenarioWithTags("outage", "@stale"))
	if hookErr == nil || !errors.Is(hookErr, godog.ErrSkip) {
		t.Fatalf("the Before hook did not skip a disabled scenario: %v", hookErr)
	}
	if len(r.disabled) != 1 {
		t.Fatalf("the disabled scenario was not recorded as disabled: %+v", r.disabled)
	}
	if len(r.skips) != 0 {
		t.Fatalf("a disabled scenario was also recorded as a capability skip: %+v", r.skips)
	}
}

func TestDisabledScenarioNameParsing(t *testing.T) {
	for _, tc := range []struct {
		summary string
		name    string
		ok      bool
	}{
		{summary: "A falsy value is a value, not an absence: empty keys cannot be seeded", name: "A falsy value is a value, not an absence", ok: true},
		{summary: "empty keys cannot be seeded", name: "", ok: false},
		{summary: ": leading colon names nothing", name: "", ok: false},
		{summary: "reason is STATIC: got DEFAULT: two colons keep the first split", name: "reason is STATIC", ok: true},
	} {
		t.Run(tc.summary, func(t *testing.T) {
			name, ok := KnownDeviation{Summary: tc.summary}.DisabledScenario()
			if name != tc.name || ok != tc.ok {
				t.Errorf("DisabledScenario() = %q, %v, want %q, %v", name, ok, tc.name, tc.ok)
			}
		})
	}
}
