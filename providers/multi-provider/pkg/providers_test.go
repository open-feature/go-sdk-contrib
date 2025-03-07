package multiprovider

import (
	"testing"
	"time"

	"github.com/open-feature/go-sdk/openfeature"
	"github.com/open-feature/go-sdk/openfeature/hooks"
	"github.com/open-feature/go-sdk/openfeature/memprovider"
	oft "github.com/open-feature/go-sdk/openfeature/testing"
)

// MockProvider utilizes openfeature's TestProvider to add testable Init & Shutdown methods to test the MultiProvider functionality
type MockProvider struct {
	oft.TestProvider
	InitCount *int
	ShutCount *int
}

func (m *MockProvider) Init(evalCtx openfeature.EvaluationContext) error {
	*m.InitCount += 1
	return nil
}

func (m *MockProvider) Shutdown() {
	*m.ShutCount += 1
}

func NewMockProvider(initCount *int, shutCount *int) *MockProvider {
	return &MockProvider{
		TestProvider: oft.NewTestProvider(),
		InitCount:    initCount,
		ShutCount:    shutCount,
	}
}

// MockProviderDelay utilizes openfeature's TestProvider to add testable Init & Shutdown methods to test the MultiProvider functionality with a small delay making sure the the go routines properly wait.
type MockProviderDelay struct {
	oft.TestProvider
	InitCount *int
	ShutCount *int
}

func (m *MockProviderDelay) Init(evalCtx openfeature.EvaluationContext) error {
	time.Sleep(1 * time.Millisecond)
	*m.InitCount += 1
	return nil
}

func (m *MockProviderDelay) Shutdown() {
	time.Sleep(2 * time.Millisecond)
	*m.ShutCount += 1
}

func NewMockProviderDelay(initCount *int, shutCount *int) *MockProviderDelay {
	return &MockProviderDelay{
		TestProvider: oft.NewTestProvider(),
		InitCount:    initCount,
		ShutCount:    shutCount,
	}
}

func TestMultiProvider_ProvidersMethod(t *testing.T) {
	testProvider1 := oft.NewTestProvider()
	testProvider2 := oft.NewTestProvider()

	defaultLogger, err := hooks.NewLoggingHook(false)
	if err != nil {
		t.Errorf("Issue setting up logger,'%s'", err)
	}

	mp, err := NewMultiProvider([]UniqueNameProvider{
		{
			Provider:   testProvider1,
			UniqueName: "provider1",
		}, {
			Provider:   testProvider2,
			UniqueName: "provider2",
		},
	}, "test", defaultLogger)

	if err != nil {
		t.Errorf("Expected the multiprovider to successfully make an instance, '%s'", err)
	}

	providers := mp.Providers()

	if len(providers) != 2 {
		t.Errorf("Expected there to be '2' providers as passed but got: '%d'", len(providers))
	}

	if providers[0].UniqueName != "provider1" {
		t.Errorf("Expected unique provider name to be: 'provider1', got: '%s'", providers[0].UniqueName)
	}
	if providers[1].UniqueName != "provider2" {
		t.Errorf("Expected unique provider name to be: 'provider2', got: '%s'", providers[1].UniqueName)
	}
}

