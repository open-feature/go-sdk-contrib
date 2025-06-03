package flagdhttpconnector

import (
	context "context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	flagdsync "github.com/open-feature/flagd/core/pkg/sync"
	"go.uber.org/zap"
)

const (
	PollingPayloadCacheKey       = "HttpConnector.polling-payload"
	DefaultPollIntervalSeconds   = 60
	DefaultConnectTimeoutSeconds = 10
	DefaultRequestTimeoutSeconds = 10
)

type HttpConnector struct {
	options                    HttpConnectorOptions
	client                     *http.Client
	ticker                     *time.Ticker
	cacheFetcher               *HttpCacheFetcher
	failSafeCache              *FailSafeCache
	shutdownChan               chan bool
	payloadCachePollTtlSeconds int
	initLock                   sync.Mutex
	isInitialized              bool
	isClosed                   bool
}

func (h *HttpConnector) Init(ctx context.Context) error {
	return nil
}

func (h *HttpConnector) IsReady() bool {
	return true
}

func (h *HttpConnector) Sync(ctx context.Context, dataSync chan<- flagdsync.DataSync) error {
	h.options.Log.Logger.Debug("Sync called")
	h.initLock.Lock()
	defer h.initLock.Unlock()
	if h.client == nil {
		h.options.Log.Logger.Error("HTTP client is not initialized")
		return errors.New("not initialized")
	}
	if h.isInitialized {
		h.options.Log.Logger.Info("HttpConnector is already initialized, skipping re-initialization")
		return nil
	}
	h.options.Log.Logger.Info("Starting HTTP connector sync",
		zap.Int("poll_interval_seconds", h.options.PollIntervalSeconds),
	)

	h.options.Log.Logger.Debug("Initial polling for updates")
	success := h.fetchAndUpdate(dataSync)
	if !success {
		h.options.Log.Logger.Warn("Failed to fetch initial data from HTTP source, using cache if available")
		h.updateFromCache(dataSync)
	}

	h.ticker = time.NewTicker(time.Duration(h.options.PollIntervalSeconds) * time.Second)
	go func() {
		for {
			select {
			case <-h.ticker.C:
				h.options.Log.Logger.Debug("Polling for updates")
				h.fetchAndUpdate(dataSync)
			case <-h.shutdownChan:
				h.options.Log.Logger.Info("Shutting down HTTP connector sync")
				return
			}
		}
	}()

	h.isInitialized = true
	return nil
}

func (h *HttpConnector) ReSync(ctx context.Context, dataSync chan<- flagdsync.DataSync) error {
	h.options.Log.Logger.Debug("ReSync called, doing nothing as HttpConnector does not support re-sync")
	return nil
}

var _ flagdsync.ISync = &HttpConnector{}

func NewHttpConnector(options HttpConnectorOptions) (*HttpConnector, error) {
	opts := options
	if opts.PollIntervalSeconds == 0 {
		opts.PollIntervalSeconds = DefaultPollIntervalSeconds
	}
	if opts.ConnectTimeoutSeconds == 0 {
		opts.ConnectTimeoutSeconds = DefaultConnectTimeoutSeconds
	}
	if opts.RequestTimeoutSeconds == 0 {
		opts.RequestTimeoutSeconds = DefaultRequestTimeoutSeconds
	}

	if err := Validate(&opts); err != nil {
		return nil, err
	}
	timeout := time.Duration(opts.RequestTimeoutSeconds) * time.Second
	transport := &http.Transport{}

	if opts.ProxyHost != "" && opts.ProxyPort != 0 {
		proxyURL := &url.URL{
			Scheme: "http",
			Host:   net.JoinHostPort(opts.ProxyHost, string(rune(opts.ProxyPort))),
		}
		transport.Proxy = http.ProxyURL(proxyURL)
	}

	h := &HttpConnector{
		options:      opts,
		shutdownChan: make(chan bool),
	}

	if opts.Client == nil {
		h.client = &http.Client{
			Timeout:   timeout,
			Transport: transport,
		}
	} else {
		h.client = opts.Client
	}

	var err error
	if opts.UseFailsafeCache {
		if opts.PayloadCache == nil || opts.PayloadCacheOptions == nil {
			return nil, errors.New("payloadCache and payloadCacheOptions must be set when UseFailsafeCache is true")
		}
		h.failSafeCache, err = NewFailSafeCache(opts.PayloadCache, opts.PayloadCacheOptions)
		if err != nil {
			return nil, err
		}
	}
	if opts.UseHttpCache {
		h.cacheFetcher = &HttpCacheFetcher{}
	}
	h.payloadCachePollTtlSeconds = opts.PollIntervalSeconds

	return h, nil
}

