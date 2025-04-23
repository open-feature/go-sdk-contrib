package strategies

import (
	"context"
	"github.com/open-feature/go-sdk-contrib/providers/multi-provider/internal/mocks"
	of "github.com/open-feature/go-sdk/openfeature"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"testing"
	"time"
)

const TestFlag = "test-flag"

func configureFlags[R bool | int64 | float64 | string | interface{}](provider *mocks.MockFeatureProvider, resultVal R, state bool, error bool, delay time.Duration) {
	var rErr of.ResolutionError
	var variant string
	var reason of.Reason
	if error {
		rErr = of.NewGeneralResolutionError("test error")
		reason = of.DisabledReason
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
			time.Sleep(delay)
			return of.BoolResolutionDetail{
				Value:                    any(resultVal).(bool),
				ProviderResolutionDetail: details,
			}
		}).MaxTimes(1)
	case string:
		provider.EXPECT().StringEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(c context.Context, flag string, defaultVal string, evalCtx of.FlattenedContext) of.StringResolutionDetail {
			time.Sleep(delay)
			return of.StringResolutionDetail{
				Value:                    any(resultVal).(string),
				ProviderResolutionDetail: details,
			}
		}).MaxTimes(1)
	case int64:
		provider.EXPECT().IntEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(c context.Context, flag string, defaultVal int64, evalCtx of.FlattenedContext) of.IntResolutionDetail {
			time.Sleep(delay)
			return of.IntResolutionDetail{
				Value:                    any(resultVal).(int64),
				ProviderResolutionDetail: details,
			}
		}).MaxTimes(1)
	case float64:
		provider.EXPECT().FloatEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(c context.Context, flag string, defaultVal float64, evalCtx of.FlattenedContext) of.FloatResolutionDetail {
			time.Sleep(delay)
			return of.FloatResolutionDetail{
				Value:                    any(resultVal).(float64),
				ProviderResolutionDetail: details,
			}
		}).MaxTimes(1)
	default:
		provider.EXPECT().ObjectEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(c context.Context, flag string, defaultVal any, evalCtx of.FlattenedContext) of.InterfaceResolutionDetail {
			time.Sleep(delay)
			return of.InterfaceResolutionDetail{
				Value:                    resultVal,
				ProviderResolutionDetail: details,
			}
		}).MaxTimes(1)
	}
}

