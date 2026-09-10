package tck

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/cucumber/godog"
	"github.com/open-feature/go-sdk/openfeature"
)

// Run executes the OpenFeature Provider Conformance Suite against the provider
// described by cfg.
//
// Each scenario becomes a Go subtest, so failures point at a scenario by name
// and -run selects one the usual way.
//
// The run fails unless every canonical scenario produced an outcome — see
// checkCanonicalCoverage. A scenario the provider declines through
// Config.Capabilities has produced one; a scenario filtered out of the run has
// not, and a report describing a subset of the canonical set is not a
// conformance result.
//
// Scenarios run serially, and that is enforced rather than merely preferred.
// Backend state — which flags are seeded, whether the backend is reachable — is
// global to the suite, so concurrent scenarios corrupt each other: one
// scenario's reconnect restores the backend underneath another's disconnect
// assertion. The resulting failure looks like a flaky provider rather than a
// broken test, which makes it expensive to diagnose.
func Run(t *testing.T, cfg Config) {
	t.Helper()

	if err := cfg.validate(); err != nil {
		t.Fatalf("provider-tck: invalid configuration:\n%v", err)
	}

	caps, err := newCapabilitySet(cfg.capabilities())
	if err != nil {
		t.Fatalf("provider-tck: invalid configuration:\n%v", err)
	}

	features, err := cfg.featureSources()
	if err != nil {
		t.Fatalf("provider-tck: invalid configuration:\n%v", err)
	}

	r := &runner{cfg: cfg, caps: caps, t: t}

	t.Logf("provider-tck [%s]: backend under test is %s; declared capabilities %s",
		cfg.Name, cfg.Control.Description(), formatCapabilities(caps.sorted()))

	// The Cucumber Messages stream is collected whether or not a report is
	// requested. It costs one buffer, and the alternative is a code path that
	// only ever runs in CI.
	registerMessagesFormatter()
	if !installMessagesSink(cfg.Name, &messagesSink{
		out:        &r.messages,
		skipReason: r.skipReason,
		executed:   r.recordExecuted,
	}) {
		t.Fatalf("provider-tck: two suites named %q are running at once; suite names must be "+
			"unique within a test binary, since they also scope the OpenFeature domain and the "+
			"report filenames", cfg.Name)
	}
	defer removeMessagesSink(cfg.Name)

	defer r.shutdown()

	status := godog.TestSuite{
		Name:                cfg.Name,
		ScenarioInitializer: r.initializeScenario,
		Options: &godog.Options{
			// Two formatters: the human-readable one on stdout, and the
			// Messages stream into the buffer the report references. Both are
			// handed the same writer by godog, so the Messages formatter
			// resolves its own sink rather than using it. See messages.go.
			Format: "pretty," + messagesFormatterName,
			Output: os.Stdout,
			// The canonical Gherkin is embedded in this package, so an adopting
			// module needs no submodule and no particular directory layout.
			// With Config.ExtensionFeatures set, the adopter's filesystem is
			// mounted alongside it and both are parsed in one pass. See
			// featureSources.
			FS:    features.fsys,
			Paths: features.paths,
			// Scenarios become subtests of t.
			TestingT: t,
			// Serial. See the doc comment. The Messages formatter also depends
			// on it: godog has no scenario-finished event, so the formatter
			// closes a test case when the run moves on, which is only
			// unambiguous while one scenario runs at a time.
			Concurrency: 1,
			// An undefined or pending step is a failure, not a silent pass. The
			// feature files come from the specification, so a step with no
			// definition means this package has fallen behind them.
			Strict:   true,
			NoColors: true,
		},
	}.Run()

	r.reportSkips()
	r.writeReport()
	r.checkCanonicalCoverage()

	if status != 0 && !t.Failed() {
		t.Fatalf("provider-tck [%s]: suite failed with exit status %d", cfg.Name, status)
	}
}

// runner holds everything that outlives a single scenario.
type runner struct {
	cfg  Config
	caps capabilitySet
	t    *testing.T

	mu    sync.Mutex
	skips []skippedScenario

	// messages receives the Cucumber Messages stream, written by the formatter
	// registered in messages.go when godog delivers its Summary event.
	messages lockedBuffer

	// skipReasons is why the capability gate refused a scenario, keyed by
	// pickle id.
	//
	// It exists because godog's formatter events cannot carry it:
	// Skipped(pickle, step, definition) has no error parameter, so a formatter
	// can see that a scenario did not run but not why. The gate is the only
	// place that knows, so it records it here and the formatter reads it back.
	//
	// Keyed by pickle id rather than by scenario name because every row of a
	// Scenario Outline shares one name, and because Gherkin allows an Examples
	// block to carry its own tags — so two rows of one outline can differ in
	// whether the gate stops them.
	skipReasons map[string]string

	// providerName is what the provider called itself, observed from the last
	// scenario that registered one.
	providerName string

	// executed is every scenario the run reported on, with its outcome, as the
	// Messages formatter derived it. It is what checkCanonicalCoverage compares
	// against the embedded canonical assets.
	executed []executedScenario
}

// recordExecuted collects one scenario's outcome from the Messages formatter.
func (r *runner) recordExecuted(scenario executedScenario) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.executed = append(r.executed, scenario)
}

// skippedScenario records a scenario that did not run because the provider did
// not declare the capability it needs.
type skippedScenario struct {
	name       string
	capability Capability
}

