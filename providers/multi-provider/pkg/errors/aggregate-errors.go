package errors

import (
	"fmt"
	"golang.org/x/exp/maps"
	"strings"
)

type (
	// ProviderError is how the error in the Init stage of a provider is reported.
	ProviderError struct {
		Err          error
		ProviderName string
	}

	// AggregateError map that contains up to one error per provider within the multi-provider
	AggregateError map[string]ProviderError
)

var (
	_ error = (*ProviderError)(nil)
	_ error = (AggregateError)(nil)
)

func (e *ProviderError) Error() string {
	return fmt.Sprintf("Provider %s: %s", e.ProviderName, e.Err.Error())
}

// NewAggregateError Creates a new AggregateError
func NewAggregateError(providerErrors []ProviderError) *AggregateError {
	err := make(AggregateError)
	for _, se := range providerErrors {
		err[se.ProviderName] = se
	}
	return &err
}

func (ae AggregateError) Error() string {
	size := len(ae)
	switch size {
	case 0:
		return ""
	case 1:
		for _, err := range ae {
			return err.Error()
		}
	default:
		errs := make([]string, 0, size)
		for _, err := range maps.Values(ae) {
			errs = append(errs, err.Error())
		}
		return strings.Join(errs, ", ")
	}

	return "" // This will never occur, switch is exhaustive
}
