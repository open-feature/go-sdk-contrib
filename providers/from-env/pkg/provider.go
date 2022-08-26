package from_env

import (
	"github.com/open-feature/golang-sdk/pkg/openfeature"
)

// FromEnvProvider implements the FeatureProvider interface and provides functions for evaluating flags
type FromEnvProvider struct {
	envFetch envFetch
}

const (
	ReasonError          = "ERROR"
	ReasonStatic         = "STATIC"
	ReasonTargetingMatch = "TARGETING_MATCH"

	ErrorTypeMismatch = "TYPE_MISMATCH"
	ErrorParse        = "PARSE_ERROR"
	ErrorFlagNotFound = "FLAG_NOT_FOUND"
)

// Metadata returns the metadata of the provider
func (p *FromEnvProvider) Metadata() openfeature.Metadata {
	return openfeature.Metadata{
		Name: "from-env-flag-evaluator",
	}
}

// Hooks returns hooks
func (p *FromEnvProvider) Hooks() []openfeature.Hook {
	return []openfeature.Hook{}
}

// BooleanEvaluation returns a boolean flag
func (p *FromEnvProvider) BooleanEvaluation(flagKey string, defaultValue bool, evalCtx openfeature.EvaluationContext) openfeature.BoolResolutionDetail {
	res := p.resolveFlag(flagKey, defaultValue, evalCtx)
	v, ok := res.Value.(bool)
	if !ok {
		return openfeature.BoolResolutionDetail{
			Value: defaultValue,
			ResolutionDetail: openfeature.ResolutionDetail{
				Reason:    ReasonError,
				Value:     defaultValue,
				ErrorCode: ErrorTypeMismatch,
			},
		}
	}
	return openfeature.BoolResolutionDetail{
		Value:            v,
		ResolutionDetail: res,
	}
}

// StringEvaluation returns a string flag
func (p *FromEnvProvider) StringEvaluation(flagKey string, defaultValue string, evalCtx openfeature.EvaluationContext) openfeature.StringResolutionDetail {
	res := p.resolveFlag(flagKey, defaultValue, evalCtx)
	v, ok := res.Value.(string)
	if !ok {
		return openfeature.StringResolutionDetail{
			Value: defaultValue,
			ResolutionDetail: openfeature.ResolutionDetail{
				Reason:    ReasonError,
				Value:     defaultValue,
				ErrorCode: ErrorTypeMismatch,
			},
		}
	}
	return openfeature.StringResolutionDetail{
		Value:            v,
		ResolutionDetail: res,
	}
}

// IntEvaluation returns an int flag
func (p *FromEnvProvider) IntEvaluation(flagKey string, defaultValue int64, evalCtx openfeature.EvaluationContext) openfeature.IntResolutionDetail {
	res := p.resolveFlag(flagKey, defaultValue, evalCtx)
	v, ok := res.Value.(float64)
	if !ok {
		return openfeature.IntResolutionDetail{
			Value: defaultValue,
			ResolutionDetail: openfeature.ResolutionDetail{
				Reason:    ReasonError,
				Value:     defaultValue,
				ErrorCode: ErrorTypeMismatch,
			},
		}
	}
	return openfeature.IntResolutionDetail{
		Value:            int64(v),
		ResolutionDetail: res,
	}
}

// FloatEvaluation returns a float flag
func (p *FromEnvProvider) FloatEvaluation(flagKey string, defaultValue float64, evalCtx openfeature.EvaluationContext) openfeature.FloatResolutionDetail {
	res := p.resolveFlag(flagKey, defaultValue, evalCtx)
	v, ok := res.Value.(float64)
	if !ok {
		return openfeature.FloatResolutionDetail{
			Value: defaultValue,
			ResolutionDetail: openfeature.ResolutionDetail{
				Reason:    ReasonError,
				Value:     defaultValue,
				ErrorCode: ErrorTypeMismatch,
			},
		}
	}
	return openfeature.FloatResolutionDetail{
		Value:            v,
		ResolutionDetail: res,
	}
}

// ObjectEvaluation returns an object flag
func (p *FromEnvProvider) ObjectEvaluation(flagKey string, defaultValue interface{}, evalCtx openfeature.EvaluationContext) openfeature.ResolutionDetail {
	return p.resolveFlag(flagKey, defaultValue, evalCtx)
}

func (p *FromEnvProvider) resolveFlag(flagKey string, defaultValue interface{}, evalCtx openfeature.EvaluationContext) openfeature.ResolutionDetail {
	res, err := p.envFetch.fetchStoredFlag(flagKey)
	if err != nil {
		return openfeature.ResolutionDetail{
			Reason:    ReasonError,
			Value:     defaultValue,
			ErrorCode: err.Error(),
		}
	}
	variant, reason, value, err := res.evaluate(evalCtx)
	if err != nil {
		return openfeature.ResolutionDetail{
			Reason:    ReasonError,
			Value:     defaultValue,
			ErrorCode: err.Error(),
		}
	}
	return openfeature.ResolutionDetail{
		Reason:  reason,
		Variant: variant,
		Value:   value,
	}
}
