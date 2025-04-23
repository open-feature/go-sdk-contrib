package strategies

import (
	"context"
	of "github.com/open-feature/go-sdk/openfeature"
)

type FirstMatchStrategy struct {
	providers []*NamedProvider
}

var _ Strategy = (*FirstMatchStrategy)(nil)

// NewFirstMatchStrategy Creates a new FirstMatchStrategy instance as a Strategy
func NewFirstMatchStrategy(providers []*NamedProvider) Strategy {
	return &FirstMatchStrategy{providers: providers}
}

func (f *FirstMatchStrategy) Name() EvaluationStrategy {
	return StrategyFirstMatch
}

func (f *FirstMatchStrategy) BooleanEvaluation(ctx context.Context, flag string, defaultValue bool, evalCtx of.FlattenedContext) of.BoolResolutionDetail {
	evalFunc := func(c context.Context, p *NamedProvider) resultWrapper[of.BoolResolutionDetail] {
		r := p.Provider.BooleanEvaluation(c, flag, defaultValue, evalCtx)
		return resultWrapper[of.BoolResolutionDetail]{
			result: &r,
			name:   p.Name,
			value:  r.Value,
			detail: r.ProviderResolutionDetail,
		}
	}
	result := evaluateFirstMatch[of.BoolResolutionDetail](ctx, f.providers, evalFunc, defaultValue)
	result.result.ProviderResolutionDetail = result.detail
	return *result.result
}

func (f *FirstMatchStrategy) StringEvaluation(ctx context.Context, flag string, defaultValue string, evalCtx of.FlattenedContext) of.StringResolutionDetail {
	evalFunc := func(c context.Context, p *NamedProvider) resultWrapper[of.StringResolutionDetail] {
		r := p.Provider.StringEvaluation(c, flag, defaultValue, evalCtx)
		return resultWrapper[of.StringResolutionDetail]{
			result: &r,
			name:   p.Name,
			value:  r.Value,
			detail: r.ProviderResolutionDetail,
		}
	}
	result := evaluateFirstMatch[of.StringResolutionDetail](ctx, f.providers, evalFunc, defaultValue)
	result.result.ProviderResolutionDetail = result.detail
	return *result.result
}

func (f *FirstMatchStrategy) FloatEvaluation(ctx context.Context, flag string, defaultValue float64, evalCtx of.FlattenedContext) of.FloatResolutionDetail {
	evalFunc := func(c context.Context, p *NamedProvider) resultWrapper[of.FloatResolutionDetail] {
		r := p.Provider.FloatEvaluation(c, flag, defaultValue, evalCtx)
		return resultWrapper[of.FloatResolutionDetail]{
			result: &r,
			name:   p.Name,
			value:  r.Value,
			detail: r.ProviderResolutionDetail,
		}
	}
	result := evaluateFirstMatch[of.FloatResolutionDetail](ctx, f.providers, evalFunc, defaultValue)
	result.result.ProviderResolutionDetail = result.detail
	return *result.result
}

func (f *FirstMatchStrategy) IntEvaluation(ctx context.Context, flag string, defaultValue int64, evalCtx of.FlattenedContext) of.IntResolutionDetail {
	evalFunc := func(c context.Context, p *NamedProvider) resultWrapper[of.IntResolutionDetail] {
		r := p.Provider.IntEvaluation(c, flag, defaultValue, evalCtx)
		return resultWrapper[of.IntResolutionDetail]{
			result: &r,
			name:   p.Name,
			value:  r.Value,
			detail: r.ProviderResolutionDetail,
		}
	}
	result := evaluateFirstMatch[of.IntResolutionDetail](ctx, f.providers, evalFunc, defaultValue)
	result.result.ProviderResolutionDetail = result.detail
	return *result.result
}

func (f *FirstMatchStrategy) ObjectEvaluation(ctx context.Context, flag string, defaultValue interface{}, evalCtx of.FlattenedContext) of.InterfaceResolutionDetail {
	evalFunc := func(c context.Context, p *NamedProvider) resultWrapper[of.InterfaceResolutionDetail] {
		r := p.Provider.ObjectEvaluation(c, flag, defaultValue, evalCtx)
		return resultWrapper[of.InterfaceResolutionDetail]{
			result: &r,
			name:   p.Name,
			value:  r.Value,
			detail: r.ProviderResolutionDetail,
		}
	}
	result := evaluateFirstMatch[of.InterfaceResolutionDetail](ctx, f.providers, evalFunc, defaultValue)
	result.result.ProviderResolutionDetail = result.detail
	return *result.result
}

func evaluateFirstMatch[R resultConstraint, DV bool | string | int64 | float64 | interface{}](ctx context.Context, providers []*NamedProvider, e evaluator[R], defaultVal DV) resultWrapper[R] {
	for _, provider := range providers {
		r := e(ctx, provider)
		if r.detail.Error() != nil && r.detail.ResolutionDetail().ErrorCode == of.FlagNotFoundCode {
			continue
		}
		if r.detail.Error() != nil {
			return buildDefaultResult[R](StrategyFirstMatch, defaultVal, r.detail.Error())
		}

		// success!
		r.detail.FlagMetadata = setFlagMetadata(StrategyFirstMatch, provider.Name, r.detail.FlagMetadata)
		return r
	}

	// Build a default result if no matches are found
	return buildDefaultResult[R](StrategyFirstMatch, defaultVal, nil)
}
