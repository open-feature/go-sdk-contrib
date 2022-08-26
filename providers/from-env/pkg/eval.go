package from_env

import (
	"errors"

	"github.com/open-feature/golang-sdk/pkg/openfeature"
)

type StoredFlag struct {
	DefaultVariant string    `json:"defaultVariant"`
	Variants       []Variant `json:"variant"`
}

type Variant struct {
	Criteria     []Criteria  `json:"criteria"`
	TargetingKey string      `json:"targetingKey"`
	Value        interface{} `json:"value"`
	Name         string      `json:"name"`
}

type Criteria struct {
	Key   string      `json:"key"`
	Value interface{} `json:"value"`
}

func (f *StoredFlag) evaluate(evalCtx openfeature.EvaluationContext) (string, string, interface{}, error) {
	var defaultVariant *Variant
	for _, variant := range f.Variants {
		if variant.Name == f.DefaultVariant {
			defaultVariant = &variant
		}
		if variant.TargetingKey != "" && variant.TargetingKey != evalCtx.TargetingKey {
			continue
		}
		match := true
		for _, criteria := range variant.Criteria {
			val, ok := evalCtx.Attributes[criteria.Key]
			if !ok || val != criteria.Value {
				match = false
				break
			}
		}
		if match {
			return variant.Name, ReasonTargetingMatch, variant.Value, nil
		}
	}
	if defaultVariant == nil {
		return "", ReasonError, nil, errors.New(ErrorParse)
	}
	return defaultVariant.Name, ReasonStatic, defaultVariant.Value, nil
}
