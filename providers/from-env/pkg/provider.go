package from_env

import (
	"github.com/open-feature/golang-sdk/pkg/openfeature"
)

// FromEnvProvider implements the FeatureProvider interface and provides functions for evaluating flags
type FromEnvProvider struct {
	envFetch envFetch
}

const (
	ReasonStatic = "static"

	ErrorTypeMismatch = "type mismatch"
	ErrorParse        = "parse error"
	ErrorFlagNotFound = "flag not found"
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
func (p *FromEnvProvider) BooleanEvaluation(flagKey string, defaultValue bool, evalCtx map[string]interface{}) openfeature.BoolResolutionDetail {
	res := p.resolveFlag(flagKey, defaultValue, evalCtx)
	v, ok := res.Value.(bool)
	if !ok {
		return openfeature.BoolResolutionDetail{
			Value: defaultValue,
			ResolutionDetail: openfeature.ResolutionDetail{
				Reason:    openfeature.ERROR,
				ErrorCode: ErrorTypeMismatch,
			},
		}
	}
	return openfeature.BoolResolutionDetail{
		Value:            v,
		ResolutionDetail: res.ResolutionDetail,
	}
}

// StringEvaluation returns a string flag
func (p *FromEnvProvider) StringEvaluation(flagKey string, defaultValue string, evalCtx map[string]interface{}) openfeature.StringResolutionDetail {
	res := p.resolveFlag(flagKey, defaultValue, evalCtx)
	v, ok := res.Value.(string)
	if !ok {
		return openfeature.StringResolutionDetail{
			Value: defaultValue,
			ResolutionDetail: openfeature.ResolutionDetail{
				Reason:    openfeature.ERROR,
				ErrorCode: ErrorTypeMismatch,
			},
		}
	}
	return openfeature.StringResolutionDetail{
		Value:            v,
		ResolutionDetail: res.ResolutionDetail,
	}
}

// IntEvaluation returns an int flag
func (p *FromEnvProvider) IntEvaluation(flagKey string, defaultValue int64, evalCtx map[string]interface{}) openfeature.IntResolutionDetail {
	res := p.resolveFlag(flagKey, defaultValue, evalCtx)
	v, ok := res.Value.(float64)
	if !ok {
		return openfeature.IntResolutionDetail{
			Value: defaultValue,
			ResolutionDetail: openfeature.ResolutionDetail{
				Reason:    openfeature.ERROR,
				ErrorCode: ErrorTypeMismatch,
			},
		}
	}
	return openfeature.IntResolutionDetail{
		Value:            int64(v),
		ResolutionDetail: res.ResolutionDetail,
	}
}

// FloatEvaluation returns a float flag
func (p *FromEnvProvider) FloatEvaluation(flagKey string, defaultValue float64, evalCtx map[string]interface{}) openfeature.FloatResolutionDetail {
	res := p.resolveFlag(flagKey, defaultValue, evalCtx)
	v, ok := res.Value.(float64)
	if !ok {
		return openfeature.FloatResolutionDetail{
			Value: defaultValue,
			ResolutionDetail: openfeature.ResolutionDetail{
				Reason:    openfeature.ERROR,
				ErrorCode: ErrorTypeMismatch,
			},
		}
	}
	return openfeature.FloatResolutionDetail{
		Value:            v,
		ResolutionDetail: res.ResolutionDetail,
	}
}

// ObjectEvaluation returns an object flag
func (p *FromEnvProvider) ObjectEvaluation(flagKey string, defaultValue interface{}, evalCtx map[string]interface{}) openfeature.InterfaceResolutionDetail {
	return p.resolveFlag(flagKey, defaultValue, evalCtx)
}

func (p *FromEnvProvider) resolveFlag(flagKey string, defaultValue interface{}, evalCtx map[string]interface{}) openfeature.InterfaceResolutionDetail {
	res, err := p.envFetch.fetchStoredFlag(flagKey)
	if err != nil {
		return openfeature.InterfaceResolutionDetail{
			Value: defaultValue,
			ResolutionDetail: openfeature.ResolutionDetail{
				Reason:    openfeature.ERROR,
				ErrorCode: err.Error(),
			},
		}
	}
	variant, reason, value, err := res.evaluate(evalCtx)
	if err != nil {
		return openfeature.InterfaceResolutionDetail{
			Value: defaultValue,
			ResolutionDetail: openfeature.ResolutionDetail{
				Reason:    openfeature.ERROR,
				ErrorCode: err.Error(),
			},
		}
	}

	return openfeature.InterfaceResolutionDetail{
		Value: value,
		ResolutionDetail: openfeature.ResolutionDetail{
			Reason:  reason,
			Variant: variant,
		},
	}
}
