package tck

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/cucumber/godog"
	"github.com/open-feature/go-sdk/openfeature"
)

// Run executes the OpenFeature Provider Conformance Suite against the provider
// described by cfg.
//
// Each scenario becomes a Go subtest, so failures point at a scenario by name
// and -run selects one the usual way.
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

	r := &runner{cfg: cfg, caps: caps, t: t}

	t.Logf("provider-tck [%s]: backend under test is %s; declared capabilities %s",
		cfg.Name, cfg.Control.Description(), formatCapabilities(caps.sorted()))

	defer r.shutdown()

	status := godog.TestSuite{
		Name:                cfg.Name,
		ScenarioInitializer: r.initializeScenario,
		Options: &godog.Options{
			Format: "pretty",
			Output: os.Stdout,
			// The canonical Gherkin is embedded in this package, so an adopting
			// module needs no submodule and no particular directory layout.
			FS:    assets,
			Paths: []string{featuresPath},
			// Scenarios become subtests of t.
			TestingT: t,
			// Serial. See the doc comment.
			Concurrency: 1,
			// An undefined or pending step is a failure, not a silent pass. The
			// feature files come from the specification, so a step with no
			// definition means this package has fallen behind them.
			Strict:   true,
			NoColors: true,
		},
	}.Run()

	r.reportSkips()
	r.reportExampleGaps()
	r.writeReport()

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

	// records is the per-scenario outcome list the conformance report is built
	// from. It is kept even when no report is requested, because it costs
	// nothing and the alternative is a code path that only ever runs in CI.
	records []scenarioRecord

	// started times scenarios by pickle id. godog gives no scenario-scoped place
	// to hang this, so a map is the only option; the key is the pickle id rather
	// than the scenario name because every row of a Scenario Outline shares one
	// name and would otherwise share one entry.
	started map[string]time.Time

	// gated names the scenarios the capability gate stopped before their first
	// step, so the after hook does not record them a second time.
	//
	// It exists because godog does not deliver the before hook's ErrSkip to the
	// after hook -- err arrives nil, indistinguishable from a scenario that ran
	// and passed. Testing for ErrSkip there silently recorded every skipped
	// scenario twice, once correctly and once as passed, which is the exact
	// failure Appendix F forbids.
	//
	// Keyed by pickle id for a second reason as well as the shared name: Gherkin
	// allows an Examples block to carry its own tags, so two rows of one outline
	// can differ in whether the gate stops them. Keyed by name, gating one row
	// would suppress the after hook for every row, and the rows that did run
	// would vanish from the report entirely.
	gated map[string]bool

	// exampleGaps collects outline rows whose Examples row could not be
	// resolved. See reportExampleGaps.
	exampleGaps []string

	// providerName is what the provider called itself, observed from the last
	// scenario that registered one.
	providerName string
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
}

// beforeScenario gates on capabilities and resets the backend.
//
// Returning an error wrapping godog.ErrSkip skips the scenario and every step
// in it without failing the suite, which is exactly the semantics an
// undeclared capability calls for: the scenario is reported, visibly, as not
// run.
func (r *runner) beforeScenario(ctx context.Context, sc *godog.Scenario) (context.Context, error) {
	if capability, missing := r.missingCapability(sc); missing {
		r.recordSkip(sc.Name, capability)
		r.recordOutcome(sc, OutcomeNotDeclared, fmt.Sprintf(
			"requires capability %s, which this provider does not declare", capability.Tag()), 0)
		r.markGated(sc.Id)
		return ctx, fmt.Errorf(
			"%w: scenario requires capability %s (Gherkin tag %s), which this provider does not declare. Declared capabilities: %s",
			godog.ErrSkip, capability, capability.Tag(), formatCapabilities(r.caps.sorted()))
	}

	r.markStarted(sc.Id)

	ctx = withState(ctx, newScenarioState(&r.cfg))

	if err := r.cfg.Control.PrepareScenario(ctx); err != nil {
		return ctx, fmt.Errorf("could not reset %s before scenario %q: %w",
			r.cfg.Control.Description(), sc.Name, err)
	}

	return ctx, nil
}

