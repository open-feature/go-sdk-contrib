package flagd

import (
	"github.com/golang/mock/gomock"
	"github.com/open-feature/go-sdk-contrib/providers/flagd/internal/mock"
	of "github.com/open-feature/go-sdk/pkg/openfeature"
	"testing"
)

func TestNewProvider(t *testing.T) {
	tests := []struct {
		name                string
		expectPort          uint16
		expectHost          string
		expectCacheType     string
		expectCertPath      string
		expectMaxRetries    int
		expectCacheSize     int
		expectOtelIntercept bool
		expectSocketPath    string
		expectTlsEnabled    bool
		options             []ProviderOption
	}{
		{
			name:                "default construction",
			expectPort:          defaultPort,
			expectHost:          defaultHost,
			expectCacheType:     defaultCache,
			expectCertPath:      "",
			expectMaxRetries:    defaultMaxEventStreamRetries,
			expectCacheSize:     defaultMaxCacheSize,
			expectOtelIntercept: false,
			expectSocketPath:    "",
			expectTlsEnabled:    false,
		},
		{
			name:                "with options",
			expectPort:          9090,
			expectHost:          "myHost",
			expectCacheType:     cacheLRUValue,
			expectCertPath:      "/path",
			expectMaxRetries:    2,
			expectCacheSize:     2500,
			expectOtelIntercept: true,
			expectSocketPath:    "/socket",
			expectTlsEnabled:    true,
			options: []ProviderOption{
				WithSocketPath("/socket"),
				WithOtelInterceptor(true),
				WithLRUCache(2500),
				WithEventStreamConnectionMaxAttempts(2),
				WithCertificatePath("/path"),
				WithHost("myHost"),
				WithPort(9090),
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
	svcMock.EXPECT().Init().AnyTimes()

	provider := NewProvider()
	provider.service = svcMock

	if provider.Status() != of.NotReadyState {
		t.Errorf("expected initial status to be not ready, but got %v", provider.Status())
	}

	err := provider.Init(of.EvaluationContext{})
	if err != nil {
		t.Fatal("error initialization provider", err)
	}

	// push events to local event channel
	go func() {
		customChan <- of.Event{
			ProviderName: "flagd",
			EventType:    of.ProviderReady,
		}

		customChan <- of.Event{
			ProviderName: "flagd",
			EventType:    of.ProviderError,
		}
	}()

	// check event emitting from provider in order
	event := <-provider.EventChannel()
	if event.EventType != of.ProviderReady {
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
