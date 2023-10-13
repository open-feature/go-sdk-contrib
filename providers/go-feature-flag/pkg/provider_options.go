package gofeatureflag

import (
	ffclient "github.com/thomaspoignant/go-feature-flag"
	"github.com/thomaspoignant/go-feature-flag/exporter"
	"time"
)

// ProviderOptions is the struct containing the provider options you can
// use while initializing GO Feature Flag.
// To have a valid configuration you need to have an Endpoint or GOFeatureFlagConfig set.
type ProviderOptions struct {
	// Endpoint contains the DNS of your GO Feature Flag relay proxy (ex: http://localhost:1031)
	Endpoint string

	// HTTPClient (optional) is the HTTP Client we will use to contact GO Feature Flag.
	// By default, we are using a custom HTTPClient with a timeout configure to 10000 milliseconds.
	HTTPClient HTTPClient

	// GOFeatureFlagConfig is the configuration struct for the GO Feature Flag module.
	// If not nil we will launch the provider using the GO Feature Flag module.
	GOFeatureFlagConfig *ffclient.Config

	// APIKey  (optional) If the relay proxy is configured to authenticate the requests, you should provide
	// an API Key to the provider. Please ask the administrator of the relay proxy to provide an API Key.
	// (This feature is available only if you are using GO Feature Flag relay proxy v1.7.0 or above)
	// Default: null
	APIKey string

	// DisableCache (optional) set to true if you would like that every flag evaluation goes to the GO Feature Flag directly.
	DisableCache bool

	// FlagCacheSize (optional) is the maximum number of flag events we keep in memory to cache your flags.
	// default: 10000
	FlagCacheSize int

	// FlagCacheTTL (optional) is the time we keep the evaluation in the cache before we consider it as obsolete.
	// If you want to keep the value forever you can set the FlagCacheTTL field to -1
	// default: 1 minute
	FlagCacheTTL time.Duration

	// DataFlushInterval (optional) interval time we use to call the relay proxy to collect data.
	// The parameter is used only if the cache is enabled, otherwise the collection of the data is done directly
	// when calling the evaluation API.
	// default: 1 minute
	DataFlushInterval time.Duration

	// DataMaxEventInMemory (optional) maximum number of item we keep in memory before calling the API.
	// If this number is reached before the DataFlushInterval we will call the API.
	// The parameter is used only if the cache is enabled, otherwise the collection of the data is done directly
	// when calling the evaluation API.
	// default: 500
	DataMaxEventInMemory int64

	// DataCollectorCustomExporter (optional) if you want to export the data somewhere else than in the GO Feature Flag
	// server, you can specify a custom data exporter.
	DataCollectorCustomExporter exporter.Exporter
}
