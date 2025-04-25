package multiprovider

import (
	"errors"
	"github.com/open-feature/go-sdk-contrib/providers/multi-provider/internal/mocks"
	"github.com/open-feature/go-sdk-contrib/providers/multi-provider/pkg/strategies"
	"github.com/open-feature/go-sdk/openfeature"
	of "github.com/open-feature/go-sdk/openfeature"
	imp "github.com/open-feature/go-sdk/openfeature/memprovider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"regexp"
	"testing"
)

func TestMultiProvider_ProvidersMethod(t *testing.T) {
	testProvider1 := imp.NewInMemoryProvider(map[string]imp.InMemoryFlag{})
	testProvider2 := imp.NewInMemoryProvider(map[string]imp.InMemoryFlag{})

	providers := make(ProviderMap)
	providers["provider1"] = testProvider1
	providers["provider2"] = testProvider2

	mp, err := NewMultiProvider(providers, strategies.StrategyFirstSuccess)
	require.NoError(t, err)

	p := mp.Providers()
	assert.Len(t, p, 2)
	assert.Regexp(t, regexp.MustCompile("provider[1-2]"), p[0].Name)
	assert.NotNil(t, p[0].Provider)
	assert.Regexp(t, regexp.MustCompile("provider[1-2]"), p[1].Name)
	assert.NotNil(t, p[1].Provider)
}

func TestMultiProvider_NewMultiProvider(t *testing.T) {
	t.Run("nil providerMap returns an error", func(t *testing.T) {
		_, err := NewMultiProvider(nil, strategies.StrategyFirstMatch)
		require.Errorf(t, err, "providerMap cannot be nil or empty")
	})

	t.Run("naming a provider the empty string returns an error", func(t *testing.T) {
		providers := make(ProviderMap)
		providers[""] = imp.NewInMemoryProvider(map[string]imp.InMemoryFlag{})
		_, err := NewMultiProvider(providers, strategies.StrategyFirstMatch)
		require.Errorf(t, err, "provider name cannot be the empty string")
	})

	t.Run("nil provider within map returns an error", func(t *testing.T) {
		providers := make(ProviderMap)
		providers["provider1"] = nil
		_, err := NewMultiProvider(providers, strategies.StrategyFirstMatch)
		require.Errorf(t, err, "provider provider1 cannot be nil")
	})

	t.Run("unknown evaluation strategy returns an error", func(t *testing.T) {
		providers := make(ProviderMap)
		providers["provider1"] = imp.NewInMemoryProvider(map[string]imp.InMemoryFlag{})
		_, err := NewMultiProvider(providers, "unknown")
		require.Errorf(t, err, "unknown is an unknown evaluation strategy")
	})

	t.Run("setting custom strategy without custom strategy option returns error", func(t *testing.T) {
		providers := make(ProviderMap)
		providers["provider1"] = imp.NewInMemoryProvider(map[string]imp.InMemoryFlag{})
		_, err := NewMultiProvider(providers, StrategyCustom)
		require.Errorf(t, err, "A custom strategy must be set via an option if StrategyCustom is set")
	})

	t.Run("success", func(t *testing.T) {
		providers := make(ProviderMap)
		providers["provider1"] = imp.NewInMemoryProvider(map[string]imp.InMemoryFlag{})
		mp, err := NewMultiProvider(providers, StrategyComparison)
		require.NoError(t, err)
		assert.NotZero(t, mp)
	})

	t.Run("success with custom provider", func(t *testing.T) {
		providers := make(ProviderMap)
		providers["provider1"] = imp.NewInMemoryProvider(map[string]imp.InMemoryFlag{})
		ctrl := gomock.NewController(t)
		strategy := strategies.NewMockStrategy(ctrl)
		mp, err := NewMultiProvider(providers, StrategyCustom, WithCustomStrategy(strategy))
		require.NoError(t, err)
		assert.NotZero(t, mp)
	})
}

func TestMultiProvider_ProvidersByNamesMethod(t *testing.T) {
	testProvider1 := imp.NewInMemoryProvider(map[string]imp.InMemoryFlag{})
	testProvider2 := imp.NewInMemoryProvider(map[string]imp.InMemoryFlag{})

	providers := make(ProviderMap)
	providers["provider1"] = testProvider1
	providers["provider2"] = testProvider2

	mp, err := NewMultiProvider(providers, strategies.StrategyFirstMatch)
	require.NoError(t, err)

	p := mp.ProvidersByName()

	assert.Equal(t, 2, p.Size())
	require.Contains(t, p, "provider1")
	assert.Equal(t, p["provider1"], testProvider1)
	require.Contains(t, p, "provider2")
	assert.Equal(t, p["provider2"], testProvider2)
}

