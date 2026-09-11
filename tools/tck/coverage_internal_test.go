package tck

import (
	"strings"
	"testing"

	messages "github.com/cucumber/messages/go/v21"
)

// The guard's job is to make a partial run impossible to mistake for a
// conformance run. Its two halves are tested separately: what the canonical set
// is, and what happens when a run does not cover it.

func TestCanonicalExpectationIsTheWholeEmbeddedSet(t *testing.T) {
	expected, err := canonicalExpectation()
	if err != nil {
		t.Fatalf("canonicalExpectation: %v", err)
	}

	features := map[string]bool{}
	total := 0
	for key, count := range expected {
		if !isCanonicalFeature(key.uri) {
			t.Errorf("the expectation includes %s, which is not a canonical feature", key.uri)
		}
		if key.name == "" {
			t.Errorf("a scenario in %s has no name, so it cannot be matched to a result", key.uri)
		}
		if count < 1 {
			t.Errorf("%s is expected %d times", key, count)
		}
		features[key.uri] = true
		total += count
	}

	// Every embedded feature file has to be represented. A file that stopped
	// being parsed would silently shrink what the guard demands, which is the
	// one way this check could fail open.
	for _, name := range []string{"errors", "evaluation", "events", "lifecycle"} {
		uri := featuresPath + "/" + name + ".feature"
		if !features[uri] {
			t.Errorf("no scenario expected from %s", uri)
		}
	}

	// A Scenario Outline contributes one entry per Examples row, which is what
	// makes a single missing row visible.
	outline := canonicalScenario{
		uri:  featuresPath + "/errors.feature",
		name: "Requesting the wrong type returns the code default",
	}
	if expected[outline] < 2 {
		t.Errorf("the type-mismatch outline is expected %d times; its Examples rows are not "+
			"being counted individually", expected[outline])
	}

	t.Logf("the canonical set is %d scenarios across %d features", total, len(features))
}

// completeRun is what a full run of the canonical set looks like to the guard.
func completeRun(t *testing.T) (map[canonicalScenario]int, []executedScenario) {
	t.Helper()

	expected, err := canonicalExpectation()
	if err != nil {
		t.Fatalf("canonicalExpectation: %v", err)
	}

	var executed []executedScenario
	for key, count := range expected {
		for range count {
			executed = append(executed, executedScenario{
				uri:    key.uri,
				name:   key.name,
				status: messages.TestStepResultStatus_PASSED,
			})
		}
	}
	return expected, executed
}

func TestCompleteRunHasNoGaps(t *testing.T) {
	expected, executed := completeRun(t)

	gaps, canonical, extensions := canonicalGaps(expected, executed)
	if len(gaps) != 0 {
		t.Errorf("a complete run reported gaps:\n%s", strings.Join(gaps, "\n"))
	}
	if canonical != len(executed) {
		t.Errorf("the canonical set is %d scenarios but the guard counted %d", len(executed), canonical)
	}
	if extensions != 0 {
		t.Errorf("a run with no extensions counted %d extension scenarios", extensions)
	}
}

// TestASkippedScenarioStillCounts keeps the guard off the capability gate's
// territory.
//
// A capability-gated scenario ran the gate and is in the results as SKIPPED with
// its reason. The question was put and declined, which is a legitimate outcome
// and not a gap; treating it as one would force every provider to declare every
// capability.
func TestASkippedScenarioStillCounts(t *testing.T) {
	expected, executed := completeRun(t)
	for i := range executed {
		executed[i].status = messages.TestStepResultStatus_SKIPPED
	}

	if gaps, _, _ := canonicalGaps(expected, executed); len(gaps) != 0 {
		t.Errorf("a run in which every scenario was skipped for an undeclared capability "+
			"reported gaps:\n%s", strings.Join(gaps, "\n"))
	}
}

func TestAnExcludedScenarioIsAGap(t *testing.T) {
	expected, executed := completeRun(t)

	dropped := executed[0]
	gaps, _, _ := canonicalGaps(expected, executed[1:])

	if len(gaps) != 1 {
		t.Fatalf("dropping one scenario produced %d gaps:\n%s", len(gaps), strings.Join(gaps, "\n"))
	}
	if !strings.Contains(gaps[0], dropped.name) || !strings.Contains(gaps[0], dropped.uri) {
		t.Errorf("the gap does not name the scenario that did not run: %s", gaps[0])
	}
}

// TestAnAnnouncedButUnexecutedScenarioIsAGap is the `go test -run` shape.
//
// The scenario is in the stream with a test case of its own and no step result
// in it, which without this check reads as a well-formed report of a run that
// never happened.
func TestAnAnnouncedButUnexecutedScenarioIsAGap(t *testing.T) {
	expected, executed := completeRun(t)
	executed[0].status = messages.TestStepResultStatus_UNKNOWN

	gaps, _, _ := canonicalGaps(expected, executed)
	if len(gaps) != 1 {
		t.Fatalf("a scenario with no outcome produced %d gaps:\n%s", len(gaps), strings.Join(gaps, "\n"))
	}
	if !strings.Contains(gaps[0], "announced but never run") {
		t.Errorf("the gap does not distinguish a scenario that never ran from one that was "+
			"never parsed: %s", gaps[0])
	}
}

// TestExtensionScenariosCannotCloseAGap is the constraint that keeps the
// canonical set canonical.
func TestExtensionScenariosCannotCloseAGap(t *testing.T) {
	expected, executed := completeRun(t)
	dropped := executed[0]
	executed = executed[1:]

	// An adopter supplying a scenario with the same name, from their own
	// filesystem, does not substitute for the canonical one.
	executed = append(executed, executedScenario{
		uri:    extensionsRoot + "/" + strings.TrimPrefix(dropped.uri, featuresPath+"/"),
		name:   dropped.name,
		status: messages.TestStepResultStatus_PASSED,
	})

	gaps, _, extensions := canonicalGaps(expected, executed)
	if len(gaps) != 1 {
		t.Fatalf("an extension scenario closed a canonical gap; gaps: %v", gaps)
	}
	if !strings.Contains(gaps[0], dropped.name) {
		t.Errorf("the gap does not name the canonical scenario that did not run: %s", gaps[0])
	}
	if extensions != 1 {
		t.Errorf("the guard counted %d extension scenarios, want 1", extensions)
	}
}
