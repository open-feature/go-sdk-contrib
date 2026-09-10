package tck_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/cucumber/godog"
	messages "github.com/cucumber/messages/go/v21"
	"github.com/open-feature/go-sdk-contrib/tools/provider-tck/pkg/tck"
	"github.com/open-feature/go-sdk/openfeature"
	"github.com/open-feature/go-sdk/openfeature/memprovider"
)

// An adopter with provider-specific behaviour has to be able to add scenarios
// that run in the TCK's backend lifecycle — same provider registration, same
// per-scenario reset — rather than in a harness of their own. These tests are
// the end-to-end proof that Config.ExtensionFeatures and Config.ExtensionSteps
// do that, and that they change nothing when unset.

// extensionFeatureURI is where testdata/tck-extensions/vendor.feature appears
// once the suite has mounted it. The prefix is what tells an extension result
// apart from a canonical one.
const extensionFeatureURI = "extensions/vendor.feature"

// vendorSteps is the fixture adopter glue: one step that resolves a flag
// through the provider the TCK registered, and one that asserts what it saw.
//
// It uses tck.ClientFromContext rather than building a client of its own, which
// is the whole point — the provider under test is registered under a domain the
// adopter never names.
type vendorSteps struct {
	mu       sync.Mutex
	resolved map[string]string
	calls    int
}

func newVendorSteps() *vendorSteps {
	return &vendorSteps{resolved: map[string]string{}}
}

func (v *vendorSteps) register(ctx *godog.ScenarioContext) {
	ctx.Step(`^the vendor step resolves "([^"]*)"$`, v.resolve)
	ctx.Step(`^the vendor step should have seen "([^"]*)"$`, v.shouldHaveSeen)
}

func (v *vendorSteps) resolve(ctx context.Context, key string) error {
	client, err := tck.ClientFromContext(ctx)
	if err != nil {
		return err
	}

	value, err := client.StringValue(ctx, key, "unset", openfeature.EvaluationContext{})
	if err != nil {
		return err
	}

	v.mu.Lock()
	defer v.mu.Unlock()
	v.calls++
	v.resolved[key] = value
	return nil
}

func (v *vendorSteps) shouldHaveSeen(want string) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	for _, got := range v.resolved {
		if got == want {
			return nil
		}
	}
	return fmt.Errorf("the vendor step resolved %v, which does not contain %q", v.resolved, want)
}

func (v *vendorSteps) count() int {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.calls
}

// vendorConfig is the reference adoption with extensions: the in-memory suite
// plus two fields. Passing nil steps gives the same suite without them.
func vendorConfig(name string, steps *vendorSteps) tck.Config {
	cfg := tck.Config{
		Name:    name,
		Control: plainMemoryControl{},
		NewProvider: func(context.Context) (openfeature.FeatureProvider, error) {
			return memprovider.NewInMemoryProvider(tck.CanonicalFlagSet()), nil
		},
		Capabilities: []tck.Capability{tck.Events, tck.Object, tck.NumericCoercion},
	}
	if steps != nil {
		cfg.ExtensionFeatures = os.DirFS("testdata/tck-extensions")
		cfg.ExtensionSteps = steps.register
	}
	return cfg
}

// TestExtensionsRunInTheCanonicalSuite is the end-to-end proof.
//
// One suite, one backend lifecycle: the canonical scenarios run, the adopter's
// scenario runs, and the adopter's scenario reaches the provider the TCK
// registered rather than one of its own.
func TestExtensionsRunInTheCanonicalSuite(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(tck.ReportDirEnv, dir)

	steps := newVendorSteps()
	tck.Run(t, vendorConfig("with-extensions", steps))

	if steps.count() == 0 {
		t.Fatal("the extension step never ran, so the extension feature did not enter the suite")
	}

	run := readRun(t, dir, "with-extensions")

	var canonical, extension int
	for _, tc := range run.cases {
		switch {
		case tc.uri == extensionFeatureURI:
			extension++
			if tc.status != messages.TestStepResultStatus_PASSED {
				t.Errorf("the extension scenario %q is reported as %s: %s",
					tc.name, tc.status, tc.message)
			}
		case isCanonicalURI(tc.uri):
			canonical++
		default:
			t.Errorf("scenario %q came from %s, which is neither canonical nor an extension",
				tc.name, tc.uri)
		}
	}

	if extension != 1 {
		t.Errorf("the stream reports %d extension scenarios, want the 1 in the fixture", extension)
	}
	if canonical == 0 {
		t.Fatal("no canonical scenario ran alongside the extension, so the two did not share a suite")
	}

	// The adopter's feature source is carried in the results like any other, so
	// a consumer can read the question behind the extension result.
	if _, ok := run.sources[extensionFeatureURI]; !ok {
		t.Error("the stream does not carry the extension feature's source")
	}

	t.Logf("one suite ran %d canonical and %d extension scenario(s)", canonical, extension)
}

