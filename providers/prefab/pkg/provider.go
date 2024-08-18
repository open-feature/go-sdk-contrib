package prefab

import (
	"context"
	"fmt"

	"github.com/open-feature/go-sdk-contrib/providers/prefab/internal"
	of "github.com/open-feature/go-sdk/openfeature"
	prefab "github.com/prefab-cloud/prefab-cloud-go/pkg"
)

const providerNotReady = "Provider not ready"
const generalError = "general error"

type Provider struct {
	providerConfig ProviderConfig
	prefabClient   *prefab.Client
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
	var prefabClient *prefab.Client
	var err error

	if p.providerConfig.APIKey != "" {
		prefabClient, err = prefab.NewClient(prefab.WithAPIKey(p.providerConfig.APIKey))
	} else if p.providerConfig.APIURLs != nil {
		prefabClient, err = prefab.NewClient(prefab.WithAPIURLs(p.providerConfig.APIURLs))
	} else if p.providerConfig.Sources != nil {
		prefabClient, err = prefab.NewClient(prefab.WithOfflineSources(p.providerConfig.Sources))
	} else {
		err = fmt.Errorf("provider config missing fields")
	}

	if err != nil {
		p.status = of.ErrorState
	} else {
		p.status = of.ReadyState
		p.prefabClient = prefabClient
	}
	return err
}

func (p *Provider) Status() of.State {
	return p.status
}

func (p *Provider) Shutdown() {
	// p.prefabClient.Close()
	p.status = of.NotReadyState
}

// provider does not have any hooks, returns empty slice
func (p *Provider) Hooks() []of.Hook {
	return []of.Hook{}
}

// Metadata returns value of Metadata (name of current service, exposed to openfeature sdk)
func (p *Provider) Metadata() of.Metadata {
	return of.Metadata{
		Name: "prefab",
	}
}

func (p *Provider) BooleanEvaluation(ctx context.Context, flag string, defaultValue bool, evalCtx of.FlattenedContext) of.BoolResolutionDetail {
	shouldReturn, returnValue := verifyStateBoolean(p, defaultValue)
	if shouldReturn {
		return returnValue
	}

	prefabContext, err := internal.ToPrefabContext(evalCtx)
	if err != nil {
		return of.BoolResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				ResolutionError: of.NewInvalidContextResolutionError(err.Error()),
				Reason:          of.ErrorReason,
			},
		}
	}

	value, _ := p.prefabClient.GetBoolValueWithDefault(flag, prefabContext, defaultValue)
	if err == nil {
		return of.BoolResolutionDetail{
			Value: value,
		}
	}

	return of.BoolResolutionDetail{
		Value: defaultValue,
		ProviderResolutionDetail: of.ProviderResolutionDetail{
			ResolutionError: of.NewInvalidContextResolutionError(err.Error()),
			Reason:          of.ErrorReason,
		},
	}
}

func verifyStateBoolean(p *Provider, defaultValue bool) (bool, of.BoolResolutionDetail) {
	if p.status != of.ReadyState {
		if p.status == of.NotReadyState {
			return true, of.BoolResolutionDetail{
				Value: defaultValue,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					ResolutionError: of.NewProviderNotReadyResolutionError(providerNotReady),
					Reason:          of.ErrorReason,
				},
			}
		} else {
			return true, of.BoolResolutionDetail{
				Value: defaultValue,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					ResolutionError: of.NewGeneralResolutionError(generalError),
					Reason:          of.ErrorReason,
				},
			}
		}
	}
	return false, of.BoolResolutionDetail{}
}

func (p *Provider) FloatEvaluation(ctx context.Context, flag string, defaultValue float64, evalCtx of.FlattenedContext) of.FloatResolutionDetail {
	shouldReturn, returnValue := verifyStateFloat(p, defaultValue)
	if shouldReturn {
		return returnValue
	}

	prefabContext, err := internal.ToPrefabContext(evalCtx)
	if err != nil {
		return of.FloatResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				ResolutionError: of.NewInvalidContextResolutionError(err.Error()),
				Reason:          of.ErrorReason,
			},
		}
	}

	value, _ := p.prefabClient.GetFloatValueWithDefault(flag, prefabContext, defaultValue)
	return of.FloatResolutionDetail{
		Value:                    value,
		ProviderResolutionDetail: of.ProviderResolutionDetail{},
	}
}

func verifyStateFloat(p *Provider, defaultValue float64) (bool, of.FloatResolutionDetail) {
	if p.status != of.ReadyState {
		if p.status == of.NotReadyState {
			return true, of.FloatResolutionDetail{
				Value: defaultValue,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					ResolutionError: of.NewProviderNotReadyResolutionError(providerNotReady),
					Reason:          of.ErrorReason,
				},
			}
		} else {
			return true, of.FloatResolutionDetail{
				Value: defaultValue,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					ResolutionError: of.NewGeneralResolutionError(generalError),
					Reason:          of.ErrorReason,
				},
			}
		}
	}
	return false, of.FloatResolutionDetail{}
}

