package multiprovider

import (
	"context"
	"errors"
	"fmt"
	"github.com/open-feature/go-sdk-contrib/providers/multi-provider/internal/logger"
	"github.com/open-feature/go-sdk-contrib/providers/multi-provider/internal/wrappers"
	"github.com/open-feature/go-sdk-contrib/providers/multi-provider/pkg/strategies"
	"golang.org/x/sync/errgroup"
	"log/slog"
	"maps"
	"slices"
	"sync"
	"time"

	mperr "github.com/open-feature/go-sdk-contrib/providers/multi-provider/pkg/errors"

	of "github.com/open-feature/go-sdk/openfeature"
)

type (
	// MultiProvider Provider used for combining multiple providers
	MultiProvider struct {
		providers          ProviderMap
		metadata           of.Metadata
		initialized        bool
		totalStatus        of.State
		totalStatusLock    sync.RWMutex
		providerStatus     map[string]of.State
		providerStatusLock sync.Mutex
		strategy           strategies.Strategy
		logger             *logger.ConditionalLogger
		outboundEvents     chan of.Event
		inboundEvents      chan namedEvent
		workerGroup        sync.WaitGroup
		shutdownFunc       context.CancelFunc
		globalHooks        []of.Hook
	}

	// Configuration MultiProvider's internal configuration
	Configuration struct {
		useFallback      bool
		fallbackProvider of.FeatureProvider
		customStrategy   strategies.Strategy
		logger           *slog.Logger
		timeout          time.Duration
		hooks            []of.Hook
		providerHooks    map[string][]of.Hook
	}

	// EvaluationStrategy Defines a strategy to use for resolving the result from multiple providers
	EvaluationStrategy = string
	// ProviderMap A map where the keys are names of providers and the values are the providers themselves
	ProviderMap map[string]of.FeatureProvider
	// Option Function used for setting Configuration via the options pattern
	Option func(*Configuration)

	// Private Types
	namedEvent struct {
		of.Event
		providerName string
	}
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

	MetadataProviderName  = "multiprovider-provider-name"
	MetadataProviderType  = "multiprovider-provider-type"
	MetadataInternalError = "multiprovider-internal-error"
)

var (
	_                of.FeatureProvider = (*MultiProvider)(nil)
	_                of.EventHandler    = (*MultiProvider)(nil)
	_                of.StateHandler    = (*MultiProvider)(nil)
	stateValues      map[of.State]int
	stateTable       [3]of.State
	eventTypeToState map[of.EventType]of.State
)

