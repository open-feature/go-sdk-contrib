package strategies

import (
	"context"
	"fmt"
	multiprovider "github.com/open-feature/go-sdk-contrib/providers/multi-provider/pkg"
	of "github.com/open-feature/go-sdk/openfeature"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"testing"
)

func createMockProviders(ctrl *gomock.Controller, count int) ([]multiprovider.UniqueNameProvider, map[string]*MockFeatureProvider) {
	providers := make([]multiprovider.UniqueNameProvider, 0, count)
	providerMocks := make(map[string]*MockFeatureProvider)
	for index := range count {
		provider := NewMockFeatureProvider(ctrl)
		namedProvider := multiprovider.UniqueNameProvider{
			Provider:   provider,
			UniqueName: fmt.Sprintf("%d", index),
		}
		providerMocks[namedProvider.UniqueName] = provider
		providers = append(providers, namedProvider)
	}

	return providers, providerMocks
}

func Test_FirstMatchStrategy_BooleanEvaluation(t *testing.T) {
	ctrl := gomock.NewController(t)

	t.Run("Single Provider Match", func(t *testing.T) {
		providers, mocks := createMockProviders(ctrl, 1)
		mocks[providers[0].UniqueName].EXPECT().
			BooleanEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(of.BoolResolutionDetail{Value: true})

		strategy := NewFirstMatchStrategy(providers)
		result := strategy.BooleanEvaluation(context.Background(), "test-string", false, of.FlattenedContext{})
		assert.True(t, result.Value)
		assert.Contains(t, result.FlagMetadata, MetadataSuccessfulProviderName)
		assert.Equal(t, providers[0].UniqueName, result.FlagMetadata[MetadataSuccessfulProviderName])
	})

	t.Run("Default Resolution", func(t *testing.T) {
		providers, mocks := createMockProviders(ctrl, 1)
		mocks[providers[0].UniqueName].EXPECT().
			BooleanEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(of.BoolResolutionDetail{
				Value: false,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					ResolutionError: of.NewFlagNotFoundResolutionError("not found"),
				},
			})
		strategy := NewFirstMatchStrategy(providers)
		result := strategy.BooleanEvaluation(context.Background(), "test-string", false, of.FlattenedContext{})
		assert.False(t, result.Value)
		assert.Equal(t, of.DefaultReason, result.Reason)
		assert.Equal(t, of.NewFlagNotFoundResolutionError("not found in any provider").Error(), result.ResolutionError.Error())
		assert.Equal(t, "none", result.FlagMetadata[MetadataSuccessfulProviderName])
		assert.Equal(t, StrategyFirstMatch, result.FlagMetadata[MetadataStrategyUsed])
	})

	t.Run("Evaluation stops after match", func(t *testing.T) {
		providers, mocks := createMockProviders(ctrl, 5)
		mocks[providers[0].UniqueName].EXPECT().
			BooleanEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(of.BoolResolutionDetail{
				Value: false,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					ResolutionError: of.NewFlagNotFoundResolutionError("Flag not found"),
				},
			})
		mocks[providers[1].UniqueName].EXPECT().
			BooleanEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(of.BoolResolutionDetail{Value: true})
		for i, p := range providers {
			if i <= 1 {
				continue
			}
			mocks[p.UniqueName].EXPECT().BooleanEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
		}

		strategy := NewFirstMatchStrategy(providers)
		result := strategy.BooleanEvaluation(context.Background(), "test-flag", false, of.FlattenedContext{})
		assert.True(t, result.Value)
		assert.Contains(t, result.FlagMetadata, MetadataSuccessfulProviderName)
		assert.Equal(t, providers[1].UniqueName, result.FlagMetadata[MetadataSuccessfulProviderName])
	})

	t.Run("Evaluation stops after first error that is not a FLAG_NOT_FOUND error", func(t *testing.T) {
		providers, mocks := createMockProviders(ctrl, 5)
		mocks[providers[0].UniqueName].EXPECT().
			BooleanEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(of.BoolResolutionDetail{
				Value: false,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					ResolutionError: of.NewGeneralResolutionError("something went wrong"),
				},
			})
		for i, p := range providers {
			if i == 0 {
				continue
			}
			mocks[p.UniqueName].EXPECT().BooleanEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
		}
		strategy := NewFirstMatchStrategy(providers)
		result := strategy.BooleanEvaluation(context.Background(), "test-string", false, of.FlattenedContext{})
		assert.False(t, result.Value)
		assert.Equal(t, of.DefaultReason, result.Reason)
		assert.Equal(t, of.NewGeneralResolutionError("something went wrong").Error(), result.ResolutionError.Error())
		assert.Equal(t, "none", result.FlagMetadata[MetadataSuccessfulProviderName])
		assert.Equal(t, StrategyFirstMatch, result.FlagMetadata[MetadataStrategyUsed])
	})
}

