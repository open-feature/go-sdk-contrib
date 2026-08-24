package tck

import (
	"strings"
	"testing"
)

// TestFailedCapabilityCarriesAReason exercises the branch no passing suite can.
//
// The schema requires a reason whenever an outcome is not "passed", and the
// self-test suites all pass, so nothing that runs end to end ever builds a failed
// capability entry. That branch was schema-invalid for a while and no test
// noticed, because the only way to reach it is to fail a scenario on purpose --
// which is what this does, by handing the report builder the records directly
// rather than by breaking a provider.
func TestFailedCapabilityCarriesAReason(t *testing.T) {
	caps, err := newCapabilitySet([]Capability{Object, Events})
	if err != nil {
		t.Fatalf("building the capability set: %v", err)
	}

	r := &runner{cfg: Config{Name: "synthetic", Control: stubControl{}}, caps: caps}
	r.records = []scenarioRecord{
		{feature: "errors", name: "a structured flag fails", tags: []string{"@object"},
			outcome: OutcomeFailed, reason: "resolved to nil, expected an object"},
		{feature: "errors", name: "a structured flag succeeds", tags: []string{"@object"},
			outcome: OutcomePassed},
		{feature: "events", name: "ready fires", tags: []string{"@events"},
			outcome: OutcomePassed},
	}

	report := r.buildReport()

	object, present := report.Capabilities[Object.Tag()]
	if !present {
		t.Fatalf("%s is missing from the report", Object.Tag())
	}
	if object.State != OutcomeFailed {
		t.Errorf("%s reported as %q, want %q", Object.Tag(), object.State, OutcomeFailed)
	}
	if object.Reason == "" {
		t.Fatalf("%s failed but carries no reason; the schema rejects a non-passed outcome "+
			"without one, so a report built this way would not validate", Object.Tag())
	}
	// The reason has to be usable, not merely present: a consumer reading a
	// comparison page wants to know how much failed before opening the detail.
	if !strings.Contains(object.Reason, "1 of 2") {
		t.Errorf("%s reason %q does not say how many of how many failed", Object.Tag(), object.Reason)
	}

	events, present := report.Capabilities[Events.Tag()]
	if !present {
		t.Fatalf("%s is missing from the report", Events.Tag())
	}
	if events.State != OutcomePassed {
		t.Errorf("%s reported as %q, want %q; one capability failing must not drag down another",
			Events.Tag(), events.State, OutcomePassed)
	}

	// Every capability the report does mention, other than a pass, must carry a
	// reason -- the same rule the schema enforces, checked here so a change to the
	// builder fails in this package rather than in a validator downstream.
	for tag, result := range report.Capabilities {
		if result.State != OutcomePassed && result.Reason == "" {
			t.Errorf("capability %s is %q with no reason", tag, result.State)
		}
	}
}
