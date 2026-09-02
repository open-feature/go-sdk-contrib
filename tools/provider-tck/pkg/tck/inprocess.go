package tck

import (
	"context"
	"errors"
	"sync"

	"github.com/open-feature/go-sdk/openfeature/memprovider"
)

// Flag keys and variant names from the canonical flag set. They are named
// constants because the step definitions, the canonical flag set and the
// control API all have to agree on them.
const (
	// ChangingFlagKey is the flag ChangeFlag mutates. The suite asserts only
	// that its resolved value differs afterwards.
	ChangingFlagKey = "changing-flag"

	changingBaseline = "foo"
	changingChanged  = "bar"
)

// CanonicalFlagSet returns the canonical flag set as Go in-memory flags.
//
// It mirrors the canonical-flags.json of the spec submodule entry for entry.
// Two details from that file are load-bearing and hold here too:
//
//   - missing-flag is absent, which is what the FLAG_NOT_FOUND scenario tests.
//     Adding it turns that scenario green for the wrong reason.
//   - no flag carries a ContextEvaluator, so every evaluation reports reason
//     STATIC, which is what the feature files expect. The TCK tests a
//     provider's mapping of a response, not a backend's evaluation logic.
//
// Integer values are int64 and float values are float64 so that the two numeric
// types stay distinct: memprovider widens int to int64 but never converts
// between integer and float, which is the behaviour StrictNumericTyping
// describes.
func CanonicalFlagSet() map[string]memprovider.InMemoryFlag {
	return map[string]memprovider.InMemoryFlag{
		"boolean-flag": {
			Key:            "boolean-flag",
			State:          memprovider.Enabled,
			DefaultVariant: "on",
			Variants: map[string]any{
				"on":  true,
				"off": false,
			},
		},
		"string-flag": {
			Key:            "string-flag",
			State:          memprovider.Enabled,
			DefaultVariant: "greeting",
			Variants: map[string]any{
				"greeting": "hi",
				"parting":  "bye",
			},
		},
		"integer-flag": {
			Key:            "integer-flag",
			State:          memprovider.Enabled,
			DefaultVariant: "ten",
			Variants: map[string]any{
				"one": int64(1),
				"ten": int64(10),
			},
		},
		"float-flag": {
			Key:            "float-flag",
			State:          memprovider.Enabled,
			DefaultVariant: "half",
			Variants: map[string]any{
				"tenth": 0.1,
				"half":  0.5,
			},
		},
		"object-flag": {
			Key:            "object-flag",
			State:          memprovider.Enabled,
			DefaultVariant: "template",
			Variants: map[string]any{
				"empty": map[string]any{},
				"template": map[string]any{
					"showImages":    true,
					"title":         "Check out these pics!",
					"imagesPerPage": int64(100),
				},
			},
		},
		// A string flag, evaluated as a boolean by the TYPE_MISMATCH scenario.
		"wrong-flag": {
			Key:            "wrong-flag",
			State:          memprovider.Enabled,
			DefaultVariant: "one",
			Variants: map[string]any{
				"one": "uno",
				"two": "dos",
			},
		},
		ChangingFlagKey: changingFlag(changingBaseline),
	}
}

func changingFlag(defaultVariant string) memprovider.InMemoryFlag {
	return memprovider.InMemoryFlag{
		Key:            ChangingFlagKey,
		State:          memprovider.Enabled,
		DefaultVariant: defaultVariant,
		Variants: map[string]any{
			changingBaseline: changingBaseline,
			changingChanged:  changingChanged,
		},
	}
}