func (p *Provider) IntEvaluation(ctx context.Context, flag string, defaultValue int64, evalCtx of.FlattenedContext) of.IntResolutionDetail {
	shouldReturn, returnValue := verifyStateInt(p, defaultValue)
	if shouldReturn {
		return returnValue
	}

	prefabContext, err := internal.ToPrefabContext(evalCtx)
	if err != nil {
		return of.IntResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				ResolutionError: of.NewInvalidContextResolutionError(err.Error()),
				Reason:          of.ErrorReason,
			},
		}
	}

	value, _ := p.prefabClient.GetIntValueWithDefault(flag, prefabContext, defaultValue)
	return of.IntResolutionDetail{
		Value:                    value,
		ProviderResolutionDetail: of.ProviderResolutionDetail{},
	}
}

func verifyStateInt(p *Provider, defaultValue int64) (bool, of.IntResolutionDetail) {
	if p.status != of.ReadyState {
		if p.status == of.NotReadyState {
			return true, of.IntResolutionDetail{
				Value: defaultValue,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					ResolutionError: of.NewProviderNotReadyResolutionError(providerNotReady),
					Reason:          of.ErrorReason,
				},
			}
		} else {
			return true, of.IntResolutionDetail{
				Value: defaultValue,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					ResolutionError: of.NewGeneralResolutionError(generalError),
					Reason:          of.ErrorReason,
				},
			}
		}
	}
	return false, of.IntResolutionDetail{}
}

func (p *Provider) StringEvaluation(ctx context.Context, flag string, defaultValue string, evalCtx of.FlattenedContext) of.StringResolutionDetail {

	shouldReturn, returnValue := verifyStateString(p, defaultValue)
	if shouldReturn {
		return returnValue
	}

	prefabContext, err := internal.ToPrefabContext(evalCtx)
	if err != nil {
		return of.StringResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				ResolutionError: of.NewInvalidContextResolutionError(err.Error()),
				Reason:          of.ErrorReason,
			},
		}
	}

	value, _ := p.prefabClient.GetStringValueWithDefault(flag, prefabContext, defaultValue)
	return of.StringResolutionDetail{
		Value:                    value,
		ProviderResolutionDetail: of.ProviderResolutionDetail{},
	}
}

func verifyStateString(p *Provider, defaultValue string) (bool, of.StringResolutionDetail) {
	if p.status != of.ReadyState {
		if p.status == of.NotReadyState {
			return true, of.StringResolutionDetail{
				Value: defaultValue,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					ResolutionError: of.NewProviderNotReadyResolutionError(providerNotReady),
					Reason:          of.ErrorReason,
				},
			}
		} else {
			return true, of.StringResolutionDetail{
				Value: defaultValue,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					ResolutionError: of.NewGeneralResolutionError(generalError),
					Reason:          of.ErrorReason,
				},
			}
		}
	}
	return false, of.StringResolutionDetail{}
}

func (p *Provider) ObjectEvaluation(ctx context.Context, flag string, defaultValue interface{}, evalCtx of.FlattenedContext) of.InterfaceResolutionDetail {
	shouldReturn, returnValue := verifyStateObject(p, defaultValue)
	if shouldReturn {
		return returnValue
	}

	prefabContext, err := internal.ToPrefabContext(evalCtx)
	if err != nil {
		return of.InterfaceResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				ResolutionError: of.NewInvalidContextResolutionError(err.Error()),
				Reason:          of.ErrorReason,
			},
		}
	}

	value, _ := p.prefabClient.GetJSONValueWithDefault(flag, prefabContext, defaultValue)
	return of.InterfaceResolutionDetail{
		Value:                    value,
		ProviderResolutionDetail: of.ProviderResolutionDetail{},
	}
}

func verifyStateObject(p *Provider, defaultValue interface{}) (bool, of.InterfaceResolutionDetail) {
	if p.status != of.ReadyState {
		if p.status == of.NotReadyState {
			return true, of.InterfaceResolutionDetail{
				Value: defaultValue,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					ResolutionError: of.NewProviderNotReadyResolutionError(providerNotReady),
					Reason:          of.ErrorReason,
				},
			}
		} else {
			return true, of.InterfaceResolutionDetail{
				Value: defaultValue,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					ResolutionError: of.NewGeneralResolutionError(generalError),
					Reason:          of.ErrorReason,
				},
			}
		}
	}
	return false, of.InterfaceResolutionDetail{}
}
