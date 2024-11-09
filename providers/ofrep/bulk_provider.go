package ofrep

import (
	"context"
	"sync"
	"time"

	"github.com/open-feature/go-sdk-contrib/providers/ofrep/internal/evaluate"
	"github.com/open-feature/go-sdk-contrib/providers/ofrep/internal/outbound"
	"github.com/open-feature/go-sdk/openfeature"
	of "github.com/open-feature/go-sdk/openfeature"
)

const providerName = "OFREP Bulk Provider"

func NewBulkProvider(baseUri string, options ...Option) *BulkProvider {
	cfg := outbound.Configuration{
		BaseURI:               baseUri,
		ClientPollingInterval: 30 * time.Second,
	}

	for _, option := range options {
		option(&cfg)
	}

	return &BulkProvider{
		events: make(chan of.Event, 3),
		cfg:    cfg,
		state:  of.NotReadyState,
	}
}

var (
	_ of.FeatureProvider = (*BulkProvider)(nil) // ensure BulkProvider implements FeatureProvider
	_ of.StateHandler    = (*BulkProvider)(nil) // ensure BulkProvider implements StateHandler
	_ of.EventHandler    = (*BulkProvider)(nil) // ensure BulkProvider implements EventHandler
)

type BulkProvider struct {
	Provider
	cfg        outbound.Configuration
	state      of.State
	mu         sync.RWMutex
	events     chan of.Event
	cancelFunc context.CancelFunc
}

func (p *BulkProvider) Metadata() openfeature.Metadata {
	return of.Metadata{
		Name: providerName,
	}
}

func (p *BulkProvider) Status() of.State {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.state
}

func (p *BulkProvider) Init(evalCtx of.EvaluationContext) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.state != of.NotReadyState {
		// avoid reinitialization if initialized
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	p.cancelFunc = cancel

	client := outbound.NewHttp(p.cfg)

	flatCtx := FlattenContext(evalCtx)

	evaluator := evaluate.NewBulkEvaluator(client, flatCtx)
	err := evaluator.Fetch(ctx)
	if err != nil {
		return err
	}

	if p.cfg.PollingEnabled() {
		p.startPolling(ctx, evaluator, p.cfg.PollingInterval())
	}

	p.evaluator = evaluator
	p.state = of.ReadyState
	return nil
}

func (p *BulkProvider) Shutdown() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.cancelFunc != nil {
		p.cancelFunc()
		p.cancelFunc = nil
	}

	p.state = of.NotReadyState
	p.evaluator = nil
}

func (p *BulkProvider) EventChannel() <-chan of.Event {
	return p.events
}

func (p *BulkProvider) startPolling(ctx context.Context, evaluator *evaluate.BulkEvaluator, pollingInterval time.Duration) {
	go func() {
		ticker := time.NewTicker(pollingInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				err := evaluator.Fetch(ctx)
				if err != nil {
					if err != context.Canceled {
						p.mu.Lock()
						p.state = of.StaleState
						p.mu.Unlock()
						p.events <- of.Event{
							ProviderName: providerName, EventType: of.ProviderStale,
							ProviderEventDetails: of.ProviderEventDetails{Message: err.Error()},
						}
					}
					continue
				}
				p.mu.RLock()
				state := p.state
				p.mu.RUnlock()
				if state != of.ReadyState {
					p.mu.Lock()
					p.state = of.ReadyState
					p.mu.Unlock()
					p.events <- of.Event{
						ProviderName: providerName, EventType: of.ProviderReady,
						ProviderEventDetails: of.ProviderEventDetails{Message: "Provider is ready"},
					}
				}
			}
		}
	}()
}

func FlattenContext(evalCtx of.EvaluationContext) of.FlattenedContext {
	flatCtx := evalCtx.Attributes()
	if evalCtx.TargetingKey() != "" {
		flatCtx[of.TargetingKey] = evalCtx.TargetingKey()
	}
	return flatCtx
}
