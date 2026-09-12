package tck

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"time"

	"github.com/cucumber/godog"
	"github.com/open-feature/go-sdk/openfeature"
)

// Default timings. Both are deliberately generous: a suite that reports a
// timeout when the provider was merely slow costs far more to diagnose than a
// suite that takes a few extra seconds to fail.
const (
	defaultEventTimeout = 12 * time.Second
	defaultReadyTimeout = 30 * time.Second
)

// ProviderFactory creates a provider under test.
//
// It is a factory rather than a single instance because each scenario gets its
// own provider, and because a provider often cannot be configured before the
// suite starts — a container stack's host ports do not exist until it is up.
type ProviderFactory func(ctx context.Context) (openfeature.FeatureProvider, error)

// EndpointProviderFactory creates a provider under test against the running
// Compose stack.
//
// The endpoint carries the dynamically mapped host ports, which is why this is
// a factory rather than a value: the ports do not exist until the stack has
// started. See WithProviderFromEndpoint.
type EndpointProviderFactory func(ctx context.Context, endpoint BackendEndpoint) (openfeature.FeatureProvider, error)

// config is the accumulated result of the options passed to Run.
//
// It is unexported on purpose. Every field here is reachable through exactly
// one Option, so the set of things an adopter can say is the set of With*
// functions in this package — which is what lets a later capability be added
// without breaking a struct literal, and what lets Run grow a second entry
// point (a TCK for something other than a provider) without a second config
// type.
type config struct {
	// Name identifies the suite in test output. Set by WithName. Required.
	Name string

	// NewProvider creates the provider under test. Set by WithProvider.
	//
	// Exactly one of NewProvider and NewProviderFromEndpoint is set; Run
	// derives the first from the second when the Compose harness owns the
	// stack.
	NewProvider ProviderFactory

	// NewProviderFromEndpoint creates the provider under test from the running
	// Compose stack's endpoint. Set by WithProviderFromEndpoint.
	NewProviderFromEndpoint EndpointProviderFactory

	// Control is the seam through which the TCK manipulates the backend. Set by
	// WithControl, or built by the Compose harness against the control API.
	Control BackendControl

	// NewUnavailableProvider creates a provider pointed at a backend that does
	// not exist. Set by WithUnavailableProvider.
	NewUnavailableProvider ProviderFactory

	// Capabilities declares which optional parts of the provider contract this
	// provider supports. Set by WithCapabilities.
	Capabilities []Capability

	// capabilitiesDeclared records that WithCapabilities was passed at all, so
	// that declaring nothing stays different from declaring nothing in
	// particular. Without it a bare WithCapabilities() would be indistinguishable
	// from silence and would quietly declare everything, which is the opposite
	// of what it says.
	capabilitiesDeclared bool

	// KnownDeviations records gaps the provider is known to have against parts
	// of the specification that are not optional. Set by WithKnownDeviations.
	KnownDeviations []KnownDeviation

	// EventTimeout is how long to wait for a provider event to arrive. Set by
	// WithEventTimeout.
	EventTimeout time.Duration

	// ReadyTimeout is how long to wait for a provider to reach READY. Set by
	// WithReadyTimeout.
	ReadyTimeout time.Duration

	// ExtensionFeatures supplies feature files of the adopter's own. Set by
	// WithFeatures.
	ExtensionFeatures fs.FS

	// ExtensionSteps registers step definitions of the adopter's own. Set by
	// WithSteps.
	ExtensionSteps func(*godog.ScenarioContext)

	// compose describes the Docker Compose stack the suite owns, or is nil when
	// the adopter supplies its own Control. Set by the compose options.
	compose *composeConfig
}

// Option configures a Run.
//
// Options rather than a struct literal, so that the suite can gain a
// capability — a Compose stack it starts itself, a tag filter, a TCK for
// something other than a provider — without every adoption having to be
// edited, and so that a required setting is named in one place rather than
// being a zero value someone has to remember means "unset".
type Option func(*config)