// TestNoExtensionsChangesNothing is the other half: a Config without them must
// produce exactly the run it produced before the fields existed.
func TestNoExtensionsChangesNothing(t *testing.T) {
	baselineDir := t.TempDir()
	t.Setenv(tck.ReportDirEnv, baselineDir)
	tck.Run(t, vendorConfig("baseline", nil))
	baseline := readRun(t, baselineDir, "baseline")

	extendedDir := t.TempDir()
	t.Setenv(tck.ReportDirEnv, extendedDir)
	tck.Run(t, vendorConfig("extended", newVendorSteps()))
	extended := readRun(t, extendedDir, "extended")

	// Same canonical scenarios, in the same order, with the same outcomes. An
	// extension that displaced, reordered or shadowed a canonical scenario
	// shows up here.
	baselineCanonical := canonicalOnly(baseline.cases)
	extendedCanonical := canonicalOnly(extended.cases)

	if len(baselineCanonical) != len(extendedCanonical) {
		t.Fatalf("the canonical suite ran %d scenarios on its own but %d alongside an extension",
			len(baselineCanonical), len(extendedCanonical))
	}
	for i := range baselineCanonical {
		a, b := baselineCanonical[i], extendedCanonical[i]
		if a.uri != b.uri || a.name != b.name || a.status != b.status {
			t.Errorf("canonical scenario %d differs: %s/%q/%s alone, %s/%q/%s with an extension",
				i, a.uri, a.name, a.status, b.uri, b.name, b.status)
		}
	}

	// The canonical feature sources must be byte-identical in both runs, which
	// is what says the extension mount did not change what was parsed.
	for uri, data := range baseline.sources {
		if !isCanonicalURI(uri) {
			t.Errorf("a Config with no extensions parsed %s", uri)
			continue
		}
		if extended.sources[uri] != data {
			t.Errorf("the source of %s differs between the two runs", uri)
		}
	}
	for _, tc := range baseline.cases {
		if !isCanonicalURI(tc.uri) {
			t.Errorf("a Config with no extensions ran %q from %s", tc.name, tc.uri)
		}
	}

	t.Logf("%d canonical scenarios, identical with and without extensions", len(baselineCanonical))
}

// TestAnExtensionCannotShadowACanonicalScenario is the constraint measured in
// Java, checked end to end.
//
// testdata/tck-shadow/evaluation.feature copies the canonical file's name, its
// Feature name and one of its Scenario names. In the Java TCK that was enough
// for a second classpath root to replace the canonical file, and the suite
// reported success having run the adopter's version of it. Here the canonical
// run must be untouched and the adopter's scenario must appear alongside it.
func TestAnExtensionCannotShadowACanonicalScenario(t *testing.T) {
	baselineDir := t.TempDir()
	t.Setenv(tck.ReportDirEnv, baselineDir)
	tck.Run(t, vendorConfig("shadow-baseline", nil))
	baseline := readRun(t, baselineDir, "shadow-baseline")

	dir := t.TempDir()
	t.Setenv(tck.ReportDirEnv, dir)
	cfg := vendorConfig("shadow", newVendorSteps())
	cfg.ExtensionFeatures = os.DirFS("testdata/tck-shadow")
	tck.Run(t, cfg)
	shadowed := readRun(t, dir, "shadow")

	// The canonical feature the fixture impersonates was parsed from the
	// embedded assets, byte for byte, and not from the adopter's copy.
	const canonicalURI = "assets/gherkin/evaluation.feature"
	if baseline.sources[canonicalURI] == "" {
		t.Fatalf("the baseline run carries no source for %s", canonicalURI)
	}
	if shadowed.sources[canonicalURI] != baseline.sources[canonicalURI] {
		t.Errorf("%s was parsed from the extension filesystem: its source differs from the "+
			"canonical one", canonicalURI)
	}

	// Every canonical scenario still ran, in the same order, with the same
	// outcome. A replacement would show up here as a missing or renamed one.
	before, after := canonicalOnly(baseline.cases), canonicalOnly(shadowed.cases)
	if len(before) != len(after) {
		t.Fatalf("the canonical suite ran %d scenarios on its own but %d alongside the shadowing "+
			"fixture", len(before), len(after))
	}
	for i := range before {
		if before[i].uri != after[i].uri || before[i].name != after[i].name ||
			before[i].status != after[i].status {
			t.Errorf("canonical scenario %d differs: %s/%q/%s alone, %s/%q/%s with the fixture",
				i, before[i].uri, before[i].name, before[i].status,
				after[i].uri, after[i].name, after[i].status)
		}
	}

	// The fixture ran too, as an addition, under its own URI.
	const shadowName = "Resolve values with variant and reason"
	fixture := 0
	for _, tc := range shadowed.cases {
		if tc.uri != "extensions/evaluation.feature" {
			continue
		}
		fixture++
		if tc.name != shadowName {
			t.Errorf("the fixture scenario is named %q, want %q", tc.name, shadowName)
		}
		if tc.status != messages.TestStepResultStatus_PASSED {
			t.Errorf("the fixture scenario is reported as %s: %s", tc.status, tc.message)
		}
	}
	if fixture != 1 {
		t.Errorf("%d results came from the shadowing fixture, want 1", fixture)
	}

	// And the canonical Scenario Outline of the same name kept all of its rows,
	// which is what says the two coexist rather than one having won.
	canonicalRows := 0
	for _, tc := range after {
		if tc.uri == canonicalURI && tc.name == shadowName {
			canonicalRows++
		}
	}
	if canonicalRows < 2 {
		t.Errorf("the canonical outline %q produced %d rows; the fixture displaced it",
			shadowName, canonicalRows)
	}

	t.Logf("%d canonical scenarios unchanged; the shadowing fixture added 1 under extensions/",
		len(after))
}

func isCanonicalURI(uri string) bool {
	return strings.HasPrefix(uri, "assets/gherkin/")
}

func canonicalOnly(cases []resultCase) []resultCase {
	out := make([]resultCase, 0, len(cases))
	for _, tc := range cases {
		if isCanonicalURI(tc.uri) {
			out = append(out, tc)
		}
	}
	return out
}