func Test_FirstMatchStrategy_StringEvaluation(t *testing.T) {
	ctrl := gomock.NewController(t)

	t.Run("Single Provider Match", func(t *testing.T) {
		providers, mocks := createMockProviders(ctrl, 1)
		mocks[providers[0].UniqueName].EXPECT().
			StringEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(of.StringResolutionDetail{Value: "test"})

		strategy := NewFirstMatchStrategy(providers)
		result := strategy.StringEvaluation(context.Background(), "test-string", "", of.FlattenedContext{})
		assert.Equal(t, "test", result.Value)
		assert.Contains(t, result.FlagMetadata, MetadataSuccessfulProviderName)
		assert.Equal(t, providers[0].UniqueName, result.FlagMetadata[MetadataSuccessfulProviderName])
	})

	t.Run("Default Resolution", func(t *testing.T) {
		providers, mocks := createMockProviders(ctrl, 1)
		mocks[providers[0].UniqueName].EXPECT().
			StringEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(of.StringResolutionDetail{
				Value: "",
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					ResolutionError: of.NewFlagNotFoundResolutionError("not found"),
				},
			})
		strategy := NewFirstMatchStrategy(providers)
		result := strategy.StringEvaluation(context.Background(), "test-string", "", of.FlattenedContext{})
		assert.Equal(t, "", result.Value)
		assert.Equal(t, of.DefaultReason, result.Reason)
		assert.Equal(t, of.NewFlagNotFoundResolutionError("not found in any provider").Error(), result.ResolutionError.Error())
		assert.Equal(t, "none", result.FlagMetadata[MetadataSuccessfulProviderName])
		assert.Equal(t, StrategyFirstMatch, result.FlagMetadata[MetadataStrategyUsed])
	})

	t.Run("Evaluation stops after match", func(t *testing.T) {
		providers, mocks := createMockProviders(ctrl, 5)
		mocks[providers[0].UniqueName].EXPECT().
			StringEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(of.StringResolutionDetail{
				Value: "",
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					ResolutionError: of.NewFlagNotFoundResolutionError("Flag not found"),
				},
			})
		mocks[providers[1].UniqueName].EXPECT().
			StringEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(of.StringResolutionDetail{Value: "test"})
		for i, p := range providers {
			if i <= 1 {
				continue
			}
			mocks[p.UniqueName].EXPECT().StringEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
		}

		strategy := NewFirstMatchStrategy(providers)
		result := strategy.StringEvaluation(context.Background(), "test-flag", "", of.FlattenedContext{})
		assert.Equal(t, "test", result.Value)
		assert.Contains(t, result.FlagMetadata, MetadataSuccessfulProviderName)
		assert.Equal(t, providers[1].UniqueName, result.FlagMetadata[MetadataSuccessfulProviderName])
	})

	t.Run("Evaluation stops after first error that is not a FLAG_NOT_FOUND error", func(t *testing.T) {
		providers, mocks := createMockProviders(ctrl, 5)
		mocks[providers[0].UniqueName].EXPECT().
			StringEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(of.StringResolutionDetail{
				Value: "",
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					ResolutionError: of.NewGeneralResolutionError("something went wrong"),
				},
			})
		for i, p := range providers {
			if i == 0 {
				continue
			}
			mocks[p.UniqueName].EXPECT().StringEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
		}
		strategy := NewFirstMatchStrategy(providers)
		result := strategy.StringEvaluation(context.Background(), "test-string", "", of.FlattenedContext{})
		assert.Equal(t, "", result.Value)
		assert.Equal(t, of.DefaultReason, result.Reason)
		assert.Equal(t, of.NewGeneralResolutionError("something went wrong").Error(), result.ResolutionError.Error())
		assert.Equal(t, "none", result.FlagMetadata[MetadataSuccessfulProviderName])
		assert.Equal(t, StrategyFirstMatch, result.FlagMetadata[MetadataStrategyUsed])
	})
}

