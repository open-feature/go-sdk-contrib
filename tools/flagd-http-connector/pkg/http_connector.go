package flagdhttpconnector

import (
	context "context"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	flagdsync "github.com/open-feature/flagd/core/pkg/sync"
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
	queue                      chan QueuePayload
	scheduler                  *time.Ticker
	cacheFetcher               *HttpCacheFetcher
	failSafeCache              *FailSafeCache
	shutdownChan               chan struct{}
	wg                         sync.WaitGroup
	payloadCachePollTtlSeconds int
}

func (h HttpConnector) Init(ctx context.Context) error {
	return nil
}

func (h HttpConnector) IsReady() bool {
	return true
}

func (h HttpConnector) Sync(ctx context.Context, dataSync chan<- flagdsync.DataSync) error {
	success := h.fetchAndUpdate()
	if !success {
		h.updateFromCache()
	}

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

func (fps HttpConnector) ReSync(ctx context.Context, dataSync chan<- flagdsync.DataSync) error {
	return nil
}

var _ flagdsync.ISync = &HttpConnector{}

// QueuePayloadType represents the type of payload in the queue.
type QueuePayloadType string

const (
	PayloadTypeData QueuePayloadType = "DATA"
)

type QueuePayload struct {
	Type    QueuePayloadType
	Payload string
}

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
		options: opts,
		client:  client,
		// queue:        make(chan QueuePayload, opts.QueueSize),
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

func (h *HttpConnector) GetStreamQueue() <-chan QueuePayload {
	success := h.fetchAndUpdate()
	if !success {
		log.Println("Initial fetch failed, attempting from cache")
		h.updateFromCache()
	}

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

	return h.queue
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
		log.Println("Failed to build request:", err)
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
			log.Printf("Received non-OK status: %d, body: %s", resp.StatusCode, string(body))
			return false
		}

		if resp.StatusCode == http.StatusNotModified {
			log.Println("Received 304 Not Modified")
			return true
		}

		payload, err := io.ReadAll(resp.Body)
		if err != nil || len(payload) == 0 {
			log.Println("Empty or unreadable payload")
			return false
		}
	}

	// select {
	// case h.queue <- QueuePayload{Type: PayloadTypeData, Payload: string(payload)}:
	// default:
	// 	log.Println("Queue full, dropping payload")
	// }

	// TODO instead of routine, use more robust
	go h.updateCache(string(payload))
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
	// if flagData != "" {
	// 	select {
	// 	case h.queue <- QueuePayload{Type: PayloadTypeData, Payload: flagData}:
	// 	default:
	// 		log.Println("Queue full, dropping cached payload")
	// 	}
	// }
}

func (h *HttpConnector) Shutdown() {
	close(h.shutdownChan)
	h.wg.Wait()
}