func TestMultiProvider_ProvidersByNamesMethod(t *testing.T) {
	testProvider1 := oft.NewTestProvider()
	testProvider2 := oft.NewTestProvider()

	defaultLogger, err := hooks.NewLoggingHook(false)
	if err != nil {
		t.Errorf("Issue setting up logger,'%s'", err)
	}

	mp, err := NewMultiProvider([]UniqueNameProvider{
		{
			Provider:   testProvider1,
			UniqueName: "provider1",
		}, {
			Provider:   testProvider2,
			UniqueName: "provider2",
		},
	}, "test", defaultLogger)

	if err != nil {
		t.Errorf("Expected the multiprovider to successfully make an instance, '%s'", err)
	}

	providers := mp.ProvidersByName()

	if len(providers) != 2 {
		t.Errorf("Expected there to be '2' providers as passed but got: '%d'", len(providers))
	}

	if provider, exists := providers["provider1"]; exists {
		if provider.UniqueName != "provider1" {
			t.Errorf("Expected unique provider name to be: 'provider1', got: '%s'", provider.UniqueName)
		}
		if provider.Provider != testProvider1 {
			t.Errorf("Expected unique provider name to be: 'provider1', got: '%s'", provider.UniqueName)
		}
	} else {
		t.Errorf("Expected there to be a provider with the key of '%s', but none was found.", "provider1")
	}

	if provider, exists := providers["provider2"]; exists {
		if provider.UniqueName != "provider2" {
			t.Errorf("Expected unique provider name to be: 'provider2', got: '%s'", provider.UniqueName)
		}
		if provider.Provider != testProvider2 {
			t.Errorf("Expected unique provider name to be: 'provider2', got: '%s'", provider.UniqueName)
		}
	} else {
		t.Errorf("Expected there to be a provider with the key of '%s', but none was found.", "provider2")
	}

}

func TestMultiProvider_ProviderByNameMethod(t *testing.T) {
	testProvider1 := oft.NewTestProvider()
	testProvider2 := oft.NewTestProvider()

	defaultLogger, err := hooks.NewLoggingHook(false)
	if err != nil {
		t.Errorf("Issue setting up logger,'%s'", err)
	}

	mp, err := NewMultiProvider([]UniqueNameProvider{
		{
			Provider:   testProvider1,
			UniqueName: "provider1",
		}, {
			Provider:   testProvider2,
			UniqueName: "provider2",
		},
	}, "test", defaultLogger)

	if err != nil {
		t.Errorf("Expected the multiprovider to successfully make an instance, '%s'", err)
	}

	providers := mp.ProvidersByName()

	if len(providers) != 2 {
		t.Errorf("Expected there to be '2' providers as passed but got: '%d'", len(providers))
	}
	if provider, exists := mp.ProviderByName("provider2"); exists {
		if provider.UniqueName != "provider2" {
			t.Errorf("Expected unique provider name to be: 'provider2', got: '%s'", provider.UniqueName)
		}
		if provider.Provider != testProvider2 {
			t.Errorf("Expected unique provider name to be: 'provider2', got: '%s'", provider.UniqueName)
		}
	} else {
		t.Errorf("Expected there to be a provider with the key of '%s', but none was found.", "provider1")
	}

}

// todo: currently the `multiProvider.Metadata()` just give the `Name` of the multi provider it doesn't aggregate the passed providers as stated in this specification https://openfeature.dev/specification/appendix-a/#metadata so this test fails
func TestMultiProvider_MetaData(t *testing.T) {
	testProvider1 := oft.NewTestProvider()
	testProvider2 := oft.NewTestProvider()

	defaultLogger, err := hooks.NewLoggingHook(false)
	if err != nil {
		t.Errorf("Issue setting up logger,'%s'", err)
	}

	mp, err := NewMultiProvider([]UniqueNameProvider{
		{
			Provider:   testProvider1,
			UniqueName: "provider1",
		}, {
			Provider:   testProvider2,
			UniqueName: "provider2",
		},
	}, "test", defaultLogger)

	if err != nil {
		t.Errorf("Expected the multiprovider to successfully make an instance, '%s'", err)
	}

	expectedMetadata := MultiMetadata{
		Name: "multiprovider",
		OriginalMetadata: map[string]openfeature.Metadata{
			"provider1": openfeature.Metadata{Name: "NoopProvider"},
			"provider2": openfeature.Metadata{Name: "NoopProvider"},
		},
	}

	if mp.Metadata().Name != "hi" {
		t.Errorf("Expected to see the aggregated metadata of all passed providers: '%s', got: '%s'", expectedMetadata, mp.Metadata().Name)
	}
}