func init() {
	// used for mapping provider event types & provider states to comparable values for evaluation
	stateValues = map[of.State]int{
		"":            -1, // Not a real state, but used for handling provider config changes
		of.ErrorState: 0,
		of.StaleState: 1,
		of.ReadyState: 2,
	}
	// used for mapping
	stateTable = [3]of.State{
		of.ReadyState, // 0
		of.StaleState, // 1
		of.ErrorState, // 2
	}
	eventTypeToState = map[of.EventType]of.State{
		of.ProviderConfigChange: "",
		of.ProviderReady:        of.ReadyState,
		of.ProviderStale:        of.StaleState,
		of.ProviderError:        of.ErrorState,
	}
}

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
		logger:        slog.Default(), // Logging enabled by default using default slog logger
		providerHooks: make(map[string][]of.Hook),
	}

	for _, opt := range options {
		opt(config)
	}

	providers := providerMap
	// Wrap any providers that include hooks
	for name, provider := range providerMap {
		if (len(provider.Hooks()) + len(config.providerHooks[name])) == 0 {
			continue
		}

		if _, ok := provider.(of.EventHandler); ok {
			providers[name] = wrappers.IsolateProviderWithEvents(provider, config.providerHooks[name])
			continue
		}

		providers[name] = wrappers.IsolateProvider(provider, config.providerHooks[name])
	}

	multiProvider := &MultiProvider{
		providers:      providers,
		outboundEvents: make(chan of.Event),
		logger:         logger.NewConditionalLogger(config.logger),
		metadata:       providerMap.buildMetadata(),
		totalStatus:    of.NotReadyState,
		providerStatus: make(map[string]of.State),
		globalHooks:    config.hooks,
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
	// wrapper type used only for initialization of event listener workers
	type namedEventHandler struct {
		of.EventHandler
		name string
	}
	mp.logger.LogDebug(context.Background(), "start initialization")
	mp.inboundEvents = make(chan namedEvent, len(mp.providers))
	handlers := make(chan namedEventHandler)
	for name, provider := range mp.providers {
		// Initialize each provider to not ready state. No locks required there are no workers running
		mp.providerStatus[name] = of.NotReadyState
		l := mp.logger.With(slog.String("multiprovider-provider-name", name))

		eg.Go(func() error {
			l.LogDebug(context.Background(), "starting initialization")
			stateHandle, ok := provider.(of.StateHandler)
			if !ok {
				l.LogDebug(context.Background(), "StateHandle not implemented, skipping initialization")
			} else if err := stateHandle.Init(evalCtx); err != nil {
				l.LogError(context.Background(), "initialization failed", slog.Any("error", err))
				return &mperr.ProviderError{
					Err:          err,
					ProviderName: name,
				}
			}
			l.LogDebug(context.Background(), "initialization successful")
			if eventer, ok := provider.(of.EventHandler); ok {
				l.LogDebug(context.Background(), "detected EventHandler implementation")
				handlers <- namedEventHandler{eventer, name}
			} else {
				// Do not yet update providers that need event handling
				mp.providerStatusLock.Lock()
				defer mp.providerStatusLock.Unlock()
				mp.providerStatus[name] = of.ReadyState
			}
			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		mp.setStatus(of.ErrorState)
		var pErr *mperr.ProviderError
		if errors.As(err, &pErr) {
			// Update provider status to error, no event needs to be emitted.
			// No locks needed as no workers are active at this point
			mp.providerStatus[pErr.ProviderName] = of.ErrorState
		} else {
			pErr = &mperr.ProviderError{
				Err:          err,
				ProviderName: "unknown",
			}
		}
		mp.outboundEvents <- of.Event{
			ProviderName: mp.Metadata().Name,
			EventType:    of.ProviderError,
			ProviderEventDetails: of.ProviderEventDetails{
				Message:     fmt.Sprintf("internal provider %s encountered an error during initialization: %+v", pErr.ProviderName, pErr.Err),
				FlagChanges: nil,
				EventMetadata: map[string]interface{}{
					MetadataProviderName:  pErr.ProviderName,
					MetadataInternalError: pErr.Error(),
				},
			},
		}
		return err
	}
	close(handlers)
	workerCtx, shutdownFunc := context.WithCancel(context.Background())
	for h := range handlers {
		go mp.startListening(workerCtx, h.name, h.EventHandler, &mp.workerGroup)
	}
	mp.shutdownFunc = shutdownFunc

	go func() {
		workerLogger := mp.logger.With(slog.String("multiprovider-worker", "event-forwarder-worker"))
		mp.workerGroup.Add(1)
		defer mp.workerGroup.Done()
		for e := range mp.inboundEvents {
			l := workerLogger.With(
				slog.String("multiprovider-provider-name", e.providerName),
				slog.String("multiprovider-provider-type", e.ProviderName),
			)
			l.LogDebug(context.Background(), fmt.Sprintf("received %s event from provider", e.EventType))
			state := mp.updateProviderStateAndEvaluateTotalState(e, l)
			if state != mp.Status() {
				mp.setStatus(state)
				mp.outboundEvents <- e.Event
				l.LogDebug(context.Background(), "forwarded state update event")
			} else {
				l.LogDebug(context.Background(), "total state not updated, inbound event will not be emitted")
			}
		}
	}()

	mp.setStatus(of.ReadyState)
	mp.outboundEvents <- of.Event{
		ProviderName: mp.Metadata().Name,
		EventType:    of.ProviderReady,
		ProviderEventDetails: of.ProviderEventDetails{
			Message:     "all internal providers initialized successfully",
			FlagChanges: nil,
			EventMetadata: map[string]interface{}{
				MetadataProviderName: "all",
			},
		},
	}
	mp.initialized = true
	return nil
}

// startListening is intended to be
func (mp *MultiProvider) startListening(ctx context.Context, name string, h of.EventHandler, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()
	for {
		select {
		case e := <-h.EventChannel():
			e.EventMetadata[MetadataProviderName] = name
			e.EventMetadata[MetadataProviderType] = h.(of.FeatureProvider).Metadata().Name
			mp.inboundEvents <- namedEvent{
				Event:        e,
				providerName: name,
			}
		case <-ctx.Done():
			return
		}
	}
}

func (mp *MultiProvider) updateProviderStateAndEvaluateTotalState(e namedEvent, l *logger.ConditionalLogger) of.State {
	if e.EventType == of.ProviderConfigChange {
		l.LogDebug(context.Background(), fmt.Sprintf("ProviderConfigChange event: %s", e.Message))
		return mp.Status()
	}
	mp.providerStatusLock.Lock()
	defer mp.providerStatusLock.Unlock()
	logProviderState(l, e, mp.providerStatus[e.providerName])
	mp.providerStatus[e.providerName] = eventTypeToState[e.EventType]
	maxState := stateValues[of.ReadyState] // initialize to the lowest state value
	for _, s := range mp.providerStatus {
		if stateValues[s] > maxState {
			// change in state due to higher priority
			maxState = stateValues[s]
		}
	}
	return stateTable[maxState]
}

func logProviderState(l *logger.ConditionalLogger, e namedEvent, previousState of.State) {
	switch eventTypeToState[e.EventType] {
	case of.ReadyState:
		if previousState != of.NotReadyState {
			l.LogInfo(context.Background(), fmt.Sprintf("provider %s has returned to ready state from %s", e.providerName, previousState))
			return
		}
		l.LogDebug(context.Background(), fmt.Sprintf("provider %s is ready", e.providerName))
	case of.StaleState:
		l.LogWarn(context.Background(), fmt.Sprintf("provider %s is stale: %s", e.providerName, e.Message))
	case of.ErrorState:
		l.LogError(context.Background(), fmt.Sprintf("provider %s is in an error state: %s", e.providerName, e.Message))
	}
}

// Shutdown Shuts down all internal providers
func (mp *MultiProvider) Shutdown() {
	if !mp.initialized {
		// Don't do anything if we were never initialized
		return
	}
	// Stop all event listener workers, shutdown events should not affect overall state
	mp.shutdownFunc()
	// Stop forwarding worker
	close(mp.inboundEvents)
	mp.logger.LogDebug(context.Background(), "triggered worker shutdown")
	// Wait for workers to stop
	mp.workerGroup.Wait()
	mp.logger.LogDebug(context.Background(), "worker shutdown completed")
	mp.logger.LogDebug(context.Background(), "starting provider shutdown")
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

	mp.logger.LogDebug(context.Background(), "waiting for provider shutdown completion")
	wg.Wait()
	mp.logger.LogDebug(context.Background(), "provider shutdown completed")
	mp.setStatus(of.NotReadyState)
	close(mp.outboundEvents)
	mp.outboundEvents = nil
	mp.inboundEvents = nil
	mp.initialized = false
}

// Status the current state of the MultiProvider
func (mp *MultiProvider) Status() of.State {
	mp.totalStatusLock.RLock()
	defer mp.totalStatusLock.RUnlock()
	return mp.totalStatus
}

func (mp *MultiProvider) setStatus(state of.State) {
	mp.totalStatusLock.Lock()
	defer mp.totalStatusLock.Unlock()
	mp.totalStatus = state
	mp.logger.LogDebug(context.Background(), "state updated", slog.String("state", string(state)))
}

// EventChannel the channel events are emitted on
func (mp *MultiProvider) EventChannel() <-chan of.Event {
	return mp.outboundEvents
}
