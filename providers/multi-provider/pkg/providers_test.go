package multiprovider

import (
	"testing"

	// "github.com/open-feature/go-sdk/openfeature"
	"github.com/open-feature/go-sdk/openfeature/hooks"
	"github.com/open-feature/go-sdk/openfeature/memprovider"
	oft "github.com/open-feature/go-sdk/openfeature/testing"
)

func TestNewMultiProvider_ProviderMetadataUniqueNames(t *testing.T) {
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

	multiProvider, err := NewMultiProvider([]UniqueNameProvider{
		{
			Provider:   testProvider1,
		}, {
			Provider: testProvider2,
		},
	}, "test", defaultLogger)

	if err != nil {
		t.Errorf("Expected the multiprovider to successfully make an instance, '%s'", err)
	}

	providerEntries := multiProvider.Providers()

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

	multiProvider, err := NewMultiProvider([]UniqueNameProvider{
		{
			Provider:   testProvider1,
		}, {
			Provider: testProvider2,
		}, {
			Provider: testProvider3,
		},{
			Provider: testProvider4,
		},
	}, "test", defaultLogger)

	if err != nil {
		t.Errorf("Expected the multiprovider to successfully make an instance, '%s'", err)
	}

	providerEntries := multiProvider.Providers()

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
func TestNewMultiProvider_(t *testing.T) {
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

	multiProvider, err := NewMultiProvider([]UniqueNameProvider{
		{
			Provider:   testProvider1,
		}, {
			Provider: testProvider2,
		}, {
			Provider: testProvider3,
		},{
			Provider: testProvider4,
		},
	}, "test", defaultLogger)

	if err != nil {
		t.Errorf("Expected the multiprovider to successfully make an instance, '%s'", err)
	}

	if multiProvider.providersEntries[0].UniqueName != "InMemoryProvider" {
		t.Errorf("Expected unique provider name to be: 'InMemoryProvider', got: '%s'", multiProvider.providersEntries[0].UniqueName)
	}
	if multiProvider.providersEntries[1].UniqueName != "NoopProvider-1" {
		t.Errorf("Expected unique provider name to be: 'NoopProvider', got: '%s'", multiProvider.providersEntries[1].UniqueName)
	}
	if multiProvider.providersEntries[2].UniqueName != "NoopProvider-2" {
		t.Errorf("Expected unique provider name to be: 'NoopProvider', got: '%s'", multiProvider.providersEntries[2].UniqueName)
	}
	if multiProvider.providersEntries[3].UniqueName != "NoopProvider-3" {
		t.Errorf("Expected unique provider name to be: 'NoopProvider', got: '%s'", multiProvider.providersEntries[3].UniqueName)
	}
}