func TestMultiProvider_MetaData(t *testing.T) {
	testProvider1 := imp.NewInMemoryProvider(map[string]imp.InMemoryFlag{})
	ctrl := gomock.NewController(t)
	testProvider2 := mocks.NewMockFeatureProvider(ctrl)
	testProvider2.EXPECT().Metadata().Return(of.Metadata{
		Name: "MockProvider",
	})

	providers := make(ProviderMap)
	providers["provider1"] = testProvider1
	providers["provider2"] = testProvider2

	mp, err := NewMultiProvider(providers, strategies.StrategyFirstSuccess)
	require.NoError(t, err)

	metadata := mp.Metadata()
	require.NotZero(t, metadata)
	assert.Equal(t, "MultiProvider {provider1: NoopProvider, provider2: MockProvider}", metadata.Name)
}

func TestMultiProvider_Init(t *testing.T) {
	ctrl := gomock.NewController(t)

	testProvider1 := mocks.NewMockFeatureProvider(ctrl)
	testProvider1.EXPECT().Metadata().Return(of.Metadata{Name: "MockProvider"})
	testProvider2 := imp.NewInMemoryProvider(map[string]imp.InMemoryFlag{})
	testProvider3 := mocks.NewMockFeatureProvider(ctrl)
	testProvider3.EXPECT().Metadata().Return(of.Metadata{Name: "MockProvider"})

	providers := make(ProviderMap)
	providers["provider1"] = testProvider1
	providers["provider2"] = testProvider2
	providers["provider3"] = testProvider3

	mp, err := NewMultiProvider(providers, strategies.StrategyFirstMatch)
	require.NoError(t, err)

	attributes := map[string]interface{}{
		"foo": "bar",
	}
	evalCtx := openfeature.NewTargetlessEvaluationContext(attributes)

	err = mp.Init(evalCtx)
	require.NoError(t, err)
	assert.Equal(t, of.ReadyState, mp.status)
}

func TestMultiProvider_InitErrorWithProvider(t *testing.T) {
	ctrl := gomock.NewController(t)
	errProvider := mocks.NewMockFeatureProvider(ctrl)
	errProvider.EXPECT().Metadata().Return(of.Metadata{Name: "MockProvider"})
	errHandler := mocks.NewMockStateHandler(ctrl)
	errHandler.EXPECT().Init(gomock.Any()).Return(errors.New("test error"))
	testProvider3 := struct {
		of.FeatureProvider
		of.StateHandler
	}{
		errProvider,
		errHandler,
	}

	testProvider1 := mocks.NewMockFeatureProvider(ctrl)
	testProvider1.EXPECT().Metadata().Return(of.Metadata{Name: "MockProvider"})
	testProvider2 := imp.NewInMemoryProvider(map[string]imp.InMemoryFlag{})

	providers := make(ProviderMap)
	providers["provider1"] = testProvider1
	providers["provider2"] = testProvider2
	providers["provider3"] = testProvider3

	mp, err := NewMultiProvider(providers, strategies.StrategyFirstMatch)
	require.NoError(t, err)

	attributes := map[string]interface{}{
		"foo": "bar",
	}
	evalCtx := openfeature.NewTargetlessEvaluationContext(attributes)

	err = mp.Init(evalCtx)
	require.Errorf(t, err, "Provider provider1: test error")
	assert.Equal(t, of.ErrorState, mp.status)
}

func TestMultiProvider_Shutdown(t *testing.T) {
	ctrl := gomock.NewController(t)

	testProvider1 := mocks.NewMockFeatureProvider(ctrl)
	testProvider1.EXPECT().Metadata().Return(of.Metadata{Name: "MockProvider"})
	testProvider2 := imp.NewInMemoryProvider(map[string]imp.InMemoryFlag{})
	testProvider3 := mocks.NewMockFeatureProvider(ctrl)
	testProvider3.EXPECT().Metadata().Return(of.Metadata{Name: "MockProvider"})

	providers := make(ProviderMap)
	providers["provider1"] = testProvider1
	providers["provider2"] = testProvider2
	providers["provider3"] = testProvider3
	mp, err := NewMultiProvider(providers, strategies.StrategyFirstMatch)
	require.NoError(t, err)

	mp.Shutdown()
}