func (r *runner) initializeScenario(ctx *godog.ScenarioContext) {
	ctx.Before(r.beforeScenario)
	ctx.After(r.afterScenario)

	registerProviderSteps(ctx)
	registerFlagSteps(ctx)
	registerEventSteps(ctx)

	// An adopter's steps go last, so a collision with a TCK expression is
	// godog's ambiguity error naming both rather than a silent override in
	// either direction.
	if r.cfg.ExtensionSteps != nil {
		r.cfg.ExtensionSteps(ctx)
	}
}

// beforeScenario gates on capabilities and resets the backend.
//
// Returning an error wrapping godog.ErrSkip skips the scenario and every step
// in it without failing the suite, which is exactly the semantics an
// undeclared capability calls for: the scenario is reported, visibly, as not
// run. godog reports every step of such a scenario to the formatter as
// SKIPPED, so the Messages stream says so too.
func (r *runner) beforeScenario(ctx context.Context, sc *godog.Scenario) (context.Context, error) {
	if capability, missing := r.missingCapability(sc); missing {
		r.recordSkip(sc.Name, capability)
		r.noteSkipReason(sc.Id, fmt.Sprintf(
			"requires capability %s, which this provider does not declare", capability.Tag()))
		return ctx, fmt.Errorf(
			"%w: scenario requires capability %s (Gherkin tag %s), which this provider does not declare. Declared capabilities: %s",
			godog.ErrSkip, capability, capability.Tag(), formatCapabilities(r.caps.sorted()))
	}

	ctx = withState(ctx, newScenarioState(&r.cfg))

	if err := r.cfg.Control.PrepareScenario(ctx); err != nil {
		return ctx, fmt.Errorf("could not reset %s before scenario %q: %w",
			r.cfg.Control.Description(), sc.Name, err)
	}

	return ctx, nil
}

// afterScenario detaches the scenario's event handlers. A skipped scenario has
// no state, which is not an error.
//
// It deliberately records nothing. godog does not deliver the Before hook's
// ErrSkip here — err arrives nil, indistinguishable from a scenario that ran
// and passed — so an outcome derived in this hook reported every capability
// skip as a pass, which is the exact failure Appendix F forbids. The outcome
// now comes from the formatter events, where a skip is a skip.
func (r *runner) afterScenario(ctx context.Context, _ *godog.Scenario, err error) (context.Context, error) {
	if state, stateErr := stateFrom(ctx); stateErr == nil {
		if state.providerName != "" {
			r.mu.Lock()
			r.providerName = state.providerName
			r.mu.Unlock()
		}
		state.teardown()
	}

	return ctx, err
}

// noteSkipReason records why the gate refused a scenario.
//
// The map is created on first use because the runner has to work as a zero
// value: it is constructed as a struct literal in tests that exercise the gate
// directly, and a nil map assignment there is a panic rather than a helpful
// failure.
func (r *runner) noteSkipReason(pickleID, reason string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.skipReasons == nil {
		r.skipReasons = map[string]string{}
	}
	r.skipReasons[pickleID] = reason
}

// skipReason is the lookup the Messages formatter consults.
func (r *runner) skipReason(pickleID string) string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.skipReasons[pickleID]
}

// messagesBytes is the Cucumber Messages stream the run produced.
func (r *runner) messagesBytes() []byte {
	return r.messages.bytes()
}

// missingCapability reports the first capability a scenario needs that the
// provider did not declare.
//
// Tags that do not map to a capability gate nothing, so the canonical feature
// files stay free to carry organisational tags.
func (r *runner) missingCapability(sc *godog.Scenario) (Capability, bool) {
	for _, tag := range sc.Tags {
		capability, gates := CapabilityForTag(tag.Name)
		if !gates {
			continue
		}
		if !r.caps.has(capability) {
			return capability, true
		}
	}
	return "", false
}

func (r *runner) recordSkip(scenario string, capability Capability) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.skips = append(r.skips, skippedScenario{name: scenario, capability: capability})
}

// reportSkips prints every capability-gated skip with its reason.
//
// This is not decoration. A conformance suite that quietly goes green on
// scenarios it did not run is worse than no suite at all, so the skips and the
// reason for each one are surfaced next to the result rather than left to be
// inferred from a scenario count.
func (r *runner) reportSkips() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.skips) == 0 {
		r.t.Logf("provider-tck [%s]: every applicable scenario ran; no capability was left undeclared",
			r.cfg.Name)
		return
	}

	sorted := make([]skippedScenario, len(r.skips))
	copy(sorted, r.skips)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].capability != sorted[j].capability {
			return sorted[i].capability < sorted[j].capability
		}
		return sorted[i].name < sorted[j].name
	})

	report := []string{
		fmt.Sprintf("provider-tck [%s]: %d scenario(s) skipped because a capability was not declared.",
			r.cfg.Name, len(sorted)),
		"These were NOT run and are NOT part of the conformance result:",
	}
	for _, s := range sorted {
		report = append(report,
			fmt.Sprintf("  - %s", s.name),
			fmt.Sprintf("      needs %s (tag %s)", s.capability, s.capability.Tag()))
	}
	report = append(report, "Declared capabilities: "+formatCapabilities(r.caps.sorted()))

	r.t.Log(strings.Join(report, "\n"))
}

// shutdown releases the last provider the suite registered.
//
// Registering a provider in a domain shuts down the one it replaces, so every
// scenario but the last cleans up after itself. Replacing the last one with the
// no-op provider closes that gap, which matters for a provider holding a
// network connection or a background goroutine.
func (r *runner) shutdown() {
	if err := openfeature.SetNamedProvider(r.cfg.domain(), openfeature.NoopProvider{}); err != nil {
		r.t.Logf("provider-tck [%s]: could not release the provider under test: %v", r.cfg.Name, err)
	}
}
