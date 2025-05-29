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
	PollingPayloadCacheKey = "HttpConnector.polling-payload"
)

type HttpConnector struct {
	options                    HttpConnectorOptions
	client                     *http.Client
	ticker                     *time.Ticker
	cacheFetcher               *HttpCacheFetcher
	failSafeCache              *FailSafeCache
	shutdownChan               chan bool
	payloadCachePollTtlSeconds int
	shutdownOnce               sync.Once
}

func (h *HttpConnector) Init(ctx context.Context) error {
	return nil
}

func (h *HttpConnector) IsReady() bool {
	return true
}

func (h *HttpConnector) Sync(ctx context.Context, dataSync chan<- flagdsync.DataSync) error {
	h.options.log.Logger.Info("Starting HTTP connector sync",
		zap.Int("poll_interval_seconds", h.options.PollIntervalSeconds),
	)

	h.options.log.Logger.Debug("Initial polling for updates")
	success := h.fetchAndUpdate(dataSync)
	if !success {
		h.options.log.Logger.Warn("Failed to fetch initial data from HTTP source, using cache if available")
		h.updateFromCache(dataSync)
	}

	h.ticker = time.NewTicker(time.Duration(h.options.PollIntervalSeconds) * time.Second)
	go func() {
		for {
			select {
			case <-h.ticker.C:
				h.options.log.Logger.Debug("Polling for updates")
				h.fetchAndUpdate(dataSync)
			case <-h.shutdownChan:
				h.options.log.Logger.Info("Shutting down HTTP connector sync")
				return
			}
		}
	}()

	return nil
}

func (h *HttpConnector) ReSync(ctx context.Context, dataSync chan<- flagdsync.DataSync) error {
	success := h.fetchAndUpdate(dataSync)
	if !success {
		h.options.log.Logger.Warn("Failed to fetch initial data from HTTP source, using cache if available")
		h.updateFromCache(dataSync)
	}
	return nil
}

var _ flagdsync.ISync = &HttpConnector{}

func NewHttpConnector(opts HttpConnectorOptions) (*HttpConnector, error) {
	if opts.log == nil {
		return nil, errors.New("log is required for HttpConnector")
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

	client := &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}

	h := &HttpConnector{
		options:      opts,
		client:       client,
		shutdownChan: make(chan bool),
	}

	var err error
	if opts.UseFailsafeCache && opts.PayloadCache != nil {
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
	h.options.log.Logger.Debug("fetchAndUpdate called")
	if h.options.UsePollingCache && h.options.PayloadCache != nil {
		payload, err := h.options.PayloadCache.Get(PollingPayloadCacheKey)
		if err != nil {
			h.options.log.Debug("Failed to get payload from cache", zap.Error(err))
		}
		if payload != "" {
			h.options.log.Logger.Debug("Using cached payload")
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
		h.options.log.Logger.Debug("Using HTTP cache fetcher")
		resp, payload, err = h.cacheFetcher.FetchContent(h.client, req)
		if err != nil {
			return false
		}
	} else {
		h.options.log.Logger.Debug("Using direct HTTP request", zap.String("url", h.options.URL))
		resp, err = h.client.Do(req)
		defer resp.Body.Close()
		if err != nil {
			h.options.log.Error("HTTP request failed", zap.Error(err), zap.String("url", h.options.URL))
			return false
		}
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotModified {
			body, _ := io.ReadAll(resp.Body)
			h.options.log.Error("HTTP request failed", zap.Error(err), zap.String("response", string(body)))
			return false
		}

		if resp.StatusCode == http.StatusNotModified {
			h.options.log.Logger.Debug("HTTP response not modified, using cached payload")
			return true
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			h.options.log.Error("Failed to read response body", zap.Error(err))
			return false
		}
		payload = string(body)
	}

	if resp.StatusCode == http.StatusNotModified {
		h.options.log.Logger.Debug("HTTP response not modified, skipping update")
		return true
	}

	go func() {
		h.options.log.Logger.Debug("scheduling cache update if needed")
		h.updateCache(payload)
	}()
	if dataSync != nil {
		h.options.log.Logger.Debug("Sending data sync")

		h.options.log.Logger.Sync()

		dataSync <- flagdsync.DataSync{
			FlagData: payload,
			Source:   h.options.URL,
			Selector: "",
			Type:     flagdsync.ALL,
		}
		h.options.log.Logger.Debug("Data sync sent successfully")
	}
	return true
}

func (h *HttpConnector) updateCache(payload string) {
	h.options.log.Logger.Debug("Updating payload cache")
	if h.failSafeCache != nil {
		h.options.log.Logger.Debug("Updating fail-safe cache with new payload")
		h.failSafeCache.UpdatePayloadIfNeeded(payload)
	}
	if h.options.PayloadCache != nil {
		h.options.log.Logger.Debug("Updating polling payload cache with new payload")
		h.options.PayloadCache.PutWithTTL(PollingPayloadCacheKey, payload,
			h.payloadCachePollTtlSeconds)
	}
}

func (h *HttpConnector) updateFromCache(dataSync chan<- flagdsync.DataSync) {
	var flagData string
	var err error
	if h.options.PayloadCache != nil {
		h.options.log.Logger.Debug("Fetching cached payload from cache")
		flagData, err = h.options.PayloadCache.Get(PollingPayloadCacheKey)
		if err == nil {
			h.options.log.Logger.Debug("Cached payload found", zap.String("payload", flagData))
		} else {
			h.options.log.Logger.Error("Failed to get cached payload", zap.Error(err))
		}
	}
	if flagData == "" && h.failSafeCache != nil {
		h.options.log.Logger.Debug("Fetching cached payload from fail-safe cache")
		flagData = h.failSafeCache.Get()
		if flagData == "" {
			h.options.log.Logger.Debug("No cached payload found in fail-safe cache")
		}
	}
	if dataSync != nil && flagData != "" {
		h.options.log.Logger.Debug("Sending cached data sync")

		h.options.log.Logger.Sync()

		dataSync <- flagdsync.DataSync{
			FlagData: flagData,
			Source:   h.options.URL,
			Selector: "",
			Type:     flagdsync.ALL,
		}
	}
}

func (h *HttpConnector) Shutdown() {
	h.shutdownOnce.Do(func() {
		h.options.log.Logger.Info("Shutting down HTTP connector")
		if h.shutdownChan != nil {
			h.options.log.Logger.Debug("Closing shutdown channel")
			close(h.shutdownChan)
		}
		if (h.ticker != nil) && (h.ticker.C != nil) {
			h.options.log.Logger.Debug("Stopping ticker")
			h.ticker.Stop()
		}
		h.options.log.Logger.Info("HTTP connector shutdown complete")
	})

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
