package tck

import (
	"context"
	"errors"
	"io/fs"
	"strings"
	"testing"
	"time"

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
	r := &runner{caps: caps, cfg: config{Name: "gate", Control: stubControl{}}}

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

// validConfig is the smallest configuration that validates, so that a test
// naming one mistake sees that mistake's error and not a pile of unrelated
// ones.
func validConfig(extra ...Option) *config {
	return newConfig(append([]Option{
		WithName("gate"),
		WithControl(stubControl{}),
		WithProvider(func(context.Context) (openfeature.FeatureProvider, error) {
			return openfeature.NoopProvider{}, nil
		}),
	}, extra...))
}

func TestValidateRejectsUnavailableInitWithoutAFactory(t *testing.T) {
	err := validConfig(WithCapabilities(UnavailableInit)).validate()
	if err == nil {
		t.Fatal("a configuration declaring UnavailableInit with no tck.WithUnavailableProvider was accepted; " +
			"the @unavailable scenarios would fail inside a step instead of being reported as misconfiguration")
	}
	if !strings.Contains(err.Error(), "WithUnavailableProvider") {
		t.Fatalf("the error does not name the missing option: %v", err)
	}
}

func TestValidateRejectsUnknownCapability(t *testing.T) {
	if err := validConfig(WithCapabilities("@not-a-capability")).validate(); err == nil {
		t.Fatal("an unknown capability was accepted; it would silently gate nothing")
	}
}

// The required options are required, and saying which one is missing is the
// whole reason validation runs before the stack starts rather than inside the
// first step. These pin the message as well as the rejection: an adopter who
// forgot an option needs to be told which.

func TestValidateNamesEachMissingRequiredOption(t *testing.T) {
	for _, tc := range []struct {
		missing string
		opts    []Option
	}{
		{
			missing: "WithName",
			opts: []Option{
				WithControl(stubControl{}),
				WithProvider(func(context.Context) (openfeature.FeatureProvider, error) {
					return openfeature.NoopProvider{}, nil
				}),
			},
		},
		{
			missing: "WithProvider",
			opts: []Option{
				WithName("gate"),
				WithControl(stubControl{}),
			},
		},
		{
			missing: "WithControl",
			opts: []Option{
				WithName("gate"),
				WithProvider(func(context.Context) (openfeature.FeatureProvider, error) {
					return openfeature.NoopProvider{}, nil
				}),
			},
		},
	} {
		t.Run(tc.missing, func(t *testing.T) {
			err := newConfig(tc.opts).validate()
			if err == nil {
				t.Fatalf("a configuration missing tck.%s was accepted", tc.missing)
			}
			if !strings.Contains(err.Error(), tc.missing) {
				t.Errorf("the error does not name tck.%s: %v", tc.missing, err)
			}
		})
	}
}

// TestValidateRejectsMixingTheComposeAndManualPaths pins the one ambiguity
// options introduce that a struct literal did not: two ways to supply the same
// thing, where taking one silently would point a provider at a stack whose
// control the suite is not driving.
func TestValidateRejectsMixingTheComposeAndManualPaths(t *testing.T) {
	for _, tc := range []struct {
		name string
		opts []Option
		want string
	}{
		{
			name: "control beside a compose file",
			opts: []Option{
				WithName("gate"),
				WithControl(stubControl{}),
				WithComposeFile("docker-compose.yaml"),
				WithBackendPorts(8013),
				WithProviderFromEndpoint(func(context.Context, BackendEndpoint) (openfeature.FeatureProvider, error) {
					return openfeature.NoopProvider{}, nil
				}),
			},
			want: "WithControl",
		},
		{
			name: "endpoint-less factory beside a compose file",
			opts: []Option{
				WithName("gate"),
				WithComposeFile("docker-compose.yaml"),
				WithBackendPorts(8013),
				WithProvider(func(context.Context) (openfeature.FeatureProvider, error) {
					return openfeature.NoopProvider{}, nil
				}),
			},
			want: "WithProviderFromEndpoint",
		},
		{
			name: "endpoint factory with no compose file",
			opts: []Option{
				WithName("gate"),
				WithControl(stubControl{}),
				WithProviderFromEndpoint(func(context.Context, BackendEndpoint) (openfeature.FeatureProvider, error) {
					return openfeature.NoopProvider{}, nil
				}),
			},
			want: "WithComposeFile",
		},
		{
			name: "both provider factories",
			opts: []Option{
				WithName("gate"),
				WithControl(stubControl{}),
				WithProvider(func(context.Context) (openfeature.FeatureProvider, error) {
					return openfeature.NoopProvider{}, nil
				}),
				WithProviderFromEndpoint(func(context.Context, BackendEndpoint) (openfeature.FeatureProvider, error) {
					return openfeature.NoopProvider{}, nil
				}),
			},
			want: "WithProviderFromEndpoint",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := newConfig(tc.opts).validate()
			if err == nil {
				t.Fatal("a configuration mixing the Compose and manual paths was accepted")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("the error does not name tck.%s: %v", tc.want, err)
			}
		})
	}
}

