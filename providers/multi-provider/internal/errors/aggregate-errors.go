package errors

import (
	"encoding/json"
	"fmt"
)

// StateErr is how the error in the Init stage of a provider is reported.
type StateErr struct {
	ProviderName string `json:"source"`
	Err          error  `json:"-"`
	ErrMessage   string `json:"error"`
}

func (e *StateErr) Error() string {
	return fmt.Sprintf("Provider %s had an error: %v", e.ProviderName, e.Err)
}

type AggregateError struct {
	Message string     `json:"message"`
	Errors  []StateErr `json:"errors"`
}

func (ae *AggregateError) Error() string {
	errorsJSON, err := json.Marshal(ae.Errors)
	if err != nil {
		return fmt.Sprintf("Error in json marshal of errors, %s", err)
	}

	return fmt.Sprintf("%s\n%s", ae.Message, string(errorsJSON))

}

func (ae *AggregateError) Construct(providerErrors []StateErr) {
	// Show first error message for convenience, but all errors in the object
	msg := fmt.Sprintf("Provider errors occurred: %s: %v", providerErrors[0].ProviderName, providerErrors[0].Err)

	ae.Message = msg
	ae.Errors = providerErrors
}