// afterScenario detaches the scenario's event handlers. A skipped scenario has
// no state, which is not an error.
func (r *runner) afterScenario(ctx context.Context, sc *godog.Scenario, err error) (context.Context, error) {
	if state, stateErr := stateFrom(ctx); stateErr == nil {
		if state.providerName != "" {
			r.mu.Lock()
			r.providerName = state.providerName
			r.mu.Unlock()
		}
		state.teardown()
	}

	// A capability skip was already recorded before the scenario started.
	// Recording it again here would put it in the report twice, the second time
	// as passed.
	if !r.wasGated(sc.Id) {
		outcome, reason := OutcomePassed, ""
		if err != nil {
			outcome, reason = OutcomeFailed, err.Error()
		}
		r.recordOutcome(sc, outcome, reason, r.elapsed(sc.Id))
	}

	return ctx, err
}

// recordOutcome appends one scenario's result.
func (r *runner) recordOutcome(sc *godog.Scenario, outcome Outcome, reason string, duration time.Duration) {
	tags := make([]string, 0, len(sc.Tags))
	for _, tag := range sc.Tags {
		tags = append(tags, tag.Name)
	}

	// Resolved before the lock, because a failure to resolve takes the lock
	// itself.
	example, order := r.exampleFor(sc)

	r.mu.Lock()
	defer r.mu.Unlock()
	r.records = append(r.records, scenarioRecord{
		feature:      featureName(sc.Uri),
		name:         sc.Name,
		example:      example,
		exampleOrder: order,
		tags:         tags,
		outcome:      outcome,
		reason:       reason,
		duration:     duration,
	})
}

// exampleFor resolves the Examples row a scenario came from, or nil for a
// scenario that is not an outline row.
//
// It is called from both the gate path and the pass/fail path, because a
// skipped outline row is exactly as ambiguous as a failed one: four skipped
// rows of the @object outline are four entries that differ in nothing without
// it.
func (r *runner) exampleFor(sc *godog.Scenario) (map[string]string, int) {
	index, err := scenarioExamples()
	if err != nil {
		r.recordExampleGap(fmt.Sprintf(
			"the embedded feature files could not be parsed for their Examples tables: %v", err))
		return nil, 0
	}

	if row, ok := index.rowFor(sc); ok {
		return row.values, row.order
	}

	if index.isOutline(sc.Uri, sc.Name) {
		r.recordExampleGap(fmt.Sprintf(
			"scenario %q in %s comes from a Scenario Outline, but no Examples row matched AST node ids %v",
			sc.Name, sc.Uri, sc.AstNodeIds))
	}

	return nil, 0
}

func (r *runner) recordExampleGap(detail string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, existing := range r.exampleGaps {
		if existing == detail {
			return
		}
	}
	r.exampleGaps = append(r.exampleGaps, detail)
}

// reportExampleGaps fails the run if any outline row went unidentified.
//
// Recovering the Examples row depends on reproducing the AST node ids godog
// assigns, which is a coupling to how godog numbers nodes. This is what keeps
// that coupling honest: if a godog release changes the numbering, the report
// would quietly go back to emitting rows that differ in nothing, and a quiet
// return to the ambiguity this field exists to remove is worse than a build
// failure.
func (r *runner) reportExampleGaps() {
	r.mu.Lock()
	gaps := make([]string, len(r.exampleGaps))
	copy(gaps, r.exampleGaps)
	r.mu.Unlock()

	if len(gaps) == 0 {
		return
	}

	message := []string{fmt.Sprintf(
		"provider-tck [%s]: %d Scenario Outline row(s) could not be identified by their Examples parameters, "+
			"so the report cannot distinguish them:", r.cfg.Name, len(gaps))}
	for _, gap := range gaps {
		message = append(message, "  - "+gap)
	}
	r.t.Error(strings.Join(message, "\n"))
}

// markGated and markStarted create their maps on first use.
//
// The runner has to work as a zero value: it is constructed as a struct literal
// in tests that exercise the gate directly, and a nil map assignment there is a
// panic rather than a helpful failure.
func (r *runner) markGated(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.gated == nil {
		r.gated = map[string]bool{}
	}
	r.gated[name] = true
}

func (r *runner) markStarted(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.started == nil {
		r.started = map[string]time.Time{}
	}
	r.started[name] = time.Now()
}

// wasGated reports whether the capability gate stopped this scenario.
func (r *runner) wasGated(name string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.gated[name]
}

func (r *runner) elapsed(name string) time.Duration {
	r.mu.Lock()
	defer r.mu.Unlock()
	start, ok := r.started[name]
	if !ok {
		return 0
	}
	return time.Since(start)
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
