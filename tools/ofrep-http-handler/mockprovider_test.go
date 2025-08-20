package ofrephandler_test

import (
	"context"
	"fmt"

	"github.com/open-feature/go-sdk/openfeature"
)

func newMockProvider(flags map[string]mockValue) *mockProvider {
	return &mockProvider{flags}
}

type mockValue struct {
	value      any
	requireCtx map[string]any
}

type mockProvider struct {
	flags map[string]mockValue
}

func (c *mockProvider) BooleanEvaluation(ctx context.Context, flag string, defaultValue bool, flatCtx openfeature.FlattenedContext) openfeature.BoolResolutionDetail {
	val, detail := evaluate(c.flags, flatCtx, flag, defaultValue)
	return openfeature.BoolResolutionDetail{Value: val, ProviderResolutionDetail: detail}
}

func (c *mockProvider) FloatEvaluation(ctx context.Context, flag string, defaultValue float64, flatCtx openfeature.FlattenedContext) openfeature.FloatResolutionDetail {
	val, detail := evaluate(c.flags, flatCtx, flag, defaultValue)
	return openfeature.FloatResolutionDetail{Value: val, ProviderResolutionDetail: detail}
}

func (c *mockProvider) IntEvaluation(ctx context.Context, flag string, defaultValue int64, flatCtx openfeature.FlattenedContext) openfeature.IntResolutionDetail {
	val, detail := evaluate(c.flags, flatCtx, flag, defaultValue)
	return openfeature.IntResolutionDetail{Value: val, ProviderResolutionDetail: detail}
}

func (c *mockProvider) ObjectEvaluation(ctx context.Context, flag string, defaultValue any, flatCtx openfeature.FlattenedContext) openfeature.InterfaceResolutionDetail {
	val, detail := evaluate(c.flags, flatCtx, flag, defaultValue)
	return openfeature.InterfaceResolutionDetail{Value: val, ProviderResolutionDetail: detail}
}

func (c *mockProvider) StringEvaluation(ctx context.Context, flag string, defaultValue string, flatCtx openfeature.FlattenedContext) openfeature.StringResolutionDetail {
	val, detail := evaluate(c.flags, flatCtx, flag, defaultValue)
	return openfeature.StringResolutionDetail{Value: val, ProviderResolutionDetail: detail}
}

func (c *mockProvider) Hooks() []openfeature.Hook {
	return nil
}

func (c *mockProvider) Metadata() openfeature.Metadata {
	return openfeature.NamedProviderMetadata("mock-provider")
}

func evaluate[T any](flags map[string]mockValue, flatCtx openfeature.FlattenedContext, flag string, defaultValue T) (T, openfeature.ProviderResolutionDetail) {
	mockVal, ok := flags[flag]
	if !ok {
		return defaultValue, openfeature.ProviderResolutionDetail{
			Reason: openfeature.DefaultReason,
		}
	}

	if mockVal.requireCtx != nil {
		for k, v := range mockVal.requireCtx {
			if ctxVal, exists := flatCtx[k]; !exists || ctxVal != v {
				return defaultValue, openfeature.ProviderResolutionDetail{
					Reason: openfeature.DefaultReason,
				}
			}
		}
	}

	parsedVal, ok := mockVal.value.(T)
	if !ok {
		return defaultValue, openfeature.ProviderResolutionDetail{
			Reason:          openfeature.ErrorReason,
			ResolutionError: openfeature.NewTypeMismatchResolutionError(fmt.Sprintf("expected type %T, got %T", defaultValue, mockVal)),
		}
	}

	return parsedVal, openfeature.ProviderResolutionDetail{
		Reason: openfeature.CachedReason,
	}
}
