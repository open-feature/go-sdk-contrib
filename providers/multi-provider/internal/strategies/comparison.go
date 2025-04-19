package strategies

import (
	"cmp"
	"context"
	"errors"
	of "github.com/open-feature/go-sdk/openfeature"
	"golang.org/x/sync/errgroup"
	"strings"
)

type (
	ComparisonStrategy struct {
		providers        []*NamedProvider
		fallbackProvider of.FeatureProvider
	}

	evaluator[R resultConstraint] func(ctx context.Context, p *NamedProvider) resultWrapper[R]
)

var _ Strategy = (*ComparisonStrategy)(nil)

func NewComparisonStrategy(providers []*NamedProvider, fallbackProvider of.FeatureProvider) *ComparisonStrategy {
	return &ComparisonStrategy{
		providers:        providers,
		fallbackProvider: fallbackProvider,
	}
}

func (c ComparisonStrategy) Name() EvaluationStrategy {
	return StrategyComparison
}

func (c ComparisonStrategy) BooleanEvaluation(ctx context.Context, flag string, defaultValue bool, evalCtx of.FlattenedContext) of.BoolResolutionDetail {
	evalFunc := func(c context.Context, p *NamedProvider) resultWrapper[of.BoolResolutionDetail] {
		result := p.Provider.BooleanEvaluation(ctx, flag, defaultValue, evalCtx)
		return resultWrapper[of.BoolResolutionDetail]{
			result: &result,
			name:   p.Name,
			value:  result.Value,
			detail: result.ProviderResolutionDetail,
		}
	}
	results, metadata := evaluateComparison[of.BoolResolutionDetail, bool](ctx, c.providers, evalFunc, c.fallbackProvider, defaultValue)
	if len(results) == 1 {
		results[0].result.FlagMetadata[MetadataSuccessfulProviderName] = results[0].name
		return *results[0].result
	}

	reason := ReasonAggregated
	if fallbackUsed, ok := metadata[MetadataFallbackUsed].(bool); fallbackUsed && ok {
		reason = ReasonAggregatedFallback
	}

	return of.BoolResolutionDetail{
		Value: results[0].result.Value,
		ProviderResolutionDetail: of.ProviderResolutionDetail{
			ResolutionError: of.ResolutionError{},
			Reason:          reason,
			Variant:         results[0].detail.Variant,
			FlagMetadata:    metadata,
		},
	}
}

func (c ComparisonStrategy) StringEvaluation(ctx context.Context, flag string, defaultValue string, evalCtx of.FlattenedContext) of.StringResolutionDetail {
	evalFunc := func(c context.Context, p *NamedProvider) resultWrapper[of.StringResolutionDetail] {
		result := p.Provider.StringEvaluation(ctx, flag, defaultValue, evalCtx)
		return resultWrapper[of.StringResolutionDetail]{
			result: &result,
			name:   p.Name,
			value:  result.Value,
			detail: result.ProviderResolutionDetail,
		}
	}
	results, metadata := evaluateComparison[of.StringResolutionDetail, string](ctx, c.providers, evalFunc, c.fallbackProvider, defaultValue)
	if len(results) == 1 {
		results[0].result.FlagMetadata[MetadataSuccessfulProviderName] = results[0].name
		return *results[0].result
	}

	reason := ReasonAggregated
	if fallbackUsed, ok := metadata[MetadataFallbackUsed].(bool); fallbackUsed && ok {
		reason = ReasonAggregatedFallback
	}

	return of.StringResolutionDetail{
		Value: results[0].result.Value,
		ProviderResolutionDetail: of.ProviderResolutionDetail{
			ResolutionError: of.ResolutionError{},
			Reason:          reason,
			Variant:         results[0].detail.Variant,
			FlagMetadata:    metadata,
		},
	}
}

func (c ComparisonStrategy) FloatEvaluation(ctx context.Context, flag string, defaultValue float64, evalCtx of.FlattenedContext) of.FloatResolutionDetail {
	evalFunc := func(c context.Context, p *NamedProvider) resultWrapper[of.FloatResolutionDetail] {
		result := p.Provider.FloatEvaluation(ctx, flag, defaultValue, evalCtx)
		return resultWrapper[of.FloatResolutionDetail]{
			result: &result,
			name:   p.Name,
			value:  result.Value,
			detail: result.ProviderResolutionDetail,
		}
	}
	results, metadata := evaluateComparison[of.FloatResolutionDetail, float64](ctx, c.providers, evalFunc, c.fallbackProvider, defaultValue)
	if len(results) == 1 {
		results[0].result.FlagMetadata[MetadataSuccessfulProviderName] = results[0].name
		return *results[0].result
	}

	reason := ReasonAggregated
	if fallbackUsed, ok := metadata[MetadataFallbackUsed].(bool); fallbackUsed && ok {
		reason = ReasonAggregatedFallback
	}

	return of.FloatResolutionDetail{
		Value: results[0].result.Value,
		ProviderResolutionDetail: of.ProviderResolutionDetail{
			ResolutionError: of.ResolutionError{},
			Reason:          reason,
			Variant:         results[0].detail.Variant,
			FlagMetadata:    metadata,
		},
	}
}

