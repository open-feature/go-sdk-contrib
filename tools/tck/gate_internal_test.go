package tck

import (
	"context"
	"errors"
	"io/fs"
	"reflect"
	"sort"
	"strings"
	"testing"
	"testing/fstest"
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

// canonicalScenarioWithTags and extensionScenarioWithTags are the same thing
// with a URI, for the checks that key on which root a scenario came from. The
// prefixes are the ones featureSources mounts, so these cannot drift from what
// a run actually produces without the mount changing too.
func canonicalScenarioWithTags(name string, tags ...string) *godog.Scenario {
	sc := scenarioWithTags(name, tags...)
	sc.Uri = featuresPath + "/errors.feature"
	return sc
}

func extensionScenarioWithTags(name string, tags ...string) *godog.Scenario {
	sc := scenarioWithTags(name, tags...)
	sc.Uri = extensionsRoot + "/vendor.feature"
	return sc
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

	// The exclusion must be exactly the two undeclarable sets, not a convenient
	// subset: a capability quietly dropped from the default is a gap an adopter
	// never sees reported.
	returned := make(map[Capability]bool, len(allCapabilities))
	for _, c := range AllCapabilities() {
		returned[c] = true
	}
	for _, c := range allCapabilities {
		_, inexpressible := c.IsInexpressible()
		undeclarable := c.IsReserved() || inexpressible
		if undeclarable == returned[c] {
			t.Errorf("AllCapabilities is wrong about %s: reserved=%v, inexpressible=%v, returned=%v",
				c, c.IsReserved(), inexpressible, returned[c])
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

// The reverse of the expiry check: a tag the assets carry and this vocabulary
// has never heard of.
//
// It is the easier of the two to leave out, because ignoring an unknown tag
// looks tolerant. It is the opposite. An unknown tag gates nothing, so the
// scenarios carrying it stay mandatory for every adopter, and a provider that
// legitimately withholds the new capability shows unexplained failures while
// every other provider stays green. Nothing in the results says why.

// TestACanonicalScenarioCarryingAnUnknownTagFailsTheRun pins the gate.
func TestACanonicalScenarioCarryingAnUnknownTagFailsTheRun(t *testing.T) {
	caps, err := newCapabilitySet(nil)
	if err != nil {
		t.Fatalf("newCapabilitySet: %v", err)
	}
	r := &runner{caps: caps}

	sc := canonicalScenarioWithTags(
		"a scenario gated on a capability this suite has not learned",
		"@events", "@some-future-capability")

	tag, unknown := unknownCapabilityTag(sc)
	if !unknown {
		t.Fatal("a canonical scenario carrying an unresolvable tag did not trip the check")
	}
	if tag != "@some-future-capability" {
		t.Fatalf("blamed %q, want %q", tag, "@some-future-capability")
	}

	// It has to fail rather than skip, which is the same error channel the
	// gate uses and so the same mistake worth guarding against: a skip here
	// would report the scenario as legitimately not run.
	_, runErr := r.beforeScenario(context.Background(), sc)
	if runErr == nil {
		t.Fatal("beforeScenario accepted a canonical scenario with an unresolvable tag")
	}
	if errors.Is(runErr, godog.ErrSkip) {
		t.Fatalf("the scenario was skipped rather than failed: %v", runErr)
	}
	if !strings.Contains(runErr.Error(), "@some-future-capability") {
		t.Errorf("the failure does not name the tag: %v", runErr)
	}
	if !strings.Contains(runErr.Error(), "capability.go") {
		t.Errorf("the failure does not say what to change: %v", runErr)
	}
}

// TestAKnownTagOnACanonicalScenarioIsAccepted is the other half, so that the
// check above cannot pass by rejecting everything.
func TestAKnownTagOnACanonicalScenarioIsAccepted(t *testing.T) {
	for _, capability := range AllCapabilities() {
		sc := canonicalScenarioWithTags("a canonical scenario", capability.Tag())
		if tag, unknown := unknownCapabilityTag(sc); unknown {
			t.Errorf("%s is in the vocabulary but %q was reported unknown", capability, tag)
		}
	}
}

// TestAnExtensionScenarioMayCarryAnyTag keeps an adopter's own features out of
// this check.
//
// A vendor's organisational tags are the vendor's to choose; this suite has no
// vocabulary for them and should not pretend to. The partition is the one
// featureSources mounts, which is also what the Messages stream keys on, rather
// than a second notion of what is canonical.
func TestAnExtensionScenarioMayCarryAnyTag(t *testing.T) {
	caps, err := newCapabilitySet(nil)
	if err != nil {
		t.Fatalf("newCapabilitySet: %v", err)
	}
	r := &runner{caps: caps}

	sc := extensionScenarioWithTags("a vendor scenario", "@fractional", "@wip")

	if tag, unknown := unknownCapabilityTag(sc); unknown {
		t.Fatalf("an extension scenario's own tag %q was rejected; a vendor's tags are not this "+
			"suite's vocabulary", tag)
	}

	// And they must not gate it either, which is the older half of the same
	// rule: an extension scenario carrying tags this suite cannot resolve runs
	// rather than being skipped for a capability nobody declared.
	if capability, missing := r.missingCapability(sc); missing {
		t.Fatalf("an extension scenario was gated on %s by a tag of the vendor's own", capability)
	}
}

// TestTheCanonicalScenariosCarryNoUnknownTag is the same check against the
// assets actually pinned, so moving the pin is what trips it rather than some
// future adopter's run.
//
// It is the tripwire and not the gate, exactly as
// TestTheCanonicalScenariosCarryNoReservedTag is, and it is deliberately crude
// in the same way: a Gherkin tag line is a line whose every token starts with
// an at-sign, and nothing else is considered. That narrowness is what keeps it
// off the prose -- errors.feature's comments discuss @string-typing and
// @fully-typed-values by name, and a looser scan would read those as tags.
//
// This is the check that fires on the pin move that adds a capability, and it
// names the file, which is what turns "some adopter's suite went red" into one
// line to add to capability.go.
func TestTheCanonicalScenariosCarryNoUnknownTag(t *testing.T) {
	features, err := fs.ReadDir(assets, featuresPath)
	if err != nil {
		t.Fatalf("could not list the canonical features: %v", err)
	}
	if len(features) == 0 {
		t.Fatal("no canonical feature files, so this test would pass vacuously")
	}

	seen := 0
	for _, entry := range features {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".feature") {
			continue
		}
		source, err := fs.ReadFile(assets, featuresPath+"/"+entry.Name())
		if err != nil {
			t.Fatalf("could not read %s: %v", entry.Name(), err)
		}
		for _, line := range strings.Split(string(source), "\n") {
			fields := strings.Fields(strings.TrimSpace(line))
			if len(fields) == 0 {
				continue
			}
			tagLine := true
			for _, field := range fields {
				if !strings.HasPrefix(field, "@") {
					tagLine = false
					break
				}
			}
			if !tagLine {
				continue
			}
			for _, tag := range fields {
				seen++
				if _, known := CapabilityForTag(tag); !known {
					t.Errorf("%s carries the tag %s, which the capability vocabulary does not "+
						"know. The pin has grown a capability this suite has not learned: add a "+
						"constant for it in capability.go and a line in allCapabilities, then "+
						"decide per adoption whether to declare it. Until that is done the tag "+
						"gates nothing, so its scenarios stay mandatory for every adopter and a "+
						"provider that legitimately withholds the capability fails them with no "+
						"explanation", entry.Name(), tag)
				}
			}
		}
	}

	if seen == 0 {
		t.Fatal("found no tag line in the canonical features, so this test would pass vacuously " +
			"-- the scan or the Gherkin's shape has changed")
	}
	t.Logf("checked %d tag(s) on the canonical scenarios against the vocabulary", seen)
}

// TestTheCanonicalAssetsMatchTheirDigest is the revision check's own tripwire,
// and the place a pin move is serviced.
//
// verifyCanonicalAssets runs inside Run, so the check itself is in force for
// every adoption. What this adds is the message: when the pin moves, an
// adopter's suite would otherwise fail with a digest mismatch and no value to
// replace it with, so this test prints the new digest, and updating the
// constant from it is the whole of the change.
//
// It is deliberately not a test of assetsDigest's arithmetic. What it pins is
// that the constant and the embedded bytes agree, which is the only thing a
// consumer of this suite depends on.
func TestTheCanonicalAssetsMatchTheirDigest(t *testing.T) {
	got, err := assetsDigest()
	if err != nil {
		t.Fatalf("the embedded assets could not be fingerprinted: %v", err)
	}

	if got != canonicalAssetsDigest {
		t.Errorf("the embedded conformance assets do not match canonicalAssetsDigest.\n"+
			"  constant: %s\n  embedded: %s\n"+
			"If the pin in go.mod has just moved, this is the expected failure and the fix is to "+
			"set canonicalAssetsDigest in assets.go to the embedded value above. Read what changed "+
			"in the assets first: a new capability tag needs a line in capability.go, and a new "+
			"feature file needs one in TestEveryCanonicalFeatureFileIsCollected, and neither is "+
			"implied by updating this digest.", canonicalAssetsDigest, got)
	}

	if err := verifyCanonicalAssets(); err != nil && got == canonicalAssetsDigest {
		t.Errorf("the digest matches but verifyCanonicalAssets still refused the run: %v", err)
	}
}

// TestTheAssetsDigestNoticesAChangedAsset keeps the check above from being
// vacuous.
//
// A fingerprint that ignored what it was given would match the constant
// forever and report every stale asset as fine, so the property worth pinning
// is that a change to the bytes changes the value. It is exercised against a
// copy of the embedded set rather than against the real one, which cannot be
// mutated -- and that is also why assetsDigest takes its input from a package
// variable.
func TestTheAssetsDigestNoticesAChangedAsset(t *testing.T) {
	original, err := assetsDigest()
	if err != nil {
		t.Fatalf("fingerprinting the embedded assets: %v", err)
	}

	// Every artifact, with one scenario tagged the way a future specification
	// revision would tag it.
	mutated := fstest.MapFS{}
	err = fs.WalkDir(assets, ".", func(name string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		body, readErr := fs.ReadFile(assets, name)
		if readErr != nil {
			return readErr
		}
		mutated[name] = &fstest.MapFile{Data: body}
		return nil
	})
	if err != nil {
		t.Fatalf("copying the embedded assets: %v", err)
	}

	target := featuresPath + "/errors.feature"
	if _, ok := mutated[target]; !ok {
		t.Fatalf("%s is not in the embedded set, so this test cannot mutate it", target)
	}

	restore := assets
	t.Cleanup(func() { assets = restore })

	assets = mutated
	unchanged, err := assetsDigest()
	if err != nil {
		t.Fatalf("fingerprinting the copy: %v", err)
	}
	if unchanged != original {
		t.Fatalf("a byte-for-byte copy of the assets fingerprinted differently:\n  %s\n  %s\n"+
			"The digest depends on something other than the paths and contents, so it would be "+
			"unstable for reasons that say nothing about the assets", original, unchanged)
	}

	mutated[target] = &fstest.MapFile{Data: append([]byte("@some-future-capability\n"), mutated[target].Data...)}
	changed, err := assetsDigest()
	if err != nil {
		t.Fatalf("fingerprinting the mutated copy: %v", err)
	}
	if changed == original {
		t.Fatal("changing a canonical feature file did not change the digest, so the revision " +
			"check would pass over exactly the drift it exists to catch")
	}
	if err := verifyCanonicalAssets(); err == nil {
		t.Fatal("verifyCanonicalAssets accepted assets that do not match the recorded digest")
	}
}

// TestEveryCanonicalFeatureFileIsCollected is the under-collection tripwire on
// the pin.
//
// The assets arrive through an embed.FS built in the spec module from
// `//go:embed gherkin/*.feature`, and this package reads whatever that pattern
// matched. A pattern is not a guarantee: a file added to the canonical set in a
// later revision, renamed, or moved into a subdirectory is picked up silently
// or not at all, and the failure mode of "not at all" is a suite that stays
// green while asking fewer questions than it advertises. Every count in this
// package's README, and every adopter's expectation about what a run covers,
// rests on the set below.
//
// So it is spelled out rather than derived. A pin that changes it fails here,
// which is the point at which somebody reads the new file and decides what it
// means for this suite -- rather than at the point where an adopter wonders why
// a scenario they read about never ran.
func TestEveryCanonicalFeatureFileIsCollected(t *testing.T) {
	want := []string{
		"errors.feature",
		"evaluation.feature",
		"events.feature",
		"lifecycle.feature",
		"metadata.feature",
		"reason.feature",
	}

	entries, err := fs.ReadDir(assets, featuresPath)
	if err != nil {
		t.Fatalf("could not list the canonical features: %v", err)
	}

	var got []string
	for _, entry := range entries {
		if entry.IsDir() {
			t.Errorf("%s/%s is a directory; the embed pattern is one level deep, so anything "+
				"below it is not collected at all", featuresPath, entry.Name())
			continue
		}
		got = append(got, entry.Name())

		// A collected but empty file is the same failure wearing a different
		// hat: the name is there and the scenarios are not.
		source, err := fs.ReadFile(assets, featuresPath+"/"+entry.Name())
		if err != nil {
			t.Errorf("could not read %s: %v", entry.Name(), err)
			continue
		}
		if !strings.Contains(string(source), "Scenario") {
			t.Errorf("%s carries no scenario", entry.Name())
		}
	}

	sort.Strings(got)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("the canonical feature set is %v, want %v. If the pin moved and this is the new "+
			"set, update the list here -- and check that the capability vocabulary in "+
			"capability.go covers whatever the new file is gated on, since an undeclarable tag "+
			"skips its scenarios for every adopter", got, want)
	}
}

// A capability the language's SDK cannot express is a different refusal from a
// reserved one, and Appendix F requires it to be the implementation's job
// rather than every adopter's. Go has no instance of it, so these tests do two
// separate things: they pin *why* Go has none, against the SDK rather than
// against a comment, and they exercise the refusal itself by installing one,
// so the path is not dead code waiting for the first adopter to discover it.

// withInexpressible installs a capability the SDK is said not to express, for
// the duration of one test.
//
// Go has no real instance, so every test of the refusal has to make one. The
// map is replaced rather than mutated so the original is restored exactly, and
// nothing in this package runs tests in parallel.
func withInexpressible(t *testing.T, c Capability, reason string) {
	t.Helper()

	original := inexpressibleCapabilities
	replacement := make(map[Capability]string, len(original)+1)
	for k, v := range original {
		replacement[k] = v
	}
	replacement[c] = reason
	inexpressibleCapabilities = replacement

	t.Cleanup(func() { inexpressibleCapabilities = original })
}

// TestGoExpressesEveryCapability is the claim the empty map makes, stated where
// it will be read if it ever stops being true.
//
// It is not a tautology: it fails the moment somebody adds an entry, which
// forces the two evidence tests below to be revisited and the table in
// Appendix F to gain a row. The rule exists precisely because a fact about a
// language, left as documentation, gets remembered wrongly somewhere.
func TestGoExpressesEveryCapability(t *testing.T) {
	if len(inexpressibleCapabilities) != 0 {
		t.Errorf("inexpressibleCapabilities is %v, and Go is recorded as having none. If the SDK "+
			"changed, this is the right place to say so -- but update Appendix F's table too, "+
			"and check the evidence tests below, which measure the two properties the other "+
			"languages fail on", inexpressibleCapabilities)
	}
}

// TestTheIntegerAccessorIsWideEnoughToAskForALargeInteger is why Go can declare
// @large-integers and Java cannot.
//
// Java's integer accessor is a 32-bit Integer, so 2^53-1 cannot be passed to it
// or returned from it and the question the scenario asks is unaskable. Go's is
// int64. Measured off the SDK's own method signature rather than off its
// documentation, so a narrowing in a future SDK fails here instead of turning
// into a provider's apparent defect.
func TestTheIntegerAccessorIsWideEnoughToAskForALargeInteger(t *testing.T) {
	const maxSafeInteger = int64(9007199254740991) // 2^53-1, the value the scenario asks for

	accessor := reflect.TypeOf((*openfeature.Client).IntValueDetails)
	// (receiver, ctx, flag, defaultValue, evalCtx, ...options)
	defaultValue := accessor.In(3)
	if defaultValue.Kind() != reflect.Int64 {
		t.Fatalf("Client.IntValueDetails takes a %s default value; @large-integers is only "+
			"expressible because it takes an int64", defaultValue)
	}

	details := accessor.Out(0)
	value, ok := details.FieldByName("Value")
	if !ok || value.Type.Kind() != reflect.Int64 {
		t.Fatalf("Client.IntValueDetails returns %s, whose Value is not an int64; the value could "+
			"not survive the round trip the scenario asserts", details)
	}

	// The round trip the scenario performs, on the accessor's own types.
	if got := int64(maxSafeInteger); got != maxSafeInteger {
		t.Fatalf("2^53-1 does not survive the integer accessor's type: %d", got)
	}
}

// TestTheIntegerAndFloatAccessorsAreDistinctTypes is why Go can declare
// @numeric-coercion and JavaScript cannot.
//
// JavaScript has one numeric type, so "a float flag requested as an integer" is
// not a question its API can put -- typeof 10 and typeof 0.5 are both 'number'
// and there is no second accessor to ask through. Go has two accessors over two
// types, which is the entire reason the three coercion scenarios mean anything
// here.
func TestTheIntegerAndFloatAccessorsAreDistinctTypes(t *testing.T) {
	integer := reflect.TypeOf((*openfeature.Client).IntValueDetails).In(3)
	float := reflect.TypeOf((*openfeature.Client).FloatValueDetails).In(3)

	if integer == float {
		t.Fatalf("the integer and float accessors both take %s, so \"a float requested as an "+
			"integer\" is not a question this SDK can put and @numeric-coercion would be "+
			"inexpressible", integer)
	}
	if integer.Kind() != reflect.Int64 || float.Kind() != reflect.Float64 {
		t.Fatalf("the accessors take %s and %s, want int64 and float64", integer, float)
	}

	// The distinction has to survive into the resolved value as well, or a
	// provider could not report a coercion it refused to perform.
	intValue, _ := reflect.TypeOf((*openfeature.Client).IntValueDetails).Out(0).FieldByName("Value")
	floatValue, _ := reflect.TypeOf((*openfeature.Client).FloatValueDetails).Out(0).FieldByName("Value")
	if intValue.Type == floatValue.Type {
		t.Fatalf("both accessors resolve a %s, so the two directions of the coercion rule are "+
			"indistinguishable", intValue.Type)
	}
}

// TestTheInexpressibleSetIsWellFormed pins the shape of an entry, so that the
// first one anybody adds is usable by every message that reads it.
func TestTheInexpressibleSetIsWellFormed(t *testing.T) {
	for c, reason := range inexpressibleCapabilities {
		if _, known := CapabilityForTag(string(c)); !known {
			t.Errorf("%s is listed as inexpressible but is not a capability, so no scenario and "+
				"no message would ever reach it", c)
		}
		if c.IsReserved() {
			t.Errorf("%s is both reserved and inexpressible; the two refusals say different "+
				"things and a capability carried by no scenario cannot also be one this "+
				"language fails to ask", c)
		}
		if strings.TrimSpace(reason) == "" {
			t.Errorf("%s is listed as inexpressible with no reason: the refusal exists to tell an "+
				"adopter which property of their SDK it is, and an empty one tells them "+
				"nothing", c)
		}
	}
}

func TestAnInexpressibleCapabilityCannotBeDeclared(t *testing.T) {
	const reason = "the integer accessor is 32 bits wide, so 2^53-1 cannot be asked for"
	withInexpressible(t, LargeIntegers, reason)

	_, err := newCapabilitySet([]Capability{Object, LargeIntegers})
	if err == nil {
		t.Fatal("a capability the SDK cannot express was accepted; it would reach a conformance " +
			"report as a claim no scenario in this language could have verified")
	}
	if !strings.Contains(err.Error(), LargeIntegers.Tag()) {
		t.Errorf("the error does not name the capability: %v", err)
	}
	if !strings.Contains(err.Error(), reason) {
		t.Errorf("the error does not say which property of the SDK makes it impossible, which is "+
			"the only part an adopter can act on: %v", err)
	}
}

// TestValidateRejectsAnInexpressibleCapability pins that an adopter meets the
// refusal as a configuration error, before the stack starts and before any
// scenario runs, rather than as a puzzling skip afterwards.
func TestValidateRejectsAnInexpressibleCapability(t *testing.T) {
	withInexpressible(t, NumericCoercion, "the language has a single numeric type")

	if err := validConfig(WithCapabilities(Object, NumericCoercion)).validate(); err == nil {
		t.Fatal("tck.Run would have started the stack with an undeclarable capability declared")
	}
}

func TestAllCapabilitiesOmitsAnInexpressibleCapability(t *testing.T) {
	withInexpressible(t, NumericCoercion, "the language has a single numeric type")

	for _, c := range AllCapabilities() {
		if c == NumericCoercion {
			t.Fatal("AllCapabilities returned a capability the SDK cannot express; an adoption " +
				"starting from the full set would be refused for something it did not choose")
		}
	}

	// The default when tck.WithCapabilities is unset is the same set, and it is
	// the shape that put unverifiable claims into a published report before.
	for _, c := range newConfig(nil).capabilities() {
		if c == NumericCoercion {
			t.Fatal("the default capability set contains a capability the SDK cannot express")
		}
	}
}

// TestTheTwoRefusalsAreDistinguishable is the part of Appendix F's rule that is
// easy to lose by tidying, and the reason the two checks are not one predicate.
//
// A reader seeing a capability absent from a report has to be able to tell
// "this provider declined" from "no provider in this language can be asked",
// because only the first says anything about the provider.
func TestTheTwoRefusalsAreDistinguishable(t *testing.T) {
	withInexpressible(t, LargeIntegers, "the integer accessor is 32 bits wide")

	_, inexpressibleErr := newCapabilitySet([]Capability{LargeIntegers})
	_, reservedErr := newCapabilitySet([]Capability{Caching})
	if inexpressibleErr == nil || reservedErr == nil {
		t.Fatal("one of the two refusals did not fire")
	}

	if inexpressibleErr.Error() == reservedErr.Error() {
		t.Fatal("the two refusals produce the same message, so an adopter cannot tell a global " +
			"reservation that expires from a permanent property of their language")
	}
	if strings.Contains(inexpressibleErr.Error(), "is reserved") {
		t.Errorf("the inexpressibility refusal describes itself as a reservation: %v", inexpressibleErr)
	}
	if strings.Contains(reservedErr.Error(), "cannot be expressed") {
		t.Errorf("the reservation refusal describes itself as an SDK limitation: %v", reservedErr)
	}
}

// TestAnInexpressibleCapabilitySkipsWithItsOwnReason is the same distinction at
// the other end: a skipped scenario has to say which of the two it is.
func TestAnInexpressibleCapabilitySkipsWithItsOwnReason(t *testing.T) {
	const reason = "the integer accessor is 32 bits wide, so 2^53-1 cannot be asked for"
	withInexpressible(t, LargeIntegers, reason)

	caps, err := newCapabilitySet([]Capability{Object})
	if err != nil {
		t.Fatalf("newCapabilitySet: %v", err)
	}
	r := &runner{caps: caps, cfg: config{Name: "gate", Control: stubControl{}}, t: t}

	_, hookErr := r.beforeScenario(context.Background(), scenarioWithTags("huge", "@large-integers"))
	if hookErr == nil {
		t.Fatal("a scenario needing a capability the SDK cannot express was allowed to run")
	}
	if !strings.Contains(hookErr.Error(), godog.ErrSkip.Error()) {
		t.Fatalf("the scenario was failed rather than skipped: %v", hookErr)
	}
	if !strings.Contains(hookErr.Error(), reason) {
		t.Errorf("the skip reason does not say the SDK cannot ask the question: %v", hookErr)
	}
	if strings.Contains(hookErr.Error(), "this provider does not declare") {
		t.Errorf("the skip reason blames the provider for a property of the SDK: %v", hookErr)
	}

	// The ordinary case must not pick up the new wording, or every withheld
	// capability starts reading as a language limitation.
	_, plainErr := r.beforeScenario(context.Background(), scenarioWithTags("outage", "@stale"))
	if plainErr == nil {
		t.Fatal("an undeclared capability did not skip its scenario")
	}
	if strings.Contains(plainErr.Error(), "cannot express") {
		t.Errorf("a capability the provider simply withheld was reported as inexpressible: %v", plainErr)
	}

	// And the end-of-run summary keeps them apart too, since that is what an
	// operator actually reads.
	r.reportSkips()
	if len(r.skips) != 2 {
		t.Fatalf("recorded %d skips, want 2", len(r.skips))
	}
	var recorded int
	for _, s := range r.skips {
		if s.capability == LargeIntegers && s.inexpressible == reason {
			recorded++
		}
		if s.capability == Stale && s.inexpressible != "" {
			t.Errorf("a withheld capability was recorded as inexpressible: %+v", s)
		}
	}
	if recorded != 1 {
		t.Error("the skip record did not carry the SDK property, so the summary cannot print it")
	}
}

func TestADeviationMayNotNameAnInexpressibleCapability(t *testing.T) {
	const reason = "the language has a single numeric type"
	withInexpressible(t, NumericCoercion, reason)

	err := deviationConfig(UntrackedDeviation(NumericCoercion, "narrows 0.5 to 0")).validate()
	if err == nil {
		t.Fatal("a deviation named a capability the SDK cannot express; it would record a " +
			"property of the SDK as a defect of the provider")
	}
	if !strings.Contains(err.Error(), reason) {
		t.Errorf("the error does not say why the capability is undeclarable: %v", err)
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
