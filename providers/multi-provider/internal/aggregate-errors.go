package internal

import "fmt"

// StateErr is how the error in the Init of Shutdown stage of a provider is reported.
type StateErr struct {
	ProviderName string
	Err          error
}

func (e *StateErr) Error() string {
	return fmt.Sprintf("Provider %s had an error: %v", e.ProviderName, e.Err)
}

type AggregateError struct {
	Message string
	Errors  []StateErr
}

func (ae *AggregateError) Error() string {
	return ae.Message
}

func (ae *AggregateError) Construct(providerErrors []StateErr) {
	// Show first error message for convenience, but all errors in the object
	msg := fmt.Sprintf("Provider errors occurred: %s: %v", providerErrors[0].ProviderName, providerErrors[0].Err)

	ae.Message = msg
	ae.Errors = providerErrors
}
