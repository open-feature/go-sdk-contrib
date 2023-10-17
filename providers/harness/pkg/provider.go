package harness

import (
	"context"
	"fmt"

	harness "github.com/harness/ff-golang-server-sdk/client"
	"github.com/harness/ff-golang-server-sdk/evaluation"
	"github.com/harness/ff-golang-server-sdk/types"
	of "github.com/open-feature/go-sdk/pkg/openfeature"
)

const providerNotReady = "Provider not ready"
const generalError = "general error"

type Provider struct {
	providerConfig ProviderConfig
	harnessClient  *harness.CfClient
	status         of.State
}

func NewProvider(providerConfig ProviderConfig) (*Provider, error) {
	provider := &Provider{
		providerConfig: providerConfig,
		status:         of.NotReadyState,
	}
	return provider, nil
}

func (p *Provider) Init(evaluationContext of.EvaluationContext) error {
	harnessClient, err := harness.NewCfClient(p.providerConfig.SdkKey, p.providerConfig.Options...)
	if err != nil {
		p.status = of.ErrorState
	} else {
		p.status = of.ReadyState
		p.harnessClient = harnessClient
	}
	return err
}

func (p *Provider) Status() of.State {
	return p.status
}

func (p *Provider) Shutdown() {
	p.harnessClient.Close()
	p.status = of.NotReadyState
}

// provider does not have any hooks, returns empty slice
func (p *Provider) Hooks() []of.Hook {
	return []of.Hook{}
}

// Metadata returns value of Metadata (name of current service, exposed to openfeature sdk)
func (p *Provider) Metadata() of.Metadata {
	return of.Metadata{
		Name: "harness",
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

	harnessTarget, err := toHarnessTarget(evalCtx)
	if err != nil {
		return of.BoolResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				ResolutionError: of.NewInvalidContextResolutionError(err.Error()),
				Reason:          of.ErrorReason,
			},
		}
	}

	res, _ := p.harnessClient.BoolVariation(flag, harnessTarget, defaultValue)
	return of.BoolResolutionDetail{
		Value:                    res,
		ProviderResolutionDetail: of.ProviderResolutionDetail{},
	}
}

func (p *Provider) FloatEvaluation(ctx context.Context, flag string, defaultValue float64, evalCtx of.FlattenedContext) of.FloatResolutionDetail {
	if p.status != of.ReadyState {
		if p.status == of.NotReadyState {
			return of.FloatResolutionDetail{
				Value: defaultValue,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					ResolutionError: of.NewProviderNotReadyResolutionError(providerNotReady),
					Reason:          of.ErrorReason,
				},
			}
		} else {
			return of.FloatResolutionDetail{
				Value: defaultValue,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					ResolutionError: of.NewGeneralResolutionError(generalError),
					Reason:          of.ErrorReason,
				},
			}
		}
	}

	harnessTarget, err := toHarnessTarget(evalCtx)
	if err != nil {
		return of.FloatResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				ResolutionError: of.NewInvalidContextResolutionError(err.Error()),
				Reason:          of.ErrorReason,
			},
		}
	}

	res, _ := p.harnessClient.NumberVariation(flag, harnessTarget, defaultValue)
	return of.FloatResolutionDetail{
		Value:                    res,
		ProviderResolutionDetail: of.ProviderResolutionDetail{},
	}
}

func (p *Provider) IntEvaluation(ctx context.Context, flag string, defaultValue int64, evalCtx of.FlattenedContext) of.IntResolutionDetail {
	if p.status != of.ReadyState {
		if p.status == of.NotReadyState {
			return of.IntResolutionDetail{
				Value: defaultValue,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					ResolutionError: of.NewProviderNotReadyResolutionError(providerNotReady),
					Reason:          of.ErrorReason,
				},
			}
		} else {
			return of.IntResolutionDetail{
				Value: defaultValue,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					ResolutionError: of.NewGeneralResolutionError(generalError),
					Reason:          of.ErrorReason,
				},
			}
		}
	}

	harnessTarget, err := toHarnessTarget(evalCtx)
	if err != nil {
		return of.IntResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				ResolutionError: of.NewInvalidContextResolutionError(err.Error()),
				Reason:          of.ErrorReason,
			},
		}
	}

	res, _ := p.harnessClient.IntVariation(flag, harnessTarget, defaultValue)
	return of.IntResolutionDetail{
		Value:                    res,
		ProviderResolutionDetail: of.ProviderResolutionDetail{},
	}
}

