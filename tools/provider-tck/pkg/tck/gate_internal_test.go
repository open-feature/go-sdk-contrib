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

// stubControl is a BackendControl that does nothing, for tests that only need
// a non-nil one.
type stubControl struct{}

func (stubControl) PrepareScenario(context.Context) error { return nil }
func (stubControl) ChangeFlag(context.Context) error      { return nil }
func (stubControl) Description() string                   { return "stub" }
