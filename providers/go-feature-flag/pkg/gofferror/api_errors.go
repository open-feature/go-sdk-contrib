package gofferror

// FlagConfigurationEndpointNotFoundError is returned when the flag configuration endpoint is not found.
type FlagConfigurationEndpointNotFoundError struct{}

func (e FlagConfigurationEndpointNotFoundError) Error() string {
	return "flag configuration endpoint not found"
}

func NewFlagConfigurationEndpointNotFoundError() FlagConfigurationEndpointNotFoundError {
	return FlagConfigurationEndpointNotFoundError{}
}

// UnauthorizedError is returned when the request is unauthorized.
type UnauthorizedError struct {
	Message string
}

func (e UnauthorizedError) Error() string {
	return e.Message
}

func NewUnauthorizedError(message string) UnauthorizedError {
	return UnauthorizedError{Message: message}
}

// ImpossibleToRetrieveConfigurationError is returned when it's impossible to retrieve the configuration.
type ImpossibleToRetrieveConfigurationError struct {
	Message string
}

func (e ImpossibleToRetrieveConfigurationError) Error() string {
	return e.Message
}

func NewImpossibleToRetrieveConfigurationError(message string) ImpossibleToRetrieveConfigurationError {
	return ImpossibleToRetrieveConfigurationError{Message: message}
}
