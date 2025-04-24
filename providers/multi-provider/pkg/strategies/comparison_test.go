package strategies

import (
	"context"
	"github.com/open-feature/go-sdk-contrib/providers/multi-provider/internal/mocks"
	of "github.com/open-feature/go-sdk/openfeature"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"testing"
)

const (
	TestErrorNone     = 0
	TestErrorNotFound = 1
	TestErrorError    = 2
)

func configureComparisonProvider[R bool | int64 | float64 | string | interface{}](provider *mocks.MockFeatureProvider, resultVal R, state bool, error int) {
	var rErr of.ResolutionError
	var variant string
	var reason of.Reason
	switch error {
	case TestErrorError:
		rErr = of.NewGeneralResolutionError("test error")
		reason = of.DisabledReason
	case TestErrorNotFound:
		rErr = of.NewFlagNotFoundResolutionError("not found")
		reason = of.DefaultReason
	}
	if state {
		variant = "on"
	} else {
		variant = "off"
	}
	details := of.ProviderResolutionDetail{
		ResolutionError: rErr,
		Reason:          reason,
		Variant:         variant,
		FlagMetadata:    make(of.FlagMetadata),
	}

	switch any(resultVal).(type) {
	case bool:
		provider.EXPECT().BooleanEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(c context.Context, flag string, defaultVal bool, evalCtx of.FlattenedContext) of.BoolResolutionDetail {
			return of.BoolResolutionDetail{
				Value:                    any(resultVal).(bool),
				ProviderResolutionDetail: details,
			}
		}).MaxTimes(1)
	case string:
		provider.EXPECT().StringEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(c context.Context, flag string, defaultVal string, evalCtx of.FlattenedContext) of.StringResolutionDetail {
			return of.StringResolutionDetail{
				Value:                    any(resultVal).(string),
				ProviderResolutionDetail: details,
			}
		}).MaxTimes(1)
	case int64:
		provider.EXPECT().IntEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(c context.Context, flag string, defaultVal int64, evalCtx of.FlattenedContext) of.IntResolutionDetail {
			return of.IntResolutionDetail{
				Value:                    any(resultVal).(int64),
				ProviderResolutionDetail: details,
			}
		}).MaxTimes(1)
	case float64:
		provider.EXPECT().FloatEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(c context.Context, flag string, defaultVal float64, evalCtx of.FlattenedContext) of.FloatResolutionDetail {
			return of.FloatResolutionDetail{
				Value:                    any(resultVal).(float64),
				ProviderResolutionDetail: details,
			}
		}).MaxTimes(1)
	default:
		provider.EXPECT().ObjectEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(c context.Context, flag string, defaultVal any, evalCtx of.FlattenedContext) of.InterfaceResolutionDetail {
			return of.InterfaceResolutionDetail{
				Value:                    resultVal,
				ProviderResolutionDetail: details,
			}
		}).MaxTimes(1)
	}
}

