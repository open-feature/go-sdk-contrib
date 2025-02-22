package multiprovider

import (
	"errors"
	"fmt"

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

type MultiMetadata struct {
	Name             string
	OriginalMetadata map[string]of.Metadata
}

// MultiProvider implements openfeature `FeatureProvider` in a way to accept an array of providers.
type MultiProvider struct {
	providersEntries       []UniqueNameProvider
	providersEntriesByName map[string]UniqueNameProvider
	AggregatedMetadata     map[string]of.Metadata
}

func NewMultiProvider(providers []UniqueNameProvider) (*MultiProvider, error) {
	multiProvider := &MultiProvider{
		providersEntries:       []UniqueNameProvider{},
		providersEntriesByName: map[string]UniqueNameProvider{},
	}
	// for i, provider := range providers {

	// }

	//

	return multiProvider, nil
}

func (mp MultiProvider) Metadata() of.Metadata {

	return of.Metadata{
		Name: fmt.Sprintf("multiprovider"),
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
