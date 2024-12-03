package flagd

import (
	"testing"

	"github.com/open-feature/flagd/core/pkg/sync"
	"github.com/open-feature/go-sdk-contrib/providers/flagd/internal/cache"
	"github.com/open-feature/go-sdk-contrib/providers/flagd/internal/mock"
	process "github.com/open-feature/go-sdk-contrib/providers/flagd/pkg/service/in_process"
	of "github.com/open-feature/go-sdk/openfeature"
	"go.uber.org/mock/gomock"
)

func TestNewProvider(t *testing.T) {
	customSyncProvider := process.NewDoNothingCustomSyncProvider()

	tests := []struct {
		name                        string
		expectedResolver            ResolverType
		expectPort                  uint16
		expectHost                  string
		expectTargetUri             string
		expectCacheType             cache.Type
		expectCertPath              string
		expectMaxRetries            int
		expectCacheSize             int
		expectOtelIntercept         bool
		expectSocketPath            string
		expectTlsEnabled            bool
		expectCustomSyncProvider    sync.ISync
		expectCustomSyncProviderUri string
		options                     []ProviderOption
	}{
		{
			name:                        "default construction",
			expectedResolver:            rpc,
			expectPort:                  defaultRpcPort,
			expectHost:                  defaultHost,
			expectTargetUri:             "",
			expectCacheType:             defaultCache,
			expectCertPath:              "",
			expectMaxRetries:            defaultMaxEventStreamRetries,
			expectCacheSize:             defaultMaxCacheSize,
			expectOtelIntercept:         false,
			expectSocketPath:            "",
			expectTlsEnabled:            false,
			expectCustomSyncProvider:    nil,
			expectCustomSyncProviderUri: "",
		},
		{
			name:                        "with options",
			expectedResolver:            inProcess,
			expectPort:                  9090,
			expectHost:                  "myHost",
			expectTargetUri:             "",
			expectCacheType:             cache.LRUValue,
			expectCertPath:              "/path",
			expectMaxRetries:            2,
			expectCacheSize:             2500,
			expectOtelIntercept:         true,
			expectSocketPath:            "/socket",
			expectTlsEnabled:            true,
			expectCustomSyncProvider:    nil,
			expectCustomSyncProviderUri: "",
			options: []ProviderOption{
				WithInProcessResolver(),
				WithSocketPath("/socket"),
				WithOtelInterceptor(true),
				WithLRUCache(2500),
				WithEventStreamConnectionMaxAttempts(2),
				WithCertificatePath("/path"),
				WithHost("myHost"),
				WithPort(9090),
			},
		},
		{
			name:                        "default port handling with in-process resolver",
			expectedResolver:            inProcess,
			expectPort:                  defaultInProcessPort,
			expectHost:                  defaultHost,
			expectCacheType:             defaultCache,
			expectTargetUri:             "",
			expectCertPath:              "",
			expectMaxRetries:            defaultMaxEventStreamRetries,
			expectCacheSize:             defaultMaxCacheSize,
			expectOtelIntercept:         false,
			expectSocketPath:            "",
			expectTlsEnabled:            false,
			expectCustomSyncProvider:    nil,
			expectCustomSyncProviderUri: "",
			options: []ProviderOption{
				WithInProcessResolver(),
			},
		},
		{
			name:                        "default port handling with in-process resolver",
			expectedResolver:            rpc,
			expectPort:                  defaultRpcPort,
			expectHost:                  defaultHost,
			expectTargetUri:             "",
			expectCacheType:             defaultCache,
			expectCertPath:              "",
			expectMaxRetries:            defaultMaxEventStreamRetries,
			expectCacheSize:             defaultMaxCacheSize,
			expectOtelIntercept:         false,
			expectSocketPath:            "",
			expectTlsEnabled:            false,
			expectCustomSyncProvider:    nil,
			expectCustomSyncProviderUri: "",
			options: []ProviderOption{
				WithRPCResolver(),
			},
		},
		{
			name:                        "with target uri with in-process resolver",
			expectedResolver:            inProcess,
			expectPort:                  defaultInProcessPort,
			expectHost:                  defaultHost,
			expectCacheType:             defaultCache,
			expectTargetUri:             "envoy://localhost:9211/test.service",
			expectCertPath:              "",
			expectMaxRetries:            defaultMaxEventStreamRetries,
			expectCacheSize:             defaultMaxCacheSize,
			expectOtelIntercept:         false,
			expectSocketPath:            "",
			expectTlsEnabled:            false,
			expectCustomSyncProvider:    nil,
			expectCustomSyncProviderUri: "",
			options: []ProviderOption{
				WithInProcessResolver(),
				WithTargetUri("envoy://localhost:9211/test.service"),
			},
		},
		{
			name:                        "with custom sync provider with in-process resolver",
			expectedResolver:            inProcess,
			expectPort:                  defaultInProcessPort,
			expectHost:                  defaultHost,
			expectCacheType:             defaultCache,
			expectTargetUri:             "",
			expectCertPath:              "",
			expectMaxRetries:            defaultMaxEventStreamRetries,
			expectCacheSize:             defaultMaxCacheSize,
			expectOtelIntercept:         false,
			expectSocketPath:            "",
			expectTlsEnabled:            false,
			expectCustomSyncProvider:    customSyncProvider,
			expectCustomSyncProviderUri: "testsyncer://custom.uri",
			options: []ProviderOption{
				WithInProcessResolver(),
				WithCustomSyncProvider(customSyncProvider, "testsyncer://custom.uri"),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			flagdProvider := NewProvider(test.options...)

			metadata := flagdProvider.Metadata()
			if metadata.Name != "flagd" {
				t.Errorf("received unexpected metadata from NewProvider, expected %s got %s", "flagd", metadata.Name)
			}

			config := flagdProvider.providerConfiguration

			if config.TLSEnabled != test.expectTlsEnabled {
				t.Errorf("incorrect configuration TLSEnabled, expected %v, got %v",
					test.expectTlsEnabled, config.TLSEnabled)
			}

			if config.CertificatePath != test.expectCertPath {
				t.Errorf("incorrect configuration CertificatePath, expected %v, got %v",
					test.expectCertPath, config.CertificatePath)
			}

			if config.OtelIntercept != test.expectOtelIntercept {
				t.Errorf("incorrect configuration OtelIntercept, expected %v, got %v",
					test.expectOtelIntercept, config.OtelIntercept)
			}

			if config.EventStreamConnectionMaxAttempts != test.expectMaxRetries {
				t.Errorf("incorrect configuration EventStreamConnectionMaxAttempts, expected %v, got %v",
					test.expectMaxRetries, config.EventStreamConnectionMaxAttempts)
			}

			if config.MaxCacheSize != test.expectCacheSize {
				t.Errorf("incorrect configuration MaxCacheSize, expected %v, got %v",
					test.expectCacheSize, config.MaxCacheSize)
			}

			if config.CacheType != test.expectCacheType {
				t.Errorf("incorrect configuration CacheType, expected %v, got %v",
					test.expectCacheType, config.CacheType)
			}

			if config.Host != test.expectHost {
				t.Errorf("incorrect configuration Host, expected %v, got %v",
					test.expectHost, config.Host)
			}

			if config.Port != test.expectPort {
				t.Errorf("incorrect configuration Port, expected %v, got %v",
					test.expectPort, config.Port)
			}

			if config.TargetUri != test.expectTargetUri {
				t.Errorf("incorrect configuration TargetUri, expected %v, got %v",
					test.expectTargetUri, config.TargetUri)
			}

			if config.CustomSyncProvider != test.expectCustomSyncProvider {
				t.Errorf("incorrect configuration CustomSyncProvider, expected %v, got %v",
					test.expectCustomSyncProvider, config.CustomSyncProvider)
			}

			if config.CustomSyncProviderUri != test.expectCustomSyncProviderUri {
				t.Errorf("incorrect configuration CustomSyncProviderUri, expected %v, got %v",
					test.expectCustomSyncProviderUri, config.CustomSyncProviderUri)
			}

			// this line will fail linting if this provider is no longer compatible with the openfeature sdk
			var _ of.FeatureProvider = flagdProvider
		})
	}
}

