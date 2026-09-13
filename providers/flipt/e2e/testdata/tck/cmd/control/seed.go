package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"sort"

	tck "github.com/open-feature/spec/specification/assets/provider-tck"
)

// canonicalFlag is one entry of the canonical flag set, in the flagd
// flag-definition format the conformance assets ship. Only the fields the
// seed needs are decoded: the canonical set carries no rollouts, segments or
// rules (those live in flagd-testbed, not here), so their Flipt equivalents
// stay hand-coded — see segmentsFor, the targeting special-case in seed, and
// the threshold synthesis below.
type canonicalFlag struct {
	State          string         `json:"state"`
	Variants       map[string]any `json:"variants"`
	DefaultVariant string         `json:"defaultVariant"`
}

// loadBaseline reads the canonical flag set out of the embedded conformance
// assets and translates it into the Flipt baseline this testbed seeds. The
// translation is total over the current canonical set: every flag the assets
// carry seeds exactly the payload the hardcoded table it replaces produced.
// A flag the translation cannot express fails the boot loudly rather than
// seeding something the suite did not ask for.
func loadBaseline() ([]flagDef, error) {
	raw, err := fs.ReadFile(tck.FS, "flags/canonical-flags.json")
	if err != nil {
		return nil, fmt.Errorf("read canonical flags: %w", err)
	}
	// UseNumber, not float64: the literal text is the value. A float64 round
	// trip changes 9007199254740991 to ...992 and renders 10.0 as 10, and
	// both are load-bearing canonical values (see the $comment in the JSON).
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var set struct {
		Flags map[string]canonicalFlag `json:"flags"`
	}
	if err := dec.Decode(&set); err != nil {
		return nil, fmt.Errorf("decode canonical flags: %w", err)
	}

	out := make([]flagDef, 0, len(set.Flags))
	for _, key := range sortedKeys(set.Flags) {
		fp, err := translateFlag(key, set.Flags[key])
		if err != nil {
			return nil, err
		}
		out = append(out, flagDef{key: key, payload: fp})
	}
	return out, nil
}

// translateFlag maps one canonical flag onto its Flipt resource payload. The
// default variant's value decides the shape: a bool becomes a threshold
// rollout (Flipt booleans carry no variants, and the canonical set has no
// rollout concept, so the 100% rollout is synthesized here), an object
// becomes attachment variants keyed by canonical name, and a scalar becomes
// key-carried variants (the REST schema rejects non-object attachments).
func translateFlag(key string, cf canonicalFlag) (flagPayload, error) {
	var enabled bool
	switch cf.State {
	case "ENABLED":
		enabled = true
	case "DISABLED":
	default:
		return flagPayload{}, fmt.Errorf("flag %q has unknown state %q", key, cf.State)
	}
	def, ok := cf.Variants[cf.DefaultVariant]
	if !ok {
		return flagPayload{}, fmt.Errorf("flag %q names unknown default variant %q", key, cf.DefaultVariant)
	}

	fp := flagPayload{
		AtType:  "flipt.core.Flag",
		Key:     key,
		Name:    key,
		Enabled: enabled,
	}
	switch v := def.(type) {
	case bool:
		fp.Type = flagTypeBoolean
		if !enabled {
			// A disabled boolean carries no rollout configuration. A Flipt
			// with the disabled-boolean guard reports FLAG_DISABLED before
			// reaching any rollout; without the guard, a synthesized
			// threshold would resolve the wrong value, while bare the flag
			// falls through to enabled:false. Either way the value stays
			// right.
			return fp, nil
		}
		fp.Rollouts = []rollout{{
			Type:      rolloutTypeThreshold,
			Threshold: &threshold{Percentage: 100, Value: v},
		}}
		return fp, nil
	case map[string]any:
		fp.Type = flagTypeVariant
		for _, name := range sortedKeys(cf.Variants) {
			obj, ok := cf.Variants[name].(map[string]any)
			if !ok {
				return flagPayload{}, fmt.Errorf("flag %q mixes object and scalar variants", key)
			}
			att, err := json.Marshal(obj)
			if err != nil {
				return flagPayload{}, fmt.Errorf("marshal variant %q of flag %q: %w", name, key, err)
			}
			fp.Variants = append(fp.Variants, objVariant(name, string(att)))
		}
		fp.DefaultVariant = cf.DefaultVariant
		return fp, nil
	default:
		fp.Type = flagTypeVariant
		for _, name := range sortedKeys(cf.Variants) {
			k, err := fliptKey(cf.Variants[name])
			if err != nil {
				return flagPayload{}, fmt.Errorf("flag %q variant %q: %w", key, name, err)
			}
			fp.Variants = append(fp.Variants, plainVariant(k))
		}
		dk, err := fliptKey(def)
		if err != nil {
			return flagPayload{}, fmt.Errorf("flag %q default variant %q: %w", key, cf.DefaultVariant, err)
		}
		fp.DefaultVariant = dk
		return fp, nil
	}
}

// fliptKey renders a canonical scalar value as the Flipt variant key that
// carries it. Numbers stay verbatim — json.Number preserves the literal text
// a float64 parse would lose. The empty string cannot be a Flipt key, so the
// canonical string-zero-flag value seeds as "zero"; the conformance suite
// reports that row as a known deviation.
func fliptKey(v any) (string, error) {
	switch t := v.(type) {
	case string:
		if t == "" {
			return "zero", nil
		}
		return t, nil
	case json.Number:
		return t.String(), nil
	default:
		return "", fmt.Errorf("value of type %T has no variant-key rendering", v)
	}
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
