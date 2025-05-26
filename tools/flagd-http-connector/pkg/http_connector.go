package flagdhttpconnector

import (
	context "context"
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

// type HttpConnector struct {
// 	PollIntervalSeconds        int
// 	RequestTimeoutSeconds      int
// 	Queue                      chan QueuePayload
// 	Client                     *http.Client
// 	HttpClientExecutor         sync.WaitGroup // Use WaitGroup or goroutines
// 	Scheduler                  *time.Ticker   // Can be used for periodic polling
// 	Headers                    map[string]string
// 	FailSafeCache              *FailSafeCache
// 	PayloadCache               *PayloadCache
// 	HttpCacheFetcher           *HttpCacheFetcher
// 	PayloadCachePollTtlSeconds int
// 	UsePollingCache            bool
// 	URL                        string
// 	URI                        *url.URL
// }

// HttpConnector polls a URL and feeds a queue.
type HttpConnector struct {
	options                    HttpConnectorOptions
	client                     *http.Client
	scheduler                  *time.Ticker
	cacheFetcher               *HttpCacheFetcher
	failSafeCache              *FailSafeCache
	shutdownChan               chan struct{}
	wg                         *sync.WaitGroup
	payloadCachePollTtlSeconds int
}

func (h HttpConnector) Init(ctx context.Context) error {
	return nil
}

func (h HttpConnector) IsReady() bool {
	return true
}

func (h HttpConnector) Sync(ctx context.Context, dataSync chan<- flagdsync.DataSync) error {
	h.scheduler = time.NewTicker(time.Duration(h.options.PollIntervalSeconds) * time.Second)
	h.wg.Add(1)
	go func() {
		defer h.wg.Done()
		for {
			select {
			case <-h.scheduler.C:
				h.fetchAndUpdate()
			case <-h.shutdownChan:
				h.scheduler.Stop()
				return
			}
		}
	}()

	return nil
}

func (h HttpConnector) ReSync(ctx context.Context, dataSync chan<- flagdsync.DataSync) error {
	success := h.fetchAndUpdate()
	if !success {
		h.updateFromCache()
	}
	return nil
}

var _ flagdsync.ISync = &HttpConnector{}

func NewHttpConnector(opts HttpConnectorOptions) (*HttpConnector, error) {
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
		shutdownChan: make(chan struct{}),
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

func (h *HttpConnector) fetchAndUpdate() bool {
	if h.options.UsePollingCache && h.options.PayloadCache != nil {
		payload, err := h.options.PayloadCache.Get(PollingPayloadCacheKey)
		if err != nil {
			return false
		}
		if payload != "" {
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
		resp, payload, err = h.cacheFetcher.FetchContent(h.client, req)
		if err != nil {
			return false
		}
	} else {
		resp, err = h.client.Do(req)
		if err != nil {
			return false
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotModified {
			body, _ := io.ReadAll(resp.Body)
			h.options.log.Error("HTTP request failed", zap.Error(err), zap.String("response", string(body)))
			return false
		}

		if resp.StatusCode == http.StatusNotModified {
			return true
		}

		payload, err := io.ReadAll(resp.Body)
		if err != nil || len(payload) == 0 {
			return false
		}
	}

	h.wg.Add(1)
	go func() {
		defer h.wg.Done()
		h.updateCache(string(payload))
	}()
	return true
}

func (h *HttpConnector) updateCache(payload string) {
	if h.options.PayloadCache != nil {
		if h.failSafeCache != nil {
			h.failSafeCache.UpdatePayloadIfNeeded(payload)
		}
		h.options.PayloadCache.PutWithTTL(PollingPayloadCacheKey, payload, h.payloadCachePollTtlSeconds)
	}
}

func (h *HttpConnector) updateFromCache() {
	var flagData string
	var err error
	if h.options.PayloadCache != nil {
		flagData, err = h.options.PayloadCache.Get(PollingPayloadCacheKey)
		if err != nil {
			return
		}
	}
	if flagData == "" && h.failSafeCache != nil {
		flagData = h.failSafeCache.Get()
	}
}

func (h *HttpConnector) Shutdown() {
	close(h.shutdownChan)
	h.wg.Wait()
}