// InProcessControl is a BackendControl that manipulates an in-process provider
// directly, with no backend, no container and no HTTP.
//
// It exists so that providers with nothing to connect to — in-memory,
// environment-variable and file-based providers — can run the TCK. For those,
// "the backend" is a data structure in the same process: seeding flags is
// building a map, and changing one is an update on the live provider, so the
// event the suite awaits is the provider's own PROVIDER_CONFIGURATION_CHANGED
// rather than one the TCK synthesised.
//
// This is not a shortcut for providers that do have a backend. Reaching into an
// external backend from inside the test process — a test-only admin client, a
// shared database handle, a hook in the provider — produces a suite that passes
// while proving nothing, because the path it exercised is not the path the
// contract describes. Those providers drive the HTTP control API instead. See
// BackendControl.
//
// # Connection control
//
// InProcessControl deliberately does not implement ConnectionControl. An
// in-memory provider has no connection to lose, and pretending otherwise with a
// no-op would report the @stale scenarios as passed. A suite using it leaves
// the Stale and UnavailableInit capabilities undeclared, and those scenarios
// are reported as skipped with the reason.
//
// # Ownership of the provider
//
// This type both seeds the flags and creates the provider that serves them,
// because in-process they are the same object: ChangeFlag has to reach the live
// provider instance to emit an event from it. A suite therefore wires both
// through one control:
//
//	control := tck.NewInProcessControl()
//	tck.Run(t, tck.Config{
//	    Name:    "in-memory",
//	    Control: control,
//	    NewProvider: func(ctx context.Context) (openfeature.FeatureProvider, error) {
//	        return control.NewProvider(), nil
//	    },
//	    Capabilities: []tck.Capability{tck.Events, tck.ConfigurationChange, tck.Object, tck.StrictNumericTyping},
//	})
type InProcessControl struct {
	mu sync.Mutex

	// current is the provider serving the scenario in flight, or nil between
	// scenarios.
	current *ControllableProvider

	// changingVariant is which variant changing-flag currently resolves to.
	changingVariant string
}

var _ BackendControl = (*InProcessControl)(nil)

// NewInProcessControl returns a control over a fresh canonical flag set.
func NewInProcessControl() *InProcessControl {
	return &InProcessControl{changingVariant: changingBaseline}
}

// NewProvider creates the provider for the scenario about to run, seeded with
// the canonical flag set.
//
// Each call returns a fresh instance over a fresh copy of the baseline, which
// is what makes PrepareScenario nothing more than dropping the previous
// reference.
func (c *InProcessControl) NewProvider() *ControllableProvider {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.changingVariant = changingBaseline
	c.current = NewControllableProvider(CanonicalFlagSet())
	return c.current
}

// Description implements BackendControl.
func (c *InProcessControl) Description() string {
	return "in-process control of an in-memory provider"
}

// PrepareScenario implements BackendControl.
//
// Dropping the reference to the previous scenario's provider is the whole
// reset: the baseline is rebuilt per provider, so the NewProvider call that
// follows produces a provider already at the baseline. Clearing the reference
// rather than leaving it dangling means a scenario that changes flags without
// creating a provider fails with a clear message instead of mutating a provider
// that has already been shut down.
func (c *InProcessControl) PrepareScenario(context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.current = nil
	return nil
}

// ChangeFlag implements BackendControl.
//
// It flips changing-flag between its two variants on the live provider, so the
// event the suite awaits is the provider's own PROVIDER_CONFIGURATION_CHANGED,
// carrying changing-flag in its FlagChanges, and not a signal the TCK
// synthesised.
//
// Alternating rather than assigning a fixed variant keeps repeated calls within
// one scenario meaningful; the suite asserts that the resolved value differs,
// not what it became.
func (c *InProcessControl) ChangeFlag(context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.current == nil {
		return errors.New("no in-memory provider exists for this scenario: in-process control " +
			"manipulates the provider itself, so the scenario must create one — with " +
			"\"Given a stable provider\" — before any step that changes flag state")
	}

	if c.changingVariant == changingChanged {
		c.changingVariant = changingBaseline
	} else {
		c.changingVariant = changingChanged
	}

	return c.current.UpdateFlag(ChangingFlagKey, changingFlag(c.changingVariant))
}
