package flagd

import (
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"sync/atomic"
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

	enrichment := make(map[string]any)
	enrichment["test"] = "test"

	expectedEnrichment := make(map[string]any)
	expectedEnrichment["test"] = "test"
	expectedEnrichment["test2"] = "test2"
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
		expectContextEnrichment       map[string]any
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
		{
			name:                    "with ContextEnricher inprocess resolver",
			expectedResolver:        inProcess,
			expectHost:              defaultHost,
			expectPort:              defaultInProcessPort,
			expectCacheType:         defaultCache,
			expectCacheSize:         defaultMaxCacheSize,
			expectMaxRetries:        defaultMaxEventStreamRetries,
			expectContextEnrichment: enrichment,
			options: []ProviderOption{
				WithInProcessResolver(),
				WithContextEnricher(func(m map[string]any) *of.EvaluationContext {
					context := of.NewTargetlessEvaluationContext(m)
					return &context
				}),
			},
		},
		{
			name:                    "with ContextEnricher inprocess resolver adding values",
			expectedResolver:        inProcess,
			expectHost:              defaultHost,
			expectPort:              defaultInProcessPort,
			expectCacheType:         defaultCache,
			expectCacheSize:         defaultMaxCacheSize,
			expectMaxRetries:        defaultMaxEventStreamRetries,
			expectContextEnrichment: expectedEnrichment,
			options: []ProviderOption{
				WithInProcessResolver(),
				WithContextEnricher(func(m map[string]any) *of.EvaluationContext {
					newMap := make(map[string]any, len(m)+1)
					for k, v := range m {
						newMap[k] = v
					}
					newMap["test2"] = "test2"
					context := of.NewTargetlessEvaluationContext(newMap)
					return &context
				}),
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

			if test.expectContextEnrichment != nil {
				enriched := config.ContextEnricher(enrichment).Attributes()
				if !reflect.DeepEqual(test.expectContextEnrichment, enriched) {
					t.Errorf("incorrect context_enrichment attribute, expected %v, got %v", test.expectContextEnrichment, enriched)
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

	provider, err := NewProvider()
	if err != nil {
		t.Fatal("error creating new provider", err)
	}

	svcMock := mock.NewMockIService(ctrl)
	provider.service = svcMock

	// Set up mock expectations after injecting the mock
	svcMock.EXPECT().EventChannel().Return(customChan).AnyTimes()

	// DoAndReturn because events are a consequence of Init() running
	done := make(chan struct{})
	svcMock.EXPECT().Init().DoAndReturn(func() error {
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
		return nil
	}).Times(1)

	if provider.Status() != of.NotReadyState {
		t.Errorf("expected initial status to be not ready, but got %v", provider.Status())
	}

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

	provider, err := NewProvider()
	if err != nil {
		t.Fatal("error creating new provider", err)
	}

	svcMock := mock.NewMockIService(ctrl)
	provider.service = svcMock
	svcMock.EXPECT().Init().DoAndReturn(func() error {
		go func() {
			eventChan <- of.Event{
				ProviderName: "mock provider",
				EventType:    of.ProviderReady,
			}
		}()
		return nil
	}).Times(1)
	svcMock.EXPECT().EventChannel().Return(eventChan).MaxTimes(2)
	svcMock.EXPECT().Shutdown().Times(1)

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

	// Create provider with short deadline
	provider, err := NewProvider(WithDeadline(100)) // 100ms deadline
	if err != nil {
		t.Fatal("error creating new provider", err)
	}

	svcMock := mock.NewMockIService(ctrl)
	provider.service = svcMock
	// Init blocks forever (no return) to simulate a hanging service
	svcMock.EXPECT().Init().DoAndReturn(func() error {
		<-make(chan struct{}) // block forever, let deadline expire
		return nil
	}).MaxTimes(1)
	svcMock.EXPECT().EventChannel().Return(eventChan).MaxTimes(1)

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

	provider, err := NewProvider()
	if err != nil {
		t.Fatal("error creating new provider", err)
	}

	svcMock := mock.NewMockIService(ctrl)
	provider.service = svcMock
	// Emit ProviderError from within Init()
	svcMock.EXPECT().Init().DoAndReturn(func() error {
		go func() {
			eventChan <- of.Event{
				ProviderName: "flagd",
				EventType:    of.ProviderError,
				ProviderEventDetails: of.ProviderEventDetails{
					Message: "connection failed",
				},
			}
		}()
		return nil
	}).Times(1)
	svcMock.EXPECT().EventChannel().Return(eventChan).MaxTimes(1)

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

	provider, err := NewProvider()
	if err != nil {
		t.Fatal("error creating new provider", err)
	}

	svcMock := mock.NewMockIService(ctrl)
	provider.service = svcMock
	// Emit ProviderStale from within Init()
	svcMock.EXPECT().Init().DoAndReturn(func() error {
		go func() {
			eventChan <- of.Event{
				ProviderName: "flagd",
				EventType:    of.ProviderStale,
				ProviderEventDetails: of.ProviderEventDetails{
					Message: "connection stale",
				},
			}
		}()
		return nil
	}).Times(1)
	svcMock.EXPECT().EventChannel().Return(eventChan).MaxTimes(1)

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

func TestEventHandlingAfterFailedInit(t *testing.T) {
	// given
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// buffered like the in-process service channel, so a missing consumer blocks the publisher
	eventChan := make(chan of.Event, 5)

	provider, err := NewProvider()
	if err != nil {
		t.Fatal("error creating new provider", err)
	}

	svcMock := mock.NewMockIService(ctrl)
	provider.service = svcMock
	svcMock.EXPECT().Init().DoAndReturn(func() error {
		eventChan <- of.Event{
			ProviderName: "flagd",
			EventType:    of.ProviderError,
			ProviderEventDetails: of.ProviderEventDetails{
				Message: "connection failed",
			},
		}
		return nil
	}).Times(1)
	svcMock.EXPECT().EventChannel().Return(eventChan).AnyTimes()

	err = provider.Init(of.EvaluationContext{})
	if err == nil {
		t.Fatal("expected error from provider error event, but got nil")
	}

	// the service keeps retrying after init gave up, then recovers
	published := make(chan struct{})
	go func() {
		defer close(published)
		for range 10 {
			eventChan <- of.Event{ProviderName: "flagd", EventType: of.ProviderError}
		}
		eventChan <- of.Event{ProviderName: "flagd", EventType: of.ProviderReady}
	}()

	timeout := time.After(time.Second)
	ready := false
	for !ready {
		select {
		case event := <-provider.EventChannel():
			ready = event.EventType == of.ProviderReady
		case <-timeout:
			t.Fatal("expected ProviderReady on the provider event channel after a failed init")
		}
	}

	select {
	case <-published:
	case <-timeout:
		t.Fatal("expected the service to publish all events without blocking")
	}

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) && provider.Status() != of.ReadyState {
		time.Sleep(time.Millisecond)
	}

	if provider.Status() != of.ReadyState {
		t.Errorf("expected status to be ready, but got %v", provider.Status())
	}
}

// countEventPumps returns how many handleEvents goroutines are currently running.
func countEventPumps() int {
	buf := make([]byte, 1<<20)
	buf = buf[:runtime.Stack(buf, true)]
	return strings.Count(string(buf), "(*Provider).handleEvents(")
}

// waitForEventPumps polls until want pumps run beyond baseline, since pumps start and stop asynchronously.
// baseline accounts for pumps leaked by other tests that never call Shutdown.
func waitForEventPumps(t *testing.T, baseline, want int) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for countEventPumps()-baseline != want {
		if time.Now().After(deadline) {
			t.Fatalf("expected %d event pump(s), got %d", want, countEventPumps()-baseline)
		}
		time.Sleep(time.Millisecond)
	}
}

// shutdownAndWait stops the provider and waits for its pump, so it cannot skew the next test's baseline.
func shutdownAndWait(t *testing.T, provider *Provider, baseline int) {
	t.Helper()
	provider.Shutdown()
	waitForEventPumps(t, baseline, 0)
}

func newFailingInitProvider(t *testing.T, ctrl *gomock.Controller) (*Provider, *mock.MockIService) {
	t.Helper()

	provider, err := NewProvider()
	if err != nil {
		t.Fatal("error creating new provider", err)
	}

	svcMock := mock.NewMockIService(ctrl)
	provider.service = svcMock
	svcMock.EXPECT().Init().Return(fmt.Errorf("connection failed")).AnyTimes()
	svcMock.EXPECT().EventChannel().Return(make(chan of.Event, 5)).AnyTimes()
	svcMock.EXPECT().Shutdown().AnyTimes()

	return provider, svcMock
}

func TestRetriedInitStartsSingleEventPump(t *testing.T) {
	baseline := countEventPumps()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	provider, _ := newFailingInitProvider(t, ctrl)
	defer shutdownAndWait(t, provider, baseline)

	for range 3 {
		if err := provider.Init(of.EvaluationContext{}); err == nil {
			t.Fatal("expected init to fail")
		}
	}

	waitForEventPumps(t, baseline, 1)
	// give a wrongly spawned extra pump time to show up
	time.Sleep(50 * time.Millisecond)
	waitForEventPumps(t, baseline, 1)
}

func TestRetriedInitReceivesReady(t *testing.T) {
	baseline := countEventPumps()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	eventChan := make(chan of.Event, 5)

	provider, err := NewProvider()
	if err != nil {
		t.Fatal("error creating new provider", err)
	}
	defer shutdownAndWait(t, provider, baseline)

	svcMock := mock.NewMockIService(ctrl)
	provider.service = svcMock
	svcMock.EXPECT().EventChannel().Return(eventChan).AnyTimes()
	svcMock.EXPECT().Shutdown().AnyTimes()

	var attempts atomic.Int32
	svcMock.EXPECT().Init().DoAndReturn(func() error {
		eventType := of.ProviderError
		if attempts.Add(1) == 2 {
			eventType = of.ProviderReady
		}
		eventChan <- of.Event{ProviderName: "flagd", EventType: eventType}
		return nil
	}).Times(2)

	if err := provider.Init(of.EvaluationContext{}); err == nil {
		t.Fatal("expected the first init to fail")
	}

	if err := provider.Init(of.EvaluationContext{}); err != nil {
		t.Fatal("expected the retried init to succeed, got", err)
	}
	if provider.Status() != of.ReadyState {
		t.Errorf("expected status to be ready, but got %v", provider.Status())
	}

	select {
	case event := <-provider.EventChannel():
		t.Errorf("unexpected event %v published", event.EventType)
	case <-time.After(50 * time.Millisecond):
	}
}

func TestRetriedInitNotBlockedByPumpHoldingEarlierEvent(t *testing.T) {
	baseline := countEventPumps()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	eventChan := make(chan of.Event, 5)

	provider, err := NewProvider()
	if err != nil {
		t.Fatal("error creating new provider", err)
	}
	defer shutdownAndWait(t, provider, baseline)

	svcMock := mock.NewMockIService(ctrl)
	provider.service = svcMock
	svcMock.EXPECT().EventChannel().Return(eventChan).AnyTimes()
	svcMock.EXPECT().Shutdown().AnyTimes()

	var attempts atomic.Int32
	retryStarted := make(chan struct{})
	svcMock.EXPECT().Init().DoAndReturn(func() error {
		if attempts.Add(1) == 1 {
			eventChan <- of.Event{ProviderName: "flagd", EventType: of.ProviderError}
			return nil
		}
		eventChan <- of.Event{ProviderName: "flagd", EventType: of.ProviderReady}
		close(retryStarted)
		return nil
	}).Times(2)

	if err := provider.Init(of.EvaluationContext{}); err == nil {
		t.Fatal("expected the first init to fail")
	}

	// the pump takes this event and blocks publishing it until the test reads the provider channel
	eventChan <- of.Event{ProviderName: "flagd", EventType: of.ProviderError}
	for len(eventChan) > 0 {
		time.Sleep(time.Millisecond)
	}
	time.Sleep(50 * time.Millisecond)

	retried := make(chan error, 1)
	go func() { retried <- provider.Init(of.EvaluationContext{}) }()

	// Init holds the lock now, so reading releases the pump into setStatus
	<-retryStarted
	<-provider.EventChannel()

	select {
	case err := <-retried:
		if err != nil {
			t.Fatal("expected the retried init to succeed, got", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("retried init never returned")
	}
}

func TestShutdownStopsEventPump(t *testing.T) {
	baseline := countEventPumps()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	provider, _ := newFailingInitProvider(t, ctrl)

	if err := provider.Init(of.EvaluationContext{}); err == nil {
		t.Fatal("expected init to fail")
	}
	waitForEventPumps(t, baseline, 1)

	provider.Shutdown()

	waitForEventPumps(t, baseline, 0)
}

func TestShutdownStopsEventPumpBlockedOnPublish(t *testing.T) {
	baseline := countEventPumps()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	eventChan := make(chan of.Event, 1)
	provider, svcMock := newFailingInitProvider(t, ctrl)
	svcMock.EXPECT().EventChannel().Return(eventChan).AnyTimes()

	if err := provider.Init(of.EvaluationContext{}); err == nil {
		t.Fatal("expected init to fail")
	}

	// nobody reads provider.EventChannel(), so the pump blocks forwarding this event
	eventChan <- of.Event{ProviderName: "flagd", EventType: of.ProviderError}
	time.Sleep(20 * time.Millisecond)

	provider.Shutdown()

	waitForEventPumps(t, baseline, 0)
}

func TestInitAfterShutdownRestartsEventPump(t *testing.T) {
	baseline := countEventPumps()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	provider, _ := newFailingInitProvider(t, ctrl)

	if err := provider.Init(of.EvaluationContext{}); err == nil {
		t.Fatal("expected init to fail")
	}
	provider.Shutdown()
	waitForEventPumps(t, baseline, 0)

	if err := provider.Init(of.EvaluationContext{}); err == nil {
		t.Fatal("expected init to fail")
	}
	waitForEventPumps(t, baseline, 1)

	provider.Shutdown()
	waitForEventPumps(t, baseline, 0)
}

func TestInitWithCustomDeadline(t *testing.T) {
	// given
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	eventChan := make(chan of.Event)

	// Create provider with custom longer deadline
	provider, err := NewProvider(WithDeadline(500)) // 500ms deadline
	if err != nil {
		t.Fatal("error creating new provider", err)
	}

	svcMock := mock.NewMockIService(ctrl)
	provider.service = svcMock
	svcMock.EXPECT().Init().DoAndReturn(func() error {
		go func() {
			eventChan <- of.Event{
				ProviderName: "flagd",
				EventType:    of.ProviderReady,
			}
		}()
		return nil
	}).Times(1)
	svcMock.EXPECT().EventChannel().Return(eventChan).AnyTimes()
	svcMock.EXPECT().Shutdown().Times(1)

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

	provider, err := NewProvider()
	if err != nil {
		t.Fatal("error creating new provider", err)
	}

	svcMock := mock.NewMockIService(ctrl)
	provider.service = svcMock
	svcMock.EXPECT().Init().DoAndReturn(func() error {
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
		return nil
	}).Times(1)
	// Allow unlimited calls to EventChannel since the event loop will call it after close
	svcMock.EXPECT().EventChannel().Return(eventChan).AnyTimes()
	svcMock.EXPECT().Shutdown().Times(1)

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

func TestStreamDeadlineAndKeepAliveDefaults(t *testing.T) {
	config, err := NewProviderConfiguration(nil)
	if err != nil {
		t.Fatal("error creating provider configuration", err)
	}

	if config.StreamDeadlineMs != defaultStreamDeadlineMs {
		t.Errorf("incorrect default StreamDeadlineMs, expected %d, got %d",
			defaultStreamDeadlineMs, config.StreamDeadlineMs)
	}
	if config.KeepAliveTime != defaultKeepAliveMs {
		t.Errorf("incorrect default KeepAliveTime, expected %d, got %d",
			defaultKeepAliveMs, config.KeepAliveTime)
	}
}

func TestStreamDeadlineAndKeepAliveOptions(t *testing.T) {
	config, err := NewProviderConfiguration([]ProviderOption{
		WithStreamDeadline(1234),
		WithKeepAliveTime(5678),
	})
	if err != nil {
		t.Fatal("error creating provider configuration", err)
	}

	if config.StreamDeadlineMs != 1234 {
		t.Errorf("incorrect StreamDeadlineMs from option, expected %d, got %d", 1234, config.StreamDeadlineMs)
	}
	if config.KeepAliveTime != 5678 {
		t.Errorf("incorrect KeepAliveTime from option, expected %d, got %d", 5678, config.KeepAliveTime)
	}
}

func TestStreamDeadlineAndKeepAliveFromEnv(t *testing.T) {
	t.Setenv(flagdStreamDeadlineMsEnvironmentVariableName, "7000")
	t.Setenv(flagdKeepAliveTimeMsEnvironmentVariableName, "8000")

	config, err := NewProviderConfiguration(nil)
	if err != nil {
		t.Fatal("error creating provider configuration", err)
	}

	if config.StreamDeadlineMs != 7000 {
		t.Errorf("incorrect StreamDeadlineMs from env, expected %d, got %d", 7000, config.StreamDeadlineMs)
	}
	if config.KeepAliveTime != 8000 {
		t.Errorf("incorrect KeepAliveTime from env, expected %d, got %d", 8000, config.KeepAliveTime)
	}
}

// TestStreamDeadlineAndKeepAliveOptionsOverrideEnv ensures explicitly-declared options take precedence
// over environment variables for the stream deadline / keepalive options.
func TestStreamDeadlineAndKeepAliveOptionsOverrideEnv(t *testing.T) {
	t.Setenv(flagdStreamDeadlineMsEnvironmentVariableName, "7000")
	t.Setenv(flagdKeepAliveTimeMsEnvironmentVariableName, "8000")

	config, err := NewProviderConfiguration([]ProviderOption{
		WithStreamDeadline(1111),
		WithKeepAliveTime(2222),
	})
	if err != nil {
		t.Fatal("error creating provider configuration", err)
	}

	if config.StreamDeadlineMs != 1111 {
		t.Errorf("expected option to override env for StreamDeadlineMs, expected %d, got %d", 1111, config.StreamDeadlineMs)
	}
	if config.KeepAliveTime != 2222 {
		t.Errorf("expected option to override env for KeepAliveTime, expected %d, got %d", 2222, config.KeepAliveTime)
	}
}

func TestOfflinePollMsDefault(t *testing.T) {
	config, err := NewProviderConfiguration(nil)
	if err != nil {
		t.Fatal("error creating provider configuration", err)
	}

	if config.OfflinePollMs != defaultOfflinePollMs {
		t.Errorf("incorrect default OfflinePollMs, expected %d, got %d",
			defaultOfflinePollMs, config.OfflinePollMs)
	}
}

func TestOfflinePollMsOption(t *testing.T) {
	config, err := NewProviderConfiguration([]ProviderOption{
		WithOfflinePollMs(1234),
	})
	if err != nil {
		t.Fatal("error creating provider configuration", err)
	}

	if config.OfflinePollMs != 1234 {
		t.Errorf("incorrect OfflinePollMs from option, expected %d, got %d", 1234, config.OfflinePollMs)
	}
}

func TestOfflinePollMsFromEnv(t *testing.T) {
	t.Setenv(flagdOfflinePollMsEnvironmentVariableName, "7000")

	config, err := NewProviderConfiguration(nil)
	if err != nil {
		t.Fatal("error creating provider configuration", err)
	}

	if config.OfflinePollMs != 7000 {
		t.Errorf("incorrect OfflinePollMs from env, expected %d, got %d", 7000, config.OfflinePollMs)
	}
}

// TestOfflinePollMsOptionOverridesEnv ensures explicitly-declared options take precedence
// over the environment variable for the offline poll interval.
func TestOfflinePollMsOptionOverridesEnv(t *testing.T) {
	t.Setenv(flagdOfflinePollMsEnvironmentVariableName, "7000")

	config, err := NewProviderConfiguration([]ProviderOption{
		WithOfflinePollMs(1111),
	})
	if err != nil {
		t.Fatal("error creating provider configuration", err)
	}

	if config.OfflinePollMs != 1111 {
		t.Errorf("expected option to override env for OfflinePollMs, expected %d, got %d", 1111, config.OfflinePollMs)
	}
}