func Test_FirstMatchStrategy_IntEvaluation(t *testing.T) {
	ctrl := gomock.NewController(t)

	t.Run("Single Provider Match", func(t *testing.T) {
		providers, mocks := createMockProviders(ctrl, 1)
		mocks[providers[0].UniqueName].EXPECT().
			IntEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(of.IntResolutionDetail{Value: 123})

		strategy := NewFirstMatchStrategy(providers)
		result := strategy.IntEvaluation(context.Background(), "test-string", 0, of.FlattenedContext{})
		assert.Equal(t, int64(123), result.Value)
		assert.Contains(t, result.FlagMetadata, MetadataSuccessfulProviderName)
		assert.Equal(t, providers[0].UniqueName, result.FlagMetadata[MetadataSuccessfulProviderName])
	})

	t.Run("Default Resolution", func(t *testing.T) {
		providers, mocks := createMockProviders(ctrl, 1)
		mocks[providers[0].UniqueName].EXPECT().
			IntEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(of.IntResolutionDetail{
				Value: 0,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					ResolutionError: of.NewFlagNotFoundResolutionError("not found"),
				},
			})
		strategy := NewFirstMatchStrategy(providers)
		result := strategy.IntEvaluation(context.Background(), "test-string", 0, of.FlattenedContext{})
		assert.Equal(t, int64(0), result.Value)
		assert.Equal(t, of.DefaultReason, result.Reason)
		assert.Equal(t, of.NewFlagNotFoundResolutionError("not found in any provider").Error(), result.ResolutionError.Error())
		assert.Equal(t, "none", result.FlagMetadata[MetadataSuccessfulProviderName])
		assert.Equal(t, StrategyFirstMatch, result.FlagMetadata[MetadataStrategyUsed])
	})

	t.Run("Evaluation stops after match", func(t *testing.T) {
		providers, mocks := createMockProviders(ctrl, 5)
		mocks[providers[0].UniqueName].EXPECT().
			IntEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(of.IntResolutionDetail{
				Value: 0,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					ResolutionError: of.NewFlagNotFoundResolutionError("Flag not found"),
				},
			})
		mocks[providers[1].UniqueName].EXPECT().
			IntEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(of.IntResolutionDetail{Value: 123})
		for i, p := range providers {
			if i <= 1 {
				continue
			}
			mocks[p.UniqueName].EXPECT().IntEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
		}

		strategy := NewFirstMatchStrategy(providers)
		result := strategy.IntEvaluation(context.Background(), "test-flag", 0, of.FlattenedContext{})
		assert.Equal(t, int64(123), result.Value)
		assert.Contains(t, result.FlagMetadata, MetadataSuccessfulProviderName)
		assert.Equal(t, providers[1].UniqueName, result.FlagMetadata[MetadataSuccessfulProviderName])
	})

	t.Run("Evaluation stops after first error that is not a FLAG_NOT_FOUND error", func(t *testing.T) {
		providers, mocks := createMockProviders(ctrl, 5)
		mocks[providers[0].UniqueName].EXPECT().
			IntEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(of.IntResolutionDetail{
				Value: 123,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					ResolutionError: of.NewGeneralResolutionError("something went wrong"),
				},
			})
		for i, p := range providers {
			if i == 0 {
				continue
			}
			mocks[p.UniqueName].EXPECT().IntEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
		}
		strategy := NewFirstMatchStrategy(providers)
		result := strategy.IntEvaluation(context.Background(), "test-string", 123, of.FlattenedContext{})
		assert.Equal(t, int64(123), result.Value)
		assert.Equal(t, of.DefaultReason, result.Reason)
		assert.Equal(t, of.NewGeneralResolutionError("something went wrong").Error(), result.ResolutionError.Error())
		assert.Equal(t, "none", result.FlagMetadata[MetadataSuccessfulProviderName])
		assert.Equal(t, StrategyFirstMatch, result.FlagMetadata[MetadataStrategyUsed])
	})
}

