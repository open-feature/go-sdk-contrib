package multiprovider

import (
	"errors"
	"fmt"
	"sync"

	of "github.com/open-feature/go-sdk/openfeature"
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
	event                  chan of.Event
	status                 of.State
	mu                     sync.Mutex
}

func NewMultiProvider(passedProviders []UniqueNameProvider, evaluationStrategy string) (*MultiProvider, error) {
	multiProvider := &MultiProvider{
		providersEntries:       []UniqueNameProvider{},
		providersEntriesByName: map[string]UniqueNameProvider{},
		AggregatedMetadata:     map[string]of.Metadata{},
	}

	err := registerProviders(multiProvider, passedProviders)
	if err != nil {
		return nil, err
	}

	// err = multiProvider.initialize()
	// if err != nil {
	// 	return nil, err
	// }

	return multiProvider, nil
}

// Metadata provides the name `multiprovider` and the names of each provider passed.
func (mp MultiProvider) Metadata() MultiMetadata {

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

type InitError struct {
	ProviderName string
	Error          error
}

// Init will run the initialize method for all of provides and aggregate the errors.
func (mp *MultiProvider) Init(evalCtx of.EvaluationContext) error {
	var wg sync.WaitGroup

	errChan := make(chan InitError, len(mp.providersEntries))
	for _, provider := range mp.providersEntries {
		wg.Add(1)
		go func(p UniqueNameProvider) {
			defer wg.Done()
			if initMethod, ok := p.Provider.(of.StateHandler); ok {
				if err := initMethod.Init(evalCtx); err != nil {
					errChan <- InitError{ProviderName: p.Name, Error: err}
				}
			}
		}(provider)
	}

	wg.Wait()
	close(errChan)


	return nil
}

func (mp *MultiProvider) Status() of.State {
	return of.ReadyState
}

func (mp *MultiProvider) Shutdown() {

}

func (mp *MultiProvider) EventChannel() <-chan of.Event {
	return
}