func TestEventHandling(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	customChan := make(chan of.Event)

	svcMock := mock.NewMockIService(ctrl)
	svcMock.EXPECT().EventChannel().Return(customChan).AnyTimes()
	svcMock.EXPECT().Init().Times(1)

	provider := NewProvider()
	provider.service = svcMock

	if provider.Status() != of.NotReadyState {
		t.Errorf("expected initial status to be not ready, but got %v", provider.Status())
	}

	// push events to local event channel
	go func() {
		// initial ready event
		customChan <- of.Event{
			ProviderName: "flagd",
			EventType:    of.ProviderReady,
		}

		customChan <- of.Event{
			ProviderName: "flagd",
			EventType:    of.ProviderConfigChange,
		}

		customChan <- of.Event{
			ProviderName: "flagd",
			EventType:    of.ProviderError,
		}
	}()

	// Check initial readiness
	err := provider.Init(of.EvaluationContext{})
	if err != nil {
		t.Fatal("error initialization provider", err)
	}

	// check event emitting from provider in order
	event := <-provider.EventChannel()
	if event.EventType != of.ProviderConfigChange {
		t.Errorf("expected event %v, got %v", of.ProviderReady, event.EventType)
	}

	if provider.Status() != of.ReadyState {
		t.Errorf("expected status to be ready, but got %v", provider.Status())
	}

	event = <-provider.EventChannel()
	if event.EventType != of.ProviderError {
		t.Errorf("expected event %v, got %v", of.ProviderError, event.EventType)
	}

	if provider.Status() != of.ErrorState {
		t.Errorf("expected status to be error, but got %v", provider.Status())
	}
}

func TestInitializeOnlyOnce(t *testing.T) {
	// given
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	eventChan := make(chan of.Event)

	svcMock := mock.NewMockIService(ctrl)
	svcMock.EXPECT().Init().Times(1)
	svcMock.EXPECT().EventChannel().Return(eventChan).MaxTimes(2)
	svcMock.EXPECT().Shutdown().Times(1)

	provider := NewProvider()
	provider.service = svcMock

	// make service ready with events
	go func() {
		eventChan <- of.Event{
			ProviderName: "mock provider",
			EventType:    of.ProviderReady,
		}
	}()

	// multiple init invokes
	_ = provider.Init(of.EvaluationContext{})
	_ = provider.Init(of.EvaluationContext{})

	if !provider.initialized {
		t.Errorf("expected provider to be ready, but got not ready")
	}

	// shutdown should make provider uninitialized
	provider.Shutdown()

	if provider.initialized {
		t.Errorf("expected provider to be not ready, but got ready")
	}

}
