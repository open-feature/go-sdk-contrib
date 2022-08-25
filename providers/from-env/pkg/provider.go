package from_env

import (
	"github.com/open-feature/golang-sdk/pkg/openfeature"
)

type Provider struct {
	EnvFetch EnvFetch
}

const (
	ReasonError          = "ERROR"
	ReasonStatic         = "STATIC"
	ReasonTargetingMatch = "TARGETING_MATCH"

	ErrorTypeMismatch = "TYPE_MISMATCH"
	ErrorParse        = "PARSE_ERROR"
	ErrorFlagNotFound = "FLAG_NOT_FOUND"
)

func (p *Provider) Metadata() openfeature.Metadata {
	return openfeature.Metadata{
		Name: "environment-flag-evaluator",
	}
}

func (p *Provider) Hooks() []openfeature.Hook {
	return []openfeature.Hook{}
}

func (p *Provider) BooleanEvaluation(flagKey string, defaultValue bool, evalCtx openfeature.EvaluationContext, _ openfeature.EvaluationOptions) openfeature.BoolResolutionDetail {
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

func (p *Provider) StringEvaluation(flagKey string, defaultValue string, evalCtx openfeature.EvaluationContext, _ openfeature.EvaluationOptions) openfeature.StringResolutionDetail {
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

func (p *Provider) IntEvaluation(flagKey string, defaultValue int64, evalCtx openfeature.EvaluationContext, _ openfeature.EvaluationOptions) openfeature.IntResolutionDetail {
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

func (p *Provider) FloatEvaluation(flagKey string, defaultValue float64, evalCtx openfeature.EvaluationContext, _ openfeature.EvaluationOptions) openfeature.FloatResolutionDetail {
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

func (p *Provider) ObjectEvaluation(flagKey string, defaultValue interface{}, evalCtx openfeature.EvaluationContext, _ openfeature.EvaluationOptions) openfeature.ResolutionDetail {
	return p.resolveFlag(flagKey, defaultValue, evalCtx)
}

func (p *Provider) resolveFlag(flagKey string, defaultValue interface{}, evalCtx openfeature.EvaluationContext) openfeature.ResolutionDetail {
	res, err := p.EnvFetch.FetchStoredFlag(flagKey)
	if err != nil {
		return openfeature.ResolutionDetail{
			Reason:    ReasonError,
			Value:     defaultValue,
			ErrorCode: err.Error(),
		}
	}
	variant, reason, value, err := res.Evaluate(evalCtx)
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
