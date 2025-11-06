package multiprovider

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"slices"
	"sync"
	"time"

	"github.com/open-feature/go-sdk-contrib/providers/multi-provider/pkg/strategies"
	"golang.org/x/sync/errgroup"

	mperr "github.com/open-feature/go-sdk-contrib/providers/multi-provider/pkg/errors"

	of "github.com/open-feature/go-sdk/openfeature"
)

type (
	// MultiProvider Provider used for combining multiple providers
	MultiProvider struct {
		providers ProviderMap
		metadata  of.Metadata
		events    chan of.Event
		status    of.State
		mu        sync.RWMutex
		strategy  strategies.Strategy
		logger    *slog.Logger
	}

	// Configuration MultiProvider's internal configuration
	Configuration struct {
		useFallback      bool
		fallbackProvider of.FeatureProvider
		customStrategy   strategies.Strategy
		logger           *slog.Logger
		publishEvents    bool
		metadata         *of.Metadata //nolint unused
		timeout          time.Duration
		hooks            []of.Hook //nolint unused - Not implemented yet
	}

	// EvaluationStrategy Defines a strategy to use for resolving the result from multiple providers
	EvaluationStrategy = string
	// ProviderMap A map where the keys are names of providers and the values are the providers themselves
	ProviderMap map[string]of.FeatureProvider
	// Option Function used for setting Configuration via the options pattern
	Option func(*Configuration)
)

const (
	// StrategyFirstMatch First provider whose response that is not FlagNotFound will be returned. This is executed
	// sequentially, and not in parallel.
	StrategyFirstMatch EvaluationStrategy = strategies.StrategyFirstMatch
	// StrategyFirstSuccess First provider response that is not an error will be returned. This is executed in parallel
	StrategyFirstSuccess EvaluationStrategy = strategies.StrategyFirstSuccess
	// StrategyComparison All providers are called in parallel. If all responses agree the value will be returned.
	// Otherwise, the value from the designated fallback provider's response will be returned. The fallback provider
	// will be assigned to the first provider registered.
	StrategyComparison EvaluationStrategy = "comparison"
	// StrategyCustom allows for using a custom Strategy implementation. If this is set you MUST use the WithCustomStrategy
	// option to set it
	StrategyCustom EvaluationStrategy = "strategy-custom"
)

var _ of.FeatureProvider = (*MultiProvider)(nil)

// AsNamedProviderSlice Converts the map into a slice of NamedProvider instances
func (m ProviderMap) AsNamedProviderSlice() []*strategies.NamedProvider {
	s := make([]*strategies.NamedProvider, 0, len(m))
	for name, provider := range m {
		s = append(s, &strategies.NamedProvider{Name: name, Provider: provider})
	}

	return s
}

// Size The size of the map. This operates in O(n) time.
func (m ProviderMap) Size() int {
	return len(m.AsNamedProviderSlice())
}

func (m ProviderMap) buildMetadata() of.Metadata {
	var separator string
	metaName := "MultiProvider {"
	names := slices.Collect(maps.Keys(m))
	slices.Sort(names)
	for _, name := range names {
		metaName = fmt.Sprintf("%s%s%s: %s", metaName, separator, name, m[name].Metadata().Name)
		if separator == "" {
			separator = ", "
		}
	}
	metaName += "}"
	return of.Metadata{
		Name: metaName,
	}
}

// NewMultiProvider returns the unified interface of multiple providers for interaction.
//
// Deprecated: Use multi.NewProvider() from github.com/open-feature/go-sdk/openfeature/multi instead.
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

	var zeroDuration time.Duration
	if config.timeout == zeroDuration {
		config.timeout = 5 * time.Second
	}

	var strategy strategies.Strategy
	switch evaluationStrategy {
	case StrategyFirstMatch:
		strategy = strategies.NewFirstMatchStrategy(multiProvider.Providers())
	case StrategyFirstSuccess:
		strategy = strategies.NewFirstSuccessStrategy(multiProvider.Providers(), config.timeout)
	case StrategyComparison:
		strategy = strategies.NewComparisonStrategy(multiProvider.Providers(), config.fallbackProvider)
	case StrategyCustom:
		if config.customStrategy != nil {
			strategy = config.customStrategy
		} else {
			return nil, fmt.Errorf("A custom strategy must be set via an option if StrategyCustom is set")
		}
	default:
		return nil, fmt.Errorf("%s is an unknown evalutation strategy", strategy)
	}
	multiProvider.strategy = strategy

	return multiProvider, nil
}

// Providers Returns slice of providers wrapped in NamedProvider structs
func (mp *MultiProvider) Providers() []*strategies.NamedProvider {
	return mp.providers.AsNamedProviderSlice()
}

// ProvidersByName Returns the internal ProviderMap of the MultiProvider
func (mp *MultiProvider) ProvidersByName() ProviderMap {
	return mp.providers
}

// EvaluationStrategy The current set strategy
func (mp *MultiProvider) EvaluationStrategy() string {
	return mp.strategy.Name()
}

// Metadata provides the name `multiprovider` and the names of each provider passed.
func (mp *MultiProvider) Metadata() of.Metadata {
	return mp.metadata
}

// Hooks returns a collection of.Hook defined by this provider
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
	var eg errgroup.Group

	for name, provider := range mp.providers {
		eg.Go(func() error {
			stateHandle, ok := provider.(of.StateHandler)
			if !ok {
				return nil
			}
			if err := stateHandle.Init(evalCtx); err != nil {
				return &mperr.ProviderError{
					Err:          err,
					ProviderName: name,
				}
			}

			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		mp.mu.Lock()
		defer mp.mu.Unlock()
		mp.status = of.ErrorState

		return err
	}

	mp.mu.Lock()
	defer mp.mu.Unlock()
	mp.status = of.ReadyState
	return nil
}

// Status the current status of the MultiProvider
func (mp *MultiProvider) Status() of.State {
	mp.mu.RLock()
	defer mp.mu.RUnlock()
	return mp.status
}

// Shutdown Shuts down all internal providers
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

// EventChannel the channel events are emitted on (Not Yet Implemented)
func (mp *MultiProvider) EventChannel() <-chan of.Event {
	return mp.events
}
