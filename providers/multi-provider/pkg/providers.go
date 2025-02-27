package multiprovider

import (
	"errors"
	"fmt"
	"sync"

	err "github.com/open-feature/go-sdk-contrib/providers/multi-provider/internal"

	of "github.com/open-feature/go-sdk/openfeature"
	ofhooks "github.com/open-feature/go-sdk/openfeature/hooks"
)

var (
	errUniqueName = errors.New("Provider names must be unique.")
)

// UniqueNameProvider allows for a unique name to be assigned to a provider during a multi-provider set up.
// The name will be used when reporting errors & results to specify the provider associated.
type UniqueNameProvider struct {
	Provider of.FeatureProvider
	Name     string
}

// MultiMetadata defines the return of the MultiProvider metadata with the aggregated data of all the providers.
type MultiMetadata struct {
	Name             string
	OriginalMetadata map[string]of.Metadata
}

// MultiProvider implements openfeature `FeatureProvider` in a way to accept an array of providers.
type MultiProvider struct {
	providersEntries       []UniqueNameProvider
	providersEntriesByName map[string]UniqueNameProvider
	AggregatedMetadata     map[string]of.Metadata
	EvaluationStrategy     string
	events                 chan of.Event
	status                 of.State
	mu                     sync.Mutex
}

// NewMultiProvider returns the unified interface of multiple providers for interaction.
func NewMultiProvider(passedProviders []UniqueNameProvider, evaluationStrategy string, logger ofhooks.LoggingHook) (*MultiProvider, error) {
	multiProvider := &MultiProvider{
		providersEntries:       []UniqueNameProvider{},
		providersEntriesByName: map[string]UniqueNameProvider{},
		AggregatedMetadata:     map[string]of.Metadata{},
	}

	err := registerProviders(multiProvider, passedProviders)
	if err != nil {
		return nil, err
	}

	// err = multiProvider.Init(of.EvaluationContext{})
	// if err != nil {
	// 	return nil, err
	// }

	return multiProvider, nil
}

// Metadata provides the name `multiprovider` and the names of each provider passed.
func (mp *MultiProvider) Metadata() MultiMetadata {

	return MultiMetadata{
		Name:             "multiprovider",
		OriginalMetadata: mp.AggregatedMetadata,
	}
}

// registerProviders ensures that when setting up an instant of MultiProvider the providers provided either have a unique name or the base `metadata.Name` is made unique by adding an indexed based number to it.
// registerProviders also stores the providers by their unique name and in an array for easy usage.
func registerProviders(mp *MultiProvider, providers []UniqueNameProvider) error {
	providersByName := make(map[string][]UniqueNameProvider)

	for _, provider := range providers {
		uniqueName := provider.Name

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
			mp.providersEntries = append(mp.providersEntries, providers[0])
			mp.providersEntriesByName[name] = providers[0]
			mp.AggregatedMetadata[name] = providers[0].Provider.Metadata()
		} else {
			for i, provider := range providers {
				uniqueName := fmt.Sprintf("%s-%d", name, i+1)
				mp.providersEntries = append(mp.providersEntries, provider)
				mp.providersEntriesByName[uniqueName] = provider
				mp.AggregatedMetadata[uniqueName] = provider.Provider.Metadata()
			}
		}
	}
	return nil
}

// Init will run the initialize method for all of provides and aggregate the errors.
func (mp *MultiProvider) Init(evalCtx of.EvaluationContext) error {
	var wg sync.WaitGroup
	errChan := make(chan err.StateErr, len(mp.providersEntries))

	for _, provider := range mp.providersEntries {
		wg.Add(1)
		go func(p UniqueNameProvider) {
			defer wg.Done()
			if stateHandle, ok := p.Provider.(of.StateHandler); ok {
				if initErr := stateHandle.Init(evalCtx); initErr != nil {
					errChan <- err.StateErr{ProviderName: p.Name, Err: initErr}
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
		return &aggErr
	}

	return nil
}

func (mp *MultiProvider) Status() of.State {
	return of.ReadyState
}

func (mp *MultiProvider) Shutdown() {
	var wg sync.WaitGroup

	for _, provider := range mp.providersEntries {
		wg.Add(1)
		go func(p UniqueNameProvider) {
			defer wg.Done()
			if stateHandle, ok := p.Provider.(of.StateHandler); ok {
				stateHandle.Shutdown()
			}
		}(provider)
	}

	wg.Wait()
}

func (mp *MultiProvider) EventChannel() <-chan of.Event {
	ev := make(chan of.Event)
	return ev
}
