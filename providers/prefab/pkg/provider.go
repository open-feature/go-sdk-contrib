package prefab

import (
	"context"
	"fmt"
	"strings"

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

	prefabContext, err := toPrefabContext(evalCtx)
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

func toPrefabContext(evalCtx of.FlattenedContext) (prefab.ContextSet, error) {
	if len(evalCtx) == 0 {
		return prefab.ContextSet{}, nil
	}

	// contextsMap := make(map[string]*PrefabContextBuilder)
	// contextsMap := make(map[string]*strings)
	prefabContext := prefab.NewContextSet()
	for k, v := range evalCtx {
		// val, ok := toStr(v)
		parts := strings.SplitN(k, ".", 2)
		if len(parts) < 2 {
			panic(fmt.Sprintf("context key structure should be in the form of x.y: %s", k))
		}
		key, subkey := parts[0], parts[1]
		if _, exists := prefabContext.Data[key]; !exists {
			// prefabContext.Data[key].Data[subkey] = map[string]interface{}{
			// 	subkey: v,
			// }
			prefabContext.WithNamedContextValues(key, map[string]interface{}{
				subkey: v,
			})
		} else {
			prefabContext.Data[key].Data[subkey] = v
		}
	}
	return *prefabContext, nil
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
