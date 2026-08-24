package tck

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// flagType is one of the five flag types the Evaluation API exposes. The
// feature files name it in the "Given a <type>-flag" step, and it decides both
// which client method resolves the flag and how the expected values in the
// scenario are parsed.
type flagType string

const (
	typeBoolean flagType = "Boolean"
	typeString  flagType = "String"
	typeInteger flagType = "Integer"
	typeFloat   flagType = "Float"
	typeObject  flagType = "Object"
)

var allFlagTypes = []flagType{typeBoolean, typeString, typeInteger, typeFloat, typeObject}

// parseFlagType resolves the type named in a scenario, case-insensitively.
func parseFlagType(raw string) (flagType, error) {
	for _, t := range allFlagTypes {
		if strings.EqualFold(string(t), raw) {
			return t, nil
		}
	}
	names := make([]string, len(allFlagTypes))
	for i, t := range allFlagTypes {
		names[i] = string(t)
	}
	return "", fmt.Errorf("unknown flag type %q: expected one of %s", raw, strings.Join(names, ", "))
}

// parseValue converts a value written in a scenario into the Go type the
// Evaluation API uses for this flag type.
//
// Everything in Gherkin is a string, so this is where "0.5" becomes a float64
// and "{}" becomes an empty object. Parsing per declared type rather than
// guessing is what keeps the integer and float scenarios distinguishable: "1"
// is an int64 in an Integer scenario and a float64 in a Float one.
func (t flagType) parseValue(raw string) (any, error) {
	switch t {
	case typeBoolean:
		v, err := strconv.ParseBool(raw)
		if err != nil {
			return nil, fmt.Errorf("%q is not a boolean: %w", raw, err)
		}
		return v, nil
	case typeString:
		return raw, nil
	case typeInteger:
		v, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("%q is not an integer: %w", raw, err)
		}
		return v, nil
	case typeFloat:
		v, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return nil, fmt.Errorf("%q is not a float: %w", raw, err)
		}
		return v, nil
	case typeObject:
		var v any
		if err := json.Unmarshal([]byte(raw), &v); err != nil {
			return nil, fmt.Errorf("%q is not valid JSON: %w", raw, err)
		}
		return v, nil
	default:
		return nil, fmt.Errorf("unknown flag type %q", t)
	}
}

// valuesEqual compares an expected value from a scenario with what a provider
// actually resolved.
//
// Numbers are compared numerically rather than by Go type. A provider that
// deserialises its backend's JSON hands back float64 for every number, so the
// 100 inside object-flag arrives as float64(100) from one provider and
// int64(100) from another while both are correct. Type distinctness is asserted
// where it belongs — by requesting a flag as a specific type and checking the
// error code — not by accident of how a number was boxed.
func valuesEqual(expected, actual any) bool {
	if ef, ok := asFloat(expected); ok {
		af, ok := asFloat(actual)
		return ok && ef == af
	}
	if em, ok := expected.(map[string]any); ok {
		am, ok := actual.(map[string]any)
		if !ok || len(em) != len(am) {
			return false
		}
		for k, ev := range em {
			av, present := am[k]
			if !present || !valuesEqual(ev, av) {
				return false
			}
		}
		return true
	}
	return reflect.DeepEqual(expected, actual)
}

// asFloat reports a numeric value as a float64, and whether it was numeric at
// all. Booleans are deliberately not numeric.
func asFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case int:
		return float64(n), true
	case int8:
		return float64(n), true
	case int16:
		return float64(n), true
	case int32:
		return float64(n), true
	case int64:
		return float64(n), true
	case uint:
		return float64(n), true
	case uint8:
		return float64(n), true
	case uint16:
		return float64(n), true
	case uint32:
		return float64(n), true
	case uint64:
		return float64(n), true
	case float32:
		return float64(n), true
	case float64:
		return n, true
	default:
		return 0, false
	}
}

// describeValue renders a value for a failure message, including its Go type,
// because "expected 100 but got 100" is the single most confusing failure a
// cross-language conformance suite can produce.
func describeValue(v any) string {
	if v == nil {
		return "<nil>"
	}
	return fmt.Sprintf("%v (%T)", v, v)
}

// objectMember looks up a member of a resolved structured value.
//
// Object values cross the provider boundary as any, and providers differ in
// what they put behind it. A plain map is the common case; anything else is
// funnelled through JSON so that a provider returning a struct still works
// without the TCK knowing its type.
func objectMember(object any, key string) (any, bool, error) {
	if m, ok := object.(map[string]any); ok {
		v, present := m[key]
		return v, present, nil
	}

	encoded, err := json.Marshal(object)
	if err != nil {
		return nil, false, fmt.Errorf(
			"resolved object value is a %T, which is neither a map[string]any nor JSON-encodable, "+
				"so member %q cannot be read: %w", object, key, err)
	}

	var m map[string]any
	if err := json.Unmarshal(encoded, &m); err != nil {
		return nil, false, fmt.Errorf(
			"resolved object value is a %T that does not encode to a JSON object, "+
				"so member %q cannot be read: %w", object, key, err)
	}

	v, present := m[key]
	return v, present, nil
}