func Test_FirstSuccessStrategy_BooleanEvaluation(t *testing.T) {
	t.Run("single success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		provider := mocks.NewMockFeatureProvider(ctrl)
		configureFlags(provider, true, true, false, 0*time.Millisecond)

		strategy := NewFirstSuccessStrategy([]*NamedProvider{
			{
				Name:     "test-provider",
				Provider: provider,
			},
		}, 2*time.Second)
		result := strategy.BooleanEvaluation(context.Background(), TestFlag, false, of.FlattenedContext{})
		assert.True(t, result.Value)
		assert.Contains(t, result.FlagMetadata, MetadataStrategyUsed)
		assert.Equal(t, StrategyFirstSuccess, result.FlagMetadata[MetadataStrategyUsed])
		assert.Contains(t, result.FlagMetadata, MetadataSuccessfulProviderName)
		assert.Equal(t, "test-provider", result.FlagMetadata[MetadataSuccessfulProviderName])
	})

	t.Run("first success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		provider1 := mocks.NewMockFeatureProvider(ctrl)
		configureFlags(provider1, true, true, false, 5*time.Millisecond)
		provider2 := mocks.NewMockFeatureProvider(ctrl)
		configureFlags(provider2, false, false, true, 50*time.Millisecond)

		strategy := NewFirstSuccessStrategy([]*NamedProvider{
			{
				Name:     "success-provider",
				Provider: provider1,
			},
			{
				Name:     "failure-provider",
				Provider: provider2,
			},
		}, 2*time.Second)

		result := strategy.BooleanEvaluation(context.Background(), TestFlag, false, of.FlattenedContext{})
		assert.True(t, result.Value)
		assert.Equal(t, StrategyFirstSuccess, result.FlagMetadata[MetadataStrategyUsed])
		assert.Contains(t, result.FlagMetadata, MetadataSuccessfulProviderName)
		assert.Equal(t, "success-provider", result.FlagMetadata[MetadataSuccessfulProviderName])
	})

	t.Run("second success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		provider1 := mocks.NewMockFeatureProvider(ctrl)
		configureFlags(provider1, true, true, false, 500*time.Millisecond)
		provider2 := mocks.NewMockFeatureProvider(ctrl)
		configureFlags(provider2, false, false, true, 5*time.Millisecond)

		strategy := NewFirstSuccessStrategy([]*NamedProvider{
			{
				Name:     "success-provider",
				Provider: provider1,
			},
			{
				Name:     "failure-provider",
				Provider: provider2,
			},
		}, 2*time.Second)

		result := strategy.BooleanEvaluation(context.Background(), TestFlag, false, of.FlattenedContext{})
		assert.True(t, result.Value)
		assert.Equal(t, StrategyFirstSuccess, result.FlagMetadata[MetadataStrategyUsed])
		assert.Contains(t, result.FlagMetadata, MetadataSuccessfulProviderName)
		assert.Equal(t, "success-provider", result.FlagMetadata[MetadataSuccessfulProviderName])
	})

	t.Run("all errors", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		provider1 := mocks.NewMockFeatureProvider(ctrl)
		configureFlags(provider1, false, false, true, 50*time.Millisecond)
		provider2 := mocks.NewMockFeatureProvider(ctrl)
		configureFlags(provider2, false, false, true, 40*time.Millisecond)
		provider3 := mocks.NewMockFeatureProvider(ctrl)
		configureFlags(provider3, false, false, true, 30*time.Millisecond)

		strategy := NewFirstSuccessStrategy([]*NamedProvider{
			{
				Name:     "provider1",
				Provider: provider1,
			},
			{
				Name:     "provider2",
				Provider: provider2,
			},
			{
				Name:     "provider3",
				Provider: provider3,
			},
		}, 2*time.Second)

		result := strategy.BooleanEvaluation(context.Background(), TestFlag, false, of.FlattenedContext{})
		assert.False(t, result.Value)
		assert.Equal(t, StrategyFirstSuccess, result.FlagMetadata[MetadataStrategyUsed])
		assert.Contains(t, result.FlagMetadata, MetadataSuccessfulProviderName)
		assert.Equal(t, "none", result.FlagMetadata[MetadataSuccessfulProviderName])
		assert.Equal(t, of.DefaultReason, result.Reason)
	})
}