// TestValidateRejectsAnIncompleteComposeStack pins the Compose contract's own
// two required halves, and the two ways of describing the same port twice.
func TestValidateRejectsAnIncompleteComposeStack(t *testing.T) {
	endpointFactory := WithProviderFromEndpoint(
		func(context.Context, BackendEndpoint) (openfeature.FeatureProvider, error) {
			return openfeature.NoopProvider{}, nil
		})

	for _, tc := range []struct {
		name string
		opts []Option
		want string
	}{
		{
			name: "ports with no compose file",
			opts: []Option{WithName("gate"), WithBackendPorts(8013), endpointFactory},
			want: "WithComposeFile",
		},
		{
			name: "compose file with no backend ports",
			opts: []Option{WithName("gate"), WithComposeFile("docker-compose.yaml"), endpointFactory},
			want: "WithBackendPorts",
		},
		{
			name: "the control port declared as a backend port",
			opts: []Option{
				WithName("gate"),
				WithComposeFile("docker-compose.yaml"),
				WithBackendPorts(8080),
				endpointFactory,
			},
			want: "control API port",
		},
		{
			name: "the backend service declared as an additional one",
			opts: []Option{
				WithName("gate"),
				WithComposeFile("docker-compose.yaml"),
				WithBackendPorts(8013),
				WithAdditionalPorts(defaultBackendService, 9211),
				endpointFactory,
			},
			want: "WithAdditionalPorts",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := newConfig(tc.opts).validate()
			if err == nil {
				t.Fatal("an incomplete Compose configuration was accepted")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("the error does not mention %q: %v", tc.want, err)
			}
		})
	}
}

// TestComposeDefaultsMatchTheCrossLanguageContract pins the four defaults every
// language's suite shares. They are a contract rather than a convenience: an
// adoption that says nothing about them must behave the same way in Go, Java,
// JavaScript and Python, so changing one here is a cross-language change.
func TestComposeDefaultsMatchTheCrossLanguageContract(t *testing.T) {
	cc := newConfig([]Option{WithComposeFile("docker-compose.yaml")}).compose

	if got := cc.service(); got != "backend" {
		t.Errorf("default backend service is %q, want %q", got, "backend")
	}
	if got := cc.control(); got != 8080 {
		t.Errorf("default control port is %d, want 8080", got)
	}
	if got := cc.backendConfig(); got != DefaultBackendConfiguration {
		t.Errorf("default backend configuration is %q, want %q", got, DefaultBackendConfiguration)
	}
	if got := cc.timeout(); got != 60*time.Second {
		t.Errorf("default startup timeout is %s, want 60s", got)
	}
	if len(cc.additionalPorts) != 0 {
		t.Errorf("additional ports default to %v, want none", cc.additionalPorts)
	}
}

// TestComposeExposesTheControlPortWithoutBeingAsked is the half of the contract
// an adopter is told not to write: the control port is exposed automatically,
// and every declared port of every service is waited for in one strategy per
// service — because the compose module keeps a single strategy per service name
// and a second call for the same service silently replaces the first.
func TestComposeExposesTheControlPortWithoutBeingAsked(t *testing.T) {
	cc := newConfig([]Option{
		WithComposeFile("docker-compose.yaml"),
		WithBackendService("flagd"),
		WithBackendPorts(8013, 8015),
		WithAdditionalPorts("envoy", 9211, 9212),
	}).compose

	exposed := cc.exposed()

	if got := exposed["flagd"]; len(got) != 3 || got[0] != 8080 {
		t.Errorf("the backend service exposes %v, want the control port 8080 plus 8013 and 8015", got)
	}
	if got := exposed["envoy"]; len(got) != 2 {
		t.Errorf("the additional service exposes %v, want both declared ports", got)
	}
}

