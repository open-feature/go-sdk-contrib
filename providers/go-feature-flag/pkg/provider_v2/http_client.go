package provider_v2

import (
	"net/http"
)

// HTTPClient is a custom interface to be able to override it by any implementation
// of an HTTP client.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// DefaultHTTPClient is the default HTTP client used to call GO Feature Flag.
// By default, we have a timeout of 10000 milliseconds.