// WithName identifies the suite in test output. Required.
//
// It also scopes the OpenFeature domain the TCK registers providers under, so
// two suites in the same test binary do not observe each other's providers.
//
// Use something that reads well in a failure message: "flagd-rpc",
// "in-memory", "go-feature-flag".
func WithName(name string) Option {
	return func(c *config) { c.Name = name }
}

// WithProvider supplies the factory that creates the provider under test,
// configured against a backend that is already running and seeded with the
// canonical flag set. Called once per scenario.
//
// Return a configured but uninitialised provider. The TCK owns the lifecycle
// from there: it registers each provider with the OpenFeature API under a
// suite-scoped domain, waits for it to become ready, and replaces it at the end
// of the suite so it is shut down. Do not call openfeature.SetProvider or
// initialise the provider yourself.
//
// Required, unless the suite owns the stack — see WithProviderFromEndpoint,
// which is the same factory with the discovered endpoint handed to it.
func WithProvider(factory ProviderFactory) Option {
	return func(c *config) { c.NewProvider = factory }
}

// WithControl supplies the seam through which the TCK manipulates the backend.
//
// See BackendControl for which implementation is right for your provider. The
// short version: a provider with a real backend drives it over the HTTP control
// API; a provider with no backend at all may control it in-process.
//
// Required, unless the suite owns the stack: with WithComposeFile the TCK
// builds an HTTPControl against the stack's control API itself, and passing
// both is refused rather than silently preferring one.
func WithControl(control BackendControl) Option {
	return func(c *config) { c.Control = control }
}

// WithUnavailableProvider supplies a factory creating a provider pointed at a
// backend that does not exist.
//
// Used by the initialisation-failure scenarios, which assert that a provider
// unable to reach its backend settles into ERROR and emits PROVIDER_ERROR
// rather than hanging or panicking out of registration.
//
// Point it at a closed port on localhost. Do not point it at the backend under
// test — that must stay up and reachable, and simulated outages belong to the
// control. Configure a short connection deadline: the scenario allows a bounded
// time for the error event, and a provider with a 30-second connect timeout will
// not make it.
//
// Required only when WithCapabilities includes UnavailableInit. Leaving both out
// is the honest configuration for a provider with no backend, and the scenarios
// that would call this are then skipped with the reason reported.
func WithUnavailableProvider(factory ProviderFactory) Option {
	return func(c *config) { c.NewUnavailableProvider = factory }
}

// WithCapabilities declares which optional parts of the provider contract this
// provider supports. Scenarios tagged with an undeclared capability are
// reported as skipped with the reason, never as passed.
//
// Omitting the option declares AllCapabilities, which excludes the reserved
// ones. Narrow rather than widen: start from the default, run the suite, and
// remove only what your provider genuinely cannot do. Passing no capability at
// all is a declaration too — it says this provider supports none of the
// optional parts — and is not the same as omitting the option.
//
// Naming a reserved capability is rejected rather than passed into a report. No
// scenario carries a reserved tag, so declaring it cannot be verified — it is a
// configuration mistake, not a conformance result.
func WithCapabilities(capabilities ...Capability) Option {
	return func(c *config) {
		c.Capabilities = capabilities
		c.capabilitiesDeclared = true
	}
}

// WithKnownDeviations records gaps the provider is known to have against parts
// of the specification that are not optional.
//
// Narrowing WithCapabilities is how a provider says a scenario was not run, but
// it cannot say why, and the two reasons are not alike: a provider with no
// streaming transport declining ConfigurationChange has made a decision, while
// one declining NumericCoercion because it narrows 0.5 to 0 with no error code
// has a bug. In the results both are a skip with the same reason, so unless the
// provider author says which happened, a consumer comparing providers reads a
// defect as a design choice.
//
// Empty by default, which is silence rather than a claim. See KnownDeviation,
// TrackedDeviation and UntrackedDeviation for what belongs here and what does
// not.
func WithKnownDeviations(deviations ...KnownDeviation) Option {
	return func(c *config) { c.KnownDeviations = deviations }
}

