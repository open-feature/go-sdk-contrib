package gofeatureflag

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

type EvaluationType string

const (
	EvaluationTypeInProcess EvaluationType = "INPROCESS"
	EvaluationTypeRemote    EvaluationType = "REMOTE"
)

// ProviderOptions is the struct containing the provider options you can
// use while initializing GO Feature Flag.
// To have a valid configuration you need to have an Endpoint or GOFeatureFlagConfig set.
type ProviderOptions struct {
	// Endpoint contains the DNS of your GO Feature Flag relay proxy (ex: http://localhost:1031)
	Endpoint string

	// HTTPClient (optional) is the HTTP Client we will use to contact GO Feature Flag.
	// By default, we are using a custom HTTPClient with a timeout configure to 10000 milliseconds.
	HTTPClient *http.Client

	// APIKey  (optional) If the relay proxy is configured to authenticate the requests, you should provide
	// an API Key to the provider. Please ask the administrator of the relay proxy to provide an API Key.
	// (This feature is available only if you are using GO Feature Flag relay proxy v1.7.0 or above)
	// Default: null
	APIKey string

	// Headers (optional) is the custom headers to be sent with the request.
	Headers map[string]string

	// ExporterMetadata (optional) is the metadata we send to the GO Feature Flag relay proxy when we report the
	// evaluation data usage.
	ExporterMetadata map[string]any

	// FlagChangePollingInterval (optional) interval time we poll the proxy to check if the configuration has changed.
	// default: 120000ms
	FlagChangePollingInterval time.Duration

	// EvaluationType (optional) is the type of evaluation to use.
	// default: INPROCESS
	EvaluationType EvaluationType

	// DataCollectorMaxEventStored (optional) is the maximum number of events we store in the data collector.
	// default: 100000
	DataCollectorMaxEventStored int64

	// DataCollectorCollectInterval (optional) is the interval time we send the data to the GO Feature Flag relay proxy.
	// default: 2 minutes
	DataCollectorCollectInterval time.Duration

	// DataCollectorDisabled (optional) is the flag to disabled the data collector.
	// default: false
	DataCollectorDisabled bool

	// Logger (optional) is the logger to be used by the provider.
	// default: slog.Default()
	Logger *slog.Logger
}

func (o *ProviderOptions) Validation() error {
	if o.Endpoint == "" {
		return fmt.Errorf("invalid option: %s", o.Endpoint)
	}
	return nil
}
