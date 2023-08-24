package flagd

import (
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

			if flagdProvider == nil {
				t.Fatal("received nil service from NewProvider")
			}

			metadata := flagdProvider.Metadata()
			if metadata.Name != "flagd" {
				t.Errorf("received unexpected metadata from NewProvider, expected %s got %s", "flagd", metadata.Name)
			}

			config := flagdProvider.providerConfiguration
			if config == nil {
				t.Fatal("configurations are not set")
			}

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