func Test_FirstSuccessStrategy_StringEvaluation(t *testing.T) {
	successVal := "success"
	defaultVal := "default"
	t.Run("single success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		provider := mocks.NewMockFeatureProvider(ctrl)
		configureFlags(provider, successVal, true, false, 0*time.Millisecond)

		strategy := NewFirstSuccessStrategy([]*NamedProvider{
			{
				Name:     "test-provider",
				Provider: provider,
			},
		}, 2*time.Second)
		result := strategy.StringEvaluation(context.Background(), TestFlag, defaultVal, of.FlattenedContext{})
		assert.Equal(t, successVal, result.Value)
		assert.Contains(t, result.FlagMetadata, MetadataStrategyUsed)
		assert.Equal(t, StrategyFirstSuccess, result.FlagMetadata[MetadataStrategyUsed])
		assert.Contains(t, result.FlagMetadata, MetadataSuccessfulProviderName)
		assert.Equal(t, "test-provider", result.FlagMetadata[MetadataSuccessfulProviderName])
	})

	t.Run("first success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		provider1 := mocks.NewMockFeatureProvider(ctrl)
		configureFlags(provider1, successVal, true, false, 5*time.Millisecond)
		provider2 := mocks.NewMockFeatureProvider(ctrl)
		configureFlags(provider2, defaultVal, false, true, 50*time.Millisecond)

		strategy := NewFirstSuccessStrategy([]*NamedProvider{
			{
				Name:     "success-provider",
				Provider: provider1,
			},
			{
				Name:     "failure-provider",
				Provider: provider2,
			},
		}, 2*time.Second)

		result := strategy.StringEvaluation(context.Background(), TestFlag, defaultVal, of.FlattenedContext{})
		assert.Equal(t, successVal, result.Value)
		assert.Equal(t, StrategyFirstSuccess, result.FlagMetadata[MetadataStrategyUsed])
		assert.Contains(t, result.FlagMetadata, MetadataSuccessfulProviderName)
		assert.Equal(t, "success-provider", result.FlagMetadata[MetadataSuccessfulProviderName])
	})

	t.Run("second success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		provider1 := mocks.NewMockFeatureProvider(ctrl)
		configureFlags(provider1, successVal, true, false, 500*time.Millisecond)
		provider2 := mocks.NewMockFeatureProvider(ctrl)
		configureFlags(provider2, defaultVal, false, true, 5*time.Millisecond)

		strategy := NewFirstSuccessStrategy([]*NamedProvider{
			{
				Name:     "success-provider",
				Provider: provider1,
			},
			{
				Name:     "failure-provider",
				Provider: provider2,
			},
		}, 2*time.Second)

		result := strategy.StringEvaluation(context.Background(), TestFlag, defaultVal, of.FlattenedContext{})
		assert.Equal(t, successVal, result.Value)
		assert.Equal(t, StrategyFirstSuccess, result.FlagMetadata[MetadataStrategyUsed])
		assert.Contains(t, result.FlagMetadata, MetadataSuccessfulProviderName)
		assert.Equal(t, "success-provider", result.FlagMetadata[MetadataSuccessfulProviderName])
	})

	t.Run("all errors", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		provider1 := mocks.NewMockFeatureProvider(ctrl)
		configureFlags(provider1, defaultVal, false, true, 50*time.Millisecond)
		provider2 := mocks.NewMockFeatureProvider(ctrl)
		configureFlags(provider2, defaultVal, false, true, 40*time.Millisecond)
		provider3 := mocks.NewMockFeatureProvider(ctrl)
		configureFlags(provider3, defaultVal, false, true, 30*time.Millisecond)

		strategy := NewFirstSuccessStrategy([]*NamedProvider{
			{
				Name:     "provider1",
				Provider: provider1,
			},
			{
				Name:     "provider2",
				Provider: provider2,
			},
			{
				Name:     "provider3",
				Provider: provider3,
			},
		}, 2*time.Second)

		result := strategy.StringEvaluation(context.Background(), TestFlag, defaultVal, of.FlattenedContext{})
		assert.Equal(t, defaultVal, result.Value)
		assert.Equal(t, StrategyFirstSuccess, result.FlagMetadata[MetadataStrategyUsed])
		assert.Contains(t, result.FlagMetadata, MetadataSuccessfulProviderName)
		assert.Equal(t, "none", result.FlagMetadata[MetadataSuccessfulProviderName])
		assert.Equal(t, of.DefaultReason, result.Reason)
	})
}

