package multiprovider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/open-feature/go-sdk-contrib/providers/multi-provider/internal/strategies"
	"sync"

	err "github.com/open-feature/go-sdk-contrib/providers/multi-provider/internal"
	strategies "github.com/open-feature/go-sdk-contrib/providers/multi-provider/strategies"

	"github.com/open-feature/go-sdk/openfeature"
	"github.com/open-feature/go-sdk/openfeature/hooks"
)

var (
	errUniqueName = errors.New("provider names must be unique")
)

// UniqueNameProvider allows for a unique name to be assigned to a provider during a multi-provider set up.
// The name will be used when reporting errors & results to specify the provider associated.
type UniqueNameProvider struct {
	Provider   openfeature.FeatureProvider
	UniqueName string
}

// MultiMetadata defines the return of the MultiProvider metadata with the aggregated data of all the providers.
type MultiMetadata struct {
	Name             string                          `json:"name"`
	OriginalMetadata map[string]openfeature.Metadata `json:"originalMetadata"`
}

var _ openfeature.FeatureProvider = (*MultiProvider)(nil)

// MultiProvider implements openfeature `FeatureProvider` in a way to accept an array of providers.
type MultiProvider struct {
	providersEntries       []UniqueNameProvider
	providersEntriesByName map[string]UniqueNameProvider
	AggregatedMetadata     MultiMetadata
	events                 chan openfeature.Event
	status                 openfeature.State
	mu                     sync.Mutex
	strategy               strategies.Strategy
}

// NewMultiProvider returns the unified interface of multiple providers for interaction.
func NewMultiProvider(passedProviders []UniqueNameProvider, evaluationStrategy strategies.EvaluationStrategy, logger *hooks.LoggingHook) (*MultiProvider, error) {
	multiProvider := &MultiProvider{
		providersEntries:       []UniqueNameProvider{},
		providersEntriesByName: map[string]UniqueNameProvider{},
		AggregatedMetadata: MultiMetadata{
			Name:             "multiprovider",
			OriginalMetadata: map[string]openfeature.Metadata{},
		},
		EvaluationStrategy: evaluationStrategy,
	}

	err := multiProvider.registerProviders(passedProviders)
	if err != nil {
		return nil, err
	}

	var strategy strategies.Strategy
	switch evaluationStrategy {
	case strategies.StrategyFirstMatch:
		strategy = strategies.NewFirstMatchStrategy(multiProvider.Providers())
	case strategies.StrategyFirstSuccess:
		strategy = strategies.NewFirstSuccessStrategy(multiProvider.Providers())
	default:
		return nil, fmt.Errorf("%s is an unknown evalutation strategy", strategy)
	}
	multiProvider.strategy = strategy

	return multiProvider, nil
}

func (mp *MultiProvider) Providers() []UniqueNameProvider {
	return mp.providersEntries
}

func (mp *MultiProvider) ProvidersByName() map[string]UniqueNameProvider {
	return mp.providersEntriesByName
}

func (mp *MultiProvider) ProviderByName(name string) (UniqueNameProvider, bool) {
	provider, exists := mp.providersEntriesByName[name]
	return provider, exists
}

func (mp *MultiProvider) EvaluationStrategy() string {
	return mp.strategy.Name()
}

// registerProviders ensures that when setting up an instant of MultiProvider the providers provided either have a unique name or the base `metadata.Name` is made unique by adding an indexed based number to it.
// registerProviders also stores the providers by their unique name and in an array for easy usage.
func (mp *MultiProvider) registerProviders(providers []UniqueNameProvider) error {
	providersByName := make(map[string][]UniqueNameProvider)

	for _, provider := range providers {
		uniqueName := provider.UniqueName

		if _, exists := providersByName[uniqueName]; exists {
			return errUniqueName
		}

		if uniqueName == "" {
			providersByName[provider.Provider.Metadata().Name] = append(providersByName[provider.Provider.Metadata().Name], provider)
		} else {
			providersByName[uniqueName] = append(providersByName[uniqueName], provider)
		}
	}

	for name, providers := range providersByName {
		if len(providers) == 1 {
			providers[0].UniqueName = name
			mp.providersEntries = append(mp.providersEntries, providers[0])
			mp.providersEntriesByName[name] = providers[0]
			mp.AggregatedMetadata.OriginalMetadata[name] = providers[0].Provider.Metadata()
		} else {
			for i, provider := range providers {
				uniqueName := fmt.Sprintf("%s-%d", name, i+1)
				provider.UniqueName = uniqueName
				mp.providersEntries = append(mp.providersEntries, provider)
				mp.providersEntriesByName[uniqueName] = provider
				mp.AggregatedMetadata.OriginalMetadata[uniqueName] = provider.Provider.Metadata()
			}
		}
	}
	return nil
}