// WithEventTimeout sets how long to wait for a provider event to arrive.
//
// This is the single most important knob for a provider author, because
// providers observe backend changes on wildly different timescales. A streaming
// provider sees a configuration change in milliseconds; one that polls every 30
// seconds may need most of a poll interval to notice. Set it to comfortably
// exceed your worst-case detection latency, or the suite reports timeouts that
// are really just impatience.
//
// Individual scenarios can tighten this with the explicit "within {int}ms"
// step, which always wins over this value.
//
// Defaults to 12 seconds.
func WithEventTimeout(timeout time.Duration) Option {
	return func(c *config) { c.EventTimeout = timeout }
}

// WithReadyTimeout sets how long to wait for a provider to reach READY during
// initialisation.
//
// It also bounds each direct Shutdown and Init the lifecycle scenarios make on
// the provider, so that one which never returns fails its step rather than
// hanging the test binary.
//
// Defaults to 30 seconds.
func WithReadyTimeout(timeout time.Duration) Option {
	return func(c *config) { c.ReadyTimeout = timeout }
}

// WithFeatures supplies feature files of your own, to run in the same suite as
// the canonical ones and under the same backend lifecycle.
//
// This is for behaviour the specification does not describe and cannot —
// flagd's fractional targeting, a vendor's segment rules — where the scenarios
// still need a provider registered per scenario, a backend reset between them,
// and the event plumbing the TCK already owns. Running them in a harness of
// your own means reimplementing that, and the two then drift.
//
// Any .feature file anywhere in the filesystem is picked up, so a directory is
// the usual thing to pass:
//
//	tck.WithFeatures(os.DirFS("testdata/tck-extensions"))
//
// The files appear to the run under an "extensions/" prefix, and the canonical
// assets keep their "gherkin/" one. Nothing you supply is reachable under the
// canonical prefix, so an extension file named evaluation.feature is an
// addition and never a replacement — which is the failure the Java TCK had,
// where a same-named file in a second classpath root silently displaced the
// canonical one. The same partition is what distinguishes the two in a
// conformance report: a result whose feature URI starts with "gherkin/" is
// canonical, one under "extensions/" is yours.
//
// A filesystem holding no .feature file is refused rather than quietly running
// the canonical suite alone, that being how mis-wired extensions otherwise go
// unnoticed.
//
// Optional. Omitting it runs the canonical suite exactly as before.
func WithFeatures(features fs.FS) Option {
	return func(c *config) { c.ExtensionFeatures = features }
}

// WithSteps registers step definitions of your own.
//
// It is called during scenario initialisation, after the TCK's own steps, so
// your definitions see the same scenario context and can use the same godog
// hooks. Steps whose expressions collide with the TCK's are ambiguous to godog
// and fail the scenario, so keep the wording distinct.
//
//	tck.WithSteps(func(ctx *godog.ScenarioContext) {
//	    ctx.Step(`^the fractional bucket for "([^"]*)" is "([^"]*)"$`, theBucketIs)
//	})
//
// Your steps run inside the TCK's scenario lifecycle: the provider is already
// registered and ready, and the control's PrepareScenario has already reset the
// backend. Reach the provider under test with ClientFromContext rather than
// building a client of your own. Optional.
func WithSteps(register func(*godog.ScenarioContext)) Option {
	return func(c *config) { c.ExtensionSteps = register }
}

// newConfig applies the options in order, so a later option wins over an
// earlier one and an adoption can layer a shared base on top of a per-suite
// difference.
func newConfig(opts []Option) *config {
	c := &config{}
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		opt(c)
	}
	return c
}

// validate reports whether the configuration can run a suite at all, naming
// what is missing rather than failing later inside a step.
func (c *config) validate() error {
	var problems []error

	if c.Name == "" {
		problems = append(problems, errors.New("tck.WithName is required: it scopes the OpenFeature domain and identifies the suite in test output"))
	}

	problems = append(problems, c.validateProviderSource()...)

	if err := validateDeviations(c.KnownDeviations); err != nil {
		problems = append(problems, err)
	}

	caps, err := newCapabilitySet(c.capabilities())
	if err != nil {
		problems = append(problems, err)
	} else if caps.has(UnavailableInit) && c.NewUnavailableProvider == nil {
		problems = append(problems, errors.New(
			"tck.WithCapabilities declares tck.UnavailableInit but no tck.WithUnavailableProvider was given: "+
				"the @unavailable scenarios need a provider pointed at a backend that does not exist. "+
				"Supply one, or remove the capability so those scenarios are skipped with a reason"))
	}

	return errors.Join(problems...)
}

// validateProviderSource checks the one place this configuration can be
// ambiguous: which of the two adoption paths the suite is on.
//
// The Compose path and the manual path each supply a provider factory and a
// control, and mixing them means one of the two is silently ignored — a
// provider pointed at a stack whose control the suite is not driving, or a
// control driving a stack no provider is pointed at. Both produce failures that
// look like provider defects, so the combination is refused here instead.
func (c *config) validateProviderSource() []error {
	var problems []error

	switch {
	case c.NewProvider != nil && c.NewProviderFromEndpoint != nil:
		problems = append(problems, errors.New(
			"tck.WithProvider and tck.WithProviderFromEndpoint are both set: the first builds a "+
				"provider against a backend you started, the second against the Compose stack the "+
				"suite starts. Pick the one that matches how the backend is run"))
	case c.NewProvider == nil && c.NewProviderFromEndpoint == nil:
		problems = append(problems, errors.New(
			"tck.WithProvider is required: the TCK has nothing to test without it. With a Compose "+
				"stack, use tck.WithProviderFromEndpoint, which receives the stack's discovered host ports"))
	}

	if c.compose == nil {
		if c.Control == nil {
			problems = append(problems, errors.New(
				"tck.WithControl is required: see tck.BackendControl for which implementation fits "+
					"your provider. With a Compose stack the suite builds one for you — use "+
					"tck.WithComposeFile and the TCK drives the stack's control API itself"))
		}
		if c.NewProviderFromEndpoint != nil {
			problems = append(problems, errors.New(
				"tck.WithProviderFromEndpoint was given without tck.WithComposeFile: there is no "+
					"stack for the endpoint to describe. Add tck.WithComposeFile, or build the "+
					"provider with tck.WithProvider"))
		}
		return problems
	}

	if c.Control != nil {
		problems = append(problems, errors.New(
			"tck.WithControl and tck.WithComposeFile are both set: the Compose harness builds an "+
				"HTTPControl against the stack's own control API, so a second control would drive a "+
				"backend the suite is not running. Remove one"))
	}
	if c.NewProvider != nil {
		problems = append(problems, errors.New(
			"tck.WithProvider was given with tck.WithComposeFile: the stack's host ports are assigned "+
				"when it starts, so a factory that cannot see them cannot reach it. Use "+
				"tck.WithProviderFromEndpoint"))
	}
	problems = append(problems, c.compose.validate()...)

	return problems
}

// capabilities returns the declared capability list, defaulting to everything
// declarable — AllCapabilities, which omits the reserved capabilities.
func (c *config) capabilities() []Capability {
	if !c.capabilitiesDeclared {
		return AllCapabilities()
	}
	return c.Capabilities
}

func (c *config) eventTimeout() time.Duration {
	if c.EventTimeout <= 0 {
		return defaultEventTimeout
	}
	return c.EventTimeout
}

func (c *config) readyTimeout() time.Duration {
	if c.ReadyTimeout <= 0 {
		return defaultReadyTimeout
	}
	return c.ReadyTimeout
}

// domain is the OpenFeature domain this suite registers its providers under.
//
// It is suite-scoped rather than scenario-scoped on purpose. Registering a new
// provider in the same domain replaces the previous one, and the SDK shuts the
// replaced provider down; a fresh domain per scenario would instead leave every
// provider of the suite registered and running, which for a provider holding a
// network connection means leaking one connection per scenario.
func (c *config) domain() string {
	return fmt.Sprintf("tck/%s", c.Name)
}
