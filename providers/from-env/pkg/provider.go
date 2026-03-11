package from_env

import (
	"context"
	"errors"

	"github.com/open-feature/go-sdk/openfeature"
)

const (
	ReasonStatic = "static"

	ErrorTypeMismatch = "type mismatch"
	ErrorParse        = "parse error"
	ErrorFlagNotFound = "flag not found"
)

// FromEnvProvider implements the FeatureProvider interface and provides functions for evaluating flags
type FromEnvProvider struct {
	envFetch envFetch
}

type ProviderOption func(*FromEnvProvider)

type FlagToEnvMapper func(string) string

func WithFlagToEnvMapper(mapper FlagToEnvMapper) ProviderOption {
	return func(p *FromEnvProvider) {
		p.envFetch.mapper = mapper
	}
}

func NewProvider(opts ...ProviderOption) *FromEnvProvider {
	p := &FromEnvProvider{}

	for _, opt := range opts {
		opt(p)
	}

	return p
}

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
func (p *FromEnvProvider) BooleanEvaluation(ctx context.Context, flagKey string, defaultValue bool, evalCtx openfeature.FlattenedContext) openfeature.BoolResolutionDetail {
	res := p.resolveFlag(flagKey, defaultValue, evalCtx)
	v, ok := res.Value.(bool)
	if !ok {
		return openfeature.BoolResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
				ResolutionError: openfeature.NewTypeMismatchResolutionError(""),
				Reason:          openfeature.ErrorReason,
			},
		}
	}

	return openfeature.BoolResolutionDetail{
		Value:                    v,
		ProviderResolutionDetail: res.ProviderResolutionDetail,
	}
}

// StringEvaluation returns a string flag
func (p *FromEnvProvider) StringEvaluation(ctx context.Context, flagKey string, defaultValue string, evalCtx openfeature.FlattenedContext) openfeature.StringResolutionDetail {
	res := p.resolveFlag(flagKey, defaultValue, evalCtx)
	v, ok := res.Value.(string)
	if !ok {
		return openfeature.StringResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
				ResolutionError: openfeature.NewTypeMismatchResolutionError(""),
				Reason:          openfeature.ErrorReason,
			},
		}
	}

	return openfeature.StringResolutionDetail{
		Value:                    v,
		ProviderResolutionDetail: res.ProviderResolutionDetail,
	}
}

// IntEvaluation returns an int flag
func (p *FromEnvProvider) IntEvaluation(ctx context.Context, flagKey string, defaultValue int64, evalCtx openfeature.FlattenedContext) openfeature.IntResolutionDetail {
	res := p.resolveFlag(flagKey, defaultValue, evalCtx)
	v, ok := res.Value.(float64)
	if !ok {
		return openfeature.IntResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
				ResolutionError: openfeature.NewTypeMismatchResolutionError(""),
				Reason:          openfeature.ErrorReason,
			},
		}
	}

	return openfeature.IntResolutionDetail{
		Value:                    int64(v),
		ProviderResolutionDetail: res.ProviderResolutionDetail,
	}
}

// FloatEvaluation returns a float flag
func (p *FromEnvProvider) FloatEvaluation(ctx context.Context, flagKey string, defaultValue float64, evalCtx openfeature.FlattenedContext) openfeature.FloatResolutionDetail {
	res := p.resolveFlag(flagKey, defaultValue, evalCtx)
	v, ok := res.Value.(float64)
	if !ok {
		return openfeature.FloatResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
				ResolutionError: openfeature.NewTypeMismatchResolutionError(""),
				Reason:          openfeature.ErrorReason,
			},
		}
	}

	return openfeature.FloatResolutionDetail{
		Value:                    v,
		ProviderResolutionDetail: res.ProviderResolutionDetail,
	}
}

// ObjectEvaluation returns an object flag
func (p *FromEnvProvider) ObjectEvaluation(ctx context.Context, flagKey string, defaultValue any, evalCtx openfeature.FlattenedContext) openfeature.InterfaceResolutionDetail {
	return p.resolveFlag(flagKey, defaultValue, evalCtx)
}

func (p *FromEnvProvider) resolveFlag(flagKey string, defaultValue any, evalCtx openfeature.FlattenedContext) openfeature.InterfaceResolutionDetail {
	// fetch the stored flag from environment variables
	res, err := p.envFetch.fetchStoredFlag(flagKey)
	if err != nil {
		var e openfeature.ResolutionError
		if !errors.As(err, &e) {
			e = openfeature.NewGeneralResolutionError(err.Error())
		}

		return openfeature.InterfaceResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
				ResolutionError: e,
				Reason:          openfeature.ErrorReason,
			},
		}
	}
	// evaluate the stored flag to return the variant, reason, value and error
	variant, reason, value, err := res.evaluate(evalCtx)
	if err != nil {
		var e openfeature.ResolutionError
		if !errors.As(err, &e) {
			e = openfeature.NewGeneralResolutionError(err.Error())
		}
		return openfeature.InterfaceResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
				ResolutionError: e,
				Reason:          openfeature.ErrorReason,
			},
		}
	}

	// return the type naive ResolutionDetail structure
	return openfeature.InterfaceResolutionDetail{
		Value: value,
		ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
			Variant: variant,
			Reason:  reason,
		},
	}
}