func Test_FirstSuccessStrategy_IntEvaluation(t *testing.T) {
	successVal := int64(150)
	defaultVal := int64(0)
	t.Run("single success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		provider := mocks.NewMockFeatureProvider(ctrl)
		configureFlags(provider, successVal, true, false, 0*time.Millisecond)

		strategy := NewFirstSuccessStrategy([]*NamedProvider{
			{
				Name:     "test-provider",
				Provider: provider,
			},
		}, 2*time.Second)
		result := strategy.IntEvaluation(context.Background(), TestFlag, defaultVal, of.FlattenedContext{})
		assert.Equal(t, successVal, result.Value)
		assert.Contains(t, result.FlagMetadata, MetadataStrategyUsed)
		assert.Equal(t, StrategyFirstSuccess, result.FlagMetadata[MetadataStrategyUsed])
		assert.Contains(t, result.FlagMetadata, MetadataSuccessfulProviderName)
		assert.Equal(t, "test-provider", result.FlagMetadata[MetadataSuccessfulProviderName])
	})

	t.Run("first success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		provider1 := mocks.NewMockFeatureProvider(ctrl)
		configureFlags(provider1, successVal, true, false, 5*time.Millisecond)
		provider2 := mocks.NewMockFeatureProvider(ctrl)
		configureFlags(provider2, defaultVal, false, true, 50*time.Millisecond)

		strategy := NewFirstSuccessStrategy([]*NamedProvider{
			{
				Name:     "success-provider",
				Provider: provider1,
			},
			{
				Name:     "failure-provider",
				Provider: provider2,
			},
		}, 2*time.Second)

		result := strategy.IntEvaluation(context.Background(), TestFlag, defaultVal, of.FlattenedContext{})
		assert.Equal(t, successVal, result.Value)
		assert.Equal(t, StrategyFirstSuccess, result.FlagMetadata[MetadataStrategyUsed])
		assert.Contains(t, result.FlagMetadata, MetadataSuccessfulProviderName)
		assert.Equal(t, "success-provider", result.FlagMetadata[MetadataSuccessfulProviderName])
	})

	t.Run("second success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		provider1 := mocks.NewMockFeatureProvider(ctrl)
		configureFlags(provider1, successVal, true, false, 500*time.Millisecond)
		provider2 := mocks.NewMockFeatureProvider(ctrl)
		configureFlags(provider2, defaultVal, false, true, 5*time.Millisecond)

		strategy := NewFirstSuccessStrategy([]*NamedProvider{
			{
				Name:     "success-provider",
				Provider: provider1,
			},
			{
				Name:     "failure-provider",
				Provider: provider2,
			},
		}, 2*time.Second)

		result := strategy.IntEvaluation(context.Background(), TestFlag, defaultVal, of.FlattenedContext{})
		assert.Equal(t, successVal, result.Value)
		assert.Equal(t, StrategyFirstSuccess, result.FlagMetadata[MetadataStrategyUsed])
		assert.Contains(t, result.FlagMetadata, MetadataSuccessfulProviderName)
		assert.Equal(t, "success-provider", result.FlagMetadata[MetadataSuccessfulProviderName])
	})

	t.Run("all errors", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		provider1 := mocks.NewMockFeatureProvider(ctrl)
		configureFlags(provider1, defaultVal, false, true, 50*time.Millisecond)
		provider2 := mocks.NewMockFeatureProvider(ctrl)
		configureFlags(provider2, defaultVal, false, true, 40*time.Millisecond)
		provider3 := mocks.NewMockFeatureProvider(ctrl)
		configureFlags(provider3, defaultVal, false, true, 30*time.Millisecond)

		strategy := NewFirstSuccessStrategy([]*NamedProvider{
			{
				Name:     "provider1",
				Provider: provider1,
			},
			{
				Name:     "provider2",
				Provider: provider2,
			},
			{
				Name:     "provider3",
				Provider: provider3,
			},
		}, 2*time.Second)

		result := strategy.IntEvaluation(context.Background(), TestFlag, defaultVal, of.FlattenedContext{})
		assert.Equal(t, defaultVal, result.Value)
		assert.Equal(t, StrategyFirstSuccess, result.FlagMetadata[MetadataStrategyUsed])
		assert.Contains(t, result.FlagMetadata, MetadataSuccessfulProviderName)
		assert.Equal(t, "none", result.FlagMetadata[MetadataSuccessfulProviderName])
		assert.Equal(t, of.DefaultReason, result.Reason)
	})
}

