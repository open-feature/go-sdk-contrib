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
// the options describe.
//
// Two required options and a third that depends on how the backend is run:
//
//	tck.Run(t,
//	    tck.WithName("my-provider"),
//	    tck.WithComposeFile("testdata/tck/docker-compose.yaml"),
//	    tck.WithBackendPorts(8013),
//	    tck.WithProviderFromEndpoint(newProvider),
//	    tck.WithCapabilities(tck.Events, tck.Object),
//	)
//
// A provider with no backend supplies its own control and builds its provider
// without an endpoint — see WithControl and WithProvider. Everything else has a
// working default, and a missing required option is reported here by name rather
// than failing later inside a step.
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
func Run(t *testing.T, opts ...Option) {
	t.Helper()

	cfg := newConfig(opts)

	if err := cfg.validate(); err != nil {
		t.Fatalf("tck: invalid configuration:\n%v", err)
	}

	caps, err := newCapabilitySet(cfg.capabilities())
	if err != nil {
		t.Fatalf("tck: invalid configuration:\n%v", err)
	}

	features, err := cfg.featureSources()
	if err != nil {
		t.Fatalf("tck: invalid configuration:\n%v", err)
	}

	if cfg.compose != nil {
		// The stack outlives every scenario and is brought down once, after
		// the last one. See WithComposeFile for why it is never restarted.
		endpoint, control, stop, err := startCompose(context.Background(), cfg.compose)
		if err != nil {
			t.Fatalf("tck [%s]: %v", cfg.Name, err)
		}
		t.Cleanup(stop)

		cfg.Control = control
		factory := cfg.NewProviderFromEndpoint
		cfg.NewProvider = func(ctx context.Context) (openfeature.FeatureProvider, error) {
			return factory(ctx, endpoint)
		}
	}

	r := &runner{cfg: *cfg, caps: caps, t: t}

	t.Logf("tck [%s]: backend under test is %s; declared capabilities %s",
		cfg.Name, cfg.Control.Description(), formatCapabilities(caps.sorted()))

	defer r.shutdown()

	status := godog.TestSuite{
		Name:                cfg.Name,
		ScenarioInitializer: r.initializeScenario,
		Options: &godog.Options{
			Format: "pretty",
			Output: os.Stdout,
			// The canonical Gherkin comes embedded in the spec module this
			// package depends on, so an adopting module needs no submodule and
			// no particular directory layout. With tck.WithFeatures
			// set, the adopter's filesystem is mounted alongside it and both
			// are parsed in one pass. See featureSources.
			FS:    features.fsys,
			Paths: features.paths,
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
		t.Fatalf("tck [%s]: suite failed with exit status %d", cfg.Name, status)
	}
}

// runner holds everything that outlives a single scenario.
type runner struct {
	cfg  config
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
// run.
func (r *runner) beforeScenario(ctx context.Context, sc *godog.Scenario) (context.Context, error) {
	if capability, expired := expiredReservation(sc); expired {
		return ctx, fmt.Errorf(
			"scenario %q carries %s, which this suite still lists as a reserved capability. A "+
				"reserved capability cannot be declared, so without this check the scenario would "+
				"be reported as skipped for a capability no adopter is able to claim -- a question "+
				"put and silently withdrawn. The specification has grown scenarios for %s: delete "+
				"it from reservedCapabilities in capability.go, which is the only change needed",
			sc.Name, capability.Tag(), capability)
	}

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

// expiredReservation reports whether a scenario carries the tag of a capability
// this suite still treats as reserved.
//
// It is the expiry check on reservedCapabilities, and it exists because the
// failure it catches is silent in both directions. A reserved capability cannot
// be declared -- newCapabilitySet refuses it -- so when the specification adds
// the first scenario for one, every adopter's run reports that scenario as
// skipped for a capability they are not permitted to claim. The report is
// well-formed, the suite is green, and the new scenario is never executed by
// anybody. That is the unclaimable-capability failure Appendix F describes, and
// nothing else in the suite would notice it: the under-collection guard is
// satisfied, because the scenario was collected and gated rather than dropped,
// and a capability-gated skip is explicitly not a gap.
//
// So a reserved tag on a real scenario fails the run. The tags come from the
// scenario godog parsed, which is the parser the runner itself uses, so the
// check cannot disagree with the run about which tags a scenario carries --
// including tags inherited from the feature and tags on an Examples block.
func expiredReservation(sc *godog.Scenario) (Capability, bool) {
	for _, tag := range sc.Tags {
		capability, known := CapabilityForTag(tag.Name)
		if known && capability.IsReserved() {
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
		r.t.Logf("tck [%s]: every applicable scenario ran; no capability was left undeclared",
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
		fmt.Sprintf("tck [%s]: %d scenario(s) skipped because a capability was not declared.",
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
		r.t.Logf("tck [%s]: could not release the provider under test: %v", r.cfg.Name, err)
	}
}
