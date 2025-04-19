package multiprovider

import (
	"context"
	"errors"
	"fmt"
	"github.com/open-feature/go-sdk-contrib/providers/multi-provider/internal/strategies"
	"log/slog"
	"sync"

	mperr "github.com/open-feature/go-sdk-contrib/providers/multi-provider/internal/errors"

	of "github.com/open-feature/go-sdk/openfeature"
)

type (
	MultiProvider struct {
		providers ProviderMap
		metadata  of.Metadata
		events    chan of.Event
		status    of.State
		mu        sync.RWMutex
		strategy  strategies.Strategy
		logger    *slog.Logger
	}

	Configuration struct {
		useFallback      bool
		fallbackProvider of.FeatureProvider
		logger           *slog.Logger
		publishEvents    bool
		metadata         *of.Metadata
		hooks            []of.Hook // Not implemented yet
	}

	// EvaluationStrategy Defines a strategy to use for resolving the result from multiple providers
	EvaluationStrategy = string
	ProviderMap        map[string]of.FeatureProvider
	Option             func(*Configuration)
)

const (
	// StrategyFirstMatch First provider whose response that is not FlagNotFound will be returned. This is executed
	// sequentially, and not in parallel.
	StrategyFirstMatch EvaluationStrategy = strategies.StrategyFirstMatch
	// StrategyFirstSuccess First provider response that is not an error will be returned. This is executed in parallel
	StrategyFirstSuccess EvaluationStrategy = strategies.StrategyFirstSuccess
	// StrategyComparison All providers are called in parallel. If all responses agree the value will be returned.
	// Otherwise, the value from the designated fallback provider's response will be returned. The fallback provider
	// will be assigned to the first provider registered. (NOT YET IMPLEMENTED, SUBJECT TO CHANGE)
	StrategyComparison EvaluationStrategy = "comparison"
)

var _ of.FeatureProvider = (*MultiProvider)(nil)

// MultiProvider implements of `FeatureProvider` in a way to accept an array of providers.

func (m ProviderMap) AsNamedProviderSlice() []*strategies.NamedProvider {
	s := make([]*strategies.NamedProvider, 0, len(m))
	for name, provider := range m {
		s = append(s, &strategies.NamedProvider{Name: name, Provider: provider})
	}

	return s
}

func (m ProviderMap) buildMetadata() of.Metadata {
	var separator string
	metaName := "MultiProvider {"
	for name, provider := range m {
		metaName = fmt.Sprintf("%s%s%s: %s", metaName, separator, name, provider.Metadata().Name)
		if separator == "" {
			separator = ", "
		}
	}
	metaName += "}"
	return of.Metadata{
		Name: metaName,
	}
}

func WithLogger(l *slog.Logger) Option {
	return func(conf *Configuration) {
		conf.logger = l
	}
}

func WithFallbackProvider(p of.FeatureProvider, name string) Option {
	return func(conf *Configuration) {
		conf.fallbackProvider = p
		conf.useFallback = true
	}
}

func WithNamedFallbackProvider(p of.FeatureProvider) Option {
	return func(conf *Configuration) {
		conf.fallbackProvider = p
		conf.useFallback = true
	}
}

func WithEventPublishing() Option {
	return func(conf *Configuration) {
		conf.publishEvents = true
	}
}

func WithoutEventPublishing() Option {
	return func(conf *Configuration) {
		conf.publishEvents = false
	}
}

// NewMultiProvider returns the unified interface of multiple providers for interaction.
func NewMultiProvider(providerMap ProviderMap, evaluationStrategy EvaluationStrategy, options ...Option) (*MultiProvider, error) {
	if len(providerMap) == 0 {
		return nil, errors.New("providerMap cannot be nil or empty")
	}
	// Validate Providers
	for name, provider := range providerMap {
		if name == "" {
			return nil, errors.New("provider name cannot be the empty string")
		}

		if provider == nil {
			return nil, fmt.Errorf("provider %s cannot be nil", name)
		}
	}

	config := &Configuration{
		logger: slog.Default(),
	}

	for _, opt := range options {
		opt(config)
	}

	var eventChannel chan of.Event
	if config.publishEvents {
		eventChannel = make(chan of.Event)
	}

	logger := config.logger
	if logger == nil {
		logger = slog.Default()
	}

	multiProvider := &MultiProvider{
		providers: providerMap,
		events:    eventChannel,
		logger:    logger,
		metadata:  providerMap.buildMetadata(),
	}

	var strategy strategies.Strategy
	switch evaluationStrategy {
	case StrategyFirstMatch:
		strategy = strategies.NewFirstMatchStrategy(multiProvider.Providers())
	case StrategyFirstSuccess:
		strategy = strategies.NewFirstSuccessStrategy(multiProvider.Providers())
	case StrategyComparison:
		strategy = strategies.NewComparisonStrategy(multiProvider.Providers(), config.fallbackProvider)
	default:
		return nil, fmt.Errorf("%s is an unknown evalutation strategy", strategy)
	}
	multiProvider.strategy = strategy

	return multiProvider, nil
}

