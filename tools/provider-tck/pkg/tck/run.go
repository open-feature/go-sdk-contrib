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
func (r *runner) afterScenario(ctx context.Context, _ *godog.Scenario, err error) (context.Context, error) {
	if state, stateErr := stateFrom(ctx); stateErr == nil {
		state.teardown()
	}
	return ctx, err
}

// missingCapability reports the first capability a scenario needs that the
// provider did not declare.
//
// Tags that do not map to a capability gate nothing, so the canonical feature
// files stay free to carry organisational tags.
func (r *runner) missingCapability(sc *godog.Scenario) (Capability, bool) {
	for _, tag := range sc.Tags {
		capability, gates := capabilityForTag(tag.Name)
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