func Test_ComparisonStrategy_BooleanEvaluation(t *testing.T) {
	t.Run("single success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		provider := mocks.NewMockFeatureProvider(ctrl)
		fallback := mocks.NewMockFeatureProvider(ctrl)
		fallback.EXPECT().BooleanEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
		configureComparisonProvider(provider, true, true, TestErrorNone)

		strategy := NewComparisonStrategy([]*NamedProvider{
			{
				Name:     "test-provider",
				Provider: provider,
			},
		}, fallback)

		result := strategy.BooleanEvaluation(context.Background(), TestFlag, false, of.FlattenedContext{})
		assert.True(t, result.Value)
		assert.Contains(t, result.FlagMetadata, MetadataStrategyUsed)
		assert.Equal(t, StrategyComparison, result.FlagMetadata[MetadataStrategyUsed])
		assert.Contains(t, result.FlagMetadata, MetadataSuccessfulProviderName)
		assert.Equal(t, "test-provider", result.FlagMetadata[MetadataSuccessfulProviderName])
		assert.False(t, result.FlagMetadata[MetadataFallbackUsed].(bool))
	})

	t.Run("two success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		fallback := mocks.NewMockFeatureProvider(ctrl)
		fallback.EXPECT().BooleanEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
		provider1 := mocks.NewMockFeatureProvider(ctrl)
		configureComparisonProvider(provider1, true, true, TestErrorNone)
		provider2 := mocks.NewMockFeatureProvider(ctrl)
		configureComparisonProvider(provider2, true, true, TestErrorNone)

		strategy := NewComparisonStrategy([]*NamedProvider{
			{
				Name:     "test-provider1",
				Provider: provider1,
			},
			{
				Name:     "test-provider2",
				Provider: provider2,
			},
		}, fallback)

		result := strategy.BooleanEvaluation(context.Background(), TestFlag, false, of.FlattenedContext{})
		assert.True(t, result.Value)
		assert.Contains(t, result.FlagMetadata, MetadataStrategyUsed)
		assert.Equal(t, StrategyComparison, result.FlagMetadata[MetadataStrategyUsed])
		assert.Contains(t, result.FlagMetadata, MetadataSuccessfulProviderName+"s")
		assert.Equal(t, "test-provider1, test-provider2", result.FlagMetadata[MetadataSuccessfulProviderName+"s"])
		assert.False(t, result.FlagMetadata[MetadataFallbackUsed].(bool))
	})

	t.Run("multiple success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		fallback := mocks.NewMockFeatureProvider(ctrl)
		fallback.EXPECT().BooleanEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
		provider1 := mocks.NewMockFeatureProvider(ctrl)
		configureComparisonProvider(provider1, true, true, TestErrorNone)
		provider2 := mocks.NewMockFeatureProvider(ctrl)
		configureComparisonProvider(provider2, true, true, TestErrorNone)
		provider3 := mocks.NewMockFeatureProvider(ctrl)
		configureComparisonProvider(provider3, true, true, TestErrorNone)

		strategy := NewComparisonStrategy([]*NamedProvider{
			{
				Name:     "test-provider1",
				Provider: provider1,
			},
			{
				Name:     "test-provider2",
				Provider: provider2,
			},
			{
				Name:     "test-provider3",
				Provider: provider3,
			},
		}, fallback)

		result := strategy.BooleanEvaluation(context.Background(), TestFlag, false, of.FlattenedContext{})
		assert.True(t, result.Value)
		assert.Contains(t, result.FlagMetadata, MetadataStrategyUsed)
		assert.Equal(t, StrategyComparison, result.FlagMetadata[MetadataStrategyUsed])
		assert.Contains(t, result.FlagMetadata, MetadataSuccessfulProviderName+"s")
		assert.Equal(t, "test-provider1, test-provider2, test-provider3", result.FlagMetadata[MetadataSuccessfulProviderName+"s"])
		assert.False(t, result.FlagMetadata[MetadataFallbackUsed].(bool))
	})

	t.Run("multiple not found with single success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		fallback := mocks.NewMockFeatureProvider(ctrl)
		fallback.EXPECT().BooleanEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
		provider1 := mocks.NewMockFeatureProvider(ctrl)
		configureComparisonProvider(provider1, false, true, TestErrorNotFound)
		provider2 := mocks.NewMockFeatureProvider(ctrl)
		configureComparisonProvider(provider2, false, true, TestErrorNotFound)
		provider3 := mocks.NewMockFeatureProvider(ctrl)
		configureComparisonProvider(provider3, true, true, TestErrorNone)

		strategy := NewComparisonStrategy([]*NamedProvider{
			{
				Name:     "test-provider1",
				Provider: provider1,
			},
			{
				Name:     "test-provider2",
				Provider: provider2,
			},
			{
				Name:     "test-provider3",
				Provider: provider3,
			},
		}, fallback)

		result := strategy.BooleanEvaluation(context.Background(), TestFlag, false, of.FlattenedContext{})
		assert.True(t, result.Value)
		assert.Contains(t, result.FlagMetadata, MetadataStrategyUsed)
		assert.Equal(t, StrategyComparison, result.FlagMetadata[MetadataStrategyUsed])
		assert.Contains(t, result.FlagMetadata, MetadataSuccessfulProviderName+"s")
		assert.Equal(t, "test-provider3", result.FlagMetadata[MetadataSuccessfulProviderName+"s"])
		assert.Contains(t, result.FlagMetadata, MetadataFallbackUsed)
		assert.False(t, result.FlagMetadata[MetadataFallbackUsed].(bool))
	})

	t.Run("multiple not found with multiple success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		fallback := mocks.NewMockFeatureProvider(ctrl)
		fallback.EXPECT().BooleanEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
		provider1 := mocks.NewMockFeatureProvider(ctrl)
		configureComparisonProvider(provider1, false, true, TestErrorNotFound)
		provider2 := mocks.NewMockFeatureProvider(ctrl)
		configureComparisonProvider(provider2, false, true, TestErrorNotFound)
		provider3 := mocks.NewMockFeatureProvider(ctrl)
		configureComparisonProvider(provider3, true, true, TestErrorNone)
		provider4 := mocks.NewMockFeatureProvider(ctrl)
		configureComparisonProvider(provider4, true, true, TestErrorNone)

		strategy := NewComparisonStrategy([]*NamedProvider{
			{
				Name:     "test-provider1",
				Provider: provider1,
			},
			{
				Name:     "test-provider2",
				Provider: provider2,
			},
			{
				Name:     "test-provider3",
				Provider: provider3,
			},
			{
				Name:     "test-provider4",
				Provider: provider4,
			},
		}, fallback)

		result := strategy.BooleanEvaluation(context.Background(), TestFlag, false, of.FlattenedContext{})
		assert.True(t, result.Value)
		assert.Contains(t, result.FlagMetadata, MetadataStrategyUsed)
		assert.Equal(t, StrategyComparison, result.FlagMetadata[MetadataStrategyUsed])
		assert.Contains(t, result.FlagMetadata, MetadataSuccessfulProviderName+"s")
		assert.Equal(t, "test-provider3, test-provider4", result.FlagMetadata[MetadataSuccessfulProviderName+"s"])
		assert.Contains(t, result.FlagMetadata, MetadataFallbackUsed)
		assert.False(t, result.FlagMetadata[MetadataFallbackUsed].(bool))
	})

	t.Run("comparison failure uses fallback", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		fallback := mocks.NewMockFeatureProvider(ctrl)
		fallback.EXPECT().BooleanEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(of.BoolResolutionDetail{
			Value: true,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				ResolutionError: of.ResolutionError{},
				Variant:         "on",
				Reason:          "",
				FlagMetadata:    make(of.FlagMetadata),
			},
		})
		provider1 := mocks.NewMockFeatureProvider(ctrl)
		configureComparisonProvider(provider1, false, true, TestErrorNone)
		provider2 := mocks.NewMockFeatureProvider(ctrl)
		configureComparisonProvider(provider2, false, true, TestErrorNone)
		provider3 := mocks.NewMockFeatureProvider(ctrl)
		configureComparisonProvider(provider3, true, true, TestErrorNone)

		strategy := NewComparisonStrategy([]*NamedProvider{
			{
				Name:     "test-provider1",
				Provider: provider1,
			},
			{
				Name:     "test-provider2",
				Provider: provider2,
			},
			{
				Name:     "test-provider3",
				Provider: provider3,
			},
		}, fallback)

		result := strategy.BooleanEvaluation(context.Background(), TestFlag, false, of.FlattenedContext{})
		assert.True(t, result.Value)
		assert.Contains(t, result.FlagMetadata, MetadataStrategyUsed)
		assert.Equal(t, StrategyComparison, result.FlagMetadata[MetadataStrategyUsed])
		assert.NotContains(t, result.FlagMetadata, MetadataSuccessfulProviderName+"s")
		assert.Contains(t, result.FlagMetadata, MetadataSuccessfulProviderName)
		assert.Equal(t, "fallback", result.FlagMetadata[MetadataSuccessfulProviderName])
		assert.True(t, result.FlagMetadata[MetadataFallbackUsed].(bool))
	})

	t.Run("comparison failure with not found", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		fallback := mocks.NewMockFeatureProvider(ctrl)
		fallback.EXPECT().BooleanEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(of.BoolResolutionDetail{
			Value: true,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				ResolutionError: of.ResolutionError{},
				Variant:         "on",
				Reason:          "",
				FlagMetadata:    make(of.FlagMetadata),
			},
		})
		provider1 := mocks.NewMockFeatureProvider(ctrl)
		configureComparisonProvider(provider1, false, true, TestErrorNotFound)
		provider2 := mocks.NewMockFeatureProvider(ctrl)
		configureComparisonProvider(provider2, false, true, TestErrorNotFound)
		provider3 := mocks.NewMockFeatureProvider(ctrl)
		configureComparisonProvider(provider3, true, true, TestErrorNone)
		provider4 := mocks.NewMockFeatureProvider(ctrl)
		configureComparisonProvider(provider4, false, true, TestErrorNone)

		strategy := NewComparisonStrategy([]*NamedProvider{
			{
				Name:     "test-provider1",
				Provider: provider1,
			},
			{
				Name:     "test-provider2",
				Provider: provider2,
			},
			{
				Name:     "test-provider3",
				Provider: provider3,
			},
			{
				Name:     "test-provider4",
				Provider: provider4,
			},
		}, fallback)

		result := strategy.BooleanEvaluation(context.Background(), TestFlag, false, of.FlattenedContext{})
		assert.True(t, result.Value)
		assert.Contains(t, result.FlagMetadata, MetadataStrategyUsed)
		assert.Equal(t, StrategyComparison, result.FlagMetadata[MetadataStrategyUsed])
		assert.NotContains(t, result.FlagMetadata, MetadataSuccessfulProviderName+"s")
		assert.Contains(t, result.FlagMetadata, MetadataSuccessfulProviderName)
		assert.Equal(t, "fallback", result.FlagMetadata[MetadataSuccessfulProviderName])
		assert.Contains(t, result.FlagMetadata, MetadataFallbackUsed)
		assert.True(t, result.FlagMetadata[MetadataFallbackUsed].(bool))
	})

	t.Run("not found all providers", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		fallback := mocks.NewMockFeatureProvider(ctrl)
		fallback.EXPECT().BooleanEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
		provider1 := mocks.NewMockFeatureProvider(ctrl)
		configureComparisonProvider(provider1, false, true, TestErrorNotFound)
		provider2 := mocks.NewMockFeatureProvider(ctrl)
		configureComparisonProvider(provider2, false, true, TestErrorNotFound)

		strategy := NewComparisonStrategy([]*NamedProvider{
			{
				Name:     "test-provider1",
				Provider: provider1,
			},
			{
				Name:     "test-provider2",
				Provider: provider2,
			},
		}, fallback)

		result := strategy.BooleanEvaluation(context.Background(), TestFlag, false, of.FlattenedContext{})
		assert.False(t, result.Value)
		assert.Contains(t, result.FlagMetadata, MetadataStrategyUsed)
		assert.Equal(t, StrategyComparison, result.FlagMetadata[MetadataStrategyUsed])
		assert.NotContains(t, result.FlagMetadata, MetadataSuccessfulProviderName+"s")
		assert.Contains(t, result.FlagMetadata, MetadataSuccessfulProviderName)
		assert.Equal(t, "none", result.FlagMetadata[MetadataSuccessfulProviderName])
		assert.Contains(t, result.FlagMetadata, MetadataFallbackUsed)
		assert.False(t, result.FlagMetadata[MetadataFallbackUsed].(bool))
	})

	t.Run("non FLAG_NOT_FOUND error causes default", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		fallback := mocks.NewMockFeatureProvider(ctrl)
		fallback.EXPECT().BooleanEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
		provider1 := mocks.NewMockFeatureProvider(ctrl)
		configureComparisonProvider(provider1, true, true, TestErrorNone)
		provider2 := mocks.NewMockFeatureProvider(ctrl)
		configureComparisonProvider(provider2, false, true, TestErrorError)

		strategy := NewComparisonStrategy([]*NamedProvider{
			{
				Name:     "test-provider1",
				Provider: provider1,
			},
			{
				Name:     "test-provider2",
				Provider: provider2,
			},
		}, fallback)

		result := strategy.BooleanEvaluation(context.Background(), TestFlag, false, of.FlattenedContext{})
		assert.False(t, result.Value)
		assert.Equal(t, of.DefaultReason, result.Reason)
		assert.Contains(t, result.FlagMetadata, MetadataStrategyUsed)
		assert.Equal(t, StrategyComparison, result.FlagMetadata[MetadataStrategyUsed])
		assert.NotContains(t, result.FlagMetadata, MetadataSuccessfulProviderName+"s")
		assert.Contains(t, result.FlagMetadata, MetadataSuccessfulProviderName)
		assert.Equal(t, "none", result.FlagMetadata[MetadataSuccessfulProviderName])

	})
}

