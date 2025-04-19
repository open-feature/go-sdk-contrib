package multiprovider

import (
	"encoding/json"
	"fmt"
	"github.com/open-feature/go-sdk-contrib/providers/multi-provider/internal/strategies"
	of "github.com/open-feature/go-sdk/openfeature"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"golang.org/x/exp/maps"
	"regexp"
	"strings"
	"testing"
	"time"

	mperrs "github.com/open-feature/go-sdk-contrib/providers/multi-provider/internal/errors"

	"github.com/open-feature/go-sdk/openfeature"
	oft "github.com/open-feature/go-sdk/openfeature/testing"
)

// MockProvider utilizes openfeature's TestProvider to add testable Init & Shutdown methods to test the MultiProvider functionality
type MockProvider struct {
	oft.TestProvider
	InitCount *int
	ShutCount *int
	TestErr   string
	InitDelay int
	ShutDelay int
	MockMeta  string
}

func (m *MockProvider) Init(evalCtx openfeature.EvaluationContext) error {
	if m.TestErr != "" {
		return fmt.Errorf(m.TestErr)
	}

	if m.InitDelay != 0 {
		time.Sleep(time.Duration(m.InitDelay) * time.Millisecond)
	}
	*m.InitCount += 1
	return nil
}

func (m *MockProvider) Shutdown() {
	if m.ShutDelay != 0 {
		time.Sleep(time.Duration(m.ShutDelay) * time.Millisecond)
	}
	*m.ShutCount += 1
}

func (m *MockProvider) Metadata() openfeature.Metadata {
	return openfeature.Metadata{Name: m.MockMeta}
}

func TestMultiProvider_ProvidersMethod(t *testing.T) {
	testProvider1 := oft.NewTestProvider()
	testProvider2 := oft.NewTestProvider()

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

func TestMultiProvider_ProvidersByNamesMethod(t *testing.T) {
	testProvider1 := oft.NewTestProvider()
	testProvider2 := oft.NewTestProvider()

	providers := make(ProviderMap)
	providers["provider1"] = testProvider1
	providers["provider2"] = testProvider2

	mp, err := NewMultiProvider(providers, strategies.StrategyFirstMatch)
	require.NoError(t, err)

	p := mp.ProvidersByName()

	require.Len(t, p, 2)
	require.Contains(t, maps.Keys(p), "provider1")
	assert.Equal(t, p["provider1"], testProvider1)
	require.Contains(t, maps.Keys(p), "provider2")
	assert.Equal(t, p["provider2"], testProvider2)
}

func TestMultiProvider_MetaData(t *testing.T) {
	testProvider1 := oft.NewTestProvider()
	ctrl := gomock.NewController(t)
	testProvider2 := strategies.NewMockFeatureProvider(ctrl)
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
	assert.Equal(t, "MultiProvider{provider1:NoopProvider, provider2:MockProvider}", metadata.Name)
}

func TestMultiProvider_Init(t *testing.T) {
	ctrl := gomock.NewController(t)

	testProvider1 := strategies.NewMockFeatureProvider(ctrl)
	testProvider2 := oft.NewTestProvider()
	testProvider3 := strategies.NewMockFeatureProvider(ctrl)

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

	testProvider1 := strategies.NewMockFeatureProvider(ctrl)
	testProvider2 := oft.NewTestProvider()
	testProvider3 := strategies.NewMockFeatureProvider(ctrl)

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
	require.Error(t, err)

	var errors []mperrs.StateErr

	fullErr := err.Error()
	fullErrArr := strings.SplitAfterN(fullErr, "end", 2)
	errJSON := fullErrArr[1]
	errMsg := fullErrArr[0]
	assert.Contains(t, errMsg, "Provider errors occurred:")

	err = json.Unmarshal([]byte(errJSON), &errors)
	require.NoError(t, err)
	assert.Len(t, errors, 2)
	assert.Equal(t, of.ErrorState, mp.status)
}

func TestMultiProvider_Shutdown(t *testing.T) {
	ctrl := gomock.NewController(t)

	testProvider1 := strategies.NewMockFeatureProvider(ctrl)
	testProvider2 := oft.NewTestProvider()
	testProvider3 := strategies.NewMockFeatureProvider(ctrl)

	providers := make(ProviderMap)
	providers["provider1"] = testProvider1
	providers["provider2"] = testProvider2
	providers["provider3"] = testProvider3
	mp, err := NewMultiProvider(providers, strategies.StrategyFirstMatch)
	require.NoError(t, err)

	mp.Shutdown()
}
