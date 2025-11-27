package flagd

import (
	"context"
	"fmt"

	parallel "sync"

	"github.com/open-feature/go-sdk-contrib/providers/flagd/internal/cache"
	process "github.com/open-feature/go-sdk-contrib/providers/flagd/pkg/service/in_process"
	rpcService "github.com/open-feature/go-sdk-contrib/providers/flagd/pkg/service/rpc"
	of "github.com/open-feature/go-sdk/openfeature"
)

const (
	defaultCustomSyncProviderUri = "syncprovider://custom"
)

type Provider struct {
	initialized           bool
	providerConfiguration *ProviderConfiguration
	service               IService
	status                of.State
	mtx                   parallel.RWMutex

	eventStream chan of.Event
}

func NewProvider(opts ...ProviderOption) (*Provider, error) {
	providerConfiguration, err := NewProviderConfiguration(opts)

	if err != nil {
		return nil, err
	}

	provider := &Provider{
		initialized:           false,
		eventStream:           make(chan of.Event),
		providerConfiguration: providerConfiguration,
		status:                of.NotReadyState,
	}

	cacheService := cache.NewCacheService(
		provider.providerConfiguration.Cache,
		provider.providerConfiguration.MaxCacheSize,
		provider.providerConfiguration.log)

	var service IService
	switch provider.providerConfiguration.Resolver {
	case rpc:
		service = rpcService.NewService(
			rpcService.Configuration{
				Host:            provider.providerConfiguration.Host,
				Port:            provider.providerConfiguration.Port,
				CertificatePath: provider.providerConfiguration.CertPath,
				SocketPath:      provider.providerConfiguration.SocketPath,
				TLSEnabled:      provider.providerConfiguration.Tls,
				OtelInterceptor: provider.providerConfiguration.OtelIntercept,
			},
			cacheService,
			provider.providerConfiguration.log,
			provider.providerConfiguration.EventStreamConnectionMaxAttempts)
	case inProcess:
		service = process.NewInProcessService(process.Configuration{
			Host:                    provider.providerConfiguration.Host,
			Port:                    provider.providerConfiguration.Port,
			ProviderID:              provider.providerConfiguration.ProviderId,
			Selector:                provider.providerConfiguration.Selector,
			TargetUri:               provider.providerConfiguration.TargetUri,
			TLSEnabled:              provider.providerConfiguration.Tls,
			CertificatePath:         provider.providerConfiguration.CertPath,
			OfflineFlagSource:       provider.providerConfiguration.OfflineFlagSourcePath,
			CustomSyncProvider:      provider.providerConfiguration.CustomSyncProvider,
			CustomSyncProviderUri:   provider.providerConfiguration.CustomSyncProviderUri,
			GrpcDialOptionsOverride: provider.providerConfiguration.GrpcDialOptionsOverride,
			RetryGracePeriod:        provider.providerConfiguration.RetryGracePeriod,
			RetryBackOffMs:          provider.providerConfiguration.RetryBackoffMs,
			RetryBackOffMaxMs:       provider.providerConfiguration.RetryBackoffMaxMs,
			FatalStatusCodes:        provider.providerConfiguration.FatalStatusCodes,
		})
	default:
		service = process.NewInProcessService(process.Configuration{
			OfflineFlagSource: provider.providerConfiguration.OfflineFlagSourcePath,
		})
	}

	provider.service = service

	return provider, nil
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
