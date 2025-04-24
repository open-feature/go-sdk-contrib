package strategies

import (
	"context"
	of "github.com/open-feature/go-sdk/openfeature"
	"time"

	mperr "github.com/open-feature/go-sdk-contrib/providers/multi-provider/pkg/errors"
)

type FirstSuccessStrategy struct {
	providers []*NamedProvider
	timeout   time.Duration
}

var _ Strategy = (*FirstSuccessStrategy)(nil)

// NewFirstSuccessStrategy Creates a new FirstSuccessStrategy instance as a Strategy
func NewFirstSuccessStrategy(providers []*NamedProvider, timeout time.Duration) Strategy {
	return &FirstSuccessStrategy{providers: providers, timeout: timeout}
}

func (f *FirstSuccessStrategy) Name() EvaluationStrategy {
	return StrategyFirstSuccess
}

func (f *FirstSuccessStrategy) BooleanEvaluation(ctx context.Context, flag string, defaultValue bool, evalCtx of.FlattenedContext) of.BoolResolutionDetail {
	evalFunc := func(c context.Context, p *NamedProvider) resultWrapper[of.BoolResolutionDetail] {
		result := p.Provider.BooleanEvaluation(c, flag, defaultValue, evalCtx)
		return resultWrapper[of.BoolResolutionDetail]{
			result: &result,
			name:   p.Name,
			value:  result.Value,
			detail: result.ProviderResolutionDetail,
		}
	}
	result, metadata := evaluateFirstSuccess[of.BoolResolutionDetail](ctx, f.providers, evalFunc, defaultValue, f.timeout)
	r := *result.result
	r.ProviderResolutionDetail.FlagMetadata = mergeFlagTags(r.ProviderResolutionDetail.FlagMetadata, metadata)
	return r
}

func (f *FirstSuccessStrategy) StringEvaluation(ctx context.Context, flag string, defaultValue string, evalCtx of.FlattenedContext) of.StringResolutionDetail {
	evalFunc := func(c context.Context, p *NamedProvider) resultWrapper[of.StringResolutionDetail] {
		result := p.Provider.StringEvaluation(c, flag, defaultValue, evalCtx)
		return resultWrapper[of.StringResolutionDetail]{
			result: &result,
			name:   p.Name,
			value:  result.Value,
			detail: result.ProviderResolutionDetail,
		}
	}
	result, metadata := evaluateFirstSuccess[of.StringResolutionDetail](ctx, f.providers, evalFunc, defaultValue, f.timeout)
	r := *result.result
	r.ProviderResolutionDetail.FlagMetadata = mergeFlagTags(r.ProviderResolutionDetail.FlagMetadata, metadata)
	return r
}

func (f *FirstSuccessStrategy) FloatEvaluation(ctx context.Context, flag string, defaultValue float64, evalCtx of.FlattenedContext) of.FloatResolutionDetail {
	evalFunc := func(c context.Context, p *NamedProvider) resultWrapper[of.FloatResolutionDetail] {
		result := p.Provider.FloatEvaluation(c, flag, defaultValue, evalCtx)
		return resultWrapper[of.FloatResolutionDetail]{
			result: &result,
			name:   p.Name,
			value:  result.Value,
			detail: result.ProviderResolutionDetail,
		}
	}
	result, metadata := evaluateFirstSuccess[of.FloatResolutionDetail](ctx, f.providers, evalFunc, defaultValue, f.timeout)
	r := *result.result
	r.ProviderResolutionDetail.FlagMetadata = mergeFlagTags(r.ProviderResolutionDetail.FlagMetadata, metadata)
	return r
}

