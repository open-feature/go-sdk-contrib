package flagd

import (
	"errors"
	"fmt"
	"github.com/go-logr/logr"
	"github.com/open-feature/flagd/core/pkg/sync"
	"github.com/open-feature/go-sdk-contrib/providers/flagd/internal/cache"
	"github.com/open-feature/go-sdk-contrib/providers/flagd/internal/logger"
	"google.golang.org/grpc"
	"os"
	"strconv"
	"strings"
)

type ResolverType string

// Naming and defaults must comply with flagd environment variables
const (
	defaultMaxCacheSize          int    = 1000
	defaultRpcPort               uint16 = 8013
	defaultInProcessPort         uint16 = 8015
	defaultMaxEventStreamRetries        = 5
	defaultTLS                   bool   = false
	defaultCache                        = cache.LRUValue
	defaultHost                         = "localhost"
	defaultResolver                     = rpc
	defaultGracePeriod                  = 5
	defaultRetryBackoffMs               = 1000
	defaultRetryBackoffMaxMs            = 120000
	defaultFatalStatusCodes             = "UNAUTHENTICATED,PERMISSION_DENIED"

	rpc       ResolverType = "rpc"
	inProcess ResolverType = "in-process"
	file      ResolverType = "file"

	flagdHostEnvironmentVariableName                  = "FLAGD_HOST"
	flagdPortEnvironmentVariableName                  = "FLAGD_PORT"
	flagdTLSEnvironmentVariableName                   = "FLAGD_TLS"
	flagdSocketPathEnvironmentVariableName            = "FLAGD_SOCKET_PATH"
	flagdServerCertPathEnvironmentVariableName        = "FLAGD_SERVER_CERT_PATH"
	flagdCacheEnvironmentVariableName                 = "FLAGD_CACHE"
	flagdMaxCacheSizeEnvironmentVariableName          = "FLAGD_MAX_CACHE_SIZE"
	flagdMaxEventStreamRetriesEnvironmentVariableName = "FLAGD_MAX_EVENT_STREAM_RETRIES"
	flagdResolverEnvironmentVariableName              = "FLAGD_RESOLVER"
	flagdSourceProviderIDEnvironmentVariableName      = "FLAGD_PROVIDER_ID"
	flagdSourceSelectorEnvironmentVariableName        = "FLAGD_SOURCE_SELECTOR"
	flagdOfflinePathEnvironmentVariableName           = "FLAGD_OFFLINE_FLAG_SOURCE_PATH"
	flagdTargetUriEnvironmentVariableName             = "FLAGD_TARGET_URI"
	flagdGracePeriodVariableName                      = "FLAGD_RETRY_GRACE_PERIOD"
	flagdRetryBackoffMsVariableName                   = "FLAGD_RETRY_BACKOFF_MS"
	flagdRetryBackoffMaxMsVariableName                = "FLAGD_RETRY_BACKOFF_MAX_MS"
	flagdFatalStatusCodesVariableName                 = "FLAGD_FATAL_STATUS_CODES"
)

type ProviderConfiguration struct {
	Cache                            cache.Type
	CertPath                         string
	EventStreamConnectionMaxAttempts int
	Host                             string
	MaxCacheSize                     int
	OfflineFlagSourcePath            string
	OtelIntercept                    bool
	Port                             uint16
	TargetUri                        string
	Resolver                         ResolverType
	ProviderId                       string
	Selector                         string
	SocketPath                       string
	Tls                              bool
	CustomSyncProvider               sync.ISync
	CustomSyncProviderUri            string
	GrpcDialOptionsOverride          []grpc.DialOption
	RetryGracePeriod                 int
	RetryBackoffMs                   int
	RetryBackoffMaxMs                int
	FatalStatusCodes                 []string

	log logr.Logger
}

func newDefaultConfiguration(log logr.Logger) *ProviderConfiguration {
	p := &ProviderConfiguration{
		Cache:                            defaultCache,
		EventStreamConnectionMaxAttempts: defaultMaxEventStreamRetries,
		Host:                             defaultHost,
		log:                              log,
		MaxCacheSize:                     defaultMaxCacheSize,
		Resolver:                         defaultResolver,
		Tls:                              defaultTLS,
		RetryGracePeriod:                 defaultGracePeriod,
		RetryBackoffMs:                   defaultRetryBackoffMs,
		RetryBackoffMaxMs:                defaultRetryBackoffMaxMs,
		FatalStatusCodes:                 strings.Split(defaultFatalStatusCodes, ","),
	}

	p.updateFromEnvVar()
	return p
}

