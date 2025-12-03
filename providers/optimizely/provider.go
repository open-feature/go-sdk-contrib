package optimizely

import (
	"context"
	"errors"
	"fmt"

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

// Error messages for evaluation method restrictions based on variable count.
const (
	errNoVariables       = "flag has no variables; use BooleanEvaluation"
	errMultipleVariables = "flag has multiple variables; use ObjectEvaluation"
)

type evaluationResult struct {
	enabled   bool
	variables map[string]any
	variant   string
	detail    openfeature.ProviderResolutionDetail
	hasError  bool
}

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

func (p *Provider) getDecision(flagKey string, evalCtx openfeature.FlattenedContext) evaluationResult {
	userID, ok := evalCtx[openfeature.TargetingKey].(string)
	if !ok {
		return evaluationResult{
			hasError: true,
			detail: openfeature.ProviderResolutionDetail{
				ResolutionError: openfeature.NewTargetingKeyMissingResolutionError(ErrTargetingKeyMissing.Error()),
				Reason:          openfeature.Reason(openfeature.TargetingKeyMissingCode),
			},
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
	sdkNotReadyMsg := decide.GetDecideMessage(decide.SDKNotReady)
	flagNotFoundMsg := decide.GetDecideMessage(decide.FlagKeyInvalid, flagKey)

	for _, reason := range decision.Reasons {
		if reason == sdkNotReadyMsg {
			return evaluationResult{
				hasError: true,
				detail: openfeature.ProviderResolutionDetail{
					ResolutionError: openfeature.NewProviderNotReadyResolutionError(reason),
					Reason:          openfeature.ErrorReason,
				},
			}
		}
		if reason == flagNotFoundMsg {
			return evaluationResult{
				hasError: true,
				detail: openfeature.ProviderResolutionDetail{
					ResolutionError: openfeature.NewFlagNotFoundResolutionError(reason),
					Reason:          openfeature.ErrorReason,
				},
			}
		}
	}

	variables := decision.Variables.ToMap()

	return evaluationResult{
		enabled:   decision.Enabled,
		variables: variables,
		variant:   decision.VariationKey,
	}
}

func getSingleVariable(variables map[string]any) any {
	for _, v := range variables {
		return v
	}
	return nil
}

// requireSingleVariable checks that the result has exactly one variable,
// then attempts to cast the variable to type T.
// Returns the typed value, variant, and nil detail on success.
// Returns zero value and error detail if validation fails or type doesn't match.
func requireSingleVariable[T any](result evaluationResult) (T, string, *openfeature.ProviderResolutionDetail) {
	var zero T
	numVars := len(result.variables)

	if numVars == 0 {
		detail := openfeature.ProviderResolutionDetail{
			ResolutionError: openfeature.NewGeneralResolutionError(errNoVariables),
			Reason:          openfeature.ErrorReason,
		}
		return zero, "", &detail
	}

	if numVars > 1 {
		detail := openfeature.ProviderResolutionDetail{
			ResolutionError: openfeature.NewGeneralResolutionError(errMultipleVariables),
			Reason:          openfeature.ErrorReason,
		}
		return zero, "", &detail
	}

	val := getSingleVariable(result.variables)
	typedVal, ok := val.(T)
	if !ok {
		detail := openfeature.ProviderResolutionDetail{
			ResolutionError: openfeature.NewTypeMismatchResolutionError(fmt.Sprintf("variable is not %T, got %T", zero, val)),
			Reason:          openfeature.ErrorReason,
		}
		return zero, "", &detail
	}

	return typedVal, result.variant, nil
}

func resolutionSuccess[T any](value T, variant string) openfeature.GenericResolutionDetail[T] {
	return openfeature.GenericResolutionDetail[T]{
		Value: value,
		ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
			Reason:  openfeature.TargetingMatchReason,
			Variant: variant,
		},
	}
}

func resolutionFromDetail[T any](value T, detail openfeature.ProviderResolutionDetail) openfeature.GenericResolutionDetail[T] {
	return openfeature.GenericResolutionDetail[T]{
		Value:                    value,
		ProviderResolutionDetail: detail,
	}
}

func (p *Provider) BooleanEvaluation(ctx context.Context, flagKey string, defaultValue bool, evalCtx openfeature.FlattenedContext) openfeature.BoolResolutionDetail {
	result := p.getDecision(flagKey, evalCtx)
	if result.hasError {
		return resolutionFromDetail(defaultValue, result.detail)
	}

	if !result.enabled {
		return resolutionFromDetail(defaultValue, openfeature.ProviderResolutionDetail{Reason: openfeature.DisabledReason})
	}

	// 0 variables: return decision.Enabled
	if len(result.variables) == 0 {
		return resolutionSuccess(result.enabled, result.variant)
	}

	val, variant, errDetail := requireSingleVariable[bool](result)
	if errDetail != nil {
		return resolutionFromDetail(defaultValue, *errDetail)
	}

	return resolutionSuccess(val, variant)
}

func (p *Provider) StringEvaluation(ctx context.Context, flagKey string, defaultValue string, evalCtx openfeature.FlattenedContext) openfeature.StringResolutionDetail {
	result := p.getDecision(flagKey, evalCtx)
	if result.hasError {
		return resolutionFromDetail(defaultValue, result.detail)
	}

	if !result.enabled {
		return resolutionFromDetail(defaultValue, openfeature.ProviderResolutionDetail{Reason: openfeature.DisabledReason})
	}

	val, variant, errDetail := requireSingleVariable[string](result)
	if errDetail != nil {
		return resolutionFromDetail(defaultValue, *errDetail)
	}

	return resolutionSuccess(val, variant)
}

func (p *Provider) FloatEvaluation(ctx context.Context, flagKey string, defaultValue float64, evalCtx openfeature.FlattenedContext) openfeature.FloatResolutionDetail {
	result := p.getDecision(flagKey, evalCtx)
	if result.hasError {
		return resolutionFromDetail(defaultValue, result.detail)
	}

	if !result.enabled {
		return resolutionFromDetail(defaultValue, openfeature.ProviderResolutionDetail{Reason: openfeature.DisabledReason})
	}

	val, variant, errDetail := requireSingleVariable[float64](result)
	if errDetail != nil {
		return resolutionFromDetail(defaultValue, *errDetail)
	}

	return resolutionSuccess(val, variant)
}

func (p *Provider) IntEvaluation(ctx context.Context, flagKey string, defaultValue int64, evalCtx openfeature.FlattenedContext) openfeature.IntResolutionDetail {
	result := p.getDecision(flagKey, evalCtx)
	if result.hasError {
		return resolutionFromDetail(defaultValue, result.detail)
	}

	if !result.enabled {
		return resolutionFromDetail(defaultValue, openfeature.ProviderResolutionDetail{Reason: openfeature.DisabledReason})
	}

	val, variant, errDetail := requireSingleVariable[int](result)
	if errDetail != nil {
		return resolutionFromDetail(defaultValue, *errDetail)
	}

	return resolutionSuccess(int64(val), variant)
}

func (p *Provider) ObjectEvaluation(ctx context.Context, flagKey string, defaultValue any, evalCtx openfeature.FlattenedContext) openfeature.InterfaceResolutionDetail {
	result := p.getDecision(flagKey, evalCtx)
	if result.hasError {
		return resolutionFromDetail(defaultValue, result.detail)
	}

	if !result.enabled {
		return resolutionFromDetail(defaultValue, openfeature.ProviderResolutionDetail{Reason: openfeature.DisabledReason})
	}

	// Multiple variables: return the full map
	if len(result.variables) > 1 {
		return resolutionSuccess[any](result.variables, result.variant)
	}

	// 0 or 1 variables
	val, variant, errDetail := requireSingleVariable[any](result)
	if errDetail != nil {
		return resolutionFromDetail(defaultValue, *errDetail)
	}

	return resolutionSuccess(val, variant)
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
