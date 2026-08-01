package from_env

import (
	"maps"
	"reflect"
	"slices"

	"github.com/open-feature/go-sdk/openfeature"
)

type StoredFlag struct {
	DefaultVariant string    `json:"defaultVariant"`
	Variants       []Variant `json:"variants"`
}

type Variant struct {
	Criteria     []Criteria `json:"criteria"`
	TargetingKey string     `json:"targetingKey"`
	Value        any        `json:"value"`
	Name         string     `json:"name"`
}

type Criteria struct {
	Key   string `json:"key"`
	Value any    `json:"value"`
}

// valuesEqual reports whether left and right represent equal values.
//
// It is intended for comparing decoded JSON-like data (the result of
// json.Unmarshal into any), but falls back to reflection so it also
// behaves correctly for arbitrary comparable and non-comparable types.
//
// Common JSON types (string, float64, bool, nil, []any, map[string]any)
// are compared directly via a type switch, avoiding reflection.
// For all other types, left and right must share the same concrete type
// (as reported by reflect.TypeOf) to be considered equal; values of
// differing concrete types — e.g. int32(5) and int64(5) — are always
// unequal, even if numerically equivalent. Comparable types (including
// typed nil pointers) are then compared with ==. Non-comparable types
// (structs or arrays containing slices/maps/funcs, etc.) fall back to
// reflect.DeepEqual.
func valuesEqual(left, right any) bool {
	switch l := left.(type) {
	case nil:
		return right == nil
	case string:
		r, ok := right.(string)
		return ok && l == r
	case float64:
		r, ok := right.(float64)
		return ok && l == r
	case bool:
		r, ok := right.(bool)
		return ok && l == r
	case []any:
		r, ok := right.([]any)
		return ok && slices.EqualFunc(l, r, valuesEqual)
	case map[string]any:
		r, ok := right.(map[string]any)
		return ok && maps.EqualFunc(l, r, valuesEqual)
	}

	leftType := reflect.TypeOf(left)
	if leftType != reflect.TypeOf(right) {
		return false
	}
	if leftType.Comparable() {
		return left == right
	}
	return reflect.DeepEqual(left, right)
}

func (f *StoredFlag) evaluate(evalCtx map[string]any) (string, openfeature.Reason, any, error) {
	var defaultVariant *Variant
	for _, variant := range f.Variants {
		if variant.Name == f.DefaultVariant {
			v := variant
			defaultVariant = &v
		}
		if variant.TargetingKey != "" && variant.TargetingKey != evalCtx["targetingKey"] {
			continue
		}
		match := true
		for _, criteria := range variant.Criteria {
			val, ok := evalCtx[criteria.Key]
			if !ok || !valuesEqual(val, criteria.Value) {
				match = false
				break
			}
		}
		if match {
			return variant.Name, openfeature.TargetingMatchReason, variant.Value, nil
		}
	}
	if defaultVariant == nil {
		return "", openfeature.ErrorReason, nil, openfeature.NewParseErrorResolutionError("")
	}
	return defaultVariant.Name, openfeature.DefaultReason, defaultVariant.Value, nil
}
