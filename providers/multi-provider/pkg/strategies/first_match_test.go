package strategies

import (
	"context"
	"fmt"
	m "github.com/open-feature/go-sdk-contrib/providers/multi-provider/internal/mocks"
	of "github.com/open-feature/go-sdk/openfeature"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"testing"
)

func createMockProviders(ctrl *gomock.Controller, count int) ([]*NamedProvider, map[string]*m.MockFeatureProvider) {
	providers := make([]*NamedProvider, 0, count)
	providerMocks := make(map[string]*m.MockFeatureProvider)
	for index := range count {
		provider := m.NewMockFeatureProvider(ctrl)
		namedProvider := NamedProvider{
			Provider: provider,
			Name:     fmt.Sprintf("%d", index),
		}
		providerMocks[namedProvider.Name] = provider
		providers = append(providers, &namedProvider)
	}

	return providers, providerMocks
}

func Test_FirstMatchStrategy_BooleanEvaluation(t *testing.T) {
	t.Run("Single Provider Match", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		providers, mocks := createMockProviders(ctrl, 1)
		mocks[providers[0].Name].EXPECT().
			BooleanEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(of.BoolResolutionDetail{Value: true})

		strategy := NewFirstMatchStrategy(providers)
		result := strategy.BooleanEvaluation(context.Background(), "test-string", false, of.FlattenedContext{})
		assert.True(t, result.Value)
		assert.Contains(t, result.FlagMetadata, MetadataSuccessfulProviderName)
		assert.Equal(t, providers[0].Name, result.FlagMetadata[MetadataSuccessfulProviderName])
	})

	t.Run("Default Resolution", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		providers, mocks := createMockProviders(ctrl, 1)
		mocks[providers[0].Name].EXPECT().
			BooleanEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(of.BoolResolutionDetail{
				Value: false,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					ResolutionError: of.NewFlagNotFoundResolutionError("not found in any provider"),
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
		ctrl := gomock.NewController(t)
		providers, mocks := createMockProviders(ctrl, 5)
		mocks[providers[0].Name].EXPECT().
			BooleanEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(of.BoolResolutionDetail{
				Value: false,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					ResolutionError: of.NewFlagNotFoundResolutionError("Flag not found"),
				},
			})
		mocks[providers[1].Name].EXPECT().
			BooleanEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(of.BoolResolutionDetail{Value: true})
		for i, p := range providers {
			if i <= 1 {
				continue
			}
			mocks[p.Name].EXPECT().BooleanEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
		}

		strategy := NewFirstMatchStrategy(providers)
		result := strategy.BooleanEvaluation(context.Background(), "test-flag", false, of.FlattenedContext{})
		assert.True(t, result.Value)
		assert.Contains(t, result.FlagMetadata, MetadataSuccessfulProviderName)
		assert.Equal(t, providers[1].Name, result.FlagMetadata[MetadataSuccessfulProviderName])
	})

	t.Run("Evaluation stops after first error that is not a FLAG_NOT_FOUND error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		providers, mocks := createMockProviders(ctrl, 5)
		expectedErr := of.NewGeneralResolutionError("something went wrong")
		mocks[providers[0].Name].EXPECT().
			BooleanEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(of.BoolResolutionDetail{
				Value: false,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					ResolutionError: expectedErr,
					Reason:          of.ErrorReason,
				},
			})
		for i, p := range providers {
			if i == 0 {
				continue
			}
			mocks[p.Name].EXPECT().BooleanEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
		}
		strategy := NewFirstMatchStrategy(providers)
		result := strategy.BooleanEvaluation(context.Background(), "test-string", false, of.FlattenedContext{})
		assert.False(t, result.Value)
		assert.Equal(t, of.ErrorReason, result.Reason)
		assert.Equal(t, expectedErr.Error(), result.ResolutionError.Error())
		assert.Equal(t, "none", result.FlagMetadata[MetadataSuccessfulProviderName])
		assert.Equal(t, StrategyFirstMatch, result.FlagMetadata[MetadataStrategyUsed])
	})
}