func NewProviderConfiguration(opts []ProviderOption) (*ProviderConfiguration, error) {

	log := logr.New(logger.Logger{})

	// initialize with default configurations
	providerConfiguration := newDefaultConfiguration(log)

	// explicitly declared options have the highest priority
	for _, opt := range opts {
		opt(providerConfiguration)
	}

	configureProviderConfiguration(providerConfiguration)
	err := validateProviderConfiguration(providerConfiguration)

	return providerConfiguration, err
}

func configureProviderConfiguration(p *ProviderConfiguration) {
	if len(p.OfflineFlagSourcePath) > 0 && p.Resolver == inProcess {
		p.Resolver = file
	}

	if p.Port == 0 {
		switch p.Resolver {
		case rpc:
			p.Port = defaultRpcPort
		case inProcess:
			p.Port = defaultInProcessPort
		}
	}
}

func validateProviderConfiguration(p *ProviderConfiguration) error {
	// We need a file path for file mode
	if len(p.OfflineFlagSourcePath) == 0 && p.Resolver == file {
		return errors.New("resolver Type 'file' requires a OfflineFlagSourcePath")
	}

	return nil
}

// updateFromEnvVar is a utility to update configurations based on current environment variables
func (cfg *ProviderConfiguration) updateFromEnvVar() {

	portS := os.Getenv(flagdPortEnvironmentVariableName)
	if portS != "" {
		port, err := strconv.Atoi(portS)
		if err != nil {
			cfg.log.Error(err,
				fmt.Sprintf(
					"invalid env config for %s provided, using default value: %d or %d depending on resolver",
					flagdPortEnvironmentVariableName, defaultRpcPort, defaultInProcessPort,
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

	if certificatePath := os.Getenv(flagdServerCertPathEnvironmentVariableName); certificatePath != "" ||
		strings.ToLower(os.Getenv(flagdTLSEnvironmentVariableName)) == "true" {

		cfg.Tls = true
		cfg.CertPath = certificatePath
	}

	cfg.MaxCacheSize = getIntFromEnvVarOrDefault(flagdMaxCacheSizeEnvironmentVariableName, defaultMaxCacheSize, cfg.log)

	if cacheValue := os.Getenv(flagdCacheEnvironmentVariableName); cacheValue != "" {
		switch cache.Type(cacheValue) {
		case cache.LRUValue:
			cfg.Cache = cache.LRUValue
		case cache.InMemValue:
			cfg.Cache = cache.InMemValue
		case cache.DisabledValue:
			cfg.Cache = cache.DisabledValue
		default:
			cfg.log.Info("invalid cache type configured: %s, falling back to default: %s", cacheValue, defaultCache)
			cfg.Cache = defaultCache
		}
	}

	cfg.EventStreamConnectionMaxAttempts = getIntFromEnvVarOrDefault(
		flagdMaxEventStreamRetriesEnvironmentVariableName, defaultMaxEventStreamRetries, cfg.log)

	if resolver := os.Getenv(flagdResolverEnvironmentVariableName); resolver != "" {
		switch strings.ToLower(resolver) {
		case "rpc":
			cfg.Resolver = rpc
		case "in-process":
			cfg.Resolver = inProcess
		case "file":
			cfg.Resolver = file
		default:
			cfg.log.Info("invalid resolver type: %s, falling back to default: %s", resolver, defaultResolver)
			cfg.Resolver = defaultResolver
		}
	}

	if offlinePath := os.Getenv(flagdOfflinePathEnvironmentVariableName); offlinePath != "" {
		cfg.OfflineFlagSourcePath = offlinePath
	}

	if providerId := os.Getenv(flagdSourceProviderIDEnvironmentVariableName); providerId != "" {
		cfg.ProviderId = providerId
	}

	if selector := os.Getenv(flagdSourceSelectorEnvironmentVariableName); selector != "" {
		cfg.Selector = selector
	}

	if targetUri := os.Getenv(flagdTargetUriEnvironmentVariableName); targetUri != "" {
		cfg.TargetUri = targetUri
	}

	cfg.RetryGracePeriod = getIntFromEnvVarOrDefault(flagdGracePeriodVariableName, defaultGracePeriod, cfg.log)
	cfg.RetryBackoffMs = getIntFromEnvVarOrDefault(flagdRetryBackoffMsVariableName, defaultRetryBackoffMs, cfg.log)
	cfg.RetryBackoffMaxMs = getIntFromEnvVarOrDefault(flagdRetryBackoffMaxMsVariableName, defaultRetryBackoffMaxMs, cfg.log)

	if fatalStatusCodes := os.Getenv(flagdFatalStatusCodesVariableName); fatalStatusCodes != "" {
		cfg.FatalStatusCodes = strings.Split(fatalStatusCodes, ",")
	}
}

// Helper

func getIntFromEnvVarOrDefault(envVarName string, defaultValue int, log logr.Logger) int {
	if valueFromEnv := os.Getenv(envVarName); valueFromEnv != "" {
		intValue, err := strconv.Atoi(valueFromEnv)
		if err != nil {
			log.Error(err,
				fmt.Sprintf("invalid env config for %s provided, using default value: %d",
					envVarName, defaultValue,
				))
		} else {
			return intValue
		}
	}
	return defaultValue
}


// ProviderOptions

type ProviderOption func(*ProviderConfiguration)

// WithSocketPath overrides the default hostname and expectPort, a unix socket connection is made to flagd instead
func WithSocketPath(socketPath string) ProviderOption {
	return func(p *ProviderConfiguration) {
		p.SocketPath = socketPath
	}
}

// WithCertificatePath specifies the location of the certificate to be used in the gRPC dial credentials.
// If certificate loading fails insecure credentials will be used instead
func WithCertificatePath(path string) ProviderOption {
	return func(p *ProviderConfiguration) {
		p.CertPath = path
		p.Tls = true
	}
}

// WithPort specifies the port of the flagd server. Defaults to 8013
func WithPort(port uint16) ProviderOption {
	return func(p *ProviderConfiguration) {
		p.Port = port
	}
}

// WithHost specifies the host name of the flagd server. Defaults to localhost
func WithHost(host string) ProviderOption {
	return func(p *ProviderConfiguration) {
		p.Host = host
	}
}

// WithTargetUri specifies the custom gRPC target URI
func WithTargetUri(targetUri string) ProviderOption {
	return func(p *ProviderConfiguration) {
		p.TargetUri = targetUri
	}
}

// WithoutCache disables caching
func WithoutCache() ProviderOption {
	return func(p *ProviderConfiguration) {
		p.Cache = cache.DisabledValue
	}
}

// WithBasicInMemoryCache applies a basic in memory cache store (with no memory limits)
func WithBasicInMemoryCache() ProviderOption {
	return func(p *ProviderConfiguration) {
		p.Cache = cache.InMemValue
	}
}

// WithLRUCache applies least recently used caching (github.com/hashicorp/golang-lru).
// The provided size is the limit of the number of cached values. Once the limit is reached each new entry replaces the
// least recently used entry.
func WithLRUCache(size int) ProviderOption {
	return func(p *ProviderConfiguration) {
		if size > 0 {
			p.MaxCacheSize = size
		}
		p.Cache = cache.LRUValue
	}
}

// WithEventStreamConnectionMaxAttempts sets the maximum number of attempts to connect to flagd's event stream.
// On successful connection the attempts are reset.
func WithEventStreamConnectionMaxAttempts(i int) ProviderOption {
	return func(p *ProviderConfiguration) {
		p.EventStreamConnectionMaxAttempts = i
	}
}

// WithLogger sets the logger used by the provider.
func WithLogger(l logr.Logger) ProviderOption {
	return func(p *ProviderConfiguration) {
		p.log = l
	}
}

// WithTLS enables TLS. If certPath is not given, system certs are used.
func WithTLS(certPath string) ProviderOption {
	return func(p *ProviderConfiguration) {
		p.Tls = true
		p.CertPath = certPath
	}
}

// WithOtelInterceptor enable/disable otel interceptor for flagd communication
func WithOtelInterceptor(intercept bool) ProviderOption {
	return func(p *ProviderConfiguration) {
		p.OtelIntercept = intercept
	}
}

// WithRPCResolver sets flag resolver to RPC. RPC is the default resolving mechanism
func WithRPCResolver() ProviderOption {
	return func(p *ProviderConfiguration) {
		p.Resolver = rpc
	}
}

// WithInProcessResolver sets flag resolver to InProcess
func WithInProcessResolver() ProviderOption {
	return func(p *ProviderConfiguration) {
		p.Resolver = inProcess
	}
}

// WithOfflineFilePath file path to obtain flags used for provider in file mode.
func WithOfflineFilePath(path string) ProviderOption {
	return func(p *ProviderConfiguration) {
		p.OfflineFlagSourcePath = path
	}
}

// WithFileResolver sets flag resolver to File
func WithFileResolver() ProviderOption {
	return func(p *ProviderConfiguration) {
		p.Resolver = file
	}
}

// WithSelector sets the selector to be used for InProcess flag sync calls
func WithSelector(selector string) ProviderOption {
	return func(p *ProviderConfiguration) {
		p.Selector = selector
	}
}

// WithProviderID sets the providerID to be used for InProcess flag sync calls
func WithProviderID(providerID string) ProviderOption {
	return func(p *ProviderConfiguration) {
		p.ProviderId = providerID
	}
}

// FromEnv sets the provider configuration from environment variables (if set)
func FromEnv() ProviderOption {
	return func(p *ProviderConfiguration) {
		p.updateFromEnvVar()
	}
}

// WithCustomSyncProvider provides a custom implementation of the sync.ISync interface used by the inProcess Service
// This is only useful with inProcess resolver type
func WithCustomSyncProvider(customSyncProvider sync.ISync) ProviderOption {
	return WithCustomSyncProviderAndUri(customSyncProvider, defaultCustomSyncProviderUri)
}

// WithCustomSyncProviderAndUri provides a custom implementation of the sync.ISync interface used by the inProcess Service
// This is only useful with inProcess resolver type
func WithCustomSyncProviderAndUri(customSyncProvider sync.ISync, customSyncProviderUri string) ProviderOption {
	return func(p *ProviderConfiguration) {
		p.CustomSyncProvider = customSyncProvider
		p.CustomSyncProviderUri = customSyncProviderUri
	}
}

// WithGrpcDialOptionsOverride provides a set of custom grps.DialOption that will fully override the gRPC dial options used by
// the InProcess resolver with gRPC syncer. All the other provider options that also set dial options (e.g. WithTLS, or WithCertificatePath)
// will be silently ignored.
// This is only useful with inProcess resolver type
func WithGrpcDialOptionsOverride(grpcDialOptionsOverride []grpc.DialOption) ProviderOption {
	return func(p *ProviderConfiguration) {
		p.GrpcDialOptionsOverride = grpcDialOptionsOverride
	}
}

// WithRetryGracePeriod allows to set a time window for the transition from stale to error state
func WithRetryGracePeriod(gracePeriod int) ProviderOption {
	return func(p *ProviderConfiguration) {
		p.RetryGracePeriod = gracePeriod
	}
}

// WithRetryBackoffMs sets the initial backoff duration (in milliseconds) for retrying failed connections
func WithRetryBackoffMs(retryBackoffMs int) ProviderOption {
	return func(p *ProviderConfiguration) {
		p.RetryBackoffMs = retryBackoffMs
	}
}

// WithRetryBackoffMaxMs sets the maximum backoff duration (in milliseconds) for retrying failed connections
func WithRetryBackoffMaxMs(retryBackoffMaxMs int) ProviderOption {
	return func(p *ProviderConfiguration) {
		p.RetryBackoffMaxMs = retryBackoffMaxMs
	}
}

// WithFatalStatusCodes allows to set a list of gRPC status codes, which will cause streams to give up
// and put the provider in a PROVIDER_FATAL state
func WithFatalStatusCodes(fatalStatusCodes []string) ProviderOption {
	return func(p *ProviderConfiguration) {
		p.FatalStatusCodes = fatalStatusCodes
	}
}