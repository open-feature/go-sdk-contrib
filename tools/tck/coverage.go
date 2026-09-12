package tck

import (
	"fmt"
	"sort"
	"strings"

	"github.com/cucumber/godog"
	messages "github.com/cucumber/messages/go/v21"
)

// WHY THE CANONICAL SET IS CHECKED AGAINST ITSELF
//
// A conformance report is a claim that a provider was asked the canonical
// questions. Nothing in the machinery so far establishes that it was asked all
// of them. A `go test -run` selector matching one scenario name, a mis-wired
// extension filesystem, a future option carrying a tag filter — each
// produces a green suite and a well-formed report describing a subset, and
// there is no field in the report a consumer could read to notice.
//
// A capability skip is not this. A gated scenario runs the gate, is announced,
// and appears in the results as SKIPPED with a reason; the question was put and
// declined. What this guards against is a scenario that never appears at all,
// or appears with no outcome — the question that was never put.
//
// The expectation is the embedded canonical assets, parsed by godog's own
// parser rather than by a second one, so what the guard expects is exactly what
// a full run would have produced. Extension scenarios are excluded: an adopter's
// features are an addition to the canonical set and can neither complete it nor
// dilute it.

// canonicalScenario identifies one canonical scenario for the guard: the
// feature it belongs to and its name.
//
// Not the pickle id, which is assigned per parse and shifts when an extension
// filesystem is mounted alongside, and so cannot be compared between the
// expectation and the run. Name within feature is stable, and the multiplicity
// of the key is what accounts for the rows of a Scenario Outline: all eleven
// rows of one outline share a name, so eleven expected against ten observed is
// a missing row.
type canonicalScenario struct {
	uri  string
	name string
}

func (s canonicalScenario) String() string {
	return fmt.Sprintf("%s: %s", s.uri, s.name)
}

// executedScenario is one scenario the run actually reported on.
type executedScenario struct {
	uri    string
	name   string
	status messages.TestStepResultStatus
}

// canonicalExpectation counts the results each canonical scenario must produce
// in a complete run.
//
// It parses the embedded assets through godog itself. A parser of this
// package's own would be a second implementation of Gherkin compilation, and
// the two would disagree about exactly the cases that matter here — an outline
// whose Examples block carries its own tags, a Rule with scenarios inside it.
func canonicalExpectation() (map[canonicalScenario]int, error) {
	features, err := godog.TestSuite{
		// No tag filter and no dialect override, which is what makes this the
		// whole canonical set rather than whatever the run happened to select.
		Options: &godog.Options{FS: assets, Paths: []string{featuresPath}},
	}.RetrieveFeatures()
	if err != nil {
		return nil, err
	}

	expected := map[canonicalScenario]int{}
	for _, feature := range features {
		if feature == nil {
			continue
		}
		for _, pickle := range feature.Pickles {
			if pickle == nil {
				continue
			}
			expected[canonicalScenario{uri: pickle.Uri, name: pickle.Name}]++
		}
	}

	if len(expected) == 0 {
		return nil, fmt.Errorf("the embedded canonical assets under %s compile to no scenario at all",
			featuresPath)
	}
	return expected, nil
}

// checkCanonicalCoverage fails the suite unless every canonical scenario
// reported an outcome.
//
// It runs after the report is written rather than before, so that the evidence
// of an incomplete run is on disk to look at. Failing the test is the only
// lever available: the report schema is closed, so there is no field in which
// to record "this run was partial", and a partial run that passes quietly is
// the failure this exists to prevent.
func (r *runner) checkCanonicalCoverage() {
	expected, err := canonicalExpectation()
	if err != nil {
		r.t.Errorf("tck [%s]: could not establish which canonical scenarios this run "+
			"should have executed, so its result cannot be trusted: %v", r.cfg.Name, err)
		return
	}

	r.mu.Lock()
	executed := make([]executedScenario, len(r.executed))
	copy(executed, r.executed)
	r.mu.Unlock()

	gaps, canonical, extensions := canonicalGaps(expected, executed)

	if len(gaps) == 0 {
		r.t.Logf("tck [%s]: all %d canonical scenarios executed%s",
			r.cfg.Name, canonical, extensionSuffix(extensions))
		return
	}

	r.t.Errorf("tck [%s]: %d canonical scenario(s) did not run, so this is not a "+
		"conformance run and its report must not be published:\n%s\n"+
		"Every scenario in the embedded assets under %s has to execute. A scenario the provider "+
		"does not support is declined through tck.WithCapabilities, which still runs the gate and "+
		"records the skip with its reason; one that is filtered out of the run instead leaves "+
		"nothing behind to read.",
		r.cfg.Name, len(gaps), strings.Join(gaps, "\n"), featuresPath)
}

// canonicalGaps compares what should have run against what did, and returns one
// human-readable line per canonical scenario that came up short, along with the
// size of the canonical set and the number of extension scenarios seen.
//
// Extension results are counted and then discarded. They are reported so the
// suite can say what it ran, but they can never close a gap: an adopter's
// feature is an addition to the canonical set, and a mechanism where supplying
// enough of your own scenarios made the canonical ones optional would defeat
// the point of the guard.
func canonicalGaps(
	expected map[canonicalScenario]int,
	executed []executedScenario,
) (gaps []string, canonical, extensions int) {
	// ran counts results that reached an outcome; unresolved counts scenarios
	// godog announced and never executed. The second is the shape a `go test
	// -run` selector leaves behind: godog reports the pickle before handing the
	// scenario to the subtest that is then filtered away, so the test case
	// exists with no step result in it and its outcome stays UNKNOWN.
	ran := map[canonicalScenario]int{}
	unresolved := map[canonicalScenario]int{}

	for _, scenario := range executed {
		if !isCanonicalFeature(scenario.uri) {
			extensions++
			continue
		}
		key := canonicalScenario{uri: scenario.uri, name: scenario.name}
		if scenario.status == messages.TestStepResultStatus_UNKNOWN {
			unresolved[key]++
			continue
		}
		ran[key]++
	}

	for key, want := range expected {
		canonical += want
		got := ran[key]
		if got >= want {
			continue
		}
		detail := fmt.Sprintf("  - %s: %d of %d executed", key, got, want)
		if n := unresolved[key]; n > 0 {
			detail += fmt.Sprintf(" (%d announced but never run, which is what a -run selector "+
				"or a tag filter leaves behind)", n)
		}
		gaps = append(gaps, detail)
	}
	sort.Strings(gaps)

	return gaps, canonical, extensions
}

func extensionSuffix(extensions int) string {
	if extensions == 0 {
		return ""
	}
	return fmt.Sprintf(", alongside %d extension scenario(s)", extensions)
}
