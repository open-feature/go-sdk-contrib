package flagd

import (
	"reflect"
	"testing"
	"time"

	"github.com/open-feature/flagd/core/pkg/sync"
	"github.com/open-feature/go-sdk-contrib/providers/flagd/internal/cache"
	"github.com/open-feature/go-sdk-contrib/providers/flagd/internal/mock"
	process "github.com/open-feature/go-sdk-contrib/providers/flagd/pkg/service/in_process"
	of "github.com/open-feature/go-sdk/openfeature"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TestNewProvider(t *testing.T) {
	t.Parallel()
	customSyncProvider := process.NewDoNothingCustomSyncProvider()
	gRPCDialOptionOverride := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithAuthority("test-authority"),
	}

	tests := []struct {
		name                          string
		expectedResolver              ResolverType
		expectPort                    uint16
		expectHost                    string
		expectTargetUri               string
		expectCacheType               cache.Type
		expectCertPath                string
		expectMaxRetries              int
		expectCacheSize               int
		expectOtelIntercept           bool
		expectSocketPath              string
		expectTlsEnabled              bool
		expectProviderID              string
		expectSelector                string
		expectCustomSyncProvider      sync.ISync
		expectCustomSyncProviderUri   string
		expectOfflineFlagSourcePath   string
		expectGrpcDialOptionsOverride []grpc.DialOption
		options                       []ProviderOption
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
			name:                        "with custom sync provider and uri with in-process resolver",
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
				WithCustomSyncProviderAndUri(customSyncProvider, "testsyncer://custom.uri"),
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
			expectCustomSyncProviderUri: defaultCustomSyncProviderUri,
			options: []ProviderOption{
				WithInProcessResolver(),
				WithCustomSyncProvider(customSyncProvider),
			},
		},
		{
			name:                          "with gRPC DialOptions override with in-process resolver",
			expectedResolver:              inProcess,
			expectHost:                    defaultHost,
			expectPort:                    defaultInProcessPort,
			expectCacheType:               defaultCache,
			expectCacheSize:               defaultMaxCacheSize,
			expectMaxRetries:              defaultMaxEventStreamRetries,
			expectGrpcDialOptionsOverride: gRPCDialOptionOverride,
			options: []ProviderOption{
				WithInProcessResolver(),
				WithGrpcDialOptionsOverride(gRPCDialOptionOverride),
			},
		},
		{
			name:             "with selector and providerID with in-process resolver",
			expectedResolver: inProcess,
			expectHost:       defaultHost,
			expectPort:       defaultInProcessPort,
			expectCacheType:  defaultCache,
			expectCacheSize:  defaultMaxCacheSize,
			expectMaxRetries: defaultMaxEventStreamRetries,
			expectProviderID: "testProvider",
			expectSelector:   "flags=test",
			options: []ProviderOption{
				WithInProcessResolver(),
				WithSelector("flags=test"),
				WithProviderID("testProvider"),
			},
		},
		{
			name:             "with selector only with in-process resolver",
			expectedResolver: inProcess,
			expectHost:       defaultHost,
			expectPort:       defaultInProcessPort,
			expectCacheType:  defaultCache,
			expectCacheSize:  defaultMaxCacheSize,
			expectMaxRetries: defaultMaxEventStreamRetries,
			expectSelector:   "flags=test",
			options: []ProviderOption{
				WithInProcessResolver(),
				WithSelector("flags=test"),
			},
		},
		{
			name:             "with providerID only with in-process resolver",
			expectedResolver: inProcess,
			expectHost:       defaultHost,
			expectPort:       defaultInProcessPort,
			expectCacheType:  defaultCache,
			expectCacheSize:  defaultMaxCacheSize,
			expectMaxRetries: defaultMaxEventStreamRetries,
			expectProviderID: "testProvider",
			options: []ProviderOption{
				WithInProcessResolver(),
				WithProviderID("testProvider"),
			},
		},
		{
			name:                        "with OfflineFilePath with in-process resolver",
			expectedResolver:            file,
			expectHost:                  defaultHost,
			expectPort:                  0,
			expectCacheType:             defaultCache,
			expectCacheSize:             defaultMaxCacheSize,
			expectMaxRetries:            defaultMaxEventStreamRetries,
			expectOfflineFlagSourcePath: "offlineFilePath",
			options: []ProviderOption{
				WithInProcessResolver(),
				WithOfflineFilePath("offlineFilePath"),
			},
		},
		{
			name:                        "with OfflineFilePath with file resolver",
			expectedResolver:            file,
			expectHost:                  defaultHost,
			expectPort:                  0,
			expectCacheType:             defaultCache,
			expectCacheSize:             defaultMaxCacheSize,
			expectMaxRetries:            defaultMaxEventStreamRetries,
			expectOfflineFlagSourcePath: "offlineFilePath",
			options: []ProviderOption{
				WithFileResolver(),
				WithOfflineFilePath("offlineFilePath"),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			flagdProvider, err := NewProvider(test.options...)

			if err != nil {
				t.Fatal("error creating new provider", err)
			}

			metadata := flagdProvider.Metadata()
			if metadata.Name != "flagd" {
				t.Errorf("received unexpected metadata from NewProvider, expected %s got %s", "flagd", metadata.Name)
			}

			config := flagdProvider.providerConfiguration

			if config.Tls != test.expectTlsEnabled {
				t.Errorf("incorrect configuration Tls, expected %v, got %v",
					test.expectTlsEnabled, config.Tls)
			}

			if config.CertPath != test.expectCertPath {
				t.Errorf("incorrect configuration CertPath, expected %v, got %v",
					test.expectCertPath, config.CertPath)
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

			if config.Cache != test.expectCacheType {
				t.Errorf("incorrect configuration Cache, expected %v, got %v",
					test.expectCacheType, config.Cache)
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

			if config.Selector != test.expectSelector {
				t.Errorf("incorrect configuration Selector, expected %v, got %v",
					test.expectSelector, config.Selector)
			}

			if config.ProviderId != test.expectProviderID {
				t.Errorf("incorrect configuration ProviderId, expected %v, got %v",
					test.expectProviderID, config.ProviderId)
			}

			if config.CustomSyncProvider != test.expectCustomSyncProvider {
				t.Errorf("incorrect configuration CustomSyncProvider, expected %v, got %v",
					test.expectCustomSyncProvider, config.CustomSyncProvider)
			}

			if config.CustomSyncProviderUri != test.expectCustomSyncProviderUri {
				t.Errorf("incorrect configuration CustomSyncProviderUri, expected %v, got %v",
					test.expectCustomSyncProviderUri, config.CustomSyncProviderUri)
			}

			if config.OfflineFlagSourcePath != test.expectOfflineFlagSourcePath {
				t.Errorf("incorrect configuration OfflineFlagSourcePath, expected %v, got %v",
					test.expectOfflineFlagSourcePath, config.OfflineFlagSourcePath)
			}

			if test.expectGrpcDialOptionsOverride != nil {
				if config.GrpcDialOptionsOverride == nil {
					t.Errorf("incorrent configuration GrpcDialOptionsOverride, expected %v, got nil", config.GrpcDialOptionsOverride)
				} else if !reflect.DeepEqual(config.GrpcDialOptionsOverride, test.expectGrpcDialOptionsOverride) {
					t.Errorf("incorrent configuration GrpcDialOptionsOverride, expected %v, got %v", test.expectGrpcDialOptionsOverride, config.GrpcDialOptionsOverride)
				}
			} else {
				if config.GrpcDialOptionsOverride != nil {
					t.Errorf("incorrent configuration GrpcDialOptionsOverride, expected nil, got %v", config.GrpcDialOptionsOverride)
				}
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

	var err error

	provider, err := NewProvider()
	provider.service = svcMock

	// Set up mock expectations after injecting the mock
	svcMock.EXPECT().EventChannel().Return(customChan).AnyTimes()
	svcMock.EXPECT().Init().Times(1)

	if err != nil {
		t.Fatal("error creating new provider", err)
	}

	if provider.Status() != of.NotReadyState {
		t.Errorf("expected initial status to be not ready, but got %v", provider.Status())
	}

	// push events to local event channel
	done := make(chan struct{})
	go func() {
		defer close(done)
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
	err = provider.Init(of.EvaluationContext{})
	if err != nil {
		t.Fatal("error initialization provider", err)
	}

	// check event emitting from provider in order
	event := <-provider.EventChannel()
	if event.EventType != of.ProviderConfigChange {
		t.Errorf("expected event %v, got %v", of.ProviderReady, event.EventType)
	}

	// Wait for status to be updated asynchronously
	// The status update happens in handleEvents() after sending the event
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if provider.Status() == of.ReadyState {
			break
		}
	}

	if provider.Status() != of.ReadyState {
		t.Errorf("expected status to be ready, but got %v", provider.Status())
	}

	event = <-provider.EventChannel()
	if event.EventType != of.ProviderError {
		t.Errorf("expected event %v, got %v", of.ProviderError, event.EventType)
	}

	// Wait for status to be updated asynchronously
	// The status update happens in handleEvents() after sending the event
	deadline2 := time.Now().Add(time.Second)
	for time.Now().Before(deadline2) {
		if provider.Status() == of.ErrorState {
			break
		}
	}

	if provider.Status() != of.ErrorState {
		t.Errorf("expected status to be error, but got %v", provider.Status())
	}

	// Wait for goroutine to complete
	<-done
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

	provider, err := NewProvider()
	provider.service = svcMock

	if err != nil {
		t.Fatal("error creating new provider", err)
	}

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

func TestInitDeadlineExceeded(t *testing.T) {
	// given
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	eventChan := make(chan of.Event)

	svcMock := mock.NewMockIService(ctrl)
	svcMock.EXPECT().Init().Times(1)
	svcMock.EXPECT().EventChannel().Return(eventChan).MaxTimes(1)

	// Create provider with short deadline
	provider, err := NewProvider(WithDeadline(100)) // 100ms deadline
	provider.service = svcMock

	if err != nil {
		t.Fatal("error creating new provider", err)
	}

	// Do not send any events, let it timeout
	err = provider.Init(of.EvaluationContext{})

	if err == nil {
		t.Fatal("expected error from deadline exceeded, but got nil")
	}

	// Verify error message contains deadline info
	if err.Error() != "provider initialization deadline exceeded (100ms)" {
		t.Errorf("expected deadline error message, got: %v", err)
	}
}

func TestInitProviderErrorEvent(t *testing.T) {
	// given
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	eventChan := make(chan of.Event)

	svcMock := mock.NewMockIService(ctrl)
	svcMock.EXPECT().Init().Times(1)
	svcMock.EXPECT().EventChannel().Return(eventChan).MaxTimes(1)

	provider, err := NewProvider()
	provider.service = svcMock

	if err != nil {
		t.Fatal("error creating new provider", err)
	}

	// Send a ProviderError event instead of ProviderReady
	go func() {
		eventChan <- of.Event{
			ProviderName: "flagd",
			EventType:    of.ProviderError,
			ProviderEventDetails: of.ProviderEventDetails{
				Message: "connection failed",
			},
		}
	}()

	err = provider.Init(of.EvaluationContext{})

	if err == nil {
		t.Fatal("expected error from provider error event, but got nil")
	}

	if err.Error() != "provider initialization failed: connection failed" {
		t.Errorf("expected error message 'provider initialization failed: connection failed', got: %v", err)
	}
}

func TestInitProviderStaleEvent(t *testing.T) {
	// given
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	eventChan := make(chan of.Event)

	svcMock := mock.NewMockIService(ctrl)
	svcMock.EXPECT().Init().Times(1)
	svcMock.EXPECT().EventChannel().Return(eventChan).MaxTimes(1)

	provider, err := NewProvider()
	provider.service = svcMock

	if err != nil {
		t.Fatal("error creating new provider", err)
	}

	// Send a ProviderStale event instead of ProviderReady
	go func() {
		eventChan <- of.Event{
			ProviderName: "flagd",
			EventType:    of.ProviderStale,
			ProviderEventDetails: of.ProviderEventDetails{
				Message: "connection stale",
			},
		}
	}()

	err = provider.Init(of.EvaluationContext{})

	if err == nil {
		t.Fatal("expected error from provider stale event, but got nil")
	}

	if err.Error() != "provider initialization failed: connection stale" {
		t.Errorf("expected error message 'provider initialization failed: connection stale', got: %v", err)
	}

	if provider.initialized {
		t.Errorf("expected provider to not be initialized after error event")
	}
}

func TestInitWithCustomDeadline(t *testing.T) {
	// given
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	eventChan := make(chan of.Event)

	svcMock := mock.NewMockIService(ctrl)
	svcMock.EXPECT().Init().Times(1)
	svcMock.EXPECT().EventChannel().Return(eventChan).AnyTimes()
	svcMock.EXPECT().Shutdown().Times(1)

	// Create provider with custom longer deadline
	provider, err := NewProvider(WithDeadline(500)) // 500ms deadline
	provider.service = svcMock

	if err != nil {
		t.Fatal("error creating new provider", err)
	}

	// Send ProviderReady before deadline
	go func() {
		eventChan <- of.Event{
			ProviderName: "flagd",
			EventType:    of.ProviderReady,
		}
	}()

	err = provider.Init(of.EvaluationContext{})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if !provider.initialized {
		t.Errorf("expected provider to be initialized")
	}

	// Clean up to avoid affecting other tests
	provider.Shutdown()
	close(eventChan)
}

func TestHandleEventsChannelClose(t *testing.T) {
	// given
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	eventChan := make(chan of.Event)

	svcMock := mock.NewMockIService(ctrl)
	svcMock.EXPECT().Init().Times(1)
	// Allow unlimited calls to EventChannel since the event loop will call it after close
	svcMock.EXPECT().EventChannel().Return(eventChan).AnyTimes()
	svcMock.EXPECT().Shutdown().Times(1)

	provider, err := NewProvider()
	provider.service = svcMock

	if err != nil {
		t.Fatal("error creating new provider", err)
	}

	// Send ProviderReady to start event loop
	go func() {
		eventChan <- of.Event{
			ProviderName: "flagd",
			EventType:    of.ProviderReady,
		}
		// Give time for handleEvents to start
		time.Sleep(10 * time.Millisecond)
		// Close the channel
		close(eventChan)
	}()

	err = provider.Init(of.EvaluationContext{})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Give time for event loop to finish
	time.Sleep(50 * time.Millisecond)

	if !provider.initialized {
		t.Errorf("expected provider to be initialized")
	}

	// Clean up to avoid affecting other tests
	provider.Shutdown()
}
