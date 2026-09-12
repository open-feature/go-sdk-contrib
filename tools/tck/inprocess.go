package tck

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
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
// It is decoded from the specification's canonical-flags.json — the bytes
// CanonicalFlags returns — rather than transcribed, so that the in-memory
// suites cannot drift from the file every other language seeds from. Three
// properties of that file are load-bearing and survive the decoding:
//
//   - missing-flag is absent, which is what the FLAG_NOT_FOUND scenario tests.
//     Adding it turns that scenario green for the wrong reason.
//   - no flag gets a ContextEvaluator, so every evaluation reports reason
//     STATIC, which is what the untargeted feature files expect. The TCK tests
//     a provider's mapping of a response, not a backend's evaluation logic.
//     targeting-key-flag is the one flag in the file carrying a targeting
//     rule, and that member is deliberately not read: translating flagd's
//     JsonLogic into a ContextEvaluator would make these suites a test of a
//     rule engine written here. So the flag resolves to its miss variant
//     whatever the context, and a suite over this flag set leaves Targeting
//     undeclared rather than failing the match scenario. Undeclared is the
//     accurate report: an in-memory flag set evaluates no rules.
//   - a number keeps the type it was written with: 10 becomes an int64 and
//     10.0 a float64. memprovider type-asserts, so that is what keeps
//     integer-flag an integer and integral-float-flag a float. Plain
//     encoding/json would decode both as float64, and a loader that then
//     turned integral floats back into int64 would make integral-float-flag
//     an integer flag — which the file's own comment warns lets the lossless
//     coercion scenario pass without coercing anything. See numberValue.
func CanonicalFlagSet() map[string]memprovider.InMemoryFlag {
	flags, err := decodeCanonicalFlags(CanonicalFlags())
	if err != nil {
		// Unreachable for a pinned spec revision: the file is embedded at
		// compile time, so a failure here means the pinned assets and this
		// decoder disagree about the file's shape, which moving the pin should
		// have surfaced.
		panic("tck: canonical flag set could not be decoded: " + err.Error())
	}
	return flags
}

// canonicalFlagFile is the shape of canonical-flags.json — the flagd
// flag-definition format — reduced to what an in-memory flag needs. The
// $comment members are ignored along with any other unknown field.
type canonicalFlagFile struct {
	Flags map[string]canonicalFlag `json:"flags"`
}

type canonicalFlag struct {
	State          string         `json:"state"`
	DefaultVariant string         `json:"defaultVariant"`
	Variants       map[string]any `json:"variants"`
}

// decodeCanonicalFlags turns the canonical flag file into in-memory flags.
func decodeCanonicalFlags(data []byte) (map[string]memprovider.InMemoryFlag, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	// Keep every number as the literal it was written as, so that the split
	// into int64 and float64 in numberValue can follow the file rather than
	// float64's inability to say whether it started life as 10 or 10.0.
	decoder.UseNumber()

	var file canonicalFlagFile
	if err := decoder.Decode(&file); err != nil {
		return nil, err
	}
	if len(file.Flags) == 0 {
		return nil, errors.New("the file defines no flags")
	}

	flags := make(map[string]memprovider.InMemoryFlag, len(file.Flags))
	for key, def := range file.Flags {
		state := memprovider.State(def.State)
		if state != memprovider.Enabled && state != memprovider.Disabled {
			return nil, fmt.Errorf("flag %q: state %q is neither %q nor %q",
				key, def.State, memprovider.Enabled, memprovider.Disabled)
		}
		if _, ok := def.Variants[def.DefaultVariant]; !ok {
			return nil, fmt.Errorf("flag %q: default variant %q is not one of its variants", key, def.DefaultVariant)
		}

		variants := make(map[string]any, len(def.Variants))
		for name, raw := range def.Variants {
			value, err := fromJSONValue(raw)
			if err != nil {
				return nil, fmt.Errorf("flag %q, variant %q: %w", key, name, err)
			}
			variants[name] = value
		}

		flags[key] = memprovider.InMemoryFlag{
			Key:            key,
			State:          state,
			DefaultVariant: def.DefaultVariant,
			Variants:       variants,
		}
	}
	return flags, nil
}

// fromJSONValue converts a value decoded with UseNumber into what memprovider's
// type assertions expect, recursing into objects and arrays so that a number
// inside object-flag is converted the same way as a top-level one.
func fromJSONValue(v any) (any, error) {
	switch v := v.(type) {
	case json.Number:
		return numberValue(v)
	case map[string]any:
		out := make(map[string]any, len(v))
		for key, member := range v {
			converted, err := fromJSONValue(member)
			if err != nil {
				return nil, fmt.Errorf("member %q: %w", key, err)
			}
			out[key] = converted
		}
		return out, nil
	case []any:
		out := make([]any, len(v))
		for i, member := range v {
			converted, err := fromJSONValue(member)
			if err != nil {
				return nil, fmt.Errorf("element %d: %w", i, err)
			}
			out[i] = converted
		}
		return out, nil
	default:
		// bool, string and nil need no conversion.
		return v, nil
	}
}

// numberValue splits a JSON number on how it was written: a literal with a
// fraction or an exponent is a float64, anything else an int64.
//
// This is the whole reason the decoder runs with UseNumber. The canonical set
// deliberately contains 10 (integer-flag), 10.0 (integral-float-flag) and
// 9007199254740991 (huge-integer-flag), and only the literal text tells the
// first two apart or carries the third exactly.
func numberValue(n json.Number) (any, error) {
	if strings.ContainsAny(n.String(), ".eE") {
		f, err := n.Float64()
		if err != nil {
			return nil, fmt.Errorf("%s is not a float64: %w", n, err)
		}
		return f, nil
	}
	i, err := n.Int64()
	if err != nil {
		return nil, fmt.Errorf("%s does not fit an int64: %w", n, err)
	}
	return i, nil
}

// changingFlag is the one flag built by hand rather than decoded, because
// ChangeFlag has to rebuild it with the other default variant. Its variant
// names are pinned to the file by TestCanonicalFlagSetMatchesTheFile.
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
//	tck.Run(t,
//	    tck.WithName("in-memory"),
//	    tck.WithControl(control),
//	    tck.WithProvider(func(context.Context) (openfeature.FeatureProvider, error) {
//	        return control.NewProvider(), nil
//	    }),
//	    tck.WithCapabilities(tck.Events, tck.ConfigurationChange, tck.Object, tck.LargeIntegers),
//	)
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

// ControlAPI implements BackendControl.
//
// ControlAPIInProcess is the narrow allowance the report schema makes for a
// provider with no backend, and this control is that case: the "backend" is a
// data structure in this process. A provider that does have a backend must not
// reach for this control to claim the value — see BackendControl.
func (c *InProcessControl) ControlAPI() ControlAPI { return ControlAPIInProcess }

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
