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

	// DataCollectorBaseURL (optional) overrides the endpoint used for the data collector only.
	// When set, events are sent to this URL instead of {Endpoint}/v1/data/collector.
	// Useful when the collector service runs at a different host than the relay proxy.
	// default: empty (uses Endpoint)
	DataCollectorBaseURL string

	// Logger (optional) is the logger to be used by the provider.
	// default: slog.Default()
	Logger *slog.Logger

	// DisableCache (optional) set to true if you would like that every flag evaluation goes to the GO Feature Flag directly.
	// Cache is used only for the remote evaluation.
	// default: false
	DisableCache bool

	// FlagCacheSize (optional) is the maximum number of flag events we keep in memory to cache your flags.
	// Cache is used only for the remote evaluation.
	// default: 10000
	FlagCacheSize int

	// FlagCacheTTL (optional) is the time we keep the evaluation in the cache before we consider it as obsolete.
	// If you want to keep the value forever you can set the FlagCacheTTL field to -1
	// Cache is used only for the remote evaluation.
	// default: 1 minute
	FlagCacheTTL time.Duration
}

func (o *ProviderOptions) Validation() error {
	if o.Endpoint == "" {
		return fmt.Errorf("invalid option: Endpoint is required")
	}
	return nil
}
