package internal

import "fmt"

// InitError is how the error in the Init stage of a provider is reported.
type InitError struct {
	ProviderName string
	Err        error
}

func (e *InitError) Error() string {
	return fmt.Sprintf("Provider %s had an error: %v", e.ProviderName, e.Err)
}

type AggregateError struct {
	Message string
	Errors []InitError
}

func (ae *AggregateError) Error() string {
	return ae.Message
}

func (ae *AggregateError) Construct(providerErrors []InitError) {
	// Show first error message for convenience, but all errors in the object
	msg := fmt.Sprintf("Provider errors occurred: %s: %v", providerErrors[0].ProviderName, providerErrors[0].Err)

	ae.Message = msg
	ae.Errors = providerErrors
}