func Test_FirstMatchStrategy_StringEvaluation(t *testing.T) {
	ctrl := gomock.NewController(t)

	t.Run("Single Provider Match", func(t *testing.T) {
		providers, mocks := createMockProviders(ctrl, 1)
		mocks[providers[0].Name].EXPECT().
			StringEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(of.StringResolutionDetail{Value: "test"})

		strategy := NewFirstMatchStrategy(providers)
		result := strategy.StringEvaluation(context.Background(), "test-string", "", of.FlattenedContext{})
		assert.Equal(t, "test", result.Value)
		assert.Contains(t, result.FlagMetadata, MetadataSuccessfulProviderName)
		assert.Equal(t, providers[0].Name, result.FlagMetadata[MetadataSuccessfulProviderName])
	})

	t.Run("Default Resolution", func(t *testing.T) {
		providers, mocks := createMockProviders(ctrl, 1)
		mocks[providers[0].Name].EXPECT().
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
		mocks[providers[0].Name].EXPECT().
			StringEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(of.StringResolutionDetail{
				Value: "",
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					ResolutionError: of.NewFlagNotFoundResolutionError("Flag not found"),
				},
			})
		mocks[providers[1].Name].EXPECT().
			StringEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(of.StringResolutionDetail{Value: "test"})
		for i, p := range providers {
			if i <= 1 {
				continue
			}
			mocks[p.Name].EXPECT().StringEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
		}

		strategy := NewFirstMatchStrategy(providers)
		result := strategy.StringEvaluation(context.Background(), "test-flag", "", of.FlattenedContext{})
		assert.Equal(t, "test", result.Value)
		assert.Contains(t, result.FlagMetadata, MetadataSuccessfulProviderName)
		assert.Equal(t, providers[1].Name, result.FlagMetadata[MetadataSuccessfulProviderName])
	})

	t.Run("Evaluation stops after first error that is not a FLAG_NOT_FOUND error", func(t *testing.T) {
		providers, mocks := createMockProviders(ctrl, 5)
		expectedErr := of.NewGeneralResolutionError("something went wrong")
		mocks[providers[0].Name].EXPECT().
			StringEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(of.StringResolutionDetail{
				Value: "",
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					ResolutionError: expectedErr,
					Reason:          of.ErrorReason,
				},
			})
		for i, p := range providers {
			if i == 0 {
				continue
			}
			mocks[p.Name].EXPECT().StringEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
		}
		strategy := NewFirstMatchStrategy(providers)
		result := strategy.StringEvaluation(context.Background(), "test-string", "", of.FlattenedContext{})
		assert.Equal(t, "", result.Value)
		assert.Equal(t, of.ErrorReason, result.Reason)
		assert.Equal(t, expectedErr.Error(), result.ResolutionError.Error())
		assert.Equal(t, "none", result.FlagMetadata[MetadataSuccessfulProviderName])
		assert.Equal(t, StrategyFirstMatch, result.FlagMetadata[MetadataStrategyUsed])
	})
}

