package flagd

import (
	"fmt"
	"github.com/go-logr/logr"
	"github.com/open-feature/go-sdk-contrib/providers/flagd/pkg/cache"
	"os"
	"strconv"
)

// Naming and defaults must comply with flagd environment variables
const (
	defaultMaxCacheSize                               int  = 1000
	defaultPort                                            = 8013
	defaultMaxEventStreamRetries                           = 5
	defaultTLS                                        bool = false
	defaultCache                                           = cache.LRUValue
	defaultHost                                            = "localhost"
	flagdHostEnvironmentVariableName                       = "FLAGD_HOST"
	flagdPortEnvironmentVariableName                       = "FLAGD_PORT"
	flagdTLSEnvironmentVariableName                        = "FLAGD_TLS"
	flagdSocketPathEnvironmentVariableName                 = "FLAGD_SOCKET_PATH"
	flagdServerCertPathEnvironmentVariableName             = "FLAGD_SERVER_CERT_PATH"
	flagdCacheEnvironmentVariableName                      = "FLAGD_CACHE"
	flagdMaxCacheSizeEnvironmentVariableName               = "FLAGD_MAX_CACHE_SIZE"
	flagdMaxEventStreamRetriesEnvironmentVariableName      = "FLAGD_MAX_EVENT_STREAM_RETRIES"
	cacheDisabledValue                                     = cache.DisabledValue
	cacheLRUValue                                          = cache.LRUValue
	cacheInMemValue                                        = cache.InMemValue
)

type providerConfiguration struct {
	CacheType                        cache.Type
	CertificatePath                  string
	EventStreamConnectionMaxAttempts int
	Host                             string
	MaxCacheSize                     int
	OtelIntercept                    bool
	Port                             uint16
	SocketPath                       string
	TLSEnabled                       bool
}

func newDefaultConfiguration() *providerConfiguration {
	return &providerConfiguration{
		CacheType:                        defaultCache,
		EventStreamConnectionMaxAttempts: defaultMaxEventStreamRetries,
		Host:                             defaultHost,
		MaxCacheSize:                     defaultMaxCacheSize,
		Port:                             defaultPort,
		TLSEnabled:                       defaultTLS,
	}
}

// updateFromEnvVar is a utility to update configurations based on current environment variables
func (cfg *providerConfiguration) updateFromEnvVar(logger logr.Logger) {
	portS := os.Getenv(flagdPortEnvironmentVariableName)
	if portS != "" {
		port, err := strconv.Atoi(portS)
		if err != nil {
			logger.Error(err,
				fmt.Sprintf(
					"invalid env config for %s provided, using default value: %d",
					flagdPortEnvironmentVariableName, defaultPort,
				))
		} else {
			cfg.Port = uint16(port)
		}
	}

	host := os.Getenv(flagdHostEnvironmentVariableName)
	if host != "" {
		cfg.Host = host
	}

	socketPath := os.Getenv(flagdSocketPathEnvironmentVariableName)
	if socketPath != "" {
		cfg.SocketPath = socketPath
	}

	certificatePath := os.Getenv(flagdServerCertPathEnvironmentVariableName)
	if certificatePath != "" || os.Getenv(flagdTLSEnvironmentVariableName) == "true" {
		cfg.TLSEnabled = true
		cfg.CertificatePath = certificatePath
	}

	maxCacheSizeS := os.Getenv(flagdMaxCacheSizeEnvironmentVariableName)
	if maxCacheSizeS != "" {
		maxCacheSizeFromEnv, err := strconv.Atoi(maxCacheSizeS)
		if err != nil {
			logger.Error(err,
				fmt.Sprintf("invalid env config for %s provided, using default value: %d",
					flagdMaxCacheSizeEnvironmentVariableName, defaultMaxCacheSize,
				))
		} else {
			cfg.MaxCacheSize = maxCacheSizeFromEnv
		}
	}

	if cacheValue := os.Getenv(flagdCacheEnvironmentVariableName); cacheValue != "" {
		switch cache.Type(cacheValue) {
		case cache.LRUValue:
			cfg.CacheType = cache.LRUValue
		case cache.InMemValue:
			cfg.CacheType = cache.InMemValue
		case cache.DisabledValue:
			cfg.CacheType = cache.DisabledValue
		default:
			logger.Info("invalid cache type configured: %s, falling back to default: %s", cacheValue, defaultCache)
			cfg.CacheType = defaultCache
		}
	}

	maxEventStreamRetriesS := os.Getenv(flagdMaxEventStreamRetriesEnvironmentVariableName)
	if maxEventStreamRetriesS != "" {
		maxEventStreamRetries, err := strconv.Atoi(maxEventStreamRetriesS)
		if err != nil {
			logger.Error(err,
				fmt.Sprintf("invalid env config for %s provided, using default value: %d",
					flagdMaxEventStreamRetriesEnvironmentVariableName, defaultMaxEventStreamRetries))
		} else {
			cfg.EventStreamConnectionMaxAttempts = maxEventStreamRetries
		}
	}

}
