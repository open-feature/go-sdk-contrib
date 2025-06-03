package flagdhttpconnector

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"

	flagdlogger "github.com/open-feature/flagd/core/pkg/logger"
)

type HttpConnectorOptions struct {
	Log                   *flagdlogger.Logger
	PollIntervalSeconds   int
	ConnectTimeoutSeconds int
	RequestTimeoutSeconds int
	Headers               map[string]string
	ProxyHost             string
	ProxyPort             int
	PayloadCacheOptions   *PayloadCacheOptions
	PayloadCache          PayloadCache
	UseHttpCache          bool
	UseFailsafeCache      bool
	UsePollingCache       bool
	URL                   string
	Client                *http.Client
}

// NewHttpConnectorOptions creates a new instance and validates it
func NewHttpConnectorOptions(opts HttpConnectorOptions) (*HttpConnectorOptions, error) {
	if err := Validate(&opts); err != nil {
		return nil, err
	}
	return &opts, nil
}

func Validate(o *HttpConnectorOptions) error {
	if err := validateURL(o.URL); err != nil {
		return err
	}
	if o.RequestTimeoutSeconds < 1 || o.RequestTimeoutSeconds > 60 {
		return errors.New("requestTimeoutSeconds must be between 1 and 60")
	}
	if o.ConnectTimeoutSeconds < 1 || o.ConnectTimeoutSeconds > 60 {
		return errors.New("connectTimeoutSeconds must be between 1 and 60")
	}
	if o.PollIntervalSeconds < 1 || o.PollIntervalSeconds > 600 {
		return errors.New("pollIntervalSeconds must be between 1 and 600")
	}
	if o.ProxyHost != "" && o.ProxyPort == 0 {
		return errors.New("proxyPort must be set if proxyHost is set")
	} else if o.ProxyHost == "" && o.ProxyPort != 0 {
		return errors.New("proxyHost must be set if proxyPort is set")
	}
	if (o.PayloadCacheOptions != nil && o.PayloadCache == nil) ||
		(o.PayloadCache != nil && o.PayloadCacheOptions == nil) {
		return errors.New("both payloadCache and payloadCacheOptions must be set together")
	}
	if (o.UseFailsafeCache || o.UsePollingCache) && o.PayloadCache == nil {
		return errors.New("payloadCache must be set if useFailsafeCache or usePollingCache is true")
	}
	if o.UsePollingCache && !implementsPutWithTTL(o.PayloadCache) {
		return errors.New("when usePollingCache is set, payloadCache must implement Put(key, payload, ttlSeconds)")
	}
	if o.Log == nil {
		return errors.New("log is required for HttpConnector")
	}
	return nil
}

func validateURL(raw string) error {
	parsed, err := url.ParseRequestURI(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("invalid URL: %s", raw)
	}
	return nil
}

// This mimics method reflection check for interface satisfaction
func implementsPutWithTTL(pc PayloadCache) bool {
	return pc != nil // add actual check if needed
}
