package flagd

import (
	"context"
	"errors"
	"fmt"

	parallel "sync"

	"github.com/go-logr/logr"
	"github.com/open-feature/flagd/core/pkg/sync"
	"github.com/open-feature/go-sdk-contrib/providers/flagd/internal/cache"
	"github.com/open-feature/go-sdk-contrib/providers/flagd/internal/logger"
	process "github.com/open-feature/go-sdk-contrib/providers/flagd/pkg/service/in_process"
	rpcService "github.com/open-feature/go-sdk-contrib/providers/flagd/pkg/service/rpc"
	of "github.com/open-feature/go-sdk/openfeature"
	"google.golang.org/grpc"
)

const (
	defaultCustomSyncProviderUri = "syncprovider://custom"
)

type Provider struct {
	initialized           bool
	logger                logr.Logger
	providerConfiguration *providerConfiguration
	service               IService
	status                of.State
	mtx                   parallel.RWMutex

	eventStream chan of.Event
}

func NewProvider(opts ...ProviderOption) (*Provider, error) {
	log := logr.New(logger.Logger{})

	// initialize with default configurations
	providerConfiguration := newDefaultConfiguration(log)

	provider := &Provider{
		initialized:           false,
		eventStream:           make(chan of.Event),
		logger:                log,
		providerConfiguration: providerConfiguration,
		status:                of.NotReadyState,
	}

	// explicitly declared options have the highest priority
	for _, opt := range opts {
		opt(provider)
	}

	UpdateProvider(provider)

	err := CheckProvider(provider)

	if err != nil {
		return nil, err
	}

	cacheService := cache.NewCacheService(
		provider.providerConfiguration.CacheType,
		provider.providerConfiguration.MaxCacheSize,
		provider.logger)

	var service IService
	if provider.providerConfiguration.Resolver == rpc {
		service = rpcService.NewService(
			rpcService.Configuration{
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
	} else if provider.providerConfiguration.Resolver == inProcess {
		service = process.NewInProcessService(process.Configuration{
			Host:                    provider.providerConfiguration.Host,
			Port:                    provider.providerConfiguration.Port,
			ProviderID:              provider.providerConfiguration.ProviderID,
			Selector:                provider.providerConfiguration.Selector,
			TargetUri:               provider.providerConfiguration.TargetUri,
			TLSEnabled:              provider.providerConfiguration.TLSEnabled,
			OfflineFlagSource:       provider.providerConfiguration.OfflineFlagSourcePath,
			CustomSyncProvider:      provider.providerConfiguration.CustomSyncProvider,
			CustomSyncProviderUri:   provider.providerConfiguration.CustomSyncProviderUri,
			GrpcDialOptionsOverride: provider.providerConfiguration.GrpcDialOptionsOverride,
		})
	} else {
		service = process.NewInProcessService(process.Configuration{
			OfflineFlagSource: provider.providerConfiguration.OfflineFlagSourcePath,
		})
	}

	provider.service = service

	return provider, nil
}

func UpdateProvider(p *Provider) {
	if len(p.providerConfiguration.OfflineFlagSourcePath) > 0 && p.providerConfiguration.Resolver == inProcess {
		p.providerConfiguration.Resolver = file
	}

	if p.providerConfiguration.Port == 0 {
		switch p.providerConfiguration.Resolver {
		case rpc:
			p.providerConfiguration.Port = defaultRpcPort
		case inProcess:
			p.providerConfiguration.Port = defaultInProcessPort
		}
	}
}

func CheckProvider(p *Provider) error {
	// We need a file path for file mode
	if len(p.providerConfiguration.OfflineFlagSourcePath) == 0 && p.providerConfiguration.Resolver == file {
		return errors.New("Resolver Type 'file' requires a OfflineFlagSourcePath")
	}

	return nil
}

func (p *Provider) Init(_ of.EvaluationContext) error {
	p.mtx.Lock()
	defer p.mtx.Unlock()

	// avoid reinitialization if initialized
	if p.initialized {
		return nil
	}

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
	p.initialized = true

	// start event handling after the first ready event
	go func() {
		for {
			event := <-p.service.EventChannel()
			p.eventStream <- event
			switch event.EventType {
			case of.ProviderReady:
			case of.ProviderConfigChange:
				p.setStatus(of.ReadyState)
			case of.ProviderError:
				p.setStatus(of.ErrorState)
			}
		}
	}()

	return nil
}

func (p *Provider) Status() of.State {
	p.mtx.RLock()
	defer p.mtx.RUnlock()
	return p.status
}

func (p *Provider) Shutdown() {
	p.mtx.Lock()
	defer p.mtx.Unlock()

	p.initialized = false
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

func (p *Provider) setStatus(status of.State) {
	p.mtx.Lock()
	defer p.mtx.Unlock()
	p.status = status
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

// WithTargetUri specifies the custom gRPC target URI
func WithTargetUri(targetUri string) ProviderOption {
	return func(p *Provider) {
		p.providerConfiguration.TargetUri = targetUri
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
		p.providerConfiguration.Resolver = rpc
	}
}

// WithInProcessResolver sets flag resolver to InProcess
func WithInProcessResolver() ProviderOption {
	return func(p *Provider) {
		p.providerConfiguration.Resolver = inProcess
	}
}

// WithOfflineFilePath file path to obtain flags used for provider in file mode.
func WithOfflineFilePath(path string) ProviderOption {
	return func(p *Provider) {
		p.providerConfiguration.OfflineFlagSourcePath = path
	}
}

// WithFileResolver sets flag resolver to File
func WithFileResolver() ProviderOption {
	return func(p *Provider) {
		p.providerConfiguration.Resolver = file
	}
}

// WithSelector sets the selector to be used for InProcess flag sync calls
func WithSelector(selector string) ProviderOption {
	return func(p *Provider) {
		p.providerConfiguration.Selector = selector
	}
}

// WithProviderID sets the providerID to be used for InProcess flag sync calls
func WithProviderID(providerID string) ProviderOption {
	return func(p *Provider) {
		p.providerConfiguration.ProviderID = providerID
	}
}

// FromEnv sets the provider configuration from environment variables (if set)
func FromEnv() ProviderOption {
	return func(p *Provider) {
		p.providerConfiguration.updateFromEnvVar()
	}
}

// WithCustomSyncProvider provides a custom implementation of the sync.ISync interface used by the inProcess Service
// This is only useful with inProcess resolver type
func WithCustomSyncProvider(customSyncProvider sync.ISync) ProviderOption {
	return WithCustomSyncProviderAndUri(customSyncProvider, defaultCustomSyncProviderUri)
}

// WithCustomSyncProvider provides a custom implementation of the sync.ISync interface used by the inProcess Service
// This is only useful with inProcess resolver type
func WithCustomSyncProviderAndUri(customSyncProvider sync.ISync, customSyncProviderUri string) ProviderOption {
	return func(p *Provider) {
		p.providerConfiguration.CustomSyncProvider = customSyncProvider
		p.providerConfiguration.CustomSyncProviderUri = customSyncProviderUri
	}
}

// WithGrpcDialOptionsOverride provides a set of custom grps.DialOption that will fully override the gRPC dial options used by
// the InProcess resolver with gRPC syncer. All the other provider options that also set dial options (e.g. WithTLS, or WithCertificatePath)
// will be silently ignored.
// This is only useful with inProcess resolver type
func WithGrpcDialOptionsOverride(grpcDialOptionsOverride []grpc.DialOption) ProviderOption {
	return func(p *Provider) {
		p.providerConfiguration.GrpcDialOptionsOverride = grpcDialOptionsOverride
	}
}