func (c ComparisonStrategy) IntEvaluation(ctx context.Context, flag string, defaultValue int64, evalCtx of.FlattenedContext) of.IntResolutionDetail {
	evalFunc := func(c context.Context, p *NamedProvider) resultWrapper[of.IntResolutionDetail] {
		result := p.Provider.IntEvaluation(ctx, flag, defaultValue, evalCtx)
		return resultWrapper[of.IntResolutionDetail]{
			result: &result,
			name:   p.Name,
			value:  result.Value,
			detail: result.ProviderResolutionDetail,
		}
	}
	results, metadata := evaluateComparison[of.IntResolutionDetail, int64](ctx, c.providers, evalFunc, c.fallbackProvider, defaultValue)
	if len(results) == 1 {
		results[0].result.FlagMetadata[MetadataSuccessfulProviderName] = results[0].name
		return *results[0].result
	}

	reason := ReasonAggregated
	if fallbackUsed, ok := metadata[MetadataFallbackUsed].(bool); fallbackUsed && ok {
		reason = ReasonAggregatedFallback
	}

	return of.IntResolutionDetail{
		Value: results[0].result.Value,
		ProviderResolutionDetail: of.ProviderResolutionDetail{
			ResolutionError: of.ResolutionError{},
			Reason:          reason,
			Variant:         results[0].detail.Variant,
			FlagMetadata:    metadata,
		},
	}
}

func (c ComparisonStrategy) ObjectEvaluation(ctx context.Context, flag string, defaultValue interface{}, evalCtx of.FlattenedContext) of.InterfaceResolutionDetail {
	metadata := make(of.FlagMetadata)
	metadata[MetadataStrategyUsed] = StrategyComparison
	metadata[MetadataSuccessfulProviderName] = "none"
	metadata[MetadataFallbackUsed] = false

	return of.InterfaceResolutionDetail{
		Value: defaultValue,
		ProviderResolutionDetail: of.ProviderResolutionDetail{
			ResolutionError: of.NewGeneralResolutionError(ErrAggregationNotAllowedText),
			Reason:          of.DefaultReason,
			Variant:         "",
			FlagMetadata:    metadata,
		},
	}
}

func evaluateComparison[R resultConstraint, DV bool | string | int64 | float64](ctx context.Context, providers []*NamedProvider, e evaluator[R], fallbackProvider of.FeatureProvider, defaultVal DV) ([]resultWrapper[R], of.FlagMetadata) {
	resultChan := make(chan resultWrapper[R])
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	errGrp, ctx := errgroup.WithContext(ctx)
	for _, provider := range providers {
		p := provider
		errGrp.Go(func() error {
			result := e(ctx, p)
			notFound := result.detail.ResolutionDetail().ErrorCode == of.FlagNotFoundCode
			if result.detail.Error() != nil {
				return &providerError{
					providerName: p.Name,
					err:          result.detail.Error(),
				}
			}
			result.name = p.Name
			if !notFound {
				resultChan <- result
			}
			return nil
		})
	}

	if err := errGrp.Wait(); err != nil {
		result := buildDefaultResult[R, DV](StrategyComparison, defaultVal, err)
		return []resultWrapper[R]{result}, result.detail.FlagMetadata
	}

	// Evaluate Results Are Equal
	agreement := true
	var resultVal DV
	results := make([]resultWrapper[R], 0, len(providers))
	metadata := make(of.FlagMetadata)
	metadata[MetadataFallbackUsed] = false
	metadata[MetadataStrategyUsed] = StrategyComparison
	success := make([]string, 0, len(providers))
	for r := range resultChan {
		results = append(results, r)
		current := *r.value.(*DV)
		resultVal = cmp.Or(resultVal, current)
		agreement = agreement && (resultVal == current)
		if !agreement {
			break
		}
		metadata[r.name] = r.detail.FlagMetadata
		success = append(success, r.name)
	}
	metadata[MetadataSuccessfulProviderName+"s"] = strings.Join(success, ", ")

	if agreement {
		return results, metadata
	}

	if fallbackProvider != nil {
		fallbackResult := e(ctx, &NamedProvider{Name: "fallback", Provider: fallbackProvider})
		metadata = fallbackResult.detail.FlagMetadata
		metadata[MetadataStrategyUsed] = StrategyComparison
		metadata[MetadataFallbackUsed] = true

		return []resultWrapper[R]{fallbackResult}, metadata
	}

	defaultResult := buildDefaultResult[R, DV](StrategyComparison, defaultVal, errors.New("no fallback provider configured"))
	metadata = defaultResult.detail.FlagMetadata
	metadata[MetadataSuccessfulProviderName] = "none"
	metadata[MetadataFallbackUsed] = false
	metadata[MetadataStrategyUsed] = StrategyComparison
	return []resultWrapper[R]{defaultResult}, metadata

}