func Test_ComparisonStrategy_StringEvaluation(t *testing.T) {
	successVal := "success"
	defaultVal := "default"
	t.Run("single success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		provider := mocks.NewMockFeatureProvider(ctrl)
		fallback := mocks.NewMockFeatureProvider(ctrl)
		fallback.EXPECT().StringEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
		configureComparisonProvider(provider, successVal, true, TestErrorNone)

		strategy := NewComparisonStrategy([]*NamedProvider{
			{
				Name:     "test-provider",
				Provider: provider,
			},
		}, fallback)

		result := strategy.StringEvaluation(context.Background(), TestFlag, defaultVal, of.FlattenedContext{})
		assert.Equal(t, successVal, result.Value)
		assert.Contains(t, result.FlagMetadata, MetadataStrategyUsed)
		assert.Equal(t, StrategyComparison, result.FlagMetadata[MetadataStrategyUsed])
		assert.Contains(t, result.FlagMetadata, MetadataSuccessfulProviderName)
		assert.Equal(t, "test-provider", result.FlagMetadata[MetadataSuccessfulProviderName])
		assert.False(t, result.FlagMetadata[MetadataFallbackUsed].(bool))
	})

	t.Run("two success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		fallback := mocks.NewMockFeatureProvider(ctrl)
		fallback.EXPECT().StringEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
		provider1 := mocks.NewMockFeatureProvider(ctrl)
		configureComparisonProvider(provider1, successVal, true, TestErrorNone)
		provider2 := mocks.NewMockFeatureProvider(ctrl)
		configureComparisonProvider(provider2, successVal, true, TestErrorNone)

		strategy := NewComparisonStrategy([]*NamedProvider{
			{
				Name:     "test-provider1",
				Provider: provider1,
			},
			{
				Name:     "test-provider2",
				Provider: provider2,
			},
		}, fallback)

		result := strategy.StringEvaluation(context.Background(), TestFlag, defaultVal, of.FlattenedContext{})
		assert.Equal(t, successVal, result.Value)
		assert.Contains(t, result.FlagMetadata, MetadataStrategyUsed)
		assert.Equal(t, StrategyComparison, result.FlagMetadata[MetadataStrategyUsed])
		assert.Contains(t, result.FlagMetadata, MetadataSuccessfulProviderName+"s")
		assert.Equal(t, "test-provider1, test-provider2", result.FlagMetadata[MetadataSuccessfulProviderName+"s"])
		assert.False(t, result.FlagMetadata[MetadataFallbackUsed].(bool))
	})

	t.Run("multiple success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		fallback := mocks.NewMockFeatureProvider(ctrl)
		fallback.EXPECT().StringEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
		provider1 := mocks.NewMockFeatureProvider(ctrl)
		configureComparisonProvider(provider1, successVal, true, TestErrorNone)
		provider2 := mocks.NewMockFeatureProvider(ctrl)
		configureComparisonProvider(provider2, successVal, true, TestErrorNone)
		provider3 := mocks.NewMockFeatureProvider(ctrl)
		configureComparisonProvider(provider3, successVal, true, TestErrorNone)

		strategy := NewComparisonStrategy([]*NamedProvider{
			{
				Name:     "test-provider1",
				Provider: provider1,
			},
			{
				Name:     "test-provider2",
				Provider: provider2,
			},
			{
				Name:     "test-provider3",
				Provider: provider3,
			},
		}, fallback)

		result := strategy.StringEvaluation(context.Background(), TestFlag, defaultVal, of.FlattenedContext{})
		assert.Equal(t, successVal, result.Value)
		assert.Contains(t, result.FlagMetadata, MetadataStrategyUsed)
		assert.Equal(t, StrategyComparison, result.FlagMetadata[MetadataStrategyUsed])
		assert.Contains(t, result.FlagMetadata, MetadataSuccessfulProviderName+"s")
		assert.Equal(t, "test-provider1, test-provider2, test-provider3", result.FlagMetadata[MetadataSuccessfulProviderName+"s"])
		assert.False(t, result.FlagMetadata[MetadataFallbackUsed].(bool))
	})

	t.Run("multiple not found with single success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		fallback := mocks.NewMockFeatureProvider(ctrl)
		fallback.EXPECT().StringEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
		provider1 := mocks.NewMockFeatureProvider(ctrl)
		configureComparisonProvider(provider1, defaultVal, true, TestErrorNotFound)
		provider2 := mocks.NewMockFeatureProvider(ctrl)
		configureComparisonProvider(provider2, defaultVal, true, TestErrorNotFound)
		provider3 := mocks.NewMockFeatureProvider(ctrl)
		configureComparisonProvider(provider3, successVal, true, TestErrorNone)

		strategy := NewComparisonStrategy([]*NamedProvider{
			{
				Name:     "test-provider1",
				Provider: provider1,
			},
			{
				Name:     "test-provider2",
				Provider: provider2,
			},
			{
				Name:     "test-provider3",
				Provider: provider3,
			},
		}, fallback)

		result := strategy.StringEvaluation(context.Background(), TestFlag, defaultVal, of.FlattenedContext{})
		assert.Equal(t, successVal, result.Value)
		assert.Contains(t, result.FlagMetadata, MetadataStrategyUsed)
		assert.Equal(t, StrategyComparison, result.FlagMetadata[MetadataStrategyUsed])
		assert.Contains(t, result.FlagMetadata, MetadataSuccessfulProviderName+"s")
		assert.Equal(t, "test-provider3", result.FlagMetadata[MetadataSuccessfulProviderName+"s"])
		assert.Contains(t, result.FlagMetadata, MetadataFallbackUsed)
		assert.False(t, result.FlagMetadata[MetadataFallbackUsed].(bool))
	})

	t.Run("multiple not found with multiple success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		fallback := mocks.NewMockFeatureProvider(ctrl)
		fallback.EXPECT().StringEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
		provider1 := mocks.NewMockFeatureProvider(ctrl)
		configureComparisonProvider(provider1, defaultVal, true, TestErrorNotFound)
		provider2 := mocks.NewMockFeatureProvider(ctrl)
		configureComparisonProvider(provider2, defaultVal, true, TestErrorNotFound)
		provider3 := mocks.NewMockFeatureProvider(ctrl)
		configureComparisonProvider(provider3, successVal, true, TestErrorNone)
		provider4 := mocks.NewMockFeatureProvider(ctrl)
		configureComparisonProvider(provider4, successVal, true, TestErrorNone)

		strategy := NewComparisonStrategy([]*NamedProvider{
			{
				Name:     "test-provider1",
				Provider: provider1,
			},
			{
				Name:     "test-provider2",
				Provider: provider2,
			},
			{
				Name:     "test-provider3",
				Provider: provider3,
			},
			{
				Name:     "test-provider4",
				Provider: provider4,
			},
		}, fallback)

		result := strategy.StringEvaluation(context.Background(), TestFlag, defaultVal, of.FlattenedContext{})
		assert.Equal(t, successVal, result.Value)
		assert.Contains(t, result.FlagMetadata, MetadataStrategyUsed)
		assert.Equal(t, StrategyComparison, result.FlagMetadata[MetadataStrategyUsed])
		assert.Contains(t, result.FlagMetadata, MetadataSuccessfulProviderName+"s")
		assert.Equal(t, "test-provider3, test-provider4", result.FlagMetadata[MetadataSuccessfulProviderName+"s"])
		assert.Contains(t, result.FlagMetadata, MetadataFallbackUsed)
		assert.False(t, result.FlagMetadata[MetadataFallbackUsed].(bool))
	})

	t.Run("comparison failure uses fallback", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		fallback := mocks.NewMockFeatureProvider(ctrl)
		fallback.EXPECT().StringEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(of.StringResolutionDetail{
			Value: successVal,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				ResolutionError: of.ResolutionError{},
				Variant:         "on",
				Reason:          "",
				FlagMetadata:    make(of.FlagMetadata),
			},
		})
		provider1 := mocks.NewMockFeatureProvider(ctrl)
		configureComparisonProvider(provider1, defaultVal, true, TestErrorNone)
		provider2 := mocks.NewMockFeatureProvider(ctrl)
		configureComparisonProvider(provider2, defaultVal, true, TestErrorNone)
		provider3 := mocks.NewMockFeatureProvider(ctrl)
		configureComparisonProvider(provider3, successVal, true, TestErrorNone)

		strategy := NewComparisonStrategy([]*NamedProvider{
			{
				Name:     "test-provider1",
				Provider: provider1,
			},
			{
				Name:     "test-provider2",
				Provider: provider2,
			},
			{
				Name:     "test-provider3",
				Provider: provider3,
			},
		}, fallback)

		result := strategy.StringEvaluation(context.Background(), TestFlag, defaultVal, of.FlattenedContext{})
		assert.Equal(t, successVal, result.Value)
		assert.Contains(t, result.FlagMetadata, MetadataStrategyUsed)
		assert.Equal(t, StrategyComparison, result.FlagMetadata[MetadataStrategyUsed])
		assert.NotContains(t, result.FlagMetadata, MetadataSuccessfulProviderName+"s")
		assert.Contains(t, result.FlagMetadata, MetadataSuccessfulProviderName)
		assert.Equal(t, "fallback", result.FlagMetadata[MetadataSuccessfulProviderName])
		assert.True(t, result.FlagMetadata[MetadataFallbackUsed].(bool))
	})

	t.Run("comparison failure with not found", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		fallback := mocks.NewMockFeatureProvider(ctrl)
		fallback.EXPECT().StringEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(of.StringResolutionDetail{
			Value: successVal,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				ResolutionError: of.ResolutionError{},
				Variant:         "on",
				Reason:          "",
				FlagMetadata:    make(of.FlagMetadata),
			},
		})
		provider1 := mocks.NewMockFeatureProvider(ctrl)
		configureComparisonProvider(provider1, defaultVal, true, TestErrorNotFound)
		provider2 := mocks.NewMockFeatureProvider(ctrl)
		configureComparisonProvider(provider2, defaultVal, true, TestErrorNotFound)
		provider3 := mocks.NewMockFeatureProvider(ctrl)
		configureComparisonProvider(provider3, successVal, true, TestErrorNone)
		provider4 := mocks.NewMockFeatureProvider(ctrl)
		configureComparisonProvider(provider4, defaultVal, true, TestErrorNone)

		strategy := NewComparisonStrategy([]*NamedProvider{
			{
				Name:     "test-provider1",
				Provider: provider1,
			},
			{
				Name:     "test-provider2",
				Provider: provider2,
			},
			{
				Name:     "test-provider3",
				Provider: provider3,
			},
			{
				Name:     "test-provider4",
				Provider: provider4,
			},
		}, fallback)

		result := strategy.StringEvaluation(context.Background(), TestFlag, defaultVal, of.FlattenedContext{})
		assert.Equal(t, successVal, result.Value)
		assert.Contains(t, result.FlagMetadata, MetadataStrategyUsed)
		assert.Equal(t, StrategyComparison, result.FlagMetadata[MetadataStrategyUsed])
		assert.NotContains(t, result.FlagMetadata, MetadataSuccessfulProviderName+"s")
		assert.Contains(t, result.FlagMetadata, MetadataSuccessfulProviderName)
		assert.Equal(t, "fallback", result.FlagMetadata[MetadataSuccessfulProviderName])
		assert.Contains(t, result.FlagMetadata, MetadataFallbackUsed)
		assert.True(t, result.FlagMetadata[MetadataFallbackUsed].(bool))
	})

	t.Run("not found all providers", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		fallback := mocks.NewMockFeatureProvider(ctrl)
		fallback.EXPECT().FloatEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
		provider1 := mocks.NewMockFeatureProvider(ctrl)
		configureComparisonProvider(provider1, defaultVal, true, TestErrorNotFound)
		provider2 := mocks.NewMockFeatureProvider(ctrl)
		configureComparisonProvider(provider2, defaultVal, true, TestErrorNotFound)

		strategy := NewComparisonStrategy([]*NamedProvider{
			{
				Name:     "test-provider1",
				Provider: provider1,
			},
			{
				Name:     "test-provider2",
				Provider: provider2,
			},
		}, fallback)

		result := strategy.StringEvaluation(context.Background(), TestFlag, defaultVal, of.FlattenedContext{})
		assert.Equal(t, defaultVal, result.Value)
		assert.Contains(t, result.FlagMetadata, MetadataStrategyUsed)
		assert.Equal(t, StrategyComparison, result.FlagMetadata[MetadataStrategyUsed])
		assert.NotContains(t, result.FlagMetadata, MetadataSuccessfulProviderName+"s")
		assert.Contains(t, result.FlagMetadata, MetadataSuccessfulProviderName)
		assert.Equal(t, "none", result.FlagMetadata[MetadataSuccessfulProviderName])
		assert.Contains(t, result.FlagMetadata, MetadataFallbackUsed)
		assert.False(t, result.FlagMetadata[MetadataFallbackUsed].(bool))
	})

	t.Run("non FLAG_NOT_FOUND error causes default", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		fallback := mocks.NewMockFeatureProvider(ctrl)
		fallback.EXPECT().StringEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
		provider1 := mocks.NewMockFeatureProvider(ctrl)
		configureComparisonProvider(provider1, successVal, true, TestErrorNone)
		provider2 := mocks.NewMockFeatureProvider(ctrl)
		configureComparisonProvider(provider2, defaultVal, true, TestErrorError)

		strategy := NewComparisonStrategy([]*NamedProvider{
			{
				Name:     "test-provider1",
				Provider: provider1,
			},
			{
				Name:     "test-provider2",
				Provider: provider2,
			},
		}, fallback)

		result := strategy.StringEvaluation(context.Background(), TestFlag, defaultVal, of.FlattenedContext{})
		assert.Equal(t, defaultVal, result.Value)
		assert.Equal(t, of.DefaultReason, result.Reason)
		assert.Contains(t, result.FlagMetadata, MetadataStrategyUsed)
		assert.Equal(t, StrategyComparison, result.FlagMetadata[MetadataStrategyUsed])
		assert.NotContains(t, result.FlagMetadata, MetadataSuccessfulProviderName+"s")
		assert.Contains(t, result.FlagMetadata, MetadataSuccessfulProviderName)
		assert.Equal(t, "none", result.FlagMetadata[MetadataSuccessfulProviderName])
		assert.False(t, result.FlagMetadata[MetadataFallbackUsed].(bool))
	})
}

