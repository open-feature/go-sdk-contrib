package gofeatureflag

import (
	"net/http"
	"net/url"
	"time"

	"github.com/open-feature/go-sdk-contrib/providers/go-feature-flag/pkg/gofferror"
)

type EvaluationType string

const (
	EvaluationTypeInProcess EvaluationType = "InProcess"
	EvaluationTypeRemote    EvaluationType = "Remote"
)

// ProviderOptions is the struct containing the provider options you can
// use while initializing GO Feature Flag.
// To have a valid configuration you need to have an Endpoint or GOFeatureFlagConfig set.
type ProviderOptions struct {
	// Endpoint contains the DNS of your GO Feature Flag relay proxy (ex: http://localhost:1031)
	Endpoint string

	// EvaluationType (optional) type of evaluation to use.
	// If not set, we will use the default evaluation type.
	// default: InProcess
	EvaluationType EvaluationType

	// HTTPClient (optional) is the HTTP Client we will use to contact GO Feature Flag.
	// By default, we are using a custom HTTPClient with a timeout configure to 10000 milliseconds.
	HTTPClient *http.Client

	// APIKey  (optional) If the relay proxy is configured to authenticate the requests, you should provide
	// an API Key to the provider. Please ask the administrator of the relay proxy to provide an API Key.
	// (This feature is available only if you are using GO Feature Flag relay proxy v1.7.0 or above)
	// Default: null
	APIKey string

	// DataFlushInterval (optional) interval time we use to call the relay proxy to collect data.
	// The parameter is used only if the cache is enabled, otherwise the collection of the data is done directly
	// when calling the evaluation API.
	// default: 1 minute
	DataFlushInterval time.Duration

	// DataCollectorMaxEventStored (optional) maximum number of event we keep in memory, if we reach this number it means
	// that we will start to drop the new events. This is a security to avoid a memory leak.
	// default: 100000
	DataCollectorMaxEventStored int64

	// DisableDataCollector (optional) set to true if you would like to disable the data collector.
	DisableDataCollector bool

	// DataCollectorEndpoint (optional) endpoint to send the data collector events to.
	// If not set, we will use the same endpoint as the provider endpoint.
	DataCollectorEndpoint string

	// ExporterMetadata (optional) is the metadata we send to the GO Feature Flag relay proxy when we report the
	// evaluation data usage.
	ExporterMetadata map[string]any
}

func (o *ProviderOptions) Validation() error {
	if err := validateEndpoint(o.Endpoint); err != nil {
		return err
	}

	if o.DataCollectorEndpoint != "" {
		if err := validateEndpoint(o.DataCollectorEndpoint); err != nil {
			return err
		}
	}
	return nil
}

func validateEndpoint(endpoint string) error {
	if endpoint == "" {
		return gofferror.NewInvalidOption("invalid option: endpoint is required")
	}

	// Validate that the endpoint is a valid URL
	parsedURL, err := url.Parse(endpoint)
	if err != nil {
		return gofferror.NewInvalidOption("invalid option: endpoint must be a valid URL")
	}

	// Validate that the URL has a scheme (http or https)
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return gofferror.NewInvalidOption("invalid option: endpoint must have http or https scheme")
	}

	// Validate that the URL has a host
	if parsedURL.Host == "" {
		return gofferror.NewInvalidOption("invalid option: endpoint must have a valid host")
	}

	return nil
}