func Test_FirstMatchStrategy_FloatEvaluation(t *testing.T) {
	ctrl := gomock.NewController(t)

	t.Run("Single Provider Match", func(t *testing.T) {
		providers, mocks := createMockProviders(ctrl, 1)
		mocks[providers[0].UniqueName].EXPECT().
			FloatEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(of.FloatResolutionDetail{Value: 123})

		strategy := NewFirstMatchStrategy(providers)
		result := strategy.FloatEvaluation(context.Background(), "test-string", 0, of.FlattenedContext{})
		assert.Equal(t, float64(123), result.Value)
		assert.Contains(t, result.FlagMetadata, MetadataSuccessfulProviderName)
		assert.Equal(t, providers[0].UniqueName, result.FlagMetadata[MetadataSuccessfulProviderName])
	})

	t.Run("Default Resolution", func(t *testing.T) {
		providers, mocks := createMockProviders(ctrl, 1)
		mocks[providers[0].UniqueName].EXPECT().
			FloatEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(of.FloatResolutionDetail{
				Value: 0,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					ResolutionError: of.NewFlagNotFoundResolutionError("not found"),
				},
			})
		strategy := NewFirstMatchStrategy(providers)
		result := strategy.FloatEvaluation(context.Background(), "test-string", 0, of.FlattenedContext{})
		assert.Equal(t, float64(0), result.Value)
		assert.Equal(t, of.DefaultReason, result.Reason)
		assert.Equal(t, of.NewFlagNotFoundResolutionError("not found in any provider").Error(), result.ResolutionError.Error())
		assert.Equal(t, "none", result.FlagMetadata[MetadataSuccessfulProviderName])
		assert.Equal(t, StrategyFirstMatch, result.FlagMetadata[MetadataStrategyUsed])
	})

	t.Run("Evaluation stops after match", func(t *testing.T) {
		providers, mocks := createMockProviders(ctrl, 5)
		mocks[providers[0].UniqueName].EXPECT().
			FloatEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(of.FloatResolutionDetail{
				Value: 0,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					ResolutionError: of.NewFlagNotFoundResolutionError("Flag not found"),
				},
			})
		mocks[providers[1].UniqueName].EXPECT().
			FloatEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(of.FloatResolutionDetail{Value: 123})
		for i, p := range providers {
			if i <= 1 {
				continue
			}
			mocks[p.UniqueName].EXPECT().FloatEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
		}

		strategy := NewFirstMatchStrategy(providers)
		result := strategy.FloatEvaluation(context.Background(), "test-flag", 0, of.FlattenedContext{})
		assert.Equal(t, float64(123), result.Value)
		assert.Contains(t, result.FlagMetadata, MetadataSuccessfulProviderName)
		assert.Equal(t, providers[1].UniqueName, result.FlagMetadata[MetadataSuccessfulProviderName])
	})

	t.Run("Evaluation stops after first error that is not a FLAG_NOT_FOUND error", func(t *testing.T) {
		providers, mocks := createMockProviders(ctrl, 5)
		mocks[providers[0].UniqueName].EXPECT().
			FloatEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(of.FloatResolutionDetail{
				Value: 123.0,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					ResolutionError: of.NewGeneralResolutionError("something went wrong"),
				},
			})
		for i, p := range providers {
			if i == 0 {
				continue
			}
			mocks[p.UniqueName].EXPECT().FloatEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
		}
		strategy := NewFirstMatchStrategy(providers)
		result := strategy.FloatEvaluation(context.Background(), "test-string", 123, of.FlattenedContext{})
		assert.Equal(t, 123.0, result.Value)
		assert.Equal(t, of.DefaultReason, result.Reason)
		assert.Equal(t, of.NewGeneralResolutionError("something went wrong").Error(), result.ResolutionError.Error())
		assert.Equal(t, "none", result.FlagMetadata[MetadataSuccessfulProviderName])
		assert.Equal(t, StrategyFirstMatch, result.FlagMetadata[MetadataStrategyUsed])
	})
}

