package optimizely

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/open-feature/go-sdk/openfeature"
	optimizely "github.com/optimizely/go-sdk/v2/pkg/client"
	"github.com/optimizely/go-sdk/v2/pkg/decide"
)

// Compile-time check that Provider implements FeatureProvider
var _ openfeature.FeatureProvider = (*Provider)(nil)

// Compile-time check that Provider implements StateHandler
var _ openfeature.StateHandler = (*Provider)(nil)

// ErrTargetingKeyMissing is returned when the targeting key is not provided in the evaluation context.
var ErrTargetingKeyMissing = errors.New("targeting key is required")

// flagNotFoundReason is the string pattern used by Optimizely to indicate a flag was not found.
const flagNotFoundReason = "No flag was found"

type Provider struct {
	client *optimizely.OptimizelyClient
}

func NewProvider(client *optimizely.OptimizelyClient) *Provider {
	return &Provider{
		client: client,
	}
}

func (p *Provider) Metadata() openfeature.Metadata {
	return openfeature.Metadata{
		Name: "Optimizely",
	}
}

func (p *Provider) evaluate(flagKey string, evalCtx openfeature.FlattenedContext) (any, openfeature.ProviderResolutionDetail) {
	userID, ok := evalCtx[openfeature.TargetingKey].(string)
	if !ok {
		return nil, openfeature.ProviderResolutionDetail{
			ResolutionError: openfeature.NewTargetingKeyMissingResolutionError(ErrTargetingKeyMissing.Error()),
			Reason:          openfeature.Reason(openfeature.TargetingKeyMissingCode),
		}
	}

	attributes := make(map[string]any)
	for k, v := range evalCtx {
		if k != openfeature.TargetingKey {
			attributes[k] = v
		}
	}

	userCtx := p.client.CreateUserContext(userID, attributes)
	decision := userCtx.Decide(flagKey, []decide.OptimizelyDecideOptions{})

	for _, reason := range decision.Reasons {
		if strings.Contains(reason, flagNotFoundReason) {
			return nil, openfeature.ProviderResolutionDetail{
				ResolutionError: openfeature.NewFlagNotFoundResolutionError(reason),
				Reason:          openfeature.ErrorReason,
			}
		}
	}

	variables := decision.Variables.ToMap()
	if !decision.Enabled || variables == nil {
		return nil, openfeature.ProviderResolutionDetail{
			Reason: openfeature.DisabledReason,
		}
	}

	variableKey := "value"
	if key, ok := evalCtx["variableKey"].(string); ok && key != "" {
		variableKey = key
	}

	val, exists := variables[variableKey]
	if !exists {
		return nil, openfeature.ProviderResolutionDetail{
			Reason: openfeature.DefaultReason,
		}
	}

	return val, openfeature.ProviderResolutionDetail{
		Reason:  openfeature.TargetingMatchReason,
		Variant: decision.VariationKey,
	}
}

func resolve[T any](p *Provider, flagKey string, defaultValue T, evalCtx openfeature.FlattenedContext) openfeature.GenericResolutionDetail[T] {
	val, detail := p.evaluate(flagKey, evalCtx)
	if val == nil {
		return openfeature.GenericResolutionDetail[T]{
			Value:                    defaultValue,
			ProviderResolutionDetail: detail,
		}
	}

	if converted, ok := val.(T); ok {
		return openfeature.GenericResolutionDetail[T]{
			Value:                    converted,
			ProviderResolutionDetail: detail,
		}
	}

	return openfeature.GenericResolutionDetail[T]{
		Value: defaultValue,
		ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
			ResolutionError: openfeature.NewTypeMismatchResolutionError(fmt.Sprintf("variable is not a %T", defaultValue)),
			Reason:          openfeature.ErrorReason,
		},
	}
}

func (p *Provider) BooleanEvaluation(ctx context.Context, flagKey string, defaultValue bool, evalCtx openfeature.FlattenedContext) openfeature.BoolResolutionDetail {
	return resolve(p, flagKey, defaultValue, evalCtx)
}

func (p *Provider) StringEvaluation(ctx context.Context, flagKey string, defaultValue string, evalCtx openfeature.FlattenedContext) openfeature.StringResolutionDetail {
	return resolve(p, flagKey, defaultValue, evalCtx)
}

func (p *Provider) FloatEvaluation(ctx context.Context, flagKey string, defaultValue float64, evalCtx openfeature.FlattenedContext) openfeature.FloatResolutionDetail {
	return resolve(p, flagKey, defaultValue, evalCtx)
}

func (p *Provider) IntEvaluation(ctx context.Context, flagKey string, defaultValue int64, evalCtx openfeature.FlattenedContext) openfeature.IntResolutionDetail {
	res := resolve(p, flagKey, int(defaultValue), evalCtx)
	return openfeature.IntResolutionDetail{
		Value:                    int64(res.Value),
		ProviderResolutionDetail: res.ProviderResolutionDetail,
	}
}

func (p *Provider) ObjectEvaluation(ctx context.Context, flagKey string, defaultValue any, evalCtx openfeature.FlattenedContext) openfeature.InterfaceResolutionDetail {
	return resolve(p, flagKey, defaultValue, evalCtx)
}

func (p *Provider) Hooks() []openfeature.Hook {
	return []openfeature.Hook{}
}

func (p *Provider) Init(evaluationContext openfeature.EvaluationContext) error {
	return nil
}

func (p *Provider) Shutdown() {
	p.client.Close()
}

func (p *Provider) Status() openfeature.State {
	return openfeature.ReadyState
}