func (mp *MultiProvider) Providers() []*strategies.NamedProvider {
	return mp.providers.AsNamedProviderSlice()
}

func (mp *MultiProvider) ProvidersByName() ProviderMap {
	return mp.providers
}

func (mp *MultiProvider) EvaluationStrategy() string {
	return mp.strategy.Name()
}

// Metadata provides the name `multiprovider` and the names of each provider passed.
func (mp *MultiProvider) Metadata() of.Metadata {
	return mp.metadata
}

// Hooks returns a collection of of.Hook defined by this provider
func (mp *MultiProvider) Hooks() []of.Hook {
	// Hooks that should be included with the provider
	return []of.Hook{}
}

// BooleanEvaluation returns a boolean flag
func (mp *MultiProvider) BooleanEvaluation(ctx context.Context, flag string, defaultValue bool, evalCtx of.FlattenedContext) of.BoolResolutionDetail {
	return mp.strategy.BooleanEvaluation(ctx, flag, defaultValue, evalCtx)
}

// StringEvaluation returns a string flag
func (mp *MultiProvider) StringEvaluation(ctx context.Context, flag string, defaultValue string, evalCtx of.FlattenedContext) of.StringResolutionDetail {
	return mp.strategy.StringEvaluation(ctx, flag, defaultValue, evalCtx)
}

// FloatEvaluation returns a float flag
func (mp *MultiProvider) FloatEvaluation(ctx context.Context, flag string, defaultValue float64, evalCtx of.FlattenedContext) of.FloatResolutionDetail {
	return mp.strategy.FloatEvaluation(ctx, flag, defaultValue, evalCtx)
}

// IntEvaluation returns an int flag
func (mp *MultiProvider) IntEvaluation(ctx context.Context, flag string, defaultValue int64, evalCtx of.FlattenedContext) of.IntResolutionDetail {
	return mp.strategy.IntEvaluation(ctx, flag, defaultValue, evalCtx)
}

// ObjectEvaluation returns an object flag
func (mp *MultiProvider) ObjectEvaluation(ctx context.Context, flag string, defaultValue interface{}, evalCtx of.FlattenedContext) of.InterfaceResolutionDetail {
	return mp.strategy.ObjectEvaluation(ctx, flag, defaultValue, evalCtx)
}

// Init will run the initialize method for all of provides and aggregate the errors.
func (mp *MultiProvider) Init(evalCtx of.EvaluationContext) error {
	var wg sync.WaitGroup
	errChan := make(chan mperr.StateErr)

	for name, provider := range mp.providers {
		wg.Add(1)
		go func(p of.FeatureProvider, name string) {
			defer wg.Done()
			if stateHandle, ok := provider.(of.StateHandler); ok {
				if initErr := stateHandle.Init(evalCtx); initErr != nil {
					errChan <- mperr.StateErr{ProviderName: name, Err: initErr, ErrMessage: initErr.Error()}
				}
			}
		}(provider, name)
	}

	go func() {
		wg.Wait()
		close(errChan)
	}()

	errs := make([]mperr.StateErr, 0, 1)
	for err := range errChan {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		var aggErr mperr.AggregateError
		aggErr.Construct(errs)
		mp.mu.RLock()
		defer mp.mu.Unlock()
		mp.status = of.ErrorState

		return &aggErr
	}

	mp.mu.RLock()
	defer mp.mu.Unlock()
	mp.status = of.ReadyState

	return nil
}

func (mp *MultiProvider) Status() of.State {
	return mp.status
}

func (mp *MultiProvider) Shutdown() {
	var wg sync.WaitGroup

	for _, provider := range mp.providers {
		wg.Add(1)
		go func(p of.FeatureProvider) {
			defer wg.Done()
			if stateHandle, ok := provider.(of.StateHandler); ok {
				stateHandle.Shutdown()
			}
		}(provider)
	}

	wg.Wait()
}

func (mp *MultiProvider) EventChannel() <-chan of.Event {
	return mp.events
}