func Test_FirstMatchStrategy_ObjectEvaluation(t *testing.T) {
	ctrl := gomock.NewController(t)

	t.Run("Single Provider Match", func(t *testing.T) {
		providers, mocks := createMockProviders(ctrl, 1)
		mocks[providers[0].UniqueName].EXPECT().
			ObjectEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(of.InterfaceResolutionDetail{Value: struct{ Field int }{Field: 123}})

		strategy := NewFirstMatchStrategy(providers)
		result := strategy.ObjectEvaluation(context.Background(), "test-string", struct{}{}, of.FlattenedContext{})
		assert.Equal(t, struct{ Field int }{Field: 123}, result.Value)
		assert.Contains(t, result.FlagMetadata, MetadataSuccessfulProviderName)
		assert.Equal(t, providers[0].UniqueName, result.FlagMetadata[MetadataSuccessfulProviderName])
	})

	t.Run("Default Resolution", func(t *testing.T) {
		providers, mocks := createMockProviders(ctrl, 1)
		mocks[providers[0].UniqueName].EXPECT().
			ObjectEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(of.InterfaceResolutionDetail{
				Value: struct{}{},
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					ResolutionError: of.NewFlagNotFoundResolutionError("not found"),
				},
			})
		strategy := NewFirstMatchStrategy(providers)
		result := strategy.ObjectEvaluation(context.Background(), "test-string", struct{}{}, of.FlattenedContext{})
		assert.Equal(t, struct{}{}, result.Value)
		assert.Equal(t, of.DefaultReason, result.Reason)
		assert.Equal(t, of.NewFlagNotFoundResolutionError("not found in any provider").Error(), result.ResolutionError.Error())
		assert.Equal(t, "none", result.FlagMetadata[MetadataSuccessfulProviderName])
		assert.Equal(t, StrategyFirstMatch, result.FlagMetadata[MetadataStrategyUsed])
	})

	t.Run("Evaluation stops after match", func(t *testing.T) {
		providers, mocks := createMockProviders(ctrl, 5)
		mocks[providers[0].UniqueName].EXPECT().
			ObjectEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(of.InterfaceResolutionDetail{
				Value: 0,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					ResolutionError: of.NewFlagNotFoundResolutionError("Flag not found"),
				},
			})
		mocks[providers[1].UniqueName].EXPECT().
			ObjectEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(of.InterfaceResolutionDetail{Value: struct{ Field int }{Field: 123}})
		for i, p := range providers {
			if i <= 1 {
				continue
			}
			mocks[p.UniqueName].EXPECT().ObjectEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
		}

		strategy := NewFirstMatchStrategy(providers)
		result := strategy.ObjectEvaluation(context.Background(), "test-flag", struct{}{}, of.FlattenedContext{})
		assert.Equal(t, struct{ Field int }{Field: 123}, result.Value)
		assert.Contains(t, result.FlagMetadata, MetadataSuccessfulProviderName)
		assert.Equal(t, providers[1].UniqueName, result.FlagMetadata[MetadataSuccessfulProviderName])
	})

	t.Run("Evaluation stops after first error that is not a FLAG_NOT_FOUND error", func(t *testing.T) {
		providers, mocks := createMockProviders(ctrl, 5)
		mocks[providers[0].UniqueName].EXPECT().
			ObjectEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(of.InterfaceResolutionDetail{
				Value: struct{}{},
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					ResolutionError: of.NewGeneralResolutionError("something went wrong"),
				},
			})
		for i, p := range providers {
			if i == 0 {
				continue
			}
			mocks[p.UniqueName].EXPECT().ObjectEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
		}
		strategy := NewFirstMatchStrategy(providers)
		result := strategy.ObjectEvaluation(context.Background(), "test-string", struct{}{}, of.FlattenedContext{})
		assert.Equal(t, struct{}{}, result.Value)
		assert.Equal(t, of.DefaultReason, result.Reason)
		assert.Equal(t, of.NewGeneralResolutionError("something went wrong").Error(), result.ResolutionError.Error())
		assert.Equal(t, "none", result.FlagMetadata[MetadataSuccessfulProviderName])
		assert.Equal(t, StrategyFirstMatch, result.FlagMetadata[MetadataStrategyUsed])
	})
}
