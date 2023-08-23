package flagd

import (
	schemaV1 "buf.build/gen/go/open-feature/flagd/protocolbuffers/go/schema/v1"
	"context"
	"github.com/go-logr/logr"
	"github.com/open-feature/go-sdk-contrib/providers/flagd/internal/logger"
	"github.com/open-feature/go-sdk-contrib/providers/flagd/pkg/cache"
	"github.com/open-feature/go-sdk-contrib/providers/flagd/pkg/service"
	of "github.com/open-feature/go-sdk/pkg/openfeature"
)

type Provider struct {
	cache                 *cache.Service
	ctx                   context.Context
	isReady               chan struct{}
	logger                logr.Logger
	providerConfiguration *providerConfiguration
	service               service.IService
}

func NewProvider(opts ...ProviderOption) *Provider {
	log := logr.New(logger.Logger{})

	// initialize with default configurations
	configuration := newDefaultConfiguration()

	// env variables have higher priority than defaults
	configuration.updateFromEnvVar(log)

	provider := &Provider{
		ctx:                   context.Background(),
		isReady:               make(chan struct{}),
		logger:                log,
		providerConfiguration: configuration,
	}

	// explicitly declared options have the highest priority
	for _, opt := range opts {
		opt(provider)
	}

	provider.cache = cache.NewCacheService(
		provider.providerConfiguration.CacheType,
		provider.providerConfiguration.MaxCacheSize,
		log)

	provider.service = service.NewService(service.NewClient(
		&service.Configuration{
			Host:            provider.providerConfiguration.Host,
			Port:            provider.providerConfiguration.Port,
			CertificatePath: provider.providerConfiguration.CertificatePath,
			SocketPath:      provider.providerConfiguration.SocketPath,
			TLSEnabled:      provider.providerConfiguration.TLSEnabled,
			OtelInterceptor: provider.providerConfiguration.otelIntercept,
		},
	), provider.cache, provider.logger)

	go func() {
		if err := provider.handleEvents(provider.ctx); err != nil {
			provider.logger.Error(err, "handle events")
		}
	}()

	return provider
}

// Hooks flagd provider does not have any hooks, returns empty slice
func (p *Provider) Hooks() []of.Hook {
	return []of.Hook{}
}

// Metadata returns value of Metadata (name of current service, exposed to openfeature sdk)
func (p *Provider) Metadata() of.Metadata {
	return of.Metadata{
		Name: "flagd",
	}
}

func (p *Provider) BooleanEvaluation(
	ctx context.Context, flagKey string, defaultValue bool, evalCtx of.FlattenedContext,
) of.BoolResolutionDetail {
	return p.service.ResolveBoolean(ctx, flagKey, defaultValue, evalCtx)
}

func (p *Provider) StringEvaluation(
	ctx context.Context, flagKey string, defaultValue string, evalCtx of.FlattenedContext,
) of.StringResolutionDetail {
	return p.service.ResolveString(ctx, flagKey, defaultValue, evalCtx)
}

func (p *Provider) FloatEvaluation(
	ctx context.Context, flagKey string, defaultValue float64, evalCtx of.FlattenedContext,
) of.FloatResolutionDetail {
	return p.service.ResolveFloat(ctx, flagKey, defaultValue, evalCtx)
}

func (p *Provider) IntEvaluation(
	ctx context.Context, flagKey string, defaultValue int64, evalCtx of.FlattenedContext,
) of.IntResolutionDetail {
	return p.service.ResolveInt(ctx, flagKey, defaultValue, evalCtx)
}

func (p *Provider) ObjectEvaluation(
	ctx context.Context, flagKey string, defaultValue interface{}, evalCtx of.FlattenedContext,
) of.InterfaceResolutionDetail {
	return p.service.ResolveObject(ctx, flagKey, defaultValue, evalCtx)
}

// ProviderOptions

type ProviderOption func(*Provider)

