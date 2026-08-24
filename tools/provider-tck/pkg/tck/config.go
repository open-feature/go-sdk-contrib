package tck

import (
	"context"
	"errors"
	"fmt"
	"time"

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

// Config is the entire contract a provider author implements to run the TCK.
//
// Three fields are required — a name, a provider factory, and the seam through
// which the backend is manipulated. Everything else has a working default.
//
// The TCK owns the provider lifecycle from here: it registers each provider
// with the OpenFeature API under a suite-scoped domain, waits for it to become
// ready, and replaces it at the end of the suite so it is shut down. Do not call
// openfeature.SetProvider or initialise the provider yourself.
type Config struct {
	// Name identifies the suite in test output. It also scopes the OpenFeature
	// domain the TCK registers providers under, so two suites in the same test
	// binary do not observe each other's providers. Required.
	//
	// Use something that reads well in a failure message: "flagd-rpc",
	// "in-memory", "go-feature-flag".
	Name string

	// NewProvider creates the provider under test, configured against a backend
	// that is already running and seeded with the canonical flag set. Called
	// once per scenario. Required.
	//
	// Return a configured but uninitialised provider. The TCK initialises it.
	NewProvider ProviderFactory

	// Control is the seam through which the TCK manipulates the backend.
	// Required.
	//
	// See BackendControl for which implementation is right for your provider.
	// The short version: a provider with a real backend drives it over the HTTP
	// control API; a provider with no backend at all may control it in-process.
	Control BackendControl

	// NewUnavailableProvider creates a provider pointed at a backend that does
	// not exist.
	//
	// Used by the initialisation-failure scenarios, which assert that a
	// provider unable to reach its backend settles into ERROR and emits
	// PROVIDER_ERROR rather than hanging or panicking out of registration.
	//
	// Point it at a closed port on localhost. Do not point it at the backend
	// under test — that must stay up and reachable, and simulated outages
	// belong to Control. Configure a short connection deadline: the scenario
	// allows a bounded time for the error event, and a provider with a
	// 30-second connect timeout will not make it.
	//
	// Required only if Capabilities includes UnavailableInit. Leaving both out
	// is the honest configuration for a provider with no backend, and the
	// scenarios that would call this are then skipped with the reason reported.
	NewUnavailableProvider ProviderFactory

	// Capabilities declares which optional parts of the provider contract this
	// provider supports. Scenarios tagged with an undeclared capability are
	// reported as skipped with the reason, never as passed.
	//
	// Defaults to AllCapabilities. Narrow it rather than widening it: start
	// from the default, run the suite, and remove only what your provider
	// genuinely cannot do.
	Capabilities []Capability

	// EventTimeout is how long to wait for a provider event to arrive.
	//
	// This is the single most important knob for a provider author, because
	// providers observe backend changes on wildly different timescales. A
	// streaming provider sees a configuration change in milliseconds; one that
	// polls every 30 seconds may need most of a poll interval to notice. Set it
	// to comfortably exceed your worst-case detection latency, or the suite
	// reports timeouts that are really just impatience.
	//
	// Individual scenarios can tighten this with the explicit "within {int}ms"
	// step, which always wins over this value.
	//
	// Defaults to 12 seconds.
	EventTimeout time.Duration

	// ReadyTimeout is how long to wait for a provider to reach READY during
	// initialisation. Defaults to 30 seconds.
	ReadyTimeout time.Duration
}

// validate reports whether the configuration can run a suite at all, naming
// what is missing rather than failing later inside a step.
func (c *Config) validate() error {
	var problems []error

	if c.Name == "" {
		problems = append(problems, errors.New("Name is required: it scopes the OpenFeature domain and identifies the suite in test output"))
	}
	if c.NewProvider == nil {
		problems = append(problems, errors.New("NewProvider is required: the TCK has nothing to test without it"))
	}
	if c.Control == nil {
		problems = append(problems, errors.New("Control is required: see tck.BackendControl for which implementation fits your provider"))
	}

	caps, err := newCapabilitySet(c.capabilities())
	if err != nil {
		problems = append(problems, err)
	} else if caps.has(UnavailableInit) && c.NewUnavailableProvider == nil {
		problems = append(problems, errors.New(
			"Capabilities declares tck.UnavailableInit but NewUnavailableProvider is nil: "+
				"the @unavailable scenarios need a provider pointed at a backend that does not exist. "+
				"Supply one, or remove the capability so those scenarios are skipped with a reason"))
	}

	return errors.Join(problems...)
}

// capabilities returns the declared capability list, defaulting to everything.
func (c *Config) capabilities() []Capability {
	if c.Capabilities == nil {
		return AllCapabilities()
	}
	return c.Capabilities
}

func (c *Config) eventTimeout() time.Duration {
	if c.EventTimeout <= 0 {
		return defaultEventTimeout
	}
	return c.EventTimeout
}

func (c *Config) readyTimeout() time.Duration {
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
func (c *Config) domain() string {
	return fmt.Sprintf("provider-tck/%s", c.Name)
}
