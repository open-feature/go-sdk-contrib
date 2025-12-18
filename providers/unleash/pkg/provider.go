package unleash

import (
	"context"
	"fmt"
	"strconv"

	"github.com/Unleash/unleash-client-go/v4"
	"github.com/Unleash/unleash-client-go/v4/api"
	unleashContext "github.com/Unleash/unleash-client-go/v4/context"
	of "go.openfeature.dev/openfeature/v2"
)

const (
	providerNotReady = "Provider not ready"
	generalError     = "general error"
)

var (
	_ of.FeatureProvider = (*Provider)(nil)
	_ of.StateHandler    = (*Provider)(nil)
)

type Provider struct {
	providerConfig ProviderConfig
	status         of.State
}

func NewProvider(providerConfig ProviderConfig) (*Provider, error) {
	provider := &Provider{
		status:         of.NotReadyState,
		providerConfig: providerConfig,
	}
	return provider, nil
}

func (p *Provider) Init(context.Context) error {
	err := unleash.Initialize(
		p.providerConfig.Options...,
	)
	if err != nil {
		p.status = of.ErrorState
	} else {
		p.status = of.ReadyState
	}
	return err
}

func (p *Provider) Status() of.State {
	return p.status
}

func (p *Provider) Shutdown(context.Context) error {
	err := unleash.Close()
	p.status = of.NotReadyState
	return err
}

// Hooks returns empty slice as provider does not have any
func (p *Provider) Hooks() []of.Hook {
	return []of.Hook{}
}

// Metadata returns value of Metadata (name of current service, exposed to openfeature sdk)
func (p *Provider) Metadata() of.Metadata {
	return of.Metadata{
		Name: "Unleash",
	}
}

func (p *Provider) BooleanEvaluation(ctx context.Context, flag string, defaultValue bool, evalCtx of.FlattenedContext) of.BoolResolutionDetail {
	if p.status != of.ReadyState {
		if p.status == of.NotReadyState {
			return of.BoolResolutionDetail{
				Value: defaultValue,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					ResolutionError: of.NewProviderNotReadyResolutionError(providerNotReady),
					Reason:          of.ErrorReason,
				},
			}
		} else {
			return of.BoolResolutionDetail{
				Value: defaultValue,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					ResolutionError: of.NewGeneralResolutionError(generalError),
					Reason:          of.ErrorReason,
				},
			}
		}
	}

	unleashContext, err := toUnleashContext(evalCtx)
	if err != nil {
		return of.BoolResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				ResolutionError: of.NewInvalidContextResolutionError(err.Error()),
				Reason:          of.ErrorReason,
			},
		}
	}

	res := unleash.IsEnabled(flag, unleash.WithFallback(defaultValue), unleash.WithContext(*unleashContext))
	flagMetadata := map[string]any{
		"enabled": res,
	}

	return of.BoolResolutionDetail{
		Value: res,
		ProviderResolutionDetail: of.ProviderResolutionDetail{
			FlagMetadata: flagMetadata,
		},
	}
}

func (p *Provider) FloatEvaluation(ctx context.Context, flag string, defaultValue float64, evalCtx of.FlattenedContext) of.FloatResolutionDetail {
	res := p.ObjectEvaluation(ctx, flag, defaultValue, evalCtx)

	if value, ok := res.Value.(float64); ok {
		return of.FloatResolutionDetail{
			Value:                    value,
			ProviderResolutionDetail: res.ProviderResolutionDetail,
		}
	}

	if strValue, ok := res.Value.(string); ok {
		value, err := strconv.ParseFloat(strValue, 64)
		if err == nil {
			return of.FloatResolutionDetail{
				Value:                    value,
				ProviderResolutionDetail: res.ProviderResolutionDetail,
			}
		}
	}
	return of.FloatResolutionDetail{
		Value: defaultValue,
		ProviderResolutionDetail: of.ProviderResolutionDetail{
			Reason:          of.ErrorReason,
			ResolutionError: of.NewTypeMismatchResolutionError(fmt.Sprintf("FloatEvaluation type error for %s", flag)),
		},
	}
}