func Test_ComparisonStrategy_IntEvaluation(t *testing.T) {
	successVal := int64(1234)
	defaultVal := int64(0)
	t.Run("single success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		provider := mocks.NewMockFeatureProvider(ctrl)
		fallback := mocks.NewMockFeatureProvider(ctrl)
		fallback.EXPECT().IntEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
		configureComparisonProvider(provider, successVal, true, TestErrorNone)

		strategy := NewComparisonStrategy([]*NamedProvider{
			{
				Name:     "test-provider",
				Provider: provider,
			},
		}, fallback)

		result := strategy.IntEvaluation(context.Background(), TestFlag, defaultVal, of.FlattenedContext{})
		assert.Equal(t, successVal, result.Value)
		assert.Contains(t, result.FlagMetadata, MetadataStrategyUsed)
		assert.Equal(t, StrategyComparison, result.FlagMetadata[MetadataStrategyUsed])
		assert.Contains(t, result.FlagMetadata, MetadataSuccessfulProviderName)
		assert.Equal(t, "test-provider", result.FlagMetadata[MetadataSuccessfulProviderName])
		assert.False(t, result.FlagMetadata[MetadataFallbackUsed].(bool))
	})

	t.Run("two success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		fallback := mocks.NewMockFeatureProvider(ctrl)
		fallback.EXPECT().IntEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
		provider1 := mocks.NewMockFeatureProvider(ctrl)
		configureComparisonProvider(provider1, successVal, true, TestErrorNone)
		provider2 := mocks.NewMockFeatureProvider(ctrl)
		configureComparisonProvider(provider2, successVal, true, TestErrorNone)

		strategy := NewComparisonStrategy([]*NamedProvider{
			{
				Name:     "test-provider1",
				Provider: provider1,
			},
			{
				Name:     "test-provider2",
				Provider: provider2,
			},
		}, fallback)

		result := strategy.IntEvaluation(context.Background(), TestFlag, defaultVal, of.FlattenedContext{})
		assert.Equal(t, successVal, result.Value)
		assert.Contains(t, result.FlagMetadata, MetadataStrategyUsed)
		assert.Equal(t, StrategyComparison, result.FlagMetadata[MetadataStrategyUsed])
		assert.Contains(t, result.FlagMetadata, MetadataSuccessfulProviderName+"s")
		assert.Equal(t, "test-provider1, test-provider2", result.FlagMetadata[MetadataSuccessfulProviderName+"s"])
		assert.False(t, result.FlagMetadata[MetadataFallbackUsed].(bool))
	})

	t.Run("multiple success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		fallback := mocks.NewMockFeatureProvider(ctrl)
		fallback.EXPECT().IntEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
		provider1 := mocks.NewMockFeatureProvider(ctrl)
		configureComparisonProvider(provider1, successVal, true, TestErrorNone)
		provider2 := mocks.NewMockFeatureProvider(ctrl)
		configureComparisonProvider(provider2, successVal, true, TestErrorNone)
		provider3 := mocks.NewMockFeatureProvider(ctrl)
		configureComparisonProvider(provider3, successVal, true, TestErrorNone)

		strategy := NewComparisonStrategy([]*NamedProvider{
			{
				Name:     "test-provider1",
				Provider: provider1,
			},
			{
				Name:     "test-provider2",
				Provider: provider2,
			},
			{
				Name:     "test-provider3",
				Provider: provider3,
			},
		}, fallback)

		result := strategy.IntEvaluation(context.Background(), TestFlag, defaultVal, of.FlattenedContext{})
		assert.Equal(t, successVal, result.Value)
		assert.Contains(t, result.FlagMetadata, MetadataStrategyUsed)
		assert.Equal(t, StrategyComparison, result.FlagMetadata[MetadataStrategyUsed])
		assert.Contains(t, result.FlagMetadata, MetadataSuccessfulProviderName+"s")
		assert.Equal(t, "test-provider1, test-provider2, test-provider3", result.FlagMetadata[MetadataSuccessfulProviderName+"s"])
		assert.False(t, result.FlagMetadata[MetadataFallbackUsed].(bool))
	})

	t.Run("multiple not found with single success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		fallback := mocks.NewMockFeatureProvider(ctrl)
		fallback.EXPECT().IntEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
		provider1 := mocks.NewMockFeatureProvider(ctrl)
		configureComparisonProvider(provider1, defaultVal, true, TestErrorNotFound)
		provider2 := mocks.NewMockFeatureProvider(ctrl)
		configureComparisonProvider(provider2, defaultVal, true, TestErrorNotFound)
		provider3 := mocks.NewMockFeatureProvider(ctrl)
		configureComparisonProvider(provider3, successVal, true, TestErrorNone)

		strategy := NewComparisonStrategy([]*NamedProvider{
			{
				Name:     "test-provider1",
				Provider: provider1,
			},
			{
				Name:     "test-provider2",
				Provider: provider2,
			},
			{
				Name:     "test-provider3",
				Provider: provider3,
			},
		}, fallback)

		result := strategy.IntEvaluation(context.Background(), TestFlag, defaultVal, of.FlattenedContext{})
		assert.Equal(t, successVal, result.Value)
		assert.Contains(t, result.FlagMetadata, MetadataStrategyUsed)
		assert.Equal(t, StrategyComparison, result.FlagMetadata[MetadataStrategyUsed])
		assert.Contains(t, result.FlagMetadata, MetadataSuccessfulProviderName+"s")
		assert.Equal(t, "test-provider3", result.FlagMetadata[MetadataSuccessfulProviderName+"s"])
		assert.Contains(t, result.FlagMetadata, MetadataFallbackUsed)
		assert.False(t, result.FlagMetadata[MetadataFallbackUsed].(bool))
	})

	t.Run("multiple not found with multiple success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		fallback := mocks.NewMockFeatureProvider(ctrl)
		fallback.EXPECT().IntEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
		provider1 := mocks.NewMockFeatureProvider(ctrl)
		configureComparisonProvider(provider1, defaultVal, true, TestErrorNotFound)
		provider2 := mocks.NewMockFeatureProvider(ctrl)
		configureComparisonProvider(provider2, defaultVal, true, TestErrorNotFound)
		provider3 := mocks.NewMockFeatureProvider(ctrl)
		configureComparisonProvider(provider3, successVal, true, TestErrorNone)
		provider4 := mocks.NewMockFeatureProvider(ctrl)
		configureComparisonProvider(provider4, successVal, true, TestErrorNone)

		strategy := NewComparisonStrategy([]*NamedProvider{
			{
				Name:     "test-provider1",
				Provider: provider1,
			},
			{
				Name:     "test-provider2",
				Provider: provider2,
			},
			{
				Name:     "test-provider3",
				Provider: provider3,
			},
			{
				Name:     "test-provider4",
				Provider: provider4,
			},
		}, fallback)

		result := strategy.IntEvaluation(context.Background(), TestFlag, defaultVal, of.FlattenedContext{})
		assert.Equal(t, successVal, result.Value)
		assert.Contains(t, result.FlagMetadata, MetadataStrategyUsed)
		assert.Equal(t, StrategyComparison, result.FlagMetadata[MetadataStrategyUsed])
		assert.Contains(t, result.FlagMetadata, MetadataSuccessfulProviderName+"s")
		assert.Equal(t, "test-provider3, test-provider4", result.FlagMetadata[MetadataSuccessfulProviderName+"s"])
		assert.Contains(t, result.FlagMetadata, MetadataFallbackUsed)
		assert.False(t, result.FlagMetadata[MetadataFallbackUsed].(bool))
	})

	t.Run("comparison failure uses fallback", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		fallback := mocks.NewMockFeatureProvider(ctrl)
		fallback.EXPECT().IntEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(of.IntResolutionDetail{
			Value: successVal,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				ResolutionError: of.ResolutionError{},
				Variant:         "on",
				Reason:          "",
				FlagMetadata:    make(of.FlagMetadata),
			},
		})
		provider1 := mocks.NewMockFeatureProvider(ctrl)
		configureComparisonProvider(provider1, defaultVal, true, TestErrorNone)
		provider2 := mocks.NewMockFeatureProvider(ctrl)
		configureComparisonProvider(provider2, defaultVal, true, TestErrorNone)
		provider3 := mocks.NewMockFeatureProvider(ctrl)
		configureComparisonProvider(provider3, successVal, true, TestErrorNone)

		strategy := NewComparisonStrategy([]*NamedProvider{
			{
				Name:     "test-provider1",
				Provider: provider1,
			},
			{
				Name:     "test-provider2",
				Provider: provider2,
			},
			{
				Name:     "test-provider3",
				Provider: provider3,
			},
		}, fallback)

		result := strategy.IntEvaluation(context.Background(), TestFlag, defaultVal, of.FlattenedContext{})
		assert.Equal(t, successVal, result.Value)
		assert.Contains(t, result.FlagMetadata, MetadataStrategyUsed)
		assert.Equal(t, StrategyComparison, result.FlagMetadata[MetadataStrategyUsed])
		assert.NotContains(t, result.FlagMetadata, MetadataSuccessfulProviderName+"s")
		assert.Contains(t, result.FlagMetadata, MetadataSuccessfulProviderName)
		assert.Equal(t, "fallback", result.FlagMetadata[MetadataSuccessfulProviderName])
		assert.True(t, result.FlagMetadata[MetadataFallbackUsed].(bool))
	})

	t.Run("not found all providers", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		fallback := mocks.NewMockFeatureProvider(ctrl)
		fallback.EXPECT().FloatEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
		provider1 := mocks.NewMockFeatureProvider(ctrl)
		configureComparisonProvider(provider1, defaultVal, true, TestErrorNotFound)
		provider2 := mocks.NewMockFeatureProvider(ctrl)
		configureComparisonProvider(provider2, defaultVal, true, TestErrorNotFound)

		strategy := NewComparisonStrategy([]*NamedProvider{
			{
				Name:     "test-provider1",
				Provider: provider1,
			},
			{
				Name:     "test-provider2",
				Provider: provider2,
			},
		}, fallback)

		result := strategy.IntEvaluation(context.Background(), TestFlag, defaultVal, of.FlattenedContext{})
		assert.Equal(t, defaultVal, result.Value)
		assert.Contains(t, result.FlagMetadata, MetadataStrategyUsed)
		assert.Equal(t, StrategyComparison, result.FlagMetadata[MetadataStrategyUsed])
		assert.NotContains(t, result.FlagMetadata, MetadataSuccessfulProviderName+"s")
		assert.Contains(t, result.FlagMetadata, MetadataSuccessfulProviderName)
		assert.Equal(t, "none", result.FlagMetadata[MetadataSuccessfulProviderName])
		assert.Contains(t, result.FlagMetadata, MetadataFallbackUsed)
		assert.False(t, result.FlagMetadata[MetadataFallbackUsed].(bool))
	})

	t.Run("comparison failure with not found", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		fallback := mocks.NewMockFeatureProvider(ctrl)
		fallback.EXPECT().IntEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(of.IntResolutionDetail{
			Value: successVal,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				ResolutionError: of.ResolutionError{},
				Variant:         "on",
				Reason:          "",
				FlagMetadata:    make(of.FlagMetadata),
			},
		})
		provider1 := mocks.NewMockFeatureProvider(ctrl)
		configureComparisonProvider(provider1, defaultVal, true, TestErrorNotFound)
		provider2 := mocks.NewMockFeatureProvider(ctrl)
		configureComparisonProvider(provider2, defaultVal, true, TestErrorNotFound)
		provider3 := mocks.NewMockFeatureProvider(ctrl)
		configureComparisonProvider(provider3, successVal, true, TestErrorNone)
		provider4 := mocks.NewMockFeatureProvider(ctrl)
		configureComparisonProvider(provider4, defaultVal, true, TestErrorNone)

		strategy := NewComparisonStrategy([]*NamedProvider{
			{
				Name:     "test-provider1",
				Provider: provider1,
			},
			{
				Name:     "test-provider2",
				Provider: provider2,
			},
			{
				Name:     "test-provider3",
				Provider: provider3,
			},
			{
				Name:     "test-provider4",
				Provider: provider4,
			},
		}, fallback)

		result := strategy.IntEvaluation(context.Background(), TestFlag, defaultVal, of.FlattenedContext{})
		assert.Equal(t, successVal, result.Value)
		assert.Contains(t, result.FlagMetadata, MetadataStrategyUsed)
		assert.Equal(t, StrategyComparison, result.FlagMetadata[MetadataStrategyUsed])
		assert.NotContains(t, result.FlagMetadata, MetadataSuccessfulProviderName+"s")
		assert.Contains(t, result.FlagMetadata, MetadataSuccessfulProviderName)
		assert.Equal(t, "fallback", result.FlagMetadata[MetadataSuccessfulProviderName])
		assert.Contains(t, result.FlagMetadata, MetadataFallbackUsed)
		assert.True(t, result.FlagMetadata[MetadataFallbackUsed].(bool))
	})

	t.Run("non FLAG_NOT_FOUND error causes default", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		fallback := mocks.NewMockFeatureProvider(ctrl)
		fallback.EXPECT().IntEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
		provider1 := mocks.NewMockFeatureProvider(ctrl)
		configureComparisonProvider(provider1, successVal, true, TestErrorNone)
		provider2 := mocks.NewMockFeatureProvider(ctrl)
		configureComparisonProvider(provider2, defaultVal, true, TestErrorError)

		strategy := NewComparisonStrategy([]*NamedProvider{
			{
				Name:     "test-provider1",
				Provider: provider1,
			},
			{
				Name:     "test-provider2",
				Provider: provider2,
			},
		}, fallback)

		result := strategy.IntEvaluation(context.Background(), TestFlag, defaultVal, of.FlattenedContext{})
		assert.Equal(t, defaultVal, result.Value)
		assert.Equal(t, of.DefaultReason, result.Reason)
		assert.Contains(t, result.FlagMetadata, MetadataStrategyUsed)
		assert.Equal(t, StrategyComparison, result.FlagMetadata[MetadataStrategyUsed])
		assert.NotContains(t, result.FlagMetadata, MetadataSuccessfulProviderName+"s")
		assert.Contains(t, result.FlagMetadata, MetadataSuccessfulProviderName)
		assert.Equal(t, "none", result.FlagMetadata[MetadataSuccessfulProviderName])
		assert.False(t, result.FlagMetadata[MetadataFallbackUsed].(bool))
	})
}