func Test_FirstMatchStrategy_IntEvaluation(t *testing.T) {
	ctrl := gomock.NewController(t)

	t.Run("Single Provider Match", func(t *testing.T) {
		providers, mocks := createMockProviders(ctrl, 1)
		mocks[providers[0].Name].EXPECT().
			IntEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(of.IntResolutionDetail{Value: 123})

		strategy := NewFirstMatchStrategy(providers)
		result := strategy.IntEvaluation(context.Background(), "test-string", 0, of.FlattenedContext{})
		assert.Equal(t, int64(123), result.Value)
		assert.Contains(t, result.FlagMetadata, MetadataSuccessfulProviderName)
		assert.Equal(t, providers[0].Name, result.FlagMetadata[MetadataSuccessfulProviderName])
	})

	t.Run("Default Resolution", func(t *testing.T) {
		providers, mocks := createMockProviders(ctrl, 1)
		mocks[providers[0].Name].EXPECT().
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
		mocks[providers[0].Name].EXPECT().
			IntEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(of.IntResolutionDetail{
				Value: 0,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					ResolutionError: of.NewFlagNotFoundResolutionError("Flag not found"),
				},
			})
		mocks[providers[1].Name].EXPECT().
			IntEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(of.IntResolutionDetail{Value: 123})
		for i, p := range providers {
			if i <= 1 {
				continue
			}
			mocks[p.Name].EXPECT().IntEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
		}

		strategy := NewFirstMatchStrategy(providers)
		result := strategy.IntEvaluation(context.Background(), "test-flag", 0, of.FlattenedContext{})
		assert.Equal(t, int64(123), result.Value)
		assert.Contains(t, result.FlagMetadata, MetadataSuccessfulProviderName)
		assert.Equal(t, providers[1].Name, result.FlagMetadata[MetadataSuccessfulProviderName])
	})

	t.Run("Evaluation stops after first error that is not a FLAG_NOT_FOUND error", func(t *testing.T) {
		providers, mocks := createMockProviders(ctrl, 5)
		expectedErr := of.NewGeneralResolutionError("something went wrong")
		mocks[providers[0].Name].EXPECT().
			IntEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(of.IntResolutionDetail{
				Value: 123,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					ResolutionError: expectedErr,
					Reason:          of.ErrorReason,
				},
			})
		for i, p := range providers {
			if i == 0 {
				continue
			}
			mocks[p.Name].EXPECT().IntEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
		}
		strategy := NewFirstMatchStrategy(providers)
		result := strategy.IntEvaluation(context.Background(), "test-string", 123, of.FlattenedContext{})
		assert.Equal(t, int64(123), result.Value)
		assert.Equal(t, of.ErrorReason, result.Reason)
		assert.Equal(t, expectedErr.Error(), result.ResolutionError.Error())
		assert.Equal(t, "none", result.FlagMetadata[MetadataSuccessfulProviderName])
		assert.Equal(t, StrategyFirstMatch, result.FlagMetadata[MetadataStrategyUsed])
	})
}

func Test_FirstMatchStrategy_FloatEvaluation(t *testing.T) {
	ctrl := gomock.NewController(t)

	t.Run("Single Provider Match", func(t *testing.T) {
		providers, mocks := createMockProviders(ctrl, 1)
		mocks[providers[0].Name].EXPECT().
			FloatEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(of.FloatResolutionDetail{Value: 123})

		strategy := NewFirstMatchStrategy(providers)
		result := strategy.FloatEvaluation(context.Background(), "test-string", 0, of.FlattenedContext{})
		assert.Equal(t, float64(123), result.Value)
		assert.Contains(t, result.FlagMetadata, MetadataSuccessfulProviderName)
		assert.Equal(t, providers[0].Name, result.FlagMetadata[MetadataSuccessfulProviderName])
	})

	t.Run("Default Resolution", func(t *testing.T) {
		providers, mocks := createMockProviders(ctrl, 1)
		mocks[providers[0].Name].EXPECT().
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
		mocks[providers[0].Name].EXPECT().
			FloatEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(of.FloatResolutionDetail{
				Value: 0,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					ResolutionError: of.NewFlagNotFoundResolutionError("Flag not found"),
				},
			})
		mocks[providers[1].Name].EXPECT().
			FloatEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(of.FloatResolutionDetail{Value: 123})
		for i, p := range providers {
			if i <= 1 {
				continue
			}
			mocks[p.Name].EXPECT().FloatEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
		}

		strategy := NewFirstMatchStrategy(providers)
		result := strategy.FloatEvaluation(context.Background(), "test-flag", 0, of.FlattenedContext{})
		assert.Equal(t, float64(123), result.Value)
		assert.Contains(t, result.FlagMetadata, MetadataSuccessfulProviderName)
		assert.Equal(t, providers[1].Name, result.FlagMetadata[MetadataSuccessfulProviderName])
	})

	t.Run("Evaluation stops after first error that is not a FLAG_NOT_FOUND error", func(t *testing.T) {
		providers, mocks := createMockProviders(ctrl, 5)
		expectedErr := of.NewGeneralResolutionError("something went wrong")
		mocks[providers[0].Name].EXPECT().
			FloatEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(of.FloatResolutionDetail{
				Value: 123.0,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					ResolutionError: expectedErr,
					Reason:          of.ErrorReason,
				},
			})
		for i, p := range providers {
			if i == 0 {
				continue
			}
			mocks[p.Name].EXPECT().FloatEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
		}
		strategy := NewFirstMatchStrategy(providers)
		result := strategy.FloatEvaluation(context.Background(), "test-string", 123, of.FlattenedContext{})
		assert.Equal(t, 123.0, result.Value)
		assert.Equal(t, of.ErrorReason, result.Reason)
		assert.Equal(t, expectedErr.Error(), result.ResolutionError.Error())
		assert.Equal(t, "none", result.FlagMetadata[MetadataSuccessfulProviderName])
		assert.Equal(t, StrategyFirstMatch, result.FlagMetadata[MetadataStrategyUsed])
	})
}