func (h *HttpConnector) fetchAndUpdate(dataSync chan<- flagdsync.DataSync) bool {
	h.options.Log.Logger.Debug("fetchAndUpdate called")
	if h.options.UsePollingCache && h.options.PayloadCache != nil {
		payload, err := h.options.PayloadCache.Get(PollingPayloadCacheKey)
		if err != nil {
			h.options.Log.Debug("Failed to get payload from cache", zap.Error(err))
		}
		if payload != "" {
			h.options.Log.Logger.Debug("Using cached payload")
			return true
		}
	}

	req, err := http.NewRequest("GET", h.options.URL, nil)
	if err != nil {
		return false
	}
	for k, v := range h.options.Headers {
		req.Header.Set(k, v)
	}

	var resp *http.Response
	var payload string
	if h.cacheFetcher != nil {
		h.options.Log.Logger.Debug("Using HTTP cache fetcher")
		resp, payload, err = h.cacheFetcher.FetchContent(h.client, req)
		if err != nil {
			return false
		}
	} else {
		h.options.Log.Logger.Debug("Using direct HTTP request", zap.String("url", h.options.URL))
		resp, err = h.client.Do(req)
		defer func() {
			if resp != nil && resp.Body != nil {
				io.Copy(io.Discard, resp.Body) // drain the body to avoid resource leaks
				resp.Body.Close()
			}
		}()
		if err != nil {
			h.options.Log.Error("HTTP request failed", zap.Error(err), zap.String("url", h.options.URL))
			return false
		}
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotModified {
			body, _ := io.ReadAll(resp.Body)
			h.options.Log.Error("HTTP request failed", zap.Error(err), zap.String("response", string(body)))
			return false
		}

		if resp.StatusCode == http.StatusNotModified {
			h.options.Log.Logger.Debug("HTTP response not modified, using cached payload")
			return true
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			h.options.Log.Error("Failed to read response body", zap.Error(err))
			return false
		}
		payload = string(body)
	}

	if resp.StatusCode == http.StatusNotModified {
		h.options.Log.Logger.Debug("HTTP response not modified, skipping update")
		return true
	}

	go func() {
		h.options.Log.Logger.Debug("scheduling cache update if needed")
		h.updateCache(payload)
	}()
	if dataSync != nil {
		h.options.Log.Logger.Debug("Sending data sync")

		h.options.Log.Logger.Sync()

		dataSync <- flagdsync.DataSync{
			FlagData: payload,
			Source:   h.options.URL,
			Selector: "",
			Type:     flagdsync.ALL,
		}
		h.options.Log.Logger.Debug("Data sync sent successfully")
	}
	return true
}

func (h *HttpConnector) updateCache(payload string) {
	h.options.Log.Logger.Debug("Updating payload cache")
	if h.failSafeCache != nil {
		h.options.Log.Logger.Debug("Updating fail-safe cache with new payload")
		h.failSafeCache.UpdatePayloadIfNeeded(payload)
	}
	if h.options.PayloadCache != nil {
		h.options.Log.Logger.Debug("Updating polling payload cache with new payload")
		h.options.PayloadCache.PutWithTTL(PollingPayloadCacheKey, payload,
			h.payloadCachePollTtlSeconds)
	}
}

func (h *HttpConnector) updateFromCache(dataSync chan<- flagdsync.DataSync) {
	var flagData string
	var err error
	if h.options.PayloadCache != nil {
		h.options.Log.Logger.Debug("Fetching cached payload from cache")
		flagData, err = h.options.PayloadCache.Get(PollingPayloadCacheKey)
		if err == nil {
			h.options.Log.Logger.Debug("Cached payload found")
		} else {
			h.options.Log.Logger.Error("Failed to get cached payload", zap.Error(err))
		}
	}
	if flagData == "" && h.failSafeCache != nil {
		h.options.Log.Logger.Debug("Fetching cached payload from fail-safe cache")
		flagData = h.failSafeCache.Get()
		if flagData == "" {
			h.options.Log.Logger.Debug("No cached payload found in fail-safe cache")
		}
	}
	if dataSync != nil && flagData != "" {
		h.options.Log.Logger.Debug("Sending cached data sync")

		h.options.Log.Logger.Sync()

		dataSync <- flagdsync.DataSync{
			FlagData: flagData,
			Source:   h.options.URL,
			Selector: "",
			Type:     flagdsync.ALL,
		}
	}
}

func waitWithTimeout(wg *sync.WaitGroup, timeout time.Duration) bool {
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return true // completed normally
	case <-time.After(timeout):
		return false // timed out
	}
}

func (h *HttpConnector) Shutdown() {
	h.options.Log.Logger.Debug("Shutdown called")
	h.initLock.Lock()
	defer h.initLock.Unlock()
	if !h.isInitialized {
		h.options.Log.Logger.Info("HTTP connector is not initialized, nothing to shutdown")
		return
	}
	if h.isClosed {
		h.options.Log.Logger.Info("HTTP connector is already closed, skipping shutdown")
		return
	}
	h.options.Log.Logger.Info("Shutting down HTTP connector")
	if h.shutdownChan != nil {
		h.options.Log.Logger.Debug("Closing shutdown channel")
		close(h.shutdownChan)
	}
	if (h.ticker != nil) && (h.ticker.C != nil) {
		h.options.Log.Logger.Debug("Stopping ticker")
		h.ticker.Stop()
	}
	h.options.Log.Logger.Info("HTTP connector shutdown complete")
	h.isClosed = true
}
