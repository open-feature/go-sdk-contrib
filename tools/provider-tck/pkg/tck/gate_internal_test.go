package tck

import (
	"context"
	"strings"
	"testing"

	"github.com/cucumber/godog"
	messages "github.com/cucumber/messages/go/v21"
	"github.com/open-feature/go-sdk/openfeature"
)

// The capability gate is the one piece of this package that decides whether a
// scenario counts. If it silently let a tagged scenario through, the suite
// would report a pass for behaviour it never exercised — which is worse than
// having no suite. These tests pin it directly rather than by inference from a
// scenario count.

func scenarioWithTags(name string, tags ...string) *godog.Scenario {
	pickle := &messages.Pickle{Name: name}
	for _, tag := range tags {
		pickle.Tags = append(pickle.Tags, &messages.PickleTag{Name: tag})
	}
	return pickle
}

func TestMissingCapabilitySkipsTaggedScenario(t *testing.T) {
	caps, err := newCapabilitySet([]Capability{Events, Object})
	if err != nil {
		t.Fatalf("newCapabilitySet: %v", err)
	}
	r := &runner{caps: caps}

	capability, missing := r.missingCapability(scenarioWithTags("outage", "@events", "@stale"))
	if !missing {
		t.Fatal("a scenario tagged @stale ran against a provider that did not declare Stale")
	}
	if capability != Stale {
		t.Fatalf("blamed %s, want %s", capability, Stale)
	}
}

func TestDeclaredCapabilityRunsTaggedScenario(t *testing.T) {
	caps, err := newCapabilitySet([]Capability{Events, ConfigurationChange})
	if err != nil {
		t.Fatalf("newCapabilitySet: %v", err)
	}
	r := &runner{caps: caps}

	if _, missing := r.missingCapability(scenarioWithTags("change", "@events", "@configuration-change")); missing {
		t.Fatal("a scenario was skipped even though every capability it needs was declared")
	}
}

// TestUnknownTagsGateNothing keeps the canonical feature files free to carry
// organisational tags without every one of them becoming a capability.
func TestUnknownTagsGateNothing(t *testing.T) {
	caps, err := newCapabilitySet(nil)
	if err != nil {
		t.Fatalf("newCapabilitySet: %v", err)
	}
	r := &runner{caps: caps}

	if _, missing := r.missingCapability(scenarioWithTags("mandatory", "@smoke", "@wip")); missing {
		t.Fatal("a tag that gates no capability caused a skip")
	}
}

// TestSkipErrorIsRecognisedByGodog is the assumption the whole gate rests on:
// the error returned from the Before hook has to be one godog treats as a skip
// rather than a failure, including after godog wraps it.
func TestSkipErrorIsRecognisedByGodog(t *testing.T) {
	caps, err := newCapabilitySet([]Capability{Events})
	if err != nil {
		t.Fatalf("newCapabilitySet: %v", err)
	}
	r := &runner{caps: caps, cfg: Config{Name: "gate", Control: stubControl{}}}

	_, hookErr := r.beforeScenario(context.Background(), scenarioWithTags("outage", "@stale"))
	if hookErr == nil {
		t.Fatal("the Before hook allowed a scenario needing an undeclared capability to run")
	}
	if !strings.Contains(hookErr.Error(), godog.ErrSkip.Error()) {
		t.Fatalf("the Before hook returned %v, which godog would treat as a failure rather than a skip", hookErr)
	}
	if !strings.Contains(hookErr.Error(), Stale.Tag()) {
		t.Fatalf("the skip reason does not name the tag that caused it: %v", hookErr)
	}
}

func TestValidateRejectsUnavailableInitWithoutAFactory(t *testing.T) {
	cfg := Config{
		Name:    "gate",
		Control: stubControl{},
		NewProvider: func(context.Context) (openfeature.FeatureProvider, error) {
			return openfeature.NoopProvider{}, nil
		},
		Capabilities: []Capability{UnavailableInit},
	}

	err := cfg.validate()
	if err == nil {
		t.Fatal("a config declaring UnavailableInit with no NewUnavailableProvider was accepted; " +
			"the @unavailable scenarios would fail inside a step instead of being reported as misconfiguration")
	}
	if !strings.Contains(err.Error(), "NewUnavailableProvider") {
		t.Fatalf("the error does not name the missing field: %v", err)
	}
}

