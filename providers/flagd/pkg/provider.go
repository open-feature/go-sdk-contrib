package flagd

import (
	"context"
	"fmt"
	"time"

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
	hooks                 []of.Hook
	eventStream           chan of.Event
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
		hooks:                 []of.Hook{},
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
				DeadlineMs:      provider.providerConfiguration.DeadlineMs,
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
			DeadlineMs:              provider.providerConfiguration.DeadlineMs,
		})
	default:
		service = process.NewInProcessService(process.Configuration{
			OfflineFlagSource: provider.providerConfiguration.OfflineFlagSourcePath,
			DeadlineMs:        provider.providerConfiguration.DeadlineMs,
		})
	}

	if provider.providerConfiguration.Resolver == inProcess {
		provider.hooks = append(provider.hooks, NewSyncContextHook(func() *of.EvaluationContext {
			return provider.providerConfiguration.ContextEnricher(service.ContextValues())
		}))
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

	// Create a timer for the initialization deadline that covers the entire init process
	deadline := time.Duration(p.providerConfiguration.DeadlineMs) * time.Millisecond
	timer := time.NewTimer(deadline)
	defer timer.Stop()

	// Get the event channel before starting goroutine to avoid race with service.Init()
	serviceEventChan := p.service.EventChannel()

	// Run service.Init() in a goroutine so we can timeout if it hangs
	initDone := make(chan error, 1)
	go func() {
		initDone <- p.service.Init()
	}()

	// Wait for service.Init completion and ProviderReady event in a single select loop
	var initDoneChan <-chan error = initDone
	for {
		select {
		case err := <-initDoneChan:
			if err != nil {
				return err
			}
			// Init succeeded, disable this case and continue loop to wait for ProviderReady
			initDoneChan = nil

		case e := <-serviceEventChan:
			if e.EventType == of.ProviderReady {
				p.status = of.ReadyState
				p.initialized = true
				// start event handling after the first ready event
				go p.handleEvents()
				return nil
			}
			// If we got a ProviderError or ProviderStale during init, return it as an error
			if e.EventType == of.ProviderError || e.EventType == of.ProviderStale {
				return fmt.Errorf("provider initialization failed: %s", e.ProviderEventDetails.Message)
			}
			return fmt.Errorf("provider initialization failed: unexpected event type %v", e.EventType)

		case <-timer.C:
			return fmt.Errorf("provider initialization deadline exceeded (%dms)", p.providerConfiguration.DeadlineMs)
		}
	}
}

// handleEvents runs in a separate goroutine and processes events from the service
func (p *Provider) handleEvents() {
	serviceEventChan := p.service.EventChannel()
	for event := range serviceEventChan {
		p.eventStream <- event
		switch event.EventType {
		case of.ProviderReady, of.ProviderConfigChange:
			p.setStatus(of.ReadyState)
		case of.ProviderError:
			p.setStatus(of.ErrorState)
		}
	}
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
	return p.hooks
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
