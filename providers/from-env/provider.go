package envvar

import (
	"context"
	"errors"

	"go.openfeature.dev/openfeature/v2"
)

const (
	ReasonStatic = "static"

	ErrorTypeMismatch = "type mismatch"
	ErrorParse        = "parse error"
	ErrorFlagNotFound = "flag not found"
)

var _ openfeature.FeatureProvider = (*EnvVarProvider)(nil)

// EnvVarProvider implements the FeatureProvider interface and provides functions for evaluating flags
type EnvVarProvider struct {
	envFetch envFetch
}

type ProviderOption func(*EnvVarProvider)

type FlagToEnvMapper func(string) string

func WithFlagToEnvMapper(mapper FlagToEnvMapper) ProviderOption {
	return func(p *EnvVarProvider) {
		p.envFetch.mapper = mapper
	}
}

func NewProvider(opts ...ProviderOption) *EnvVarProvider {
	p := &EnvVarProvider{}

	for _, opt := range opts {
		opt(p)
	}

	return p
}

// Metadata returns the metadata of the provider
func (p *EnvVarProvider) Metadata() openfeature.Metadata {
	return openfeature.Metadata{
		Name: "from-env-flag-evaluator",
	}
}

// Hooks returns hooks
func (p *EnvVarProvider) Hooks() []openfeature.Hook {
	return []openfeature.Hook{}
}

// BooleanEvaluation returns a boolean flag
func (p *EnvVarProvider) BooleanEvaluation(ctx context.Context, flagKey string, defaultValue bool, evalCtx openfeature.FlattenedContext) openfeature.BoolResolutionDetail {
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
func (p *EnvVarProvider) StringEvaluation(ctx context.Context, flagKey string, defaultValue string, evalCtx openfeature.FlattenedContext) openfeature.StringResolutionDetail {
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
func (p *EnvVarProvider) IntEvaluation(ctx context.Context, flagKey string, defaultValue int64, evalCtx openfeature.FlattenedContext) openfeature.IntResolutionDetail {
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
func (p *EnvVarProvider) FloatEvaluation(ctx context.Context, flagKey string, defaultValue float64, evalCtx openfeature.FlattenedContext) openfeature.FloatResolutionDetail {
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
func (p *EnvVarProvider) ObjectEvaluation(ctx context.Context, flagKey string, defaultValue any, evalCtx openfeature.FlattenedContext) openfeature.ObjectResolutionDetail {
	return p.resolveFlag(flagKey, defaultValue, evalCtx)
}

func (p *EnvVarProvider) resolveFlag(flagKey string, defaultValue any, evalCtx openfeature.FlattenedContext) openfeature.ObjectResolutionDetail {
	// fetch the stored flag from environment variables
	res, err := p.envFetch.fetchStoredFlag(flagKey)
	if err != nil {
		var e openfeature.ResolutionError
		if !errors.As(err, &e) {
			e = openfeature.NewGeneralResolutionError(err.Error())
		}

		return openfeature.ObjectResolutionDetail{
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
		return openfeature.ObjectResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
				ResolutionError: e,
				Reason:          openfeature.ErrorReason,
			},
		}
	}

	// return the type naive ResolutionDetail structure
	return openfeature.ObjectResolutionDetail{
		Value: value,
		ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
			Variant: variant,
			Reason:  reason,
		},
	}
}
