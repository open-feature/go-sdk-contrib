package vercel

import (
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	defaultHost            = "https://flags.vercel.com"
	defaultPollingInterval = 30 * time.Second
	defaultRequestTimeout  = 10 * time.Second
	providerName           = "vercel-go-provider"
)

type providerOptions struct {
	sdkKeyOrConnectionString string
	host                     string
	httpClient               *http.Client
	datafile                 *Datafile
	pollingEnabled           bool
	pollingInterval          time.Duration
}

// Option configures the Vercel OpenFeature provider.
type Option func(*providerOptions)

func defaultOptions() providerOptions {
	return providerOptions{
		sdkKeyOrConnectionString: os.Getenv("FLAGS"),
		host:                     defaultHost,
		httpClient:               &http.Client{Timeout: defaultRequestTimeout},
		pollingEnabled:           true,
		pollingInterval:          defaultPollingInterval,
	}
}

// WithSDKKey configures the provider with a raw Vercel Flags SDK key. For
// convenience, a FLAGS connection string is also accepted.
func WithSDKKey(sdkKeyOrConnectionString string) Option {
	return func(o *providerOptions) {
		o.sdkKeyOrConnectionString = sdkKeyOrConnectionString
	}
}

// WithConnectionString configures the provider with the FLAGS connection
// string emitted by Vercel.
func WithConnectionString(connectionString string) Option {
	return WithSDKKey(connectionString)
}

// WithHost overrides the Vercel Flags service host. This is primarily useful
// for tests.
func WithHost(host string) Option {
	return func(o *providerOptions) {
		o.host = strings.TrimRight(host, "/")
	}
}

// WithHTTPClient configures the HTTP client used for datafile requests.
func WithHTTPClient(client *http.Client) Option {
	return func(o *providerOptions) {
		if client != nil {
			o.httpClient = client
		}
	}
}

// WithDatafile seeds the provider with an already-fetched Vercel Flags datafile.
func WithDatafile(datafile Datafile) Option {
	return func(o *providerOptions) {
		o.datafile = &datafile
	}
}

// WithPollingInterval configures how often the provider refreshes the datafile
// after initialization.
func WithPollingInterval(interval time.Duration) Option {
	return func(o *providerOptions) {
		o.pollingEnabled = true
		o.pollingInterval = interval
	}
}

// WithPollingDisabled disables background datafile refreshes.
func WithPollingDisabled() Option {
	return func(o *providerOptions) {
		o.pollingEnabled = false
	}
}
