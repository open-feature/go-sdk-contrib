package rocketflag

import (
	"context"

	"github.com/open-feature/go-sdk/openfeature"
	rocketflag "github.com/rocketflag/go-sdk"
)

// Client is an interface for the RocketFlag client.
type Client interface {
	GetFlag(flag string, user rocketflag.UserContext) (*rocketflag.FlagStatus, error)
}

type Provider struct {
	client Client
}

type ProviderOption func(*Provider)

func NewProvider(client Client, opts ...ProviderOption) *Provider {
	provider := &Provider{
		client: client,
	}
	for _, opt := range opts {
		opt(provider)
	}
	return provider

}

// Metadata returns value of Metadata (name of current service, exposed to openfeature sdk)
func (p *Provider) Metadata() openfeature.Metadata {
	return openfeature.Metadata{
		Name: "RocketFlag",
	}
}

// Hooks are not supported by RocketFlag.
func (p *Provider) Hooks() []openfeature.Hook {
	return []openfeature.Hook{}
}

func (p *Provider) BooleanEvaluation(ctx context.Context, flag string, defaultValue bool, flatCtx openfeature.FlattenedContext) openfeature.BoolResolutionDetail {
	rocketflagUserContext := rocketflag.UserContext{}

	if targetingKey, ok := flatCtx[openfeature.TargetingKey].(string); ok {
		rocketflagUserContext["cohort"] = targetingKey
	}

	value, err := p.client.GetFlag(flag, rocketflagUserContext)
	if err != nil {
		return openfeature.BoolResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
				ResolutionError: openfeature.NewGeneralResolutionError(err.Error()),
				Reason:          openfeature.ErrorReason,
			},
		}
	}

	reason := openfeature.DefaultReason
	if cohort, ok := rocketflagUserContext["cohort"]; ok && cohort != nil {
		reason = openfeature.TargetingMatchReason
	}

	return openfeature.BoolResolutionDetail{
		Value: value.Enabled,
		ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
			Reason: reason,
		},
	}
}

func (p *Provider) StringEvaluation(ctx context.Context, flag string, defaultValue string, evalCtx openfeature.FlattenedContext) openfeature.StringResolutionDetail {
	return openfeature.StringResolutionDetail{
		Value: defaultValue,
		ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
			ResolutionError: openfeature.NewTypeMismatchResolutionError("RocketFlag: String flags are not yet supported."),
			Reason:          openfeature.ErrorReason,
		},
	}
}

func (p *Provider) FloatEvaluation(ctx context.Context, flag string, defaultValue float64, evalCtx openfeature.FlattenedContext) openfeature.FloatResolutionDetail {
	return openfeature.FloatResolutionDetail{
		Value: defaultValue,
		ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
			ResolutionError: openfeature.NewTypeMismatchResolutionError("RocketFlag: Float flags are not yet supported."),
			Reason:          openfeature.ErrorReason,
		},
	}
}

func (p *Provider) IntEvaluation(ctx context.Context, flag string, defaultValue int64, evalCtx openfeature.FlattenedContext) openfeature.IntResolutionDetail {
	return openfeature.IntResolutionDetail{
		Value: defaultValue,
		ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
			ResolutionError: openfeature.NewTypeMismatchResolutionError("RocketFlag: Int flags are not yet supported."),
			Reason:          openfeature.ErrorReason,
		},
	}
}

func (p *Provider) ObjectEvaluation(ctx context.Context, flag string, defaultValue any, evalCtx openfeature.FlattenedContext) openfeature.InterfaceResolutionDetail {
	return openfeature.InterfaceResolutionDetail{
		Value: defaultValue,
		ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
			ResolutionError: openfeature.NewTypeMismatchResolutionError("RocketFlag: Object flags are not yet supported."),
			Reason:          openfeature.ErrorReason,
		},
	}
}