func (f *FirstSuccessStrategy) IntEvaluation(ctx context.Context, flag string, defaultValue int64, evalCtx of.FlattenedContext) of.IntResolutionDetail {
	evalFunc := func(c context.Context, p *NamedProvider) resultWrapper[of.IntResolutionDetail] {
		result := p.Provider.IntEvaluation(c, flag, defaultValue, evalCtx)
		return resultWrapper[of.IntResolutionDetail]{
			result: &result,
			name:   p.Name,
			value:  result.Value,
			detail: result.ProviderResolutionDetail,
		}
	}
	result, metadata := evaluateFirstSuccess[of.IntResolutionDetail](ctx, f.providers, evalFunc, defaultValue, f.timeout)
	r := *result.result
	r.ProviderResolutionDetail.FlagMetadata = mergeFlagTags(r.ProviderResolutionDetail.FlagMetadata, metadata)
	return r
}

func (f *FirstSuccessStrategy) ObjectEvaluation(ctx context.Context, flag string, defaultValue interface{}, evalCtx of.FlattenedContext) of.InterfaceResolutionDetail {
	evalFunc := func(c context.Context, p *NamedProvider) resultWrapper[of.InterfaceResolutionDetail] {
		result := p.Provider.ObjectEvaluation(c, flag, defaultValue, evalCtx)
		return resultWrapper[of.InterfaceResolutionDetail]{
			result: &result,
			name:   p.Name,
			value:  result.Value,
			detail: result.ProviderResolutionDetail,
		}
	}
	result, metadata := evaluateFirstSuccess[of.InterfaceResolutionDetail](ctx, f.providers, evalFunc, defaultValue, f.timeout)
	r := *result.result
	r.ProviderResolutionDetail.FlagMetadata = mergeFlagTags(r.ProviderResolutionDetail.FlagMetadata, metadata)
	return r
}

func evaluateFirstSuccess[R resultConstraint, DV bool | string | int64 | float64 | interface{}](ctx context.Context, providers []*NamedProvider, e evaluator[R], defaultVal DV, timeout time.Duration) (resultWrapper[R], of.FlagMetadata) {
	metadata := make(of.FlagMetadata)
	metadata[MetadataStrategyUsed] = StrategyFirstSuccess
	errChan := make(chan mperr.ProviderError, len(providers))
	notFoundChan := make(chan interface{})
	finishChan := make(chan *resultWrapper[R], len(providers))
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	for _, provider := range providers {
		go func(c context.Context, p *NamedProvider) {
			resultChan := make(chan *resultWrapper[R])
			go func() {
				r := e(c, provider)
				resultChan <- &r
			}()

			select {
			case <-c.Done():
				return
			case r := <-resultChan:
				if r.detail.Error() != nil && r.detail.ResolutionDetail().ErrorCode == of.FlagNotFoundCode {
					notFoundChan <- struct{}{}
					return
				} else if r.detail.Error() != nil {
					errChan <- mperr.ProviderError{
						Err:          r.detail.ResolutionError,
						ProviderName: p.Name,
					}
					return
				}
				finishChan <- r
			}

		}(ctx, provider)
	}

	errs := make([]mperr.ProviderError, 0, len(providers))
	notFoundCount := 0
	for {
		if len(errs) == len(providers) {
			err := mperr.NewAggregateError(errs)
			r := buildDefaultResult[R](StrategyFirstSuccess, defaultVal, err)
			return r, r.detail.FlagMetadata
		}

		select {
		case result := <-finishChan:
			metadata[MetadataSuccessfulProviderName] = result.name
			cancel()
			return *result, metadata
		case err := <-errChan:
			errs = append(errs, err)
		case <-notFoundChan:
			notFoundCount += 1
			if notFoundCount == len(providers) {
				r := buildDefaultResult[R](StrategyFirstSuccess, defaultVal, nil)
				return r, r.detail.FlagMetadata
			}
		case <-ctx.Done():
			var err error
			if len(errs) > 0 {
				err = mperr.NewAggregateError(errs)
			} else {
				err = ctx.Err()
			}
			r := buildDefaultResult[R](StrategyFirstSuccess, defaultVal, err)
			return r, r.detail.FlagMetadata
		}
	}
}
