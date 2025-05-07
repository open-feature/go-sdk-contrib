package multiprovider

import (
	"context"
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
	testProvider2.EXPECT().Hooks().Return([]of.Hook{}).MinTimes(1)

	providers := make(ProviderMap)
	providers["provider1"] = testProvider1
	providers["provider2"] = testProvider2

	mp, err := NewMultiProvider(providers, strategies.StrategyFirstSuccess)
	require.NoError(t, err)

	metadata := mp.Metadata()
	require.NotZero(t, metadata)
	assert.Equal(t, "MultiProvider {provider1: InMemoryProvider, provider2: MockProvider}", metadata.Name)
}

func TestMultiProvider_Init(t *testing.T) {
	ctrl := gomock.NewController(t)

	testProvider1 := mocks.NewMockFeatureProvider(ctrl)
	testProvider1.EXPECT().Metadata().Return(of.Metadata{Name: "MockProvider"})
	testProvider1.EXPECT().Hooks().Return([]of.Hook{}).MinTimes(1)
	testProvider2 := imp.NewInMemoryProvider(map[string]imp.InMemoryFlag{})
	testProvider3 := mocks.NewMockFeatureProvider(ctrl)
	testProvider3.EXPECT().Metadata().Return(of.Metadata{Name: "MockProvider"})
	testProvider3.EXPECT().Hooks().Return([]of.Hook{}).MinTimes(1)

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
	eventChan := make(chan of.Event)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		select {
		case e := <-mp.EventChannel():
			eventChan <- e
		case <-ctx.Done():
			return
		}
	}()
	err = mp.Init(evalCtx)
	require.NoError(t, err)
	assert.Equal(t, of.ReadyState, mp.Status())
	cancel()
	event := <-eventChan
	assert.NotZero(t, event)
	assert.Equal(t, mp.Metadata().Name, event.ProviderName)
	assert.Equal(t, of.ProviderReady, event.EventType)
	assert.Equal(t, of.ProviderEventDetails{
		Message:     "all internal providers initialized successfully",
		FlagChanges: nil,
		EventMetadata: map[string]interface{}{
			MetadataProviderName: "all",
		},
	}, event.ProviderEventDetails)
	t.Cleanup(func() {
		mp.Shutdown()
	})
}

func TestMultiProvider_InitErrorWithProvider(t *testing.T) {
	ctrl := gomock.NewController(t)
	errProvider := mocks.NewMockFeatureProvider(ctrl)
	errProvider.EXPECT().Metadata().Return(of.Metadata{Name: "MockProvider"})
	errProvider.EXPECT().Hooks().Return([]of.Hook{}).MinTimes(1)
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
	testProvider1.EXPECT().Hooks().Return([]of.Hook{}).MinTimes(1)
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
	eventChan := make(chan of.Event)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		select {
		case e := <-mp.EventChannel():
			eventChan <- e
		case <-ctx.Done():
			return
		}
	}()
	err = mp.Init(evalCtx)
	require.Errorf(t, err, "Provider provider3: test error")
	assert.Equal(t, of.ErrorState, mp.totalStatus)
	cancel()
	event := <-eventChan
	assert.NotZero(t, event)
	assert.Equal(t, mp.Metadata().Name, event.ProviderName)
	assert.Equal(t, of.ProviderError, event.EventType)
	assert.Equal(t, of.ProviderEventDetails{
		Message:     "internal provider provider3 encountered an error during initialization: test error",
		FlagChanges: nil,
		EventMetadata: map[string]interface{}{
			MetadataProviderName:  "provider3",
			MetadataInternalError: "Provider provider3: test error",
		},
	}, event.ProviderEventDetails)
}

func TestMultiProvider_Shutdown_WithoutInit(t *testing.T) {
	ctrl := gomock.NewController(t)

	testProvider1 := mocks.NewMockFeatureProvider(ctrl)
	testProvider1.EXPECT().Metadata().Return(of.Metadata{Name: "MockProvider"})
	testProvider1.EXPECT().Hooks().Return([]of.Hook{}).MinTimes(1)
	testProvider2 := imp.NewInMemoryProvider(map[string]imp.InMemoryFlag{})
	testProvider3 := mocks.NewMockFeatureProvider(ctrl)
	testProvider3.EXPECT().Metadata().Return(of.Metadata{Name: "MockProvider"})
	testProvider3.EXPECT().Hooks().Return([]of.Hook{}).MinTimes(1)

	providers := make(ProviderMap)
	providers["provider1"] = testProvider1
	providers["provider2"] = testProvider2
	providers["provider3"] = testProvider3
	mp, err := NewMultiProvider(providers, strategies.StrategyFirstMatch)
	require.NoError(t, err)

	mp.Shutdown()
}

func TestMultiProvider_Shutdown_WithInit(t *testing.T) {
	ctrl := gomock.NewController(t)

	testProvider1 := mocks.NewMockFeatureProvider(ctrl)
	testProvider1.EXPECT().Metadata().Return(of.Metadata{Name: "MockProvider"})
	testProvider1.EXPECT().Hooks().Return([]of.Hook{}).MinTimes(1)
	testProvider2 := imp.NewInMemoryProvider(map[string]imp.InMemoryFlag{})
	handlingProvider := mocks.NewMockFeatureProvider(ctrl)
	handlingProvider.EXPECT().Metadata().Return(of.Metadata{Name: "MockProvider"})
	handlingProvider.EXPECT().Hooks().Return([]of.Hook{}).MinTimes(1)
	handledHandler := mocks.NewMockStateHandler(ctrl)
	handledHandler.EXPECT().Init(gomock.Any()).Return(nil)
	handledHandler.EXPECT().Shutdown()
	testProvider3 := struct {
		of.FeatureProvider
		of.StateHandler
	}{
		handlingProvider,
		handledHandler,
	}

	providers := make(ProviderMap)
	providers["provider1"] = testProvider1
	providers["provider2"] = testProvider2
	providers["provider3"] = testProvider3
	mp, err := NewMultiProvider(providers, strategies.StrategyFirstMatch)
	require.NoError(t, err)
	evalCtx := openfeature.NewTargetlessEvaluationContext(map[string]interface{}{
		"foo": "bar",
	})
	eventChan := make(chan of.Event)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		select {
		case e := <-mp.EventChannel():
			eventChan <- e
		case <-ctx.Done():
			return
		}
	}()
	err = mp.Init(evalCtx)
	require.NoError(t, err)
	assert.Equal(t, of.ReadyState, mp.Status())
	cancel()
	mp.Shutdown()
	assert.Equal(t, of.NotReadyState, mp.Status())
}
