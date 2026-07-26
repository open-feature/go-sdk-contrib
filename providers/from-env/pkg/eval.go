package from_env

import (
	"reflect"

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

func valuesEqual(left, right any) bool {
	leftType := reflect.TypeOf(left)
	if leftType != reflect.TypeOf(right) {
		return false
	}
	if leftType == nil || leftType.Comparable() {
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