func Test_ComparisonStrategy_FloatEvaluation(t *testing.T) {
	successVal := float64(1234)
	defaultVal := float64(0)
	t.Run("single success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		provider := mocks.NewMockFeatureProvider(ctrl)
		fallback := mocks.NewMockFeatureProvider(ctrl)
		fallback.EXPECT().FloatEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
		configureComparisonProvider(provider, successVal, true, TestErrorNone)

		strategy := NewComparisonStrategy([]*NamedProvider{
			{
				Name:     "test-provider",
				Provider: provider,
			},
		}, fallback)

		result := strategy.FloatEvaluation(context.Background(), TestFlag, defaultVal, of.FlattenedContext{})
		assert.Equal(t, successVal, result.Value)
		assert.Contains(t, result.FlagMetadata, MetadataStrategyUsed)
		assert.Equal(t, StrategyComparison, result.FlagMetadata[MetadataStrategyUsed])
		assert.Contains(t, result.FlagMetadata, MetadataSuccessfulProviderName)
		assert.Equal(t, "test-provider", result.FlagMetadata[MetadataSuccessfulProviderName])
		assert.False(t, result.FlagMetadata[MetadataFallbackUsed].(bool))
	})

	t.Run("two success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		fallback := mocks.NewMockFeatureProvider(ctrl)
		fallback.EXPECT().FloatEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
		provider1 := mocks.NewMockFeatureProvider(ctrl)
		configureComparisonProvider(provider1, successVal, true, TestErrorNone)
		provider2 := mocks.NewMockFeatureProvider(ctrl)
		configureComparisonProvider(provider2, successVal, true, TestErrorNone)

		strategy := NewComparisonStrategy([]*NamedProvider{
			{
				Name:     "test-provider1",
				Provider: provider1,
			},
			{
				Name:     "test-provider2",
				Provider: provider2,
			},
		}, fallback)

		result := strategy.FloatEvaluation(context.Background(), TestFlag, defaultVal, of.FlattenedContext{})
		assert.Equal(t, successVal, result.Value)
		assert.Contains(t, result.FlagMetadata, MetadataStrategyUsed)
		assert.Equal(t, StrategyComparison, result.FlagMetadata[MetadataStrategyUsed])
		assert.Contains(t, result.FlagMetadata, MetadataSuccessfulProviderName+"s")
		assert.Equal(t, "test-provider1, test-provider2", result.FlagMetadata[MetadataSuccessfulProviderName+"s"])
		assert.False(t, result.FlagMetadata[MetadataFallbackUsed].(bool))
	})

	t.Run("multiple success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		fallback := mocks.NewMockFeatureProvider(ctrl)
		fallback.EXPECT().FloatEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
		provider1 := mocks.NewMockFeatureProvider(ctrl)
		configureComparisonProvider(provider1, successVal, true, TestErrorNone)
		provider2 := mocks.NewMockFeatureProvider(ctrl)
		configureComparisonProvider(provider2, successVal, true, TestErrorNone)
		provider3 := mocks.NewMockFeatureProvider(ctrl)
		configureComparisonProvider(provider3, successVal, true, TestErrorNone)

		strategy := NewComparisonStrategy([]*NamedProvider{
			{
				Name:     "test-provider1",
				Provider: provider1,
			},
			{
				Name:     "test-provider2",
				Provider: provider2,
			},
			{
				Name:     "test-provider3",
				Provider: provider3,
			},
		}, fallback)

		result := strategy.FloatEvaluation(context.Background(), TestFlag, defaultVal, of.FlattenedContext{})
		assert.Equal(t, successVal, result.Value)
		assert.Contains(t, result.FlagMetadata, MetadataStrategyUsed)
		assert.Equal(t, StrategyComparison, result.FlagMetadata[MetadataStrategyUsed])
		assert.Contains(t, result.FlagMetadata, MetadataSuccessfulProviderName+"s")
		assert.Equal(t, "test-provider1, test-provider2, test-provider3", result.FlagMetadata[MetadataSuccessfulProviderName+"s"])
		assert.False(t, result.FlagMetadata[MetadataFallbackUsed].(bool))
	})

	t.Run("multiple not found with single success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		fallback := mocks.NewMockFeatureProvider(ctrl)
		fallback.EXPECT().FloatEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
		provider1 := mocks.NewMockFeatureProvider(ctrl)
		configureComparisonProvider(provider1, defaultVal, true, TestErrorNotFound)
		provider2 := mocks.NewMockFeatureProvider(ctrl)
		configureComparisonProvider(provider2, defaultVal, true, TestErrorNotFound)
		provider3 := mocks.NewMockFeatureProvider(ctrl)
		configureComparisonProvider(provider3, successVal, true, TestErrorNone)

		strategy := NewComparisonStrategy([]*NamedProvider{
			{
				Name:     "test-provider1",
				Provider: provider1,
			},
			{
				Name:     "test-provider2",
				Provider: provider2,
			},
			{
				Name:     "test-provider3",
				Provider: provider3,
			},
		}, fallback)

		result := strategy.FloatEvaluation(context.Background(), TestFlag, defaultVal, of.FlattenedContext{})
		assert.Equal(t, successVal, result.Value)
		assert.Contains(t, result.FlagMetadata, MetadataStrategyUsed)
		assert.Equal(t, StrategyComparison, result.FlagMetadata[MetadataStrategyUsed])
		assert.Contains(t, result.FlagMetadata, MetadataSuccessfulProviderName+"s")
		assert.Equal(t, "test-provider3", result.FlagMetadata[MetadataSuccessfulProviderName+"s"])
		assert.Contains(t, result.FlagMetadata, MetadataFallbackUsed)
		assert.False(t, result.FlagMetadata[MetadataFallbackUsed].(bool))
	})

	t.Run("multiple not found with multiple success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		fallback := mocks.NewMockFeatureProvider(ctrl)
		fallback.EXPECT().FloatEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
		provider1 := mocks.NewMockFeatureProvider(ctrl)
		configureComparisonProvider(provider1, defaultVal, true, TestErrorNotFound)
		provider2 := mocks.NewMockFeatureProvider(ctrl)
		configureComparisonProvider(provider2, defaultVal, true, TestErrorNotFound)
		provider3 := mocks.NewMockFeatureProvider(ctrl)
		configureComparisonProvider(provider3, successVal, true, TestErrorNone)
		provider4 := mocks.NewMockFeatureProvider(ctrl)
		configureComparisonProvider(provider4, successVal, true, TestErrorNone)

		strategy := NewComparisonStrategy([]*NamedProvider{
			{
				Name:     "test-provider1",
				Provider: provider1,
			},
			{
				Name:     "test-provider2",
				Provider: provider2,
			},
			{
				Name:     "test-provider3",
				Provider: provider3,
			},
			{
				Name:     "test-provider4",
				Provider: provider4,
			},
		}, fallback)

		result := strategy.FloatEvaluation(context.Background(), TestFlag, defaultVal, of.FlattenedContext{})
		assert.Equal(t, successVal, result.Value)
		assert.Contains(t, result.FlagMetadata, MetadataStrategyUsed)
		assert.Equal(t, StrategyComparison, result.FlagMetadata[MetadataStrategyUsed])
		assert.Contains(t, result.FlagMetadata, MetadataSuccessfulProviderName+"s")
		assert.Equal(t, "test-provider3, test-provider4", result.FlagMetadata[MetadataSuccessfulProviderName+"s"])
		assert.Contains(t, result.FlagMetadata, MetadataFallbackUsed)
		assert.False(t, result.FlagMetadata[MetadataFallbackUsed].(bool))
	})

	t.Run("comparison failure uses fallback", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		fallback := mocks.NewMockFeatureProvider(ctrl)
		fallback.EXPECT().FloatEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(of.FloatResolutionDetail{
			Value: successVal,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				ResolutionError: of.ResolutionError{},
				Variant:         "on",
				Reason:          "",
				FlagMetadata:    make(of.FlagMetadata),
			},
		})
		provider1 := mocks.NewMockFeatureProvider(ctrl)
		configureComparisonProvider(provider1, defaultVal, true, TestErrorNone)
		provider2 := mocks.NewMockFeatureProvider(ctrl)
		configureComparisonProvider(provider2, defaultVal, true, TestErrorNone)
		provider3 := mocks.NewMockFeatureProvider(ctrl)
		configureComparisonProvider(provider3, successVal, true, TestErrorNone)

		strategy := NewComparisonStrategy([]*NamedProvider{
			{
				Name:     "test-provider1",
				Provider: provider1,
			},
			{
				Name:     "test-provider2",
				Provider: provider2,
			},
			{
				Name:     "test-provider3",
				Provider: provider3,
			},
		}, fallback)

		result := strategy.FloatEvaluation(context.Background(), TestFlag, defaultVal, of.FlattenedContext{})
		assert.Equal(t, successVal, result.Value)
		assert.Contains(t, result.FlagMetadata, MetadataStrategyUsed)
		assert.Equal(t, StrategyComparison, result.FlagMetadata[MetadataStrategyUsed])
		assert.NotContains(t, result.FlagMetadata, MetadataSuccessfulProviderName+"s")
		assert.Contains(t, result.FlagMetadata, MetadataSuccessfulProviderName)
		assert.Equal(t, "fallback", result.FlagMetadata[MetadataSuccessfulProviderName])
		assert.True(t, result.FlagMetadata[MetadataFallbackUsed].(bool))
	})

	t.Run("comparison failure with not found", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		fallback := mocks.NewMockFeatureProvider(ctrl)
		fallback.EXPECT().FloatEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(of.FloatResolutionDetail{
			Value: successVal,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				ResolutionError: of.ResolutionError{},
				Variant:         "on",
				Reason:          "",
				FlagMetadata:    make(of.FlagMetadata),
			},
		})
		provider1 := mocks.NewMockFeatureProvider(ctrl)
		configureComparisonProvider(provider1, defaultVal, true, TestErrorNotFound)
		provider2 := mocks.NewMockFeatureProvider(ctrl)
		configureComparisonProvider(provider2, defaultVal, true, TestErrorNotFound)
		provider3 := mocks.NewMockFeatureProvider(ctrl)
		configureComparisonProvider(provider3, successVal, true, TestErrorNone)
		provider4 := mocks.NewMockFeatureProvider(ctrl)
		configureComparisonProvider(provider4, defaultVal, true, TestErrorNone)

		strategy := NewComparisonStrategy([]*NamedProvider{
			{
				Name:     "test-provider1",
				Provider: provider1,
			},
			{
				Name:     "test-provider2",
				Provider: provider2,
			},
			{
				Name:     "test-provider3",
				Provider: provider3,
			},
			{
				Name:     "test-provider4",
				Provider: provider4,
			},
		}, fallback)

		result := strategy.FloatEvaluation(context.Background(), TestFlag, defaultVal, of.FlattenedContext{})
		assert.Equal(t, successVal, result.Value)
		assert.Contains(t, result.FlagMetadata, MetadataStrategyUsed)
		assert.Equal(t, StrategyComparison, result.FlagMetadata[MetadataStrategyUsed])
		assert.NotContains(t, result.FlagMetadata, MetadataSuccessfulProviderName+"s")
		assert.Contains(t, result.FlagMetadata, MetadataSuccessfulProviderName)
		assert.Equal(t, "fallback", result.FlagMetadata[MetadataSuccessfulProviderName])
		assert.Contains(t, result.FlagMetadata, MetadataFallbackUsed)
		assert.True(t, result.FlagMetadata[MetadataFallbackUsed].(bool))
	})

	t.Run("not found all providers", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		fallback := mocks.NewMockFeatureProvider(ctrl)
		fallback.EXPECT().FloatEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
		provider1 := mocks.NewMockFeatureProvider(ctrl)
		configureComparisonProvider(provider1, defaultVal, true, TestErrorNotFound)
		provider2 := mocks.NewMockFeatureProvider(ctrl)
		configureComparisonProvider(provider2, defaultVal, true, TestErrorNotFound)

		strategy := NewComparisonStrategy([]*NamedProvider{
			{
				Name:     "test-provider1",
				Provider: provider1,
			},
			{
				Name:     "test-provider2",
				Provider: provider2,
			},
		}, fallback)

		result := strategy.FloatEvaluation(context.Background(), TestFlag, defaultVal, of.FlattenedContext{})
		assert.Equal(t, defaultVal, result.Value)
		assert.Contains(t, result.FlagMetadata, MetadataStrategyUsed)
		assert.Equal(t, StrategyComparison, result.FlagMetadata[MetadataStrategyUsed])
		assert.NotContains(t, result.FlagMetadata, MetadataSuccessfulProviderName+"s")
		assert.Contains(t, result.FlagMetadata, MetadataSuccessfulProviderName)
		assert.Equal(t, "none", result.FlagMetadata[MetadataSuccessfulProviderName])
		assert.Contains(t, result.FlagMetadata, MetadataFallbackUsed)
		assert.False(t, result.FlagMetadata[MetadataFallbackUsed].(bool))
	})

	t.Run("non FLAG_NOT_FOUND error causes default", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		fallback := mocks.NewMockFeatureProvider(ctrl)
		fallback.EXPECT().FloatEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
		provider1 := mocks.NewMockFeatureProvider(ctrl)
		configureComparisonProvider(provider1, successVal, true, TestErrorNone)
		provider2 := mocks.NewMockFeatureProvider(ctrl)
		configureComparisonProvider(provider2, defaultVal, true, TestErrorError)

		strategy := NewComparisonStrategy([]*NamedProvider{
			{
				Name:     "test-provider1",
				Provider: provider1,
			},
			{
				Name:     "test-provider2",
				Provider: provider2,
			},
		}, fallback)

		result := strategy.FloatEvaluation(context.Background(), TestFlag, defaultVal, of.FlattenedContext{})
		assert.Equal(t, defaultVal, result.Value)
		assert.Equal(t, of.DefaultReason, result.Reason)
		assert.Contains(t, result.FlagMetadata, MetadataStrategyUsed)
		assert.Equal(t, StrategyComparison, result.FlagMetadata[MetadataStrategyUsed])
		assert.NotContains(t, result.FlagMetadata, MetadataSuccessfulProviderName+"s")
		assert.Contains(t, result.FlagMetadata, MetadataSuccessfulProviderName)
		assert.Equal(t, "none", result.FlagMetadata[MetadataSuccessfulProviderName])
		assert.False(t, result.FlagMetadata[MetadataFallbackUsed].(bool))
	})
}