// WithSocketPath overrides the default hostname and port, a unix socket connection is made to flagd instead
func WithSocketPath(socketPath string) ProviderOption {
	return func(p *Provider) {
		p.providerConfiguration.SocketPath = socketPath
	}
}

// WithCertificatePath specifies the location of the certificate to be used in the gRPC dial credentials.
// If certificate loading fails insecure credentials will be used instead
func WithCertificatePath(path string) ProviderOption {
	return func(p *Provider) {
		p.providerConfiguration.CertificatePath = path
		p.providerConfiguration.TLSEnabled = true
	}
}

// WithPort specifies the port of the flagd server. Defaults to 8013
func WithPort(port uint16) ProviderOption {
	return func(p *Provider) {
		p.providerConfiguration.Port = port
	}
}

// WithHost specifies the host name of the flagd server. Defaults to localhost
func WithHost(host string) ProviderOption {
	return func(p *Provider) {
		p.providerConfiguration.Host = host
	}
}

// WithoutCache disables caching
func WithoutCache() ProviderOption {
	return func(p *Provider) {
		p.providerConfiguration.CacheType = cacheDisabledValue
	}
}

// WithBasicInMemoryCache applies a basic in memory cache store (with no memory limits)
func WithBasicInMemoryCache() ProviderOption {
	return func(p *Provider) {
		p.providerConfiguration.CacheType = cacheInMemValue
	}
}

// WithLRUCache applies least recently used caching (github.com/hashicorp/golang-lru).
// The provided size is the limit of the number of cached values. Once the limit is reached each new entry replaces the
// least recently used entry.
func WithLRUCache(size int) ProviderOption {
	return func(p *Provider) {
		if size > 0 {
			p.providerConfiguration.MaxCacheSize = size
		}
		p.providerConfiguration.CacheType = cacheLRUValue
	}
}

// WithContext supplies the given context to the event stream. Not to be confused with the context used in individual
// flag evaluation requests.
func WithContext(ctx context.Context) ProviderOption {
	return func(p *Provider) {
		p.ctx = ctx
	}
}

// WithEventStreamConnectionMaxAttempts sets the maximum number of attempts to connect to flagd's event stream.
// On successful connection the attempts are reset.
func WithEventStreamConnectionMaxAttempts(i int) ProviderOption {
	return func(p *Provider) {
		p.providerConfiguration.EventStreamConnectionMaxAttempts = i
	}
}

// WithLogger sets the logger used by the provider.
func WithLogger(l logr.Logger) ProviderOption {
	return func(p *Provider) {
		p.logger = l
	}
}

// WithTLS enables TLS. If certPath is not given, system certs are used.
func WithTLS(certPath string) ProviderOption {
	return func(p *Provider) {
		p.providerConfiguration.TLSEnabled = true
		p.providerConfiguration.CertificatePath = certPath
	}
}

// WithOtelInterceptor enable/disable otel interceptor for flagd communication
func WithOtelInterceptor(intercept bool) ProviderOption {
	return func(p *Provider) {
		p.providerConfiguration.otelIntercept = intercept
	}
}

// FromEnv sets the provider configuration from environment variables (if set)
func FromEnv() ProviderOption {
	return func(p *Provider) {
		p.providerConfiguration.updateFromEnvVar(p.logger)
	}
}

// todo: Event handling to be isolated to own component

func (p *Provider) handleEvents(ctx context.Context) error {
	eventChan := make(chan *schemaV1.EventStreamResponse)
	errChan := make(chan error)

	handler := eventHandler{
		cache:     p.cache,
		errChan:   errChan,
		eventChan: eventChan,
		isReady:   p.isReady,
		logger:    p.logger,
	}

	go func() {
		p.service.EventStream(ctx, eventChan, p.providerConfiguration.EventStreamConnectionMaxAttempts, errChan)
	}()

	err := handler.handle(ctx)
	if err != nil {
		return err
	}

	return nil
}