// TestCapabilitiesDeclaredEmptyIsNotSilence pins the one semantic an option can
// express that a struct field could not: declaring no optional capability at
// all, which is different from not saying anything and being given every one.
func TestCapabilitiesDeclaredEmptyIsNotSilence(t *testing.T) {
	if got := newConfig(nil).capabilities(); len(got) != len(AllCapabilities()) {
		t.Errorf("a configuration that omits tck.WithCapabilities declared %v, want every declarable capability", got)
	}
	if got := newConfig([]Option{WithCapabilities()}).capabilities(); len(got) != 0 {
		t.Errorf("tck.WithCapabilities() declared %v, want nothing at all", got)
	}
}

// TestLaterOptionsWin pins the ordering an adoption relies on when it layers a
// per-suite difference over a shared base.
func TestLaterOptionsWin(t *testing.T) {
	cfg := newConfig([]Option{
		WithName("first"),
		WithReadyTimeout(time.Second),
		WithName("second"),
		WithReadyTimeout(2 * time.Second),
	})

	if cfg.Name != "second" {
		t.Errorf("Name = %q, want the later option's value", cfg.Name)
	}
	if cfg.readyTimeout() != 2*time.Second {
		t.Errorf("ReadyTimeout = %s, want the later option's value", cfg.readyTimeout())
	}
}

// A reserved capability is one the vocabulary names and no scenario carries.
// Declaring it cannot be verified and cannot even produce a skip, so a report
// that says it was declared claims something nothing examined. These tests pin
// the three places that could put one into a report: the declare-everything
// convenience, the default when tck.WithCapabilities is omitted, and an adopter
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
	cfg := newConfig(nil)
	for _, c := range cfg.capabilities() {
		if c.IsReserved() {
			t.Errorf("the default capability set contains the reserved capability %s", c)
		}
	}
}

func TestValidateRejectsAnExplicitlyDeclaredReservedCapability(t *testing.T) {
	for _, reserved := range reservedCapabilities {
		err := validConfig(WithCapabilities(Object, reserved)).validate()
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
// The check lives in newCapabilitySet rather than only in the configuration's validation
// because the set is the single thing a declaration is derived from and the only
// constructor, so there is no second route by which a reserved capability could
// reach one. Validation calls it, so an adopter naming a reserved
// capability is refused before any scenario runs.
func TestAReservedCapabilityCannotBeDeclared(t *testing.T) {
	for _, reserved := range reservedCapabilities {
		if _, err := newCapabilitySet([]Capability{reserved}); err == nil {
			t.Errorf("newCapabilitySet accepted the reserved capability %s, so it could still "+
				"reach a conformance declaration", reserved)
		}
	}

	caps, err := newCapabilitySet(newConfig(nil).capabilities())
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

// The expiry check on reservedCapabilities.
//
// A reserved capability cannot be declared, so the day the specification adds
// the first scenario carrying one, every adopter's run would report that
// scenario as skipped for a capability nobody is permitted to claim: a green
// suite, a well-formed report, and a question silently withdrawn. Nothing else
// here would catch it -- the scenario was collected, so the under-collection
// guard is satisfied, and a capability-gated skip is explicitly not a gap.
func TestAScenarioCarryingAReservedTagFailsTheRun(t *testing.T) {
	caps, err := newCapabilitySet(nil)
	if err != nil {
		t.Fatalf("newCapabilitySet: %v", err)
	}
	r := &runner{caps: caps}

	for _, reserved := range reservedCapabilities {
		sc := scenarioWithTags("a scenario the specification has just added", reserved.Tag())

		capability, expired := expiredReservation(sc)
		if !expired {
			t.Fatalf("a scenario tagged %s did not trip the expiry check", reserved.Tag())
		}
		if capability != reserved {
			t.Fatalf("blamed %s, want %s", capability, reserved)
		}

		// And the runner must fail rather than skip. It is the same error
		// channel the gate uses, so the distinction is only in whether the
		// error wraps godog.ErrSkip -- which is exactly the mistake this guards
		// against.
		_, runErr := r.beforeScenario(context.Background(), sc)
		if runErr == nil {
			t.Fatalf("beforeScenario accepted a scenario tagged %s", reserved.Tag())
		}
		if errors.Is(runErr, godog.ErrSkip) {
			t.Fatalf("a scenario tagged %s was skipped rather than failed: %v", reserved.Tag(), runErr)
		}
		if !strings.Contains(runErr.Error(), "reservedCapabilities") {
			t.Fatalf("the failure does not say what to change: %v", runErr)
		}
	}
}

// TestTheCanonicalScenariosCarryNoReservedTag is the same check against the
// assets actually pinned, so that moving the pin is what trips it rather than
// some future adopter's run.
//
// It is a tripwire on the pin and not the gate. The gate is expiredReservation,
// which reads the scenario godog parsed and so cannot disagree with the run.
// This one reads the feature source, which is why it is deliberately crude: it
// looks for the tag as a whole token on a line that is not a Gherkin comment,
// and nothing else. The first version searched the whole source and failed
// immediately -- events.feature has a comment saying which scenario belongs
// behind @caching once someone writes it, which is the opposite of the thing
// being guarded against. Appendix F's warning about a second parser is about
// computing an expectation with one; asking "does this tag appear on a tag
// line" is narrow enough to be safe, and the runtime check is what is
// authoritative.
func TestTheCanonicalScenariosCarryNoReservedTag(t *testing.T) {
	features, err := fs.ReadDir(assets, featuresPath)
	if err != nil {
		t.Fatalf("could not list the canonical features: %v", err)
	}
	if len(features) == 0 {
		t.Fatal("no canonical feature files, so this test would pass vacuously")
	}

	for _, entry := range features {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".feature") {
			continue
		}
		source, err := fs.ReadFile(assets, featuresPath+"/"+entry.Name())
		if err != nil {
			t.Fatalf("could not read %s: %v", entry.Name(), err)
		}
		for _, line := range strings.Split(string(source), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			for _, tag := range strings.Fields(line) {
				capability, known := CapabilityForTag(tag)
				if known && capability.IsReserved() {
					t.Errorf("%s carries %s, which is still listed as reserved. Delete %s from "+
						"reservedCapabilities in capability.go; it is the only change needed, and "+
						"until it is made every adopter reports those scenarios as skipped for a "+
						"capability they cannot declare", entry.Name(), tag, capability)
				}
			}
		}
	}
}

// TestTheReportDeclarationCannotClaimAReservedCapability checks the emitted
// declaration rather than only the configuration that feeds it.
//
// The test above pins the constructor, which is what makes this guarantee
// structural. This one pins the document, because that is what a consumer
// actually reads: a future change that built a declaration by some other route
// would satisfy the constructor test and still publish the claim.
func TestTheReportDeclarationCannotClaimAReservedCapability(t *testing.T) {
	caps, err := newCapabilitySet((&Config{}).capabilities())
	if err != nil {
		t.Fatalf("the default capability set does not validate: %v", err)
	}

	r := &runner{cfg: Config{Name: "gate", Control: stubControl{}}, caps: caps}
	report := r.buildReport("gate.ndjson", "sha256:0")

	if report.Declaration.Declared == nil {
		t.Fatal("declared is nil; the schema requires an array")
	}
	for _, tag := range report.Declaration.Declared {
		if capability, known := CapabilityForTag(tag); known && capability.IsReserved() {
			t.Errorf("declaration.declared contains the reserved tag %s", tag)
		}
	}
}

// stubControl is a BackendControl that does nothing, for tests that only need
// a non-nil one.
type stubControl struct{}

func (stubControl) PrepareScenario(context.Context) error { return nil }
func (stubControl) ChangeFlag(context.Context) error      { return nil }
func (stubControl) Description() string                   { return "stub" }
func (stubControl) ControlAPI() ControlAPI                { return ControlAPIInProcess }