func Test_FirstMatchStrategy_ObjectEvaluation(t *testing.T) {
	ctrl := gomock.NewController(t)

	t.Run("Single Provider Match", func(t *testing.T) {
		providers, mocks := createMockProviders(ctrl, 1)
		mocks[providers[0].Name].EXPECT().
			ObjectEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(of.InterfaceResolutionDetail{Value: struct{ Field int }{Field: 123}})

		strategy := NewFirstMatchStrategy(providers)
		result := strategy.ObjectEvaluation(context.Background(), "test-string", struct{}{}, of.FlattenedContext{})
		assert.Equal(t, struct{ Field int }{Field: 123}, result.Value)
		assert.Contains(t, result.FlagMetadata, MetadataSuccessfulProviderName)
		assert.Equal(t, providers[0].Name, result.FlagMetadata[MetadataSuccessfulProviderName])
	})

	t.Run("Default Resolution", func(t *testing.T) {
		providers, mocks := createMockProviders(ctrl, 1)
		mocks[providers[0].Name].EXPECT().
			ObjectEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(of.InterfaceResolutionDetail{
				Value: struct{}{},
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					ResolutionError: of.NewFlagNotFoundResolutionError("not found"),
					Reason:          of.DefaultReason,
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
		mocks[providers[0].Name].EXPECT().
			ObjectEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(of.InterfaceResolutionDetail{
				Value: 0,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					ResolutionError: of.NewFlagNotFoundResolutionError("Flag not found"),
				},
			})
		mocks[providers[1].Name].EXPECT().
			ObjectEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(of.InterfaceResolutionDetail{Value: struct{ Field int }{Field: 123}})
		for i, p := range providers {
			if i <= 1 {
				continue
			}
			mocks[p.Name].EXPECT().ObjectEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
		}

		strategy := NewFirstMatchStrategy(providers)
		result := strategy.ObjectEvaluation(context.Background(), "test-flag", struct{}{}, of.FlattenedContext{})
		assert.Equal(t, struct{ Field int }{Field: 123}, result.Value)
		assert.Contains(t, result.FlagMetadata, MetadataSuccessfulProviderName)
		assert.Equal(t, providers[1].Name, result.FlagMetadata[MetadataSuccessfulProviderName])
	})

	t.Run("Evaluation stops after first error that is not a FLAG_NOT_FOUND error", func(t *testing.T) {
		providers, mocks := createMockProviders(ctrl, 5)
		expectedErr := of.NewGeneralResolutionError("something went wrong")
		mocks[providers[0].Name].EXPECT().
			ObjectEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(of.InterfaceResolutionDetail{
				Value: struct{}{},
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					ResolutionError: expectedErr,
					Reason:          of.ErrorReason,
				},
			})
		for i, p := range providers {
			if i == 0 {
				continue
			}
			mocks[p.Name].EXPECT().ObjectEvaluation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
		}
		strategy := NewFirstMatchStrategy(providers)
		result := strategy.ObjectEvaluation(context.Background(), "test-string", struct{}{}, of.FlattenedContext{})
		assert.Equal(t, struct{}{}, result.Value)
		assert.Equal(t, of.ErrorReason, result.Reason)
		assert.Equal(t, expectedErr.Error(), result.ResolutionError.Error())
		assert.Equal(t, "none", result.FlagMetadata[MetadataSuccessfulProviderName])
		assert.Equal(t, StrategyFirstMatch, result.FlagMetadata[MetadataStrategyUsed])
	})
}
