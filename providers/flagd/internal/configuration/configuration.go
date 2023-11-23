package configuration

import (
	"fmt"
	"github.com/go-logr/logr"
	"github.com/open-feature/go-sdk-contrib/providers/flagd/internal/cache"
	"os"
	"strconv"
)

type ResolverType string

// Naming and defaults must comply with flagd environment variables
const (
	DefaultMaxCacheSize          int  = 1000
	DefaultPort                       = 8013
	DefaultMaxEventStreamRetries      = 5
	defaultTLS                   bool = false
	DefaultCache                      = cache.LRUValue
	DefaultHost                       = "localhost"
	DefaultResolver                   = RPC
	DefaultSourceSelector             = ""

	RPC       ResolverType = "rpc"
	InProcess ResolverType = "in-process"

	flagdHostEnvironmentVariableName                  = "FLAGD_HOST"
	flagdPortEnvironmentVariableName                  = "FLAGD_PORT"
	flagdTLSEnvironmentVariableName                   = "FLAGD_TLS"
	flagdSocketPathEnvironmentVariableName            = "FLAGD_SOCKET_PATH"
	flagdServerCertPathEnvironmentVariableName        = "FLAGD_SERVER_CERT_PATH"
	flagdCacheEnvironmentVariableName                 = "FLAGD_CACHE"
	flagdMaxCacheSizeEnvironmentVariableName          = "FLAGD_MAX_CACHE_SIZE"
	flagdMaxEventStreamRetriesEnvironmentVariableName = "FLAGD_MAX_EVENT_STREAM_RETRIES"
	flagdResolverEnvironmentVariableName              = "FLAGD_RESOLVER"
	flagdSourceSelectorEnvironmentVariableName        = "FLAGD_SOURCE_SELECTOR"
)

type ProviderConfiguration struct {
	CacheType                        cache.Type
	CertificatePath                  string
	EventStreamConnectionMaxAttempts int
	Host                             string
	MaxCacheSize                     int
	OtelIntercept                    bool
	Port                             uint16
	Resolver                         ResolverType
	Selector                         string
	SocketPath                       string
	TLSEnabled                       bool

	log logr.Logger
}

func NewDefaultConfiguration(log logr.Logger) *ProviderConfiguration {
	p := &ProviderConfiguration{
		CacheType:                        DefaultCache,
		EventStreamConnectionMaxAttempts: DefaultMaxEventStreamRetries,
		Host:                             DefaultHost,
		log:                              log,
		MaxCacheSize:                     DefaultMaxCacheSize,
		Port:                             DefaultPort,
		Resolver:                         DefaultResolver,
		TLSEnabled:                       defaultTLS,
	}

	p.UpdateFromEnvVar()
	return p
}

// UpdateFromEnvVar is a utility to update configurations based on current environment variables
func (cfg *ProviderConfiguration) UpdateFromEnvVar() {
	portS := os.Getenv(flagdPortEnvironmentVariableName)
	if portS != "" {
		port, err := strconv.Atoi(portS)
		if err != nil {
			cfg.log.Error(err,
				fmt.Sprintf(
					"invalid env config for %s provided, using default value: %d",
					flagdPortEnvironmentVariableName, DefaultPort,
				))
		} else {
			cfg.Port = uint16(port)
		}
	}

	if host := os.Getenv(flagdHostEnvironmentVariableName); host != "" {
		cfg.Host = host
	}

	if socketPath := os.Getenv(flagdSocketPathEnvironmentVariableName); socketPath != "" {
		cfg.SocketPath = socketPath
	}

	if certificatePath := os.Getenv(flagdServerCertPathEnvironmentVariableName); certificatePath != "" || os.Getenv(
		flagdTLSEnvironmentVariableName) == "true" {

		cfg.TLSEnabled = true
		cfg.CertificatePath = certificatePath
	}

	if maxCacheSizeS := os.Getenv(flagdMaxCacheSizeEnvironmentVariableName); maxCacheSizeS != "" {
		maxCacheSizeFromEnv, err := strconv.Atoi(maxCacheSizeS)
		if err != nil {
			cfg.log.Error(err,
				fmt.Sprintf("invalid env config for %s provided, using default value: %d",
					flagdMaxCacheSizeEnvironmentVariableName, DefaultMaxCacheSize,
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
			cfg.log.Info("invalid cache type configured: %s, falling back to default: %s", cacheValue, DefaultCache)
			cfg.CacheType = DefaultCache
		}
	}

	if maxEventStreamRetriesS := os.Getenv(
		flagdMaxEventStreamRetriesEnvironmentVariableName); maxEventStreamRetriesS != "" {

		maxEventStreamRetries, err := strconv.Atoi(maxEventStreamRetriesS)
		if err != nil {
			cfg.log.Error(err,
				fmt.Sprintf("invalid env config for %s provided, using default value: %d",
					flagdMaxEventStreamRetriesEnvironmentVariableName, DefaultMaxEventStreamRetries))
		} else {
			cfg.EventStreamConnectionMaxAttempts = maxEventStreamRetries
		}
	}

	if resolver := os.Getenv(flagdResolverEnvironmentVariableName); resolver != "" {
		switch ResolverType(resolver) {
		case RPC:
			cfg.Resolver = RPC
		case InProcess:
			cfg.Resolver = InProcess
		default:
			cfg.log.Info("invalid resolver type: %s, falling back to default: %s", resolver, DefaultResolver)
			cfg.Resolver = DefaultResolver
		}
	}

	if selector := os.Getenv(flagdSourceSelectorEnvironmentVariableName); selector != "" {
		cfg.Selector = selector
	}

}