func Test_ComparisonStrategy_ObjectEvaluation_AlwaysReturnsDefault(t *testing.T) {
	successVal := struct{ Name string }{Name: "test"}
	defaultVal := struct{}{}
	ctrl := gomock.NewController(t)
	fallback := mocks.NewMockFeatureProvider(ctrl)
	fallback.EXPECT().FloatEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	provider1 := mocks.NewMockFeatureProvider(ctrl)
	configureComparisonProvider(provider1, successVal, true, TestErrorNone)
	provider2 := mocks.NewMockFeatureProvider(ctrl)
	configureComparisonProvider(provider2, defaultVal, true, TestErrorError)

	strategy := NewComparisonStrategy([]*NamedProvider{
		{
			Name:     "test-provider1",
			Provider: provider1,
		},
		{
			Name:     "test-provider2",
			Provider: provider2,
		},
	}, fallback)

	result := strategy.ObjectEvaluation(context.Background(), TestFlag, defaultVal, of.FlattenedContext{})
	assert.Equal(t, defaultVal, result.Value)
	assert.Equal(t, of.DefaultReason, result.Reason)
	assert.Equal(t, of.NewGeneralResolutionError(ErrAggregationNotAllowedText), result.ResolutionError)
	assert.Contains(t, result.FlagMetadata, MetadataStrategyUsed)
	assert.Equal(t, StrategyComparison, result.FlagMetadata[MetadataStrategyUsed])
	assert.NotContains(t, result.FlagMetadata, MetadataSuccessfulProviderName+"s")
	assert.Contains(t, result.FlagMetadata, MetadataSuccessfulProviderName)
	assert.Equal(t, "none", result.FlagMetadata[MetadataSuccessfulProviderName])
	assert.False(t, result.FlagMetadata[MetadataFallbackUsed].(bool))
}