// Metadata provides the name `multiprovider` and the names of each provider passed.
func (mp *MultiProvider) Metadata() openfeature.Metadata {
	metaJSON, _ := json.Marshal(mp.AggregatedMetadata)

	return openfeature.Metadata{Name: string(metaJSON)}
}

// Hooks returns a collection of openfeature.Hook defined by this provider
func (mp *MultiProvider) Hooks() []openfeature.Hook {
	// Hooks that should be included with the provider
	return []openfeature.Hook{}
}

// BooleanEvaluation returns a boolean flag
func (mp *MultiProvider) BooleanEvaluation(ctx context.Context, flag string, defaultValue bool, evalCtx openfeature.FlattenedContext) openfeature.BoolResolutionDetail {
	return mp.strategy.BooleanEvaluation(ctx, flag, defaultValue, evalCtx)
}

// StringEvaluation returns a string flag
func (mp *MultiProvider) StringEvaluation(ctx context.Context, flag string, defaultValue string, evalCtx openfeature.FlattenedContext) openfeature.StringResolutionDetail {
	return mp.strategy.StringEvaluation(ctx, flag, defaultValue, evalCtx)
}

// FloatEvaluation returns a float flag
func (mp *MultiProvider) FloatEvaluation(ctx context.Context, flag string, defaultValue float64, evalCtx openfeature.FlattenedContext) openfeature.FloatResolutionDetail {
	return mp.strategy.FloatEvaluation(ctx, flag, defaultValue, evalCtx)
}

// IntEvaluation returns an int flag
func (imp *MultiProvider) IntEvaluation(ctx context.Context, flag string, defaultValue int64, evalCtx openfeature.FlattenedContext) openfeature.IntResolutionDetail {
	return mp.strategy.IntEvaluation(ctx, flag, defaultValue, evalCtx)
}

// ObjectEvaluation returns an object flag
func (mp *MultiProvider) ObjectEvaluation(ctx context.Context, flag string, defaultValue interface{}, evalCtx openfeature.FlattenedContext) openfeature.InterfaceResolutionDetail {
	return mp.strategy.ObjectEvaluation(ctx, flag, defaultValue, evalCtx)
}

// Init will run the initialize method for all of provides and aggregate the errors.
func (mp *MultiProvider) Init(evalCtx openfeature.EvaluationContext) error {
	var wg sync.WaitGroup
	errChan := make(chan err.StateErr, len(mp.providersEntries))

	for _, provider := range mp.providersEntries {
		wg.Add(1)
		go func(p UniqueNameProvider) {
			defer wg.Done()
			if stateHandle, ok := p.Provider.(openfeature.StateHandler); ok {
				if initErr := stateHandle.Init(evalCtx); initErr != nil {
					errChan <- err.StateErr{ProviderName: p.UniqueName, Err: initErr, ErrMessage: initErr.Error()}
				}
			}
		}(provider)
	}

	go func() {
		wg.Wait()
		close(errChan)
	}()

	var errors []err.StateErr
	for err := range errChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		var aggErr err.AggregateError
		aggErr.Construct(errors)
		mp.status = openfeature.ErrorState
		return &aggErr
	}

	mp.status = openfeature.ReadyState

	return nil
}

func (mp *MultiProvider) Status() openfeature.State {
	return mp.status
}

func (mp *MultiProvider) Shutdown() {
	var wg sync.WaitGroup

	for _, provider := range mp.providersEntries {
		wg.Add(1)
		go func(p UniqueNameProvider) {
			defer wg.Done()
			if stateHandle, ok := p.Provider.(openfeature.StateHandler); ok {
				stateHandle.Shutdown()
			}
		}(provider)
	}

	wg.Wait()
}

func (mp *MultiProvider) EventChannel() <-chan openfeature.Event {
	return mp.events
}