func (p *Provider) IntEvaluation(ctx context.Context, flag string, defaultValue int64, evalCtx of.FlattenedContext) of.IntResolutionDetail {
	res := p.ObjectEvaluation(ctx, flag, defaultValue, evalCtx)

	if value, ok := res.Value.(int64); ok {
		return of.IntResolutionDetail{
			Value:                    value,
			ProviderResolutionDetail: res.ProviderResolutionDetail,
		}
	}

	if strValue, ok := res.Value.(string); ok {
		value, err := strconv.ParseInt(strValue, 10, 64)
		if err == nil {
			return of.IntResolutionDetail{
				Value:                    value,
				ProviderResolutionDetail: res.ProviderResolutionDetail,
			}
		}
	}
	return of.IntResolutionDetail{
		Value: defaultValue,
		ProviderResolutionDetail: of.ProviderResolutionDetail{
			Reason:          of.ErrorReason,
			ResolutionError: of.NewTypeMismatchResolutionError(fmt.Sprintf("IntEvaluation type error for %s", flag)),
		},
	}
}

func (p *Provider) StringEvaluation(ctx context.Context, flag string, defaultValue string, evalCtx of.FlattenedContext) of.StringResolutionDetail {
	res := p.ObjectEvaluation(ctx, flag, defaultValue, evalCtx)
	return of.StringResolutionDetail{
		Value: fmt.Sprint(res.Value),
		ProviderResolutionDetail: of.ProviderResolutionDetail{
			ResolutionError: res.ProviderResolutionDetail.ResolutionError,
			Reason:          res.ProviderResolutionDetail.Reason,
			Variant:         res.Variant,
			FlagMetadata:    res.FlagMetadata,
		},
	}
}

func (p *Provider) ObjectEvaluation(ctx context.Context, flag string, defaultValue any, evalCtx of.FlattenedContext) of.ObjectResolutionDetail {
	if p.status != of.ReadyState {
		if p.status == of.NotReadyState {
			return of.ObjectResolutionDetail{
				Value: defaultValue,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					ResolutionError: of.NewProviderNotReadyResolutionError(providerNotReady),
					Reason:          of.ErrorReason,
				},
			}
		} else {
			return of.ObjectResolutionDetail{
				Value: defaultValue,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					ResolutionError: of.NewGeneralResolutionError(generalError),
					Reason:          of.ErrorReason,
				},
			}
		}
	}

	unleashContext, err := toUnleashContext(evalCtx)
	if err != nil {
		return of.ObjectResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				ResolutionError: of.NewGeneralResolutionError(err.Error()),
				Reason:          of.ErrorReason,
			},
		}
	}

	variant := unleash.GetVariant(flag, unleash.WithVariantContext(*unleashContext))
	flagMetadata := map[string]any{
		"enabled": variant.Enabled,
	}
	if variant.Name == api.DISABLED_VARIANT.Name {
		return of.ObjectResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				Variant:      "",
				FlagMetadata: flagMetadata,
			},
		}
	} else {
		return of.ObjectResolutionDetail{
			Value: variant.Payload.Value,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				Variant:      variant.Name,
				FlagMetadata: flagMetadata,
			},
		}
	}
}

func toUnleashContext(evalCtx of.FlattenedContext) (*unleashContext.Context, error) {
	if len(evalCtx) == 0 {
		return &unleashContext.Context{}, nil
	}

	unleashContext := &unleashContext.Context{}

	custom := make(map[string]string)
	for key, origVal := range evalCtx {
		val, ok := toStr(origVal)
		if !ok {
			return nil, fmt.Errorf("key `%s` can not be converted to string", key)
		}

		switch key {
		case "AppName":
			unleashContext.AppName = val
		case "CurrentTime":
			unleashContext.CurrentTime = val
		case "Environment":
			unleashContext.Environment = val
		case "RemoteAddress":
			unleashContext.RemoteAddress = val
		case "SessionId":
			unleashContext.SessionId = val
		case "UserId":
			unleashContext.UserId = val
		default:
			custom[key] = val
		}
	}

	unleashContext.Properties = custom
	return unleashContext, nil
}

func toStr(val any) (string, bool) {
	switch v := val.(type) {
	case string:
		return v, true
	case int, int8, int16, int32, int64:
		return fmt.Sprintf("%d", v), true
	case float32, float64:
		return fmt.Sprintf("%.6f", v), true
	case bool:
		return fmt.Sprintf("%t", v), true
	default:
		return "", false
	}
}