func TestValidateRejectsUnknownCapability(t *testing.T) {
	cfg := Config{
		Name:    "gate",
		Control: stubControl{},
		NewProvider: func(context.Context) (openfeature.FeatureProvider, error) {
			return openfeature.NoopProvider{}, nil
		},
		Capabilities: []Capability{"@not-a-capability"},
	}

	if err := cfg.validate(); err == nil {
		t.Fatal("an unknown capability was accepted; it would silently gate nothing")
	}
}

// A reserved capability is one the vocabulary names and no scenario carries.
// Declaring it cannot be verified and cannot even produce a skip, so a report
// that says it was declared claims something nothing examined. These tests pin
// the three places that could put one into a report: the declare-everything
// convenience, the default when Config.Capabilities is unset, and an adopter
// naming one outright.

func TestAllCapabilitiesOmitsReservedCapabilities(t *testing.T) {
	for _, c := range AllCapabilities() {
		if c.IsReserved() {
			t.Errorf("AllCapabilities returned the reserved capability %s; an adoption starting "+
				"from the full set would declare a tag no scenario carries", c)
		}
	}

	// The exclusion must be exactly the reserved set, not a convenient subset:
	// a capability quietly dropped from the default is a gap an adopter never
	// sees reported.
	returned := make(map[Capability]bool, len(allCapabilities))
	for _, c := range AllCapabilities() {
		returned[c] = true
	}
	for _, c := range allCapabilities {
		if c.IsReserved() == returned[c] {
			t.Errorf("AllCapabilities is wrong about %s: reserved=%v, returned=%v",
				c, c.IsReserved(), returned[c])
		}
	}
}

func TestTheDefaultCapabilitySetOmitsReservedCapabilities(t *testing.T) {
	// nil means "declare everything", which is the shape that put @targeting
	// and @caching into a published report elsewhere.
	cfg := &Config{}
	for _, c := range cfg.capabilities() {
		if c.IsReserved() {
			t.Errorf("the default capability set contains the reserved capability %s", c)
		}
	}
}

func TestValidateRejectsAnExplicitlyDeclaredReservedCapability(t *testing.T) {
	for _, reserved := range reservedCapabilities {
		cfg := Config{
			Name:    "gate",
			Control: stubControl{},
			NewProvider: func(context.Context) (openfeature.FeatureProvider, error) {
				return openfeature.NoopProvider{}, nil
			},
			Capabilities: []Capability{Object, reserved},
		}

		err := cfg.validate()
		if err == nil {
			t.Errorf("declaring the reserved capability %s was accepted; it would reach a "+
				"conformance report as a claim nothing examined", reserved)
			continue
		}
		if !strings.Contains(err.Error(), reserved.Tag()) {
			t.Errorf("the error for %s does not name the tag: %v", reserved, err)
		}
	}
}

// TestAReservedCapabilityCannotBeDeclared pins the rule at the only place a
// capability set is built.
//
// The check lives in newCapabilitySet rather than only in Config.validate
// because the set is the single thing a declaration is derived from and the only
// constructor, so there is no second route by which a reserved capability could
// reach one. Config.validate calls it, so an adopter naming a reserved
// capability is refused before any scenario runs.
//
// The report half of this property -- that declaration.declared never contains a
// reserved tag -- is asserted where the report exists, on the branch that emits
// one. Here there is no report to inspect, and asserting the constructor is what
// makes the report's guarantee structural rather than incidental.
func TestAReservedCapabilityCannotBeDeclared(t *testing.T) {
	for _, reserved := range reservedCapabilities {
		if _, err := newCapabilitySet([]Capability{reserved}); err == nil {
			t.Errorf("newCapabilitySet accepted the reserved capability %s, so it could still "+
				"reach a conformance declaration", reserved)
		}
	}

	caps, err := newCapabilitySet((&Config{}).capabilities())
	if err != nil {
		t.Fatalf("the default capability set does not validate: %v", err)
	}
	for _, capability := range caps.sorted() {
		if capability.IsReserved() {
			t.Errorf("the default capability set contains the reserved capability %s; a "+
				"declare-everything default must not hand out tags nothing tests", capability)
		}
	}
}

// stubControl is a BackendControl that does nothing, for tests that only need
// a non-nil one.
type stubControl struct{}

func (stubControl) PrepareScenario(context.Context) error { return nil }
func (stubControl) ChangeFlag(context.Context) error      { return nil }
func (stubControl) Description() string                   { return "stub" }