func TestMultiProvider_Init(t *testing.T) {
	initializations := 0
	shutdowns := 0

	testProvider1 := NewMockProvider(&initializations, &shutdowns)
	testProvider2 := oft.NewTestProvider()
	testProvider3 := NewMockProviderDelay(&initializations, &shutdowns)

	defaultLogger, err := hooks.NewLoggingHook(false)
	if err != nil {
		t.Errorf("Issue setting up logger,'%s'", err)
	}

	mp, err := NewMultiProvider([]UniqueNameProvider{
		{
			Provider:   testProvider1,
			UniqueName: "provider1",
		}, {
			Provider:   testProvider2,
			UniqueName: "provider2",
		}, {
			Provider:   testProvider3,
			UniqueName: "provider3",
		},
	}, "test", defaultLogger)

	if err != nil {
		t.Errorf("Expected the multiprovider to successfully make an instance, '%s'", err)
	}

	attributes := map[string]interface{}{
		"foo": "bar",
	}
	evalCtx := openfeature.NewTargetlessEvaluationContext(attributes)

	err = mp.Init(evalCtx)
	if err != nil {
		t.Errorf("Expected the initialization process to be successful, got error: '%s'", err)
	}

	if initializations == 0 {
		t.Errorf("Expected there to be initializations, but none were ran.")
	}

	if initializations != 2 {
		t.Errorf("Expected there to be '2' init steps ran, but got: '%d'.", initializations)
	}

}

func TestMultiProvider_Shutdown(t *testing.T) {
	initializations := 0
	shutdowns := 0

	testProvider1 := NewMockProvider(&initializations, &shutdowns)
	testProvider2 := oft.NewTestProvider()
	testProvider3 := NewMockProviderDelay(&initializations, &shutdowns)

	defaultLogger, err := hooks.NewLoggingHook(false)
	if err != nil {
		t.Errorf("Issue setting up logger,'%s'", err)
	}

	mp, err := NewMultiProvider([]UniqueNameProvider{
		{
			Provider:   testProvider1,
			UniqueName: "provider1",
		}, {
			Provider:   testProvider2,
			UniqueName: "provider2",
		}, {
			Provider:   testProvider3,
			UniqueName: "provider3",
		},
	}, "test", defaultLogger)

	if err != nil {
		t.Errorf("Expected the multiprovider to successfully make an instance, '%s'", err)
	}

	mp.Shutdown()

	if shutdowns == 0 {
		t.Errorf("Expected there to be shutdowns, but none were ran.")
	}

	if shutdowns != 2 {
		t.Errorf("Expected there to be '2' shutdown steps ran, but got: '%d'.", shutdowns)
	}
}

func TestNewMultiProvider_ProviderUniqueNames(t *testing.T) {
	testProvider1 := memprovider.NewInMemoryProvider(map[string]memprovider.InMemoryFlag{
		"boolFlag": {
			Key:            "boolFlag",
			State:          memprovider.Enabled,
			DefaultVariant: "true",
			Variants: map[string]interface{}{
				"true":  true,
				"false": false,
			},
			ContextEvaluator: nil,
		},
	})
	testProvider2 := oft.NewTestProvider()

	defaultLogger, err := hooks.NewLoggingHook(false)
	if err != nil {
		t.Errorf("Issue setting up logger,'%s'", err)
	}

	mp, err := NewMultiProvider([]UniqueNameProvider{
		{
			Provider: testProvider1,
		}, {
			Provider: testProvider2,
		},
	}, "test", defaultLogger)

	if err != nil {
		t.Errorf("Expected the multiprovider to successfully make an instance, '%s'", err)
	}

	providerEntries := mp.Providers()

	if providerEntries[0].UniqueName != "InMemoryProvider" {
		t.Errorf("Expected unique provider name to be: 'InMemoryProvider', got: '%s'", providerEntries[0].UniqueName)
	}
	if providerEntries[1].UniqueName != "NoopProvider" {
		t.Errorf("Expected unique provider name to be: 'NoopProvider', got: '%s'", providerEntries[1].UniqueName)
	}

	if len(providerEntries) != 2 {
		t.Errorf("Expected there to be 2 provider entries, got: '%d'", len(providerEntries))
	}
}

