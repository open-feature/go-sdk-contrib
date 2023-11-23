package flagd

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/open-feature/go-sdk-contrib/providers/flagd/internal/cache"
	"github.com/open-feature/go-sdk-contrib/providers/flagd/internal/configuration"
	"github.com/open-feature/go-sdk-contrib/providers/flagd/internal/logger"
	"github.com/open-feature/go-sdk-contrib/providers/flagd/pkg/service/rpc"
	of "github.com/open-feature/go-sdk/pkg/openfeature"
)

type Provider struct {
	logger                logr.Logger
	providerConfiguration *configuration.ProviderConfiguration
	service               IService
	status                of.State

	eventStream chan of.Event
}

func NewProvider(opts ...ProviderOption) *Provider {
	log := logr.New(logger.Logger{})

	// initialize with default configurations
	providerConfiguration := configuration.NewDefaultConfiguration(log)

	provider := &Provider{
		eventStream:           make(chan of.Event),
		logger:                log,
		providerConfiguration: providerConfiguration,
		status:                of.NotReadyState,
	}

	// explicitly declared options have the highest priority
	for _, opt := range opts {
		opt(provider)
	}

	cacheService := cache.NewCacheService(
		provider.providerConfiguration.CacheType,
		provider.providerConfiguration.MaxCacheSize,
		provider.logger)

	provider.service = rpc.NewService(
		rpc.Configuration{
			Host:            provider.providerConfiguration.Host,
			Port:            provider.providerConfiguration.Port,
			CertificatePath: provider.providerConfiguration.CertificatePath,
			SocketPath:      provider.providerConfiguration.SocketPath,
			TLSEnabled:      provider.providerConfiguration.TLSEnabled,
			OtelInterceptor: provider.providerConfiguration.OtelIntercept,
		},
		cacheService,
		provider.logger,
		provider.providerConfiguration.EventStreamConnectionMaxAttempts)

	return provider
}

func (p *Provider) Init(evaluationContext of.EvaluationContext) error {
	err := p.service.Init()
	if err != nil {
		return err
	}

	// wait for initialization from the service
	e := <-p.service.EventChannel()
	if e.EventType != of.ProviderReady {
		return fmt.Errorf("provider initialization failed: %s", e.ProviderEventDetails.Message)
	}

	p.status = of.ReadyState

	// start event handling after the first ready event
	go func() {
		for {
			event := <-p.service.EventChannel()
			p.eventStream <- event
			switch event.EventType {
			case of.ProviderReady:
				p.status = of.ReadyState
			case of.ProviderError:
				p.status = of.ErrorState
			}
		}
	}()

	return nil
}

func (p *Provider) Status() of.State {
	return p.status
}

func (p *Provider) Shutdown() {
	p.service.Shutdown()
}

func (p *Provider) EventChannel() <-chan of.Event {
	return p.eventStream
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

// WithSocketPath overrides the default hostname and expectPort, a unix socket connection is made to flagd instead
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
		p.providerConfiguration.CacheType = cache.DisabledValue
	}
}

// WithBasicInMemoryCache applies a basic in memory cache store (with no memory limits)
func WithBasicInMemoryCache() ProviderOption {
	return func(p *Provider) {
		p.providerConfiguration.CacheType = cache.InMemValue
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
		p.providerConfiguration.CacheType = cache.LRUValue
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
		p.providerConfiguration.OtelIntercept = intercept
	}
}

// WithRPCResolver sets flag resolver to RPC. RPC is the default resolving mechanism
func WithRPCResolver() ProviderOption {
	return func(p *Provider) {
		p.providerConfiguration.Resolver = configuration.RPC
	}
}

// WithInProcessResolver sets flag resolver to InProcess
func WithInProcessResolver() ProviderOption {
	return func(p *Provider) {
		p.providerConfiguration.Resolver = configuration.InProcess
	}
}

// WithSelector sets the selector to be used for InProcess flag sync calls
func WithSelector(selector string) ProviderOption {
	return func(p *Provider) {
		p.providerConfiguration.Selector = selector
	}
}

// FromEnv sets the provider configuration from environment variables (if set)
func FromEnv() ProviderOption {
	return func(p *Provider) {
		p.providerConfiguration.UpdateFromEnvVar()
	}
}