func (p *Provider) StringEvaluation(ctx context.Context, flag string, defaultValue string, evalCtx of.FlattenedContext) of.StringResolutionDetail {

	if p.status != of.ReadyState {
		if p.status == of.NotReadyState {
			return of.StringResolutionDetail{
				Value: defaultValue,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					ResolutionError: of.NewProviderNotReadyResolutionError(providerNotReady),
					Reason:          of.ErrorReason,
				},
			}
		} else {
			return of.StringResolutionDetail{
				Value: defaultValue,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					ResolutionError: of.NewGeneralResolutionError(generalError),
					Reason:          of.ErrorReason,
				},
			}
		}
	}

	harnessTarget, err := toHarnessTarget(evalCtx)
	if err != nil {
		return of.StringResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				ResolutionError: of.NewInvalidContextResolutionError(err.Error()),
				Reason:          of.ErrorReason,
			},
		}
	}

	res, _ := p.harnessClient.StringVariation(flag, harnessTarget, defaultValue)
	return of.StringResolutionDetail{
		Value:                    res,
		ProviderResolutionDetail: of.ProviderResolutionDetail{},
	}
}

func (p *Provider) ObjectEvaluation(ctx context.Context, flag string, defaultValue interface{}, evalCtx of.FlattenedContext) of.InterfaceResolutionDetail {
	if p.status != of.ReadyState {
		if p.status == of.NotReadyState {
			return of.InterfaceResolutionDetail{
				Value: defaultValue,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					ResolutionError: of.NewProviderNotReadyResolutionError(providerNotReady),
					Reason:          of.ErrorReason,
				},
			}
		} else {
			return of.InterfaceResolutionDetail{
				Value: defaultValue,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					ResolutionError: of.NewGeneralResolutionError(generalError),
					Reason:          of.ErrorReason,
				},
			}
		}
	}

	harnessTarget, err := toHarnessTarget(evalCtx)
	if err != nil {
		return of.InterfaceResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				ResolutionError: of.NewInvalidContextResolutionError(err.Error()),
				Reason:          of.ErrorReason,
			},
		}
	}

	defaultValueJson, ok := defaultValue.(types.JSON)
	if !ok {
		return of.InterfaceResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				ResolutionError: of.NewInvalidContextResolutionError("Could not get defaultValue as JSON map"),
				Reason:          of.ErrorReason,
			},
		}
	}

	res, _ := p.harnessClient.JSONVariation(flag, harnessTarget, defaultValueJson)
	return of.InterfaceResolutionDetail{
		Value:                    res,
		ProviderResolutionDetail: of.ProviderResolutionDetail{},
	}
}

func toHarnessTarget(evalCtx of.FlattenedContext) (*evaluation.Target, error) {
	if len(evalCtx) == 0 {
		return &evaluation.Target{}, nil
	}

	harnessTarget := &evaluation.Target{}

	custom := make(map[string]interface{})
	for key, origVal := range evalCtx {
		val, ok := toStr(origVal)
		if !ok {
			return nil, fmt.Errorf("key `%s` can not be converted to string", key)
		}

		switch key {
		case of.TargetingKey:
			harnessTarget.Identifier = val
		case "Name":
			harnessTarget.Name = val
		default:
			custom[key] = val
		}
	}

	harnessTarget.Attributes = &custom
	return harnessTarget, nil
}

func toStr(val interface{}) (string, bool) {
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
