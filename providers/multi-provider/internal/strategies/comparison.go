package strategies

import (
	"cmp"
	"context"
	"errors"
	of "github.com/open-feature/go-sdk/openfeature"
	"golang.org/x/sync/errgroup"
	"slices"
	"strings"
)

const (
	MetadataIsDefault       = "multiprovider-is-default-result"
	MetadataNoneFound       = "multiprovider-flag-not-found-all-providers"
	MetadataEvaluationError = "multiprovider-comparison-first-error"
)

type (
	ComparisonStrategy struct {
		providers        []*NamedProvider
		fallbackProvider of.FeatureProvider
	}

	comparator[R bool | string | int64 | float64] func(values []R) bool
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
	compFunc := func(values []bool) bool {
		current := values[0]
		match := true
		for i, v := range values {
			if i == 0 {
				continue
			}
			if current != v {
				match = false
				break
			}
		}

		return match
	}
	results, metadata := evaluateComparison[of.BoolResolutionDetail, bool](ctx, c.providers, evalFunc, compFunc, c.fallbackProvider, defaultValue)

	return of.BoolResolutionDetail{
		Value: results[0].result.Value,
		ProviderResolutionDetail: of.ProviderResolutionDetail{
			ResolutionError: comparisonResolutionError(metadata),
			Reason:          comparisonResolutionReason(metadata),
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
	compFunc := func(values []string) bool {
		current := values[0]
		match := true
		for i, v := range values {
			if i == 0 {
				continue
			}
			if current != v {
				match = false
				break
			}
		}

		return match
	}

	results, metadata := evaluateComparison[of.StringResolutionDetail, string](ctx, c.providers, evalFunc, compFunc, c.fallbackProvider, defaultValue)
	return of.StringResolutionDetail{
		Value: results[0].result.Value,
		ProviderResolutionDetail: of.ProviderResolutionDetail{
			ResolutionError: comparisonResolutionError(metadata),
			Reason:          comparisonResolutionReason(metadata),
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
	compFunc := func(values []float64) bool {
		current := values[0]
		match := true
		for i, v := range values {
			if i == 0 {
				continue
			}
			if current != v {
				match = false
				break
			}
		}

		return match
	}

	results, metadata := evaluateComparison[of.FloatResolutionDetail, float64](ctx, c.providers, evalFunc, compFunc, c.fallbackProvider, defaultValue)
	return of.FloatResolutionDetail{
		Value: results[0].result.Value,
		ProviderResolutionDetail: of.ProviderResolutionDetail{
			ResolutionError: comparisonResolutionError(metadata),
			Reason:          comparisonResolutionReason(metadata),
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
	compFunc := func(values []int64) bool {
		current := values[0]
		match := true
		for i, v := range values {
			if i == 0 {
				continue
			}
			if current != v {
				match = false
				break
			}
		}

		return match
	}
	results, metadata := evaluateComparison[of.IntResolutionDetail, int64](ctx, c.providers, evalFunc, compFunc, c.fallbackProvider, defaultValue)
	return of.IntResolutionDetail{
		Value: results[0].result.Value,
		ProviderResolutionDetail: of.ProviderResolutionDetail{
			ResolutionError: comparisonResolutionError(metadata),
			Reason:          comparisonResolutionReason(metadata),
			Variant:         results[0].detail.Variant,
			FlagMetadata:    metadata,
		},
	}
}

func comparisonResolutionReason(metadata of.FlagMetadata) of.Reason {
	reason := ReasonAggregated
	if fallbackUsed, err := metadata.GetBool(MetadataFallbackUsed); fallbackUsed && err == nil {
		reason = ReasonAggregatedFallback
	} else if defaultUsed, err := metadata.GetBool(MetadataIsDefault); defaultUsed && err == nil {
		reason = of.DefaultReason
	}
	return reason
}

func comparisonResolutionError(metadata of.FlagMetadata) of.ResolutionError {
	if isDefault, err := metadata.GetBool(MetadataIsDefault); err != nil || !isDefault {
		return of.ResolutionError{}
	}

	if notFound, err := metadata.GetBool(MetadataNoneFound); err == nil && notFound {
		return of.NewFlagNotFoundResolutionError("not found in any providers")
	}

	if evalErr, err := metadata.GetString(MetadataEvaluationError); evalErr != "" && err != nil {
		return of.NewGeneralResolutionError(evalErr)
	}

	return of.NewGeneralResolutionError("comparison failure")
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

func evaluateComparison[R resultConstraint, DV bool | string | int64 | float64](ctx context.Context, providers []*NamedProvider, e evaluator[R], comp comparator[DV], fallbackProvider of.FeatureProvider, defaultVal DV) ([]resultWrapper[R], of.FlagMetadata) {
	if len(providers) == 1 {
		result := e(ctx, providers[0])
		metadata := setFlagMetadata(StrategyComparison, cmp.Or(result.name, providers[0].Name), make(of.FlagMetadata))
		metadata[MetadataFallbackUsed] = false
		return []resultWrapper[R]{result}, metadata
	}

	resultChan := make(chan *resultWrapper[R], len(providers))
	notFoundChan := make(chan interface{})
	errGrp, ctx := errgroup.WithContext(ctx)
	for _, provider := range providers {
		errGrp.Go(func() error {
			localChan := make(chan *resultWrapper[R])

			go func(c context.Context, p *NamedProvider) {
				result := e(c, p)
				localChan <- &result
			}(ctx, provider)

			select {
			case r := <-localChan:
				notFound := r.detail.ResolutionDetail().ErrorCode == of.FlagNotFoundCode
				if !notFound && r.detail.Error() != nil {
					return &providerError{
						providerName: r.name,
						err:          r.detail.Error(),
					}
				}
				if !notFound {
					resultChan <- r
				} else {
					notFoundChan <- struct{}{}
				}
				return nil
			case <-ctx.Done():
				return nil
			}
		})
	}

	results := make([]resultWrapper[R], 0, len(providers))
	resultValues := make([]DV, 0, len(providers))
	notFoundCount := 0
	for {
		select {
		case <-ctx.Done():
			// Error occurred
			result := buildDefaultResult[R, DV](StrategyComparison, defaultVal, ctx.Err())
			metadata := result.detail.FlagMetadata
			metadata[MetadataFallbackUsed] = false
			metadata[MetadataIsDefault] = true
			metadata[MetadataEvaluationError] = ctx.Err().Error()
			return []resultWrapper[R]{result}, metadata
		case r := <-resultChan:
			results = append(results, *r)
			resultValues = append(resultValues, r.value.(DV))
			if (len(results) + notFoundCount) == len(providers) {
				goto continueComparison
			}
		case <-notFoundChan:
			notFoundCount += 1
			if notFoundCount == len(providers) {
				result := buildDefaultResult[R, DV](StrategyComparison, defaultVal, ctx.Err())
				metadata := result.detail.FlagMetadata
				metadata[MetadataFallbackUsed] = false
				metadata[MetadataIsDefault] = true
				return []resultWrapper[R]{result}, metadata
			}
			if (len(results) + notFoundCount) == len(providers) {
				goto continueComparison
			}
		}
	}
continueComparison:
	// Evaluate Results Are Equal
	metadata := make(of.FlagMetadata)
	metadata[MetadataStrategyUsed] = StrategyComparison
	agreement := comp(resultValues)
	if agreement {
		metadata[MetadataFallbackUsed] = false
		metadata[MetadataIsDefault] = false
		success := make([]string, 0, len(providers))
		for _, r := range results {
			metadata[r.name] = r.detail.FlagMetadata
			success = append(success, r.name)
		}
		// maintain stable order of metadata results
		slices.Sort(success)
		metadata[MetadataSuccessfulProviderName+"s"] = strings.Join(success, ", ")
		return results, metadata
	}

	if fallbackProvider != nil {
		fallbackResult := e(ctx, &NamedProvider{Name: "fallback", Provider: fallbackProvider})
		metadata = fallbackResult.detail.FlagMetadata
		metadata[MetadataFallbackUsed] = true
		metadata[MetadataIsDefault] = false
		metadata[MetadataSuccessfulProviderName] = "fallback"
		metadata[MetadataStrategyUsed] = StrategyComparison

		return []resultWrapper[R]{fallbackResult}, metadata
	}

	defaultResult := buildDefaultResult[R, DV](StrategyComparison, defaultVal, errors.New("no fallback provider configured"))
	metadata = defaultResult.detail.FlagMetadata
	metadata[MetadataFallbackUsed] = false
	metadata[MetadataIsDefault] = true

	return []resultWrapper[R]{defaultResult}, metadata
}