func TestNewMultiProvider_DuplicateProviderGenerateUniqueNames(t *testing.T) {
	testProvider1 := memprovider.NewInMemoryProvider(map[string]memprovider.InMemoryFlag{
		"boolFlag": {
			Key:            "boolFlag",
			State:          memprovider.Enabled,
			DefaultVariant: "true",
			Variants: map[string]interface{}{
				"true":  true,
				"false": false,
			},
			ContextEvaluator: nil,
		},
	})
	testProvider2 := oft.NewTestProvider()
	testProvider3 := oft.NewTestProvider()
	testProvider4 := oft.NewTestProvider()

	defaultLogger, err := hooks.NewLoggingHook(false)
	if err != nil {
		t.Errorf("Issue setting up logger,'%s'", err)
	}

	mp, err := NewMultiProvider([]UniqueNameProvider{
		{
			Provider: testProvider1,
		}, {
			Provider: testProvider2,
		}, {
			Provider: testProvider3,
		}, {
			Provider: testProvider4,
		},
	}, "test", defaultLogger)

	if err != nil {
		t.Errorf("Expected the multiprovider to successfully make an instance, '%s'", err)
	}

	providerEntries := mp.Providers()

	if providerEntries[0].UniqueName != "InMemoryProvider" {
		t.Errorf("Expected unique provider name to be: 'InMemoryProvider', got: '%s'", providerEntries[0].UniqueName)
	}
	if providerEntries[1].UniqueName != "NoopProvider-1" {
		t.Errorf("Expected unique provider name to be: 'NoopProvider', got: '%s'", providerEntries[1].UniqueName)
	}
	if providerEntries[2].UniqueName != "NoopProvider-2" {
		t.Errorf("Expected unique provider name to be: 'NoopProvider', got: '%s'", providerEntries[2].UniqueName)
	}
	if providerEntries[3].UniqueName != "NoopProvider-3" {
		t.Errorf("Expected unique provider name to be: 'NoopProvider', got: '%s'", providerEntries[3].UniqueName)
	}

	if len(providerEntries) != 4 {
		t.Errorf("Expected there to be 4 provider entries, got: '%d'", len(providerEntries))
	}
}
func TestNewMultiProvider_ProvidersUsePassedNames(t *testing.T) {
	testProvider1 := oft.NewTestProvider()
	testProvider2 := oft.NewTestProvider()

	defaultLogger, err := hooks.NewLoggingHook(false)
	if err != nil {
		t.Errorf("Issue setting up logger,'%s'", err)
	}

	mp, err := NewMultiProvider([]UniqueNameProvider{
		{
			Provider:   testProvider1,
			UniqueName: "theFirst",
		}, {
			Provider:   testProvider2,
			UniqueName: "theSecond",
		},
	}, "test", defaultLogger)

	if err != nil {
		t.Errorf("Expected the multiprovider to successfully make an instance, '%s'", err)
	}

	providerEntries := mp.Providers()

	if providerEntries[0].UniqueName != "theFirst" {
		t.Errorf("Expected unique provider name to be: 'theFirst', got: '%s'", providerEntries[0].UniqueName)
	}
	if providerEntries[1].UniqueName != "theSecond" {
		t.Errorf("Expected unique provider name to be: 'theSecond', got: '%s'", providerEntries[1].UniqueName)
	}

	if len(providerEntries) != 2 {
		t.Errorf("Expected there to be 2 provider entries, got: '%d'", len(providerEntries))
	}
}

func TestNewMultiProvider_ProvidersErrorNameNotUnique(t *testing.T) {
	testProvider1 := oft.NewTestProvider()
	testProvider2 := oft.NewTestProvider()

	defaultLogger, err := hooks.NewLoggingHook(false)
	if err != nil {
		t.Errorf("Issue setting up logger,'%s'", err)
	}

	_, err = NewMultiProvider([]UniqueNameProvider{
		{
			Provider:   testProvider1,
			UniqueName: "provider",
		}, {
			Provider:   testProvider2,
			UniqueName: "provider",
		},
	}, "test", defaultLogger)

	if err == nil {
		t.Errorf("Expected the multiprovider to have an error")
	}

	if err.Error() != "provider names must be unique" {
		t.Errorf("Expected the multiprovider to have an error of: '%s', got: '%s'", errUniqueName, err.Error())
	}
}
