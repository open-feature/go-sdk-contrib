package strategies

import (
	"context"
	"errors"
	of "github.com/open-feature/go-sdk/openfeature"
	"sync"
)

type FirstSuccessStrategy struct {
	providers []*NamedProvider
}

var _ Strategy = (*FirstSuccessStrategy)(nil)

// NewFirstSuccessStrategy Creates a new FirstSuccessStrategy instance as a Strategy
func NewFirstSuccessStrategy(providers []*NamedProvider) Strategy {
	return &FirstSuccessStrategy{providers: providers}
}

func (f *FirstSuccessStrategy) Name() EvaluationStrategy {
	return StrategyFirstSuccess
}

func (f *FirstSuccessStrategy) BooleanEvaluation(ctx context.Context, flag string, defaultValue bool, evalCtx of.FlattenedContext) of.BoolResolutionDetail {
	return *evaluateFirstSuccess[of.BoolResolutionDetail](ctx, f.providers, flag, defaultValue, evalCtx).result
}

func (f *FirstSuccessStrategy) StringEvaluation(ctx context.Context, flag string, defaultValue string, evalCtx of.FlattenedContext) of.StringResolutionDetail {
	return *evaluateFirstSuccess[of.StringResolutionDetail](ctx, f.providers, flag, defaultValue, evalCtx).result
}

func (f *FirstSuccessStrategy) FloatEvaluation(ctx context.Context, flag string, defaultValue float64, evalCtx of.FlattenedContext) of.FloatResolutionDetail {
	return *evaluateFirstSuccess[of.FloatResolutionDetail](ctx, f.providers, flag, defaultValue, evalCtx).result
}

func (f *FirstSuccessStrategy) IntEvaluation(ctx context.Context, flag string, defaultValue int64, evalCtx of.FlattenedContext) of.IntResolutionDetail {
	return *evaluateFirstSuccess[of.IntResolutionDetail](ctx, f.providers, flag, defaultValue, evalCtx).result
}

func (f *FirstSuccessStrategy) ObjectEvaluation(ctx context.Context, flag string, defaultValue interface{}, evalCtx of.FlattenedContext) of.InterfaceResolutionDetail {
	return *evaluateFirstSuccess[of.InterfaceResolutionDetail](ctx, f.providers, flag, defaultValue, evalCtx).result
}

func evaluateFirstSuccess[R resultConstraint, DV bool | string | int64 | float64 | interface{}](ctx context.Context, providers []*NamedProvider, flag string, defaultValue DV, evalCtx of.FlattenedContext) resultWrapper[R] {
	var (
		mutex  sync.Mutex
		wg     sync.WaitGroup
		result *resultWrapper[R]
	)
	errChan := make(chan error)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	for _, provider := range providers {
		wg.Add(1)
		go func(p *NamedProvider) {
			defer wg.Done()
			switch any(defaultValue).(type) {
			case bool:
				r := provider.Provider.BooleanEvaluation(ctx, flag, any(defaultValue).(bool), evalCtx)
				if r.Error() == nil {
					mutex.Lock()
					result = &resultWrapper[R]{result: any(r).(*R)}
					cancel()
					mutex.Unlock()
				}
				errChan <- r.Error()
			case string:
				r := provider.Provider.StringEvaluation(ctx, flag, any(defaultValue).(string), evalCtx)
				if r.Error() == nil {
					mutex.Lock()
					result = &resultWrapper[R]{result: any(r).(*R)}
					cancel()
					mutex.Unlock()
				}
				errChan <- r.Error()
			case int64:
				r := provider.Provider.IntEvaluation(ctx, flag, any(defaultValue).(int64), evalCtx)
				if r.Error() == nil {
					mutex.Lock()
					result = &resultWrapper[R]{result: any(r).(*R)}
					cancel()
					mutex.Unlock()
				}
				errChan <- r.Error()
			case float64:
				r := provider.Provider.FloatEvaluation(ctx, flag, any(defaultValue).(float64), evalCtx)
				if r.Error() == nil {
					mutex.Lock()
					result = &resultWrapper[R]{result: any(r).(*R)}
					cancel()
					mutex.Unlock()
				}
				errChan <- r.Error()
			default:
				r := provider.Provider.ObjectEvaluation(ctx, flag, defaultValue, evalCtx)
				if r.Error() == nil {
					mutex.Lock()
					result = &resultWrapper[R]{result: any(r).(*R)}
					cancel()
					mutex.Unlock()
				}
				errChan <- r.Error()
			}

		}(provider)
	}
	go func() {
		wg.Wait()
		close(errChan)
	}()

	errs := make([]error, 0, 1)
	for e := range errChan {
		errs = append(errs, e)
	}

	if result != nil {
		return *result
	}

	err := errors.Join(errs...)
	return buildDefaultResult[R](StrategyFirstSuccess, defaultValue, err)
}