func Test_FirstSuccessStrategy_FloatEvaluation(t *testing.T) {
	successVal := float64(15.5)
	defaultVal := float64(0)
	t.Run("single success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		provider := mocks.NewMockFeatureProvider(ctrl)
		configureFlags(provider, successVal, true, false, 0*time.Millisecond)

		strategy := NewFirstSuccessStrategy([]*NamedProvider{
			{
				Name:     "test-provider",
				Provider: provider,
			},
		}, 2*time.Second)
		result := strategy.FloatEvaluation(context.Background(), TestFlag, defaultVal, of.FlattenedContext{})
		assert.Equal(t, successVal, result.Value)
		assert.Contains(t, result.FlagMetadata, MetadataStrategyUsed)
		assert.Equal(t, StrategyFirstSuccess, result.FlagMetadata[MetadataStrategyUsed])
		assert.Contains(t, result.FlagMetadata, MetadataSuccessfulProviderName)
		assert.Equal(t, "test-provider", result.FlagMetadata[MetadataSuccessfulProviderName])
	})

	t.Run("first success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		provider1 := mocks.NewMockFeatureProvider(ctrl)
		configureFlags(provider1, successVal, true, false, 5*time.Millisecond)
		provider2 := mocks.NewMockFeatureProvider(ctrl)
		configureFlags(provider2, defaultVal, false, true, 50*time.Millisecond)

		strategy := NewFirstSuccessStrategy([]*NamedProvider{
			{
				Name:     "success-provider",
				Provider: provider1,
			},
			{
				Name:     "failure-provider",
				Provider: provider2,
			},
		}, 2*time.Second)

		result := strategy.FloatEvaluation(context.Background(), TestFlag, defaultVal, of.FlattenedContext{})
		assert.Equal(t, successVal, result.Value)
		assert.Equal(t, StrategyFirstSuccess, result.FlagMetadata[MetadataStrategyUsed])
		assert.Contains(t, result.FlagMetadata, MetadataSuccessfulProviderName)
		assert.Equal(t, "success-provider", result.FlagMetadata[MetadataSuccessfulProviderName])
	})

	t.Run("second success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		provider1 := mocks.NewMockFeatureProvider(ctrl)
		configureFlags(provider1, successVal, true, false, 500*time.Millisecond)
		provider2 := mocks.NewMockFeatureProvider(ctrl)
		configureFlags(provider2, defaultVal, false, true, 5*time.Millisecond)

		strategy := NewFirstSuccessStrategy([]*NamedProvider{
			{
				Name:     "success-provider",
				Provider: provider1,
			},
			{
				Name:     "failure-provider",
				Provider: provider2,
			},
		}, 2*time.Second)

		result := strategy.FloatEvaluation(context.Background(), TestFlag, defaultVal, of.FlattenedContext{})
		assert.Equal(t, successVal, result.Value)
		assert.Equal(t, StrategyFirstSuccess, result.FlagMetadata[MetadataStrategyUsed])
		assert.Contains(t, result.FlagMetadata, MetadataSuccessfulProviderName)
		assert.Equal(t, "success-provider", result.FlagMetadata[MetadataSuccessfulProviderName])
	})

	t.Run("all errors", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		provider1 := mocks.NewMockFeatureProvider(ctrl)
		configureFlags(provider1, defaultVal, false, true, 50*time.Millisecond)
		provider2 := mocks.NewMockFeatureProvider(ctrl)
		configureFlags(provider2, defaultVal, false, true, 40*time.Millisecond)
		provider3 := mocks.NewMockFeatureProvider(ctrl)
		configureFlags(provider3, defaultVal, false, true, 30*time.Millisecond)

		strategy := NewFirstSuccessStrategy([]*NamedProvider{
			{
				Name:     "provider1",
				Provider: provider1,
			},
			{
				Name:     "provider2",
				Provider: provider2,
			},
			{
				Name:     "provider3",
				Provider: provider3,
			},
		}, 2*time.Second)

		result := strategy.FloatEvaluation(context.Background(), TestFlag, defaultVal, of.FlattenedContext{})
		assert.Equal(t, defaultVal, result.Value)
		assert.Equal(t, StrategyFirstSuccess, result.FlagMetadata[MetadataStrategyUsed])
		assert.Contains(t, result.FlagMetadata, MetadataSuccessfulProviderName)
		assert.Equal(t, "none", result.FlagMetadata[MetadataSuccessfulProviderName])
		assert.Equal(t, of.DefaultReason, result.Reason)
	})
}

