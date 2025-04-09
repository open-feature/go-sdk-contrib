package strategies

import (
	"context"
	multiprovider "github.com/open-feature/go-sdk-contrib/providers/multi-provider/pkg"
	of "github.com/open-feature/go-sdk/openfeature"
)

type FirstMatchStrategy struct {
	providers []multiprovider.UniqueNameProvider
}

var _ Strategy = (*FirstMatchStrategy)(nil)

// NewFirstMatchStrategy Creates a new FirstMatchStrategy instance as a Strategy
func NewFirstMatchStrategy(providers []multiprovider.UniqueNameProvider) Strategy {
	return &FirstMatchStrategy{providers: providers}
}

func (f *FirstMatchStrategy) Name() EvaluationStrategy {
	return StrategyFirstMatch
}

func (f *FirstMatchStrategy) BooleanEvaluation(ctx context.Context, flag string, defaultValue bool, evalCtx of.FlattenedContext) of.BoolResolutionDetail {
	return *evaluateFirstMatch[of.BoolResolutionDetail](ctx, f.providers, flag, defaultValue, evalCtx).result
}

func (f *FirstMatchStrategy) StringEvaluation(ctx context.Context, flag string, defaultValue string, evalCtx of.FlattenedContext) of.StringResolutionDetail {
	return *evaluateFirstMatch[of.StringResolutionDetail](ctx, f.providers, flag, defaultValue, evalCtx).result
}

func (f *FirstMatchStrategy) FloatEvaluation(ctx context.Context, flag string, defaultValue float64, evalCtx of.FlattenedContext) of.FloatResolutionDetail {
	return *evaluateFirstMatch[of.FloatResolutionDetail](ctx, f.providers, flag, defaultValue, evalCtx).result
}

func (f *FirstMatchStrategy) IntEvaluation(ctx context.Context, flag string, defaultValue int64, evalCtx of.FlattenedContext) of.IntResolutionDetail {
	return *evaluateFirstMatch[of.IntResolutionDetail](ctx, f.providers, flag, defaultValue, evalCtx).result
}

func (f *FirstMatchStrategy) ObjectEvaluation(ctx context.Context, flag string, defaultValue interface{}, evalCtx of.FlattenedContext) of.InterfaceResolutionDetail {
	return *evaluateFirstMatch[of.InterfaceResolutionDetail](ctx, f.providers, flag, defaultValue, evalCtx).result
}

func evaluateFirstMatch[R resultConstraint, DV bool | string | int64 | float64 | interface{}](ctx context.Context, providers []multiprovider.UniqueNameProvider, flag string, defaultValue DV, evalCtx of.FlattenedContext) resultWrapper[R] {
	for _, provider := range providers {
		switch any(defaultValue).(type) {
		case bool:
			r := provider.Provider.BooleanEvaluation(ctx, flag, any(defaultValue).(bool), evalCtx)
			if r.Error() != nil && r.ResolutionDetail().ErrorCode == of.FlagNotFoundCode {
				continue
			} else if r.Error() != nil {
				return buildDefaultResult[R](StrategyFirstMatch, defaultValue, r.Error())
			}
			rp := &r
			rp.FlagMetadata = setFlagMetadata(StrategyFirstMatch, provider.UniqueName, r.FlagMetadata)
			return resultWrapper[R]{result: any(rp).(*R)}
		case string:
			r := provider.Provider.StringEvaluation(ctx, flag, any(defaultValue).(string), evalCtx)
			if r.Error() != nil && r.ResolutionDetail().ErrorCode == of.FlagNotFoundCode {
				continue
			} else if r.Error() != nil {
				return buildDefaultResult[R](StrategyFirstMatch, defaultValue, r.Error())
			}
			rp := &r
			rp.FlagMetadata = setFlagMetadata(StrategyFirstMatch, provider.UniqueName, r.FlagMetadata)
			return resultWrapper[R]{result: any(rp).(*R)}
		case int64:
			r := provider.Provider.IntEvaluation(ctx, flag, any(defaultValue).(int64), evalCtx)
			if r.Error() != nil && r.ResolutionDetail().ErrorCode == of.FlagNotFoundCode {
				continue
			} else if r.Error() != nil {
				return buildDefaultResult[R](StrategyFirstMatch, defaultValue, r.Error())
			}
			rp := &r
			rp.FlagMetadata = setFlagMetadata(StrategyFirstMatch, provider.UniqueName, r.FlagMetadata)
			return resultWrapper[R]{result: any(rp).(*R)}
		case float64:
			r := provider.Provider.FloatEvaluation(ctx, flag, any(defaultValue).(float64), evalCtx)
			if r.Error() != nil && r.ResolutionDetail().ErrorCode == of.FlagNotFoundCode {
				continue
			} else if r.Error() != nil {
				return buildDefaultResult[R](StrategyFirstMatch, defaultValue, r.Error())
			}
			rp := &r
			rp.FlagMetadata = setFlagMetadata(StrategyFirstMatch, provider.UniqueName, r.FlagMetadata)
			return resultWrapper[R]{result: any(rp).(*R)}
		default:
			r := provider.Provider.ObjectEvaluation(ctx, flag, defaultValue, evalCtx)
			if r.Error() != nil && r.ResolutionDetail().ErrorCode == of.FlagNotFoundCode {
				continue
			} else if r.Error() != nil {
				return buildDefaultResult[R](StrategyFirstMatch, defaultValue, r.Error())
			}
			rp := &r
			rp.FlagMetadata = setFlagMetadata(StrategyFirstMatch, provider.UniqueName, r.FlagMetadata)
			return resultWrapper[R]{result: any(rp).(*R)}
		}
	}

	// Build a default result if no matches are found
	return buildDefaultResult[R](StrategyFirstMatch, defaultValue, nil)
}
