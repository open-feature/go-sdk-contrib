package flagsmith

import (
	"context"
	"fmt"
	"strconv"
	flagsmithClient "github.com/Flagsmith/flagsmith-go-client/v2"
	of "github.com/open-feature/go-sdk/pkg/openfeature"
)


type Provider struct {
	client *flagsmithClient.Client
	usingBooleanConfigValue bool
}

type ProviderOption func(*Provider)

func NewProvider(client *flagsmithClient.Client, opts ...ProviderOption) *Provider {
	provider := &Provider{
		client: client,
	}
	for _, opt := range opts {
		opt(provider)
	}
	return provider

}

// flagsmith provider does not have any hooks, returns empty slice
func (p *Provider) Hooks() []of.Hook {
	return []of.Hook{}
}

func (provider *Provider) Metadata() of.Metadata {
	return of.Metadata{
		Name: "Flagsmith",
	}
}

const TraitsKey = "traits"

func (p *Provider) resolveFlag(ctx context.Context, flag string, defaultValue interface{}, evalCtx of.FlattenedContext) of.InterfaceResolutionDetail {

	var flags flagsmithClient.Flags
	var err error
	_, ok := evalCtx[of.TargetingKey]
	if ok {
		targetKey, ok := evalCtx[of.TargetingKey].(string)
		if !ok {
			e := of.NewInvalidContextResolutionError(fmt.Sprintf("targeting key: %s is not a string", of.TargetingKey))
			return of.InterfaceResolutionDetail{
				Value: defaultValue,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					ResolutionError: e,
					Reason:          of.ErrorReason,
				},
			}

		}
		var traits []*flagsmithClient.Trait
		userTraits, ok := evalCtx[TraitsKey]
		if ok {
			traits, ok = userTraits.([]*flagsmithClient.Trait)
			if !ok {
				e := of.NewInvalidContextResolutionError(fmt.Sprintf("traits: expected type []*flagsmithClient.Trait, got %T", userTraits))
				return of.InterfaceResolutionDetail{
					Value: defaultValue,
					ProviderResolutionDetail: of.ProviderResolutionDetail{
						ResolutionError: e,
						Reason:          of.ErrorReason,
					},
				}
			}
		}
		flags, err = p.client.GetIdentityFlags(targetKey, traits)
		if err != nil {
			var e of.ResolutionError
			e = of.NewGeneralResolutionError(err.Error())

			return of.InterfaceResolutionDetail{
				Value: defaultValue,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					ResolutionError: e,
					Reason:          of.ErrorReason,
				},
			}
		}

	} else {
		flags, err = p.client.GetEnvironmentFlags()
		if err != nil {
			var e of.ResolutionError
			e = of.NewGeneralResolutionError(err.Error())

			return of.InterfaceResolutionDetail{
				Value: defaultValue,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					ResolutionError: e,
					Reason:          of.ErrorReason,
				},
			}
		}
	}

	flagObj, err := flags.GetFlag(flag)

	if err != nil {
		var e of.ResolutionError
		e = of.NewGeneralResolutionError(err.Error())

		return of.InterfaceResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				ResolutionError: e,
				Reason:          of.ErrorReason,
			},
		}
	}
	if !flagObj.Enabled {
		return of.InterfaceResolutionDetail{
			Value:  flagObj.Value,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				Reason:          of.DisabledReason,
			},
		}
	}
	return of.InterfaceResolutionDetail{
		Value: flagObj.Value,
		ProviderResolutionDetail: of.ProviderResolutionDetail{
			Reason: of.TargetingMatchReason,
		},
	}

}
func (p *Provider) BooleanEvaluation(ctx context.Context, flag string, defaultValue bool, evalCtx of.FlattenedContext) of.BoolResolutionDetail {
	res := p.resolveFlag(ctx, flag, defaultValue, evalCtx)
	if p.usingBooleanConfigValue {
		value := !(res.ProviderResolutionDetail.Reason == of.DisabledReason)
		return of.BoolResolutionDetail{
			Value: value,
			ProviderResolutionDetail: res.ProviderResolutionDetail,
		}
	}

	value, ok := res.Value.(bool)
	if !ok {
		return of.BoolResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				ResolutionError: of.NewTypeMismatchResolutionError(""),
				Reason:          of.ErrorReason,
			},
		}
	}
	return of.BoolResolutionDetail{
		Value:                    value,
		ProviderResolutionDetail: res.ProviderResolutionDetail,
	}
}

func (p *Provider) StringEvaluation(ctx context.Context, flag string, defaultValue string, evalCtx of.FlattenedContext) of.StringResolutionDetail {
	res := p.resolveFlag(ctx, flag, defaultValue, evalCtx)
	value, ok := res.Value.(string)
	if !ok {
		return of.StringResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				ResolutionError: of.NewTypeMismatchResolutionError(""),
				Reason:          of.ErrorReason,
			},
		}
	}

	return of.StringResolutionDetail{
		Value: value,
		ProviderResolutionDetail: res.ProviderResolutionDetail,
	}
}

func (p *Provider) FloatEvaluation(ctx context.Context, flag string, defaultValue float64, evalCtx of.FlattenedContext) of.FloatResolutionDetail {
	res := p.resolveFlag(ctx, flag, defaultValue, evalCtx)
	stringValue, ok := res.Value.(string)
	if !ok {
		return of.FloatResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				ResolutionError: of.NewTypeMismatchResolutionError(""),
				Reason:          of.ErrorReason,
			},
		}
	}
	value, err := strconv.ParseFloat(stringValue, 64)
	if err != nil {
		return of.FloatResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				ResolutionError: of.NewTypeMismatchResolutionError(""),
				Reason:          of.ErrorReason,
			},
		}
	}

	return of.FloatResolutionDetail{
		Value: value,
		ProviderResolutionDetail: res.ProviderResolutionDetail,
	}

}

func (p *Provider) IntEvaluation(ctx context.Context, flag string, defaultValue int64, evalCtx of.FlattenedContext) of.IntResolutionDetail {
	res := p.resolveFlag(ctx, flag, defaultValue, evalCtx)
	value, ok := res.Value.(float64)
	if !ok {
		return of.IntResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				ResolutionError: of.NewTypeMismatchResolutionError(""),
				Reason:          of.ErrorReason,
			},
		}
	}
	int64Value := int64(value)
	return of.IntResolutionDetail{
		Value: int64Value,
		ProviderResolutionDetail: res.ProviderResolutionDetail,
	}
}

func (p *Provider) ObjectEvaluation(ctx context.Context, flag string, defaultValue interface{}, evalCtx of.FlattenedContext) of.InterfaceResolutionDetail {
	return p.resolveFlag(ctx, flag, defaultValue, evalCtx)
}

// WithBooleanConfigValue configures the provider to use the result of isFeatureEnabled as the boolean value of the flag
// i.e: if the flag is enabled, the value will be true, otherwise it will be false
func WithUsingBooleanConfigValue() ProviderOption {
	return func(p *Provider) {
		p.usingBooleanConfigValue = true
	}
}