func Test_FirstSuccessStrategy_ObjectEvaluation(t *testing.T) {
	successVal := struct{ Field string }{Field: "test"}
	defaultVal := struct{}{}
	t.Run("single success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		provider := mocks.NewMockFeatureProvider(ctrl)
		configureFlags(provider, successVal, true, false, 0*time.Millisecond)

		strategy := NewFirstSuccessStrategy([]*NamedProvider{
			{
				Name:     "test-provider",
				Provider: provider,
			},
		}, 2*time.Second)
		result := strategy.ObjectEvaluation(context.Background(), TestFlag, defaultVal, of.FlattenedContext{})
		assert.Equal(t, successVal, result.Value)
		assert.Contains(t, result.FlagMetadata, MetadataStrategyUsed)
		assert.Equal(t, StrategyFirstSuccess, result.FlagMetadata[MetadataStrategyUsed])
		assert.Contains(t, result.FlagMetadata, MetadataSuccessfulProviderName)
		assert.Equal(t, "test-provider", result.FlagMetadata[MetadataSuccessfulProviderName])
	})

	t.Run("first success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		provider1 := mocks.NewMockFeatureProvider(ctrl)
		configureFlags(provider1, successVal, true, false, 5*time.Millisecond)
		provider2 := mocks.NewMockFeatureProvider(ctrl)
		configureFlags(provider2, defaultVal, false, true, 50*time.Millisecond)

		strategy := NewFirstSuccessStrategy([]*NamedProvider{
			{
				Name:     "success-provider",
				Provider: provider1,
			},
			{
				Name:     "failure-provider",
				Provider: provider2,
			},
		}, 2*time.Second)

		result := strategy.ObjectEvaluation(context.Background(), TestFlag, defaultVal, of.FlattenedContext{})
		assert.Equal(t, successVal, result.Value)
		assert.Equal(t, StrategyFirstSuccess, result.FlagMetadata[MetadataStrategyUsed])
		assert.Contains(t, result.FlagMetadata, MetadataSuccessfulProviderName)
		assert.Equal(t, "success-provider", result.FlagMetadata[MetadataSuccessfulProviderName])
	})

	t.Run("second success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		provider1 := mocks.NewMockFeatureProvider(ctrl)
		configureFlags(provider1, successVal, true, false, 500*time.Millisecond)
		provider2 := mocks.NewMockFeatureProvider(ctrl)
		configureFlags(provider2, defaultVal, false, true, 5*time.Millisecond)

		strategy := NewFirstSuccessStrategy([]*NamedProvider{
			{
				Name:     "success-provider",
				Provider: provider1,
			},
			{
				Name:     "failure-provider",
				Provider: provider2,
			},
		}, 2*time.Second)

		result := strategy.ObjectEvaluation(context.Background(), TestFlag, defaultVal, of.FlattenedContext{})
		assert.Equal(t, successVal, result.Value)
		assert.Equal(t, StrategyFirstSuccess, result.FlagMetadata[MetadataStrategyUsed])
		assert.Contains(t, result.FlagMetadata, MetadataSuccessfulProviderName)
		assert.Equal(t, "success-provider", result.FlagMetadata[MetadataSuccessfulProviderName])
	})

	t.Run("all errors", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		provider1 := mocks.NewMockFeatureProvider(ctrl)
		configureFlags(provider1, defaultVal, false, true, 50*time.Millisecond)
		provider2 := mocks.NewMockFeatureProvider(ctrl)
		configureFlags(provider2, defaultVal, false, true, 40*time.Millisecond)
		provider3 := mocks.NewMockFeatureProvider(ctrl)
		configureFlags(provider3, defaultVal, false, true, 30*time.Millisecond)

		strategy := NewFirstSuccessStrategy([]*NamedProvider{
			{
				Name:     "provider1",
				Provider: provider1,
			},
			{
				Name:     "provider2",
				Provider: provider2,
			},
			{
				Name:     "provider3",
				Provider: provider3,
			},
		}, 2*time.Second)

		result := strategy.ObjectEvaluation(context.Background(), TestFlag, defaultVal, of.FlattenedContext{})
		assert.Equal(t, defaultVal, result.Value)
		assert.Equal(t, StrategyFirstSuccess, result.FlagMetadata[MetadataStrategyUsed])
		assert.Contains(t, result.FlagMetadata, MetadataSuccessfulProviderName)
		assert.Equal(t, "none", result.FlagMetadata[MetadataSuccessfulProviderName])
		assert.Equal(t, of.DefaultReason, result.Reason)
	})
}
