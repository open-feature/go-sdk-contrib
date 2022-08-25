package from_env

import (
	"errors"

	"github.com/open-feature/golang-sdk/pkg/openfeature"
)

type StoredFlag struct {
	DefaultVariant string             `json:"defaultVariant"`
	Variants       map[string]Variant `json:"variant"`
}

type Variant struct {
	Criteria     []Criteria  `json:"criteria"`
	TargetingKey string      `json:"targetingKey"`
	Value        interface{} `json:"value"`
}

type Criteria struct {
	Key   string      `json:"key"`
	Value interface{} `json:"value"`
}

func (f *StoredFlag) Evaluate(evalCtx openfeature.EvaluationContext) (string, string, interface{}, error) {
	for name, variant := range f.Variants {
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
			return name, ReasonTargetingMatch, variant.Value, nil
		}
	}
	defaultVariant, ok := f.Variants[f.DefaultVariant]
	if !ok {
		return "", ReasonError, nil, errors.New(ErrorParse)
	}
	return f.DefaultVariant, ReasonStatic, defaultVariant.Value, nil
}
