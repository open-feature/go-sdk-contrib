package flagdhttpconnector

// generate full unit tests
import (
	context "context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/open-feature/flagd/core/pkg/logger"
	flagdsync "github.com/open-feature/flagd/core/pkg/sync"
	flagd "github.com/open-feature/go-sdk-contrib/providers/flagd/pkg"
	of "github.com/open-feature/go-sdk/openfeature"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// MockPayloadCache PayloadCache implementation based on map
type MockPayloadCache struct {
	cache           sync.Map
	SuccessGetCount atomic.Int32
}

func NewMockPayloadCache() *MockPayloadCache {
	return &MockPayloadCache{}
}

// Get retrieves a payload from the cache by key.
func (m *MockPayloadCache) Get(key string) (string, error) {
	slog.Info("Getting payload from cache", "key", key)
	value, ok := m.cache.Load(key)
	if !ok {
		return "", errors.New("key not found")
	}
	payload, ok := value.(string)
	if !ok {
		return "", errors.New("invalid payload type")
	}
	m.SuccessGetCount.Add(1)
	return payload, nil
}

// Put adds or updates a payload in the cache by key.
func (m *MockPayloadCache) Put(key, payload string) error {
	slog.Info("Putting payload in cache", "key", key)
	m.cache.Store(key, payload)
	return nil
}

// PutWithTTL adds a payload to the cache with a time-to-live (TTL).
func (m *MockPayloadCache) PutWithTTL(key, payload string, ttlSeconds int) error {
	slog.Info("MockPayloadCache.PutWithTTL payload in cache", "key", key, "ttlSeconds", ttlSeconds)
	m.cache.Store(key, payload)

	// Start a goroutine to remove the key after the TTL expires.
	go func() {
		time.Sleep(time.Duration(ttlSeconds) * time.Second)
		slog.Info("Removing key from cache after TTL", "key", key)
		m.cache.Delete(key)
	}()
	return nil
}

type mockRoundTripper struct {
	mu        sync.Mutex
	callCount int
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callCount++
	body := io.NopCloser(strings.NewReader(`{"$schema":"https://flagd.dev/schema/v0/flags.json","flags":{"myBoolFlag":{"state":"ENABLED","variants":{"on":true,"off":false},"defaultVariant":"on"}}}`))
	return &http.Response{
		StatusCode: 200,
		Body:       body,
		Header:     make(http.Header),
	}, nil
}

type mock304RoundTripper struct {
	mu        sync.Mutex
	callCount int
}

func (m *mock304RoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callCount++
	if m.callCount == 1 {
		// Return a fake response only on the first call
		body := io.NopCloser(strings.NewReader(`{"status": "ok"}`))
		return &http.Response{
			StatusCode: 200,
			Body:       body,
			Header:     make(http.Header),
		}, nil
	}
	// Simulate no new data (cache hit)
	body := io.NopCloser(strings.NewReader(""))
	return &http.Response{
		StatusCode: 304, // Not Modified
		Body:       body,
		Header:     make(http.Header),
	}, nil
}

type mockFailureRoundTripper struct{}

func (m *mockFailureRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// Return a fake response
	body := io.NopCloser(strings.NewReader(`{"status": "error"}`))
	return &http.Response{
		StatusCode: 400,
		Body:       body,
		Header:     make(http.Header),
	}, nil
}

func TestNewHttpConnectorOptions(t *testing.T) {
	zapLogger, err := logger.NewZapLogger(zapcore.LevelOf(zap.DebugLevel), "json")
	logger := logger.NewLogger(zapLogger, false)
	opts := &HttpConnectorOptions{
		PollIntervalSeconds:   10,
		ConnectTimeoutSeconds: 5,
		RequestTimeoutSeconds: 15,
		Headers:               map[string]string{"User-Agent": "Flagd"},
		ProxyHost:             "proxy.example.com",
		ProxyPort:             8080,
		PayloadCacheOptions:   nil,
		PayloadCache:          nil,
		UseHttpCache:          true,
		UseFailsafeCache:      false,
		UsePollingCache:       false,
		URL:                   "http://example.com",
		Client: &http.Client{
			Transport: &mockRoundTripper{},
		},
		Log: logger,
	}
	expectedOpts := &HttpConnectorOptions{
		PollIntervalSeconds:   10,
		ConnectTimeoutSeconds: 5,
		RequestTimeoutSeconds: 15,
		Headers:               map[string]string{"User-Agent": "Flagd"},
		ProxyHost:             "proxy.example.com",
		ProxyPort:             8080,
		PayloadCache:          nil,
		UseHttpCache:          true,
		UseFailsafeCache:      false,
		UsePollingCache:       false,
		URL:                   "http://example.com",
		Client: &http.Client{
			Transport: &mockRoundTripper{},
		},
		Log: logger,
	}
	createdOpts, err := NewHttpConnectorOptions(*opts)
	require.NoError(t, err)
	assert.Equal(t, expectedOpts.PollIntervalSeconds, createdOpts.PollIntervalSeconds)
	assert.Equal(t, expectedOpts.ConnectTimeoutSeconds, createdOpts.ConnectTimeoutSeconds)
	assert.Equal(t, expectedOpts.RequestTimeoutSeconds, createdOpts.RequestTimeoutSeconds)
	assert.Equal(t, expectedOpts.Headers, createdOpts.Headers)
	assert.Equal(t, expectedOpts.ProxyHost, createdOpts.ProxyHost)
	assert.Equal(t, expectedOpts.ProxyPort, createdOpts.ProxyPort)
	assert.Equal(t, expectedOpts.PayloadCache, createdOpts.PayloadCache)
	assert.Equal(t, expectedOpts.UseHttpCache, createdOpts.UseHttpCache)
	assert.Equal(t, expectedOpts.UseFailsafeCache, createdOpts.UseFailsafeCache)
	assert.Equal(t, expectedOpts.UsePollingCache, createdOpts.UsePollingCache)
	assert.Equal(t, expectedOpts.URL, createdOpts.URL)
}

func TestValidateHttpConnectorOptions(t *testing.T) {
	zapLogger, err := logger.NewZapLogger(zapcore.LevelOf(zap.DebugLevel), "json")
	logger := logger.NewLogger(zapLogger, false)
	opts := &HttpConnectorOptions{
		PollIntervalSeconds:   10,
		ConnectTimeoutSeconds: 5,
		RequestTimeoutSeconds: 15,
		Headers:               map[string]string{"User-Agent": "Flagd"},
		ProxyHost:             "proxy.example.com",
		ProxyPort:             8080,
		PayloadCacheOptions:   nil,
		PayloadCache:          nil,
		UseHttpCache:          true,
		UseFailsafeCache:      false,
		UsePollingCache:       false,
		URL:                   "http://example.com",
		Client: &http.Client{
			Transport: &mockRoundTripper{},
		},
		Log: logger,
	}
	err = Validate(opts)
	require.NoError(t, err)
}

func TestValidateHttpConnectorOptions_InvalidURL(t *testing.T) {
	opts := &HttpConnectorOptions{
		PollIntervalSeconds:   10,
		ConnectTimeoutSeconds: 5,
		RequestTimeoutSeconds: 15,
		Headers:               map[string]string{"User-Agent": "Flagd"},
		ProxyHost:             "proxy.example.com",
		ProxyPort:             8080,
		PayloadCacheOptions:   nil,
		PayloadCache:          nil,
		UseHttpCache:          true,
		UseFailsafeCache:      false,
		UsePollingCache:       false,
		URL:                   "invalid-url",
		Client: &http.Client{
			Transport: &mockRoundTripper{},
		},
	}
	err := Validate(opts)
	assert.Error(t, err)
}

func TestValidateHttpConnectorOptions_InvalidFailsafeConfig(t *testing.T) {
	opts := &HttpConnectorOptions{
		PollIntervalSeconds:   10,
		ConnectTimeoutSeconds: 5,
		RequestTimeoutSeconds: 15,
		Headers:               map[string]string{"User-Agent": "Flagd"},
		ProxyHost:             "proxy.example.com",
		ProxyPort:             8080,
		PayloadCacheOptions:   nil,
		PayloadCache:          nil,
		UseHttpCache:          true,
		UseFailsafeCache:      true,
		UsePollingCache:       false,
		URL:                   "http://example.com",
		Client: &http.Client{
			Transport: &mockRoundTripper{},
		},
	}
	err := Validate(opts)

	if !strings.Contains(err.Error(), "payloadCache must be set if useFailsafeCache or usePollingCache is true") {
		t.Errorf("Expected error about payloadCache, got: %v", err)
	}

	assert.Error(t, err)
}

func TestValidateHttpConnectorOptions_InvalidRequestTimeout(t *testing.T) {
	opts := &HttpConnectorOptions{
		PollIntervalSeconds:   10,
		ConnectTimeoutSeconds: 5,
		RequestTimeoutSeconds: 0, // Invalid
		Headers:               map[string]string{"User-Agent": "Flagd"},
		ProxyHost:             "proxy.example.com",
		ProxyPort:             8080,
		PayloadCacheOptions:   nil,
		PayloadCache:          nil,
		UseHttpCache:          true,
		UseFailsafeCache:      false,
		UsePollingCache:       false,
		URL:                   "http://example.com",
		Client: &http.Client{
			Transport: &mockRoundTripper{},
		},
	}
	_, err := NewHttpConnector(*opts)
	assert.Error(t, err)
}

func TestValidateHttpConnectorOptions_InvalidConnectTimeout(t *testing.T) {
	opts := &HttpConnectorOptions{
		PollIntervalSeconds:   10,
		ConnectTimeoutSeconds: 0, // Invalid
		RequestTimeoutSeconds: 15,
		Headers:               map[string]string{"User-Agent": "Flagd"},
		ProxyHost:             "proxy.example.com",
		ProxyPort:             8080,
		PayloadCacheOptions:   nil,
		PayloadCache:          nil,
		UseHttpCache:          true,
		UseFailsafeCache:      false,
		UsePollingCache:       false,
		URL:                   "http://example.com",
		Client: &http.Client{
			Transport: &mockRoundTripper{},
		},
	}
	err := Validate(opts)
	assert.Error(t, err)
}

func TestValidateHttpConnectorOptions_InvalidPollInterval(t *testing.T) {
	opts := &HttpConnectorOptions{
		PollIntervalSeconds:   0, // Invalid
		ConnectTimeoutSeconds: 5,
		RequestTimeoutSeconds: 15,
		Headers:               map[string]string{"User-Agent": "Flagd"},
		ProxyHost:             "proxy.example.com",
		ProxyPort:             8080,
		PayloadCacheOptions:   nil,
		PayloadCache:          nil,
		UseHttpCache:          true,
		UseFailsafeCache:      false,
		UsePollingCache:       false,
		URL:                   "http://example.com",
		Client: &http.Client{
			Transport: &mockRoundTripper{},
		},
	}
	err := Validate(opts)
	assert.Error(t, err)
}

func TestValidateHttpConnectorOptions_InvalidProxyConfig(t *testing.T) {
	opts := &HttpConnectorOptions{
		PollIntervalSeconds:   10,
		ConnectTimeoutSeconds: 5,
		RequestTimeoutSeconds: 15,
		Headers:               map[string]string{"User-Agent": "Flagd"},
		ProxyHost:             "proxy.example.com",
		ProxyPort:             0,
		PayloadCacheOptions:   nil,
		PayloadCache:          nil,
		UseHttpCache:          true,
		UseFailsafeCache:      false,
		UsePollingCache:       false,
		URL:                   "http://example.com",
		Client: &http.Client{
			Transport: &mockRoundTripper{},
		},
	}
	err := Validate(opts)
	assert.Error(t, err)
	if !strings.Contains(err.Error(), "proxyPort must be set if proxyHost is set") {
		t.Errorf("Expected error about proxyPort, got: %v", err)
	}
}

func TestValidateHttpConnectorOptions_MissingLogger(t *testing.T) {
	opts := &HttpConnectorOptions{
		PollIntervalSeconds:   10,
		ConnectTimeoutSeconds: 5,
		RequestTimeoutSeconds: 15,
		Headers:               map[string]string{"User-Agent": "Flagd"},
		PayloadCacheOptions:   nil,
		PayloadCache:          nil,
		UseHttpCache:          true,
		UseFailsafeCache:      false,
		UsePollingCache:       false,
		URL:                   "http://example.com",
		Client: &http.Client{
			Transport: &mockRoundTripper{},
		},
	}
	err := Validate(opts)
	assert.Error(t, err)
	if !strings.Contains(err.Error(), "log is required for HttpConnector") {
		t.Errorf("Expected error about missing logger, got: %v", err)
	}
}

func TestValidateHttpConnectorOptions_ValidProxyConfig(t *testing.T) {
	zapLogger, err := logger.NewZapLogger(zapcore.LevelOf(zap.DebugLevel), "json")
	logger := logger.NewLogger(zapLogger, false)
	opts := &HttpConnectorOptions{
		PollIntervalSeconds:   10,
		ConnectTimeoutSeconds: 5,
		RequestTimeoutSeconds: 15,
		Headers:               map[string]string{"User-Agent": "Flagd"},
		ProxyHost:             "proxy.example.com",
		ProxyPort:             8080, // Valid
		PayloadCacheOptions:   nil,
		PayloadCache:          nil,
		UseHttpCache:          true,
		UseFailsafeCache:      false,
		UsePollingCache:       false,
		URL:                   "http://example.com",
		Log:                   logger,
	}
	err = Validate(opts)
	assert.NoError(t, err)

	_, err = NewHttpConnector(*opts)
	require.NoError(t, err)
}

func TestValidateHttpConnectorOptions_ValidPayloadCacheConfig(t *testing.T) {
	zapLogger, err := logger.NewZapLogger(zapcore.LevelOf(zap.DebugLevel), "json")
	logger := logger.NewLogger(zapLogger, false)
	opts := &HttpConnectorOptions{
		PollIntervalSeconds:   10,
		ConnectTimeoutSeconds: 5,
		RequestTimeoutSeconds: 15,
		Headers:               map[string]string{"User-Agent": "Flagd"},
		ProxyHost:             "proxy.example.com",
		ProxyPort:             8080,
		PayloadCacheOptions:   &PayloadCacheOptions{UpdateIntervalSeconds: 60}, // Valid
		PayloadCache:          &MockPayloadCache{},                             // Assuming MockPayloadCache implements PayloadCache
		UseHttpCache:          true,
		UseFailsafeCache:      false,
		UsePollingCache:       false,
		URL:                   "http://example.com",
		Client: &http.Client{
			Transport: &mockRoundTripper{},
		},
		Log: logger,
	}
	err = Validate(opts)
	assert.NoError(t, err)
}
func TestValidateHttpConnectorOptions_ValidPayloadCacheWithPolling(t *testing.T) {
	zapLogger, err := logger.NewZapLogger(zapcore.LevelOf(zap.DebugLevel), "json")
	logger := logger.NewLogger(zapLogger, false)
	opts := &HttpConnectorOptions{
		PollIntervalSeconds:   10,
		ConnectTimeoutSeconds: 5,
		RequestTimeoutSeconds: 15,
		Headers:               map[string]string{"User-Agent": "Flagd"},
		ProxyHost:             "proxy.example.com",
		ProxyPort:             8080,
		PayloadCacheOptions:   &PayloadCacheOptions{UpdateIntervalSeconds: 60},
		PayloadCache:          &MockPayloadCache{},
		UseHttpCache:          true,
		UseFailsafeCache:      false,
		UsePollingCache:       true, // Valid with PutWithTTL support
		URL:                   "http://example.com",
		Client: &http.Client{
			Transport: &mockRoundTripper{},
		},
		Log: logger,
	}
	err = Validate(opts)
	assert.NoError(t, err)
}

func TestWithFlagdProvider(t *testing.T) {
	zapLogger, err := logger.NewZapLogger(zapcore.LevelOf(zap.DebugLevel), "json")
	logger := logger.NewLogger(zapLogger, false)
	opts := &HttpConnectorOptions{
		PollIntervalSeconds:   10,
		ConnectTimeoutSeconds: 5,
		RequestTimeoutSeconds: 15,
		URL:                   "http://example.com",
		Client: &http.Client{
			Transport: &mockRoundTripper{},
		},
		Log: logger,
	}

	connector, err := NewHttpConnector(*opts)
	require.NoError(t, err)
	assert.NotNil(t, connector)

	provider, err := flagd.NewProvider(
		flagd.WithInProcessResolver(),
		flagd.WithCustomSyncProvider(connector),
	)
	require.NoError(t, err)
	assert.NotNil(t, provider)
	defer provider.Shutdown()

	err = provider.Init(of.EvaluationContext{})
	if err != nil {
		t.Fatal("error initialization provider", err)
	}

	if provider.Status() != of.ReadyState {
		t.Errorf("expected status to be ready, but got %v", provider.Status())
	}

	assert.True(t, connector.IsReady(), "Connector should be ready after initialization")

	evalResult := provider.BooleanEvaluation(context.Background(), "myBoolFlag", false, of.FlattenedContext{})
	assert.True(t, evalResult.Value, "Expected myBoolFlag to be true after initialization")
}

func TestShutdownHttpConnector(t *testing.T) {
	zapLogger, err := logger.NewZapLogger(zapcore.LevelOf(zap.DebugLevel), "json")
	logger := logger.NewLogger(zapLogger, false)
	opts := &HttpConnectorOptions{
		Log:                   logger,
		PollIntervalSeconds:   10,
		ConnectTimeoutSeconds: 5,
		RequestTimeoutSeconds: 15,
		URL:                   "http://example.com",
		Client: &http.Client{
			Transport: &mockRoundTripper{},
		},
	}

	connector, err := NewHttpConnector(*opts)
	require.NoError(t, err)
	assert.NotNil(t, connector)

	connector.Init(context.Background())
	syncChan := make(chan flagdsync.DataSync, 1)

	go func() {
		select {
		case _ = <-syncChan:
			return
		}
	}()

	connector.Sync(context.Background(), syncChan)

	connector.Shutdown()
	assert.NotPanics(t, func() { connector.Shutdown() }) // Ensure shutdown is idempotent
}

func TestShutdownWithoutSyncHttpConnector(t *testing.T) {
	zapLogger, err := logger.NewZapLogger(zapcore.LevelOf(zap.DebugLevel), "json")
	logger := logger.NewLogger(zapLogger, false)
	opts := &HttpConnectorOptions{
		Log:                   logger,
		PollIntervalSeconds:   10,
		ConnectTimeoutSeconds: 5,
		RequestTimeoutSeconds: 15,
		URL:                   "http://example.com",
		Client: &http.Client{
			Transport: &mockRoundTripper{},
		},
	}

	connector, err := NewHttpConnector(*opts)
	require.NoError(t, err)
	assert.NotNil(t, connector)

	connector.Shutdown()
	assert.NotPanics(t, func() { connector.Shutdown() }) // Ensure shutdown is idempotent
}

func TestSyncNotInitialized(t *testing.T) {
	zapLogger, err := logger.NewZapLogger(zapcore.LevelOf(zap.DebugLevel), "json")
	logger := logger.NewLogger(zapLogger, false)
	opts := &HttpConnectorOptions{
		Log:                   logger,
		PollIntervalSeconds:   10,
		ConnectTimeoutSeconds: 5,
		RequestTimeoutSeconds: 15,
		URL:                   "http://example.com",
	}

	connector := &HttpConnector{
		options: *opts,
	}

	err = connector.Sync(context.Background(), nil)
	require.Error(t, err)
	assert.Equal(t, "not initialized", err.Error())
}

func TestSyncHttpConnector(t *testing.T) {
	zapLogger, err := logger.NewZapLogger(zapcore.LevelOf(zap.DebugLevel), "json")
	logger := logger.NewLogger(zapLogger, false)
	opts := &HttpConnectorOptions{
		Log: logger,
		URL: "http://example.com",
		Client: &http.Client{
			Transport: &mockRoundTripper{},
		},
	}

	connector, err := NewHttpConnector(*opts)
	require.NoError(t, err)
	assert.NotNil(t, connector)

	connector.Init(context.Background())
	syncChan := make(chan flagdsync.DataSync, 1)
	connector.Sync(context.Background(), syncChan)

	err = connector.ReSync(context.Background(), syncChan)
	require.NoError(t, err)

	assert.NotPanics(t, func() { connector.Sync(context.Background(), syncChan) }) // Ensure Sync is idempotent
}

// integration tests with mock http client
func TestGithubRawContent(t *testing.T) {

	testURL := "https://raw.githubusercontent.com/open-feature/java-sdk-contrib/main/tools/flagd-http-connector/src/test/resources/testing-flags.json"

	zapLogger, err := logger.NewZapLogger(zapcore.LevelOf(zap.DebugLevel), "json")
	logger := logger.NewLogger(zapLogger, false)
	opts := HttpConnectorOptions{
		URL:                   testURL,
		PollIntervalSeconds:   10,
		ConnectTimeoutSeconds: 5,
		RequestTimeoutSeconds: 15,
		Log:                   logger,
		Client: &http.Client{
			Transport: &mockRoundTripper{},
		},
		Headers: map[string]string{
			"User-Agent": "Flagd-Http-Connector-Test",
		},
	}

	connector, err := NewHttpConnector(opts)
	require.NoError(t, err)
	defer connector.Shutdown()

	connector.Init(context.Background())
	syncChan := make(chan flagdsync.DataSync, 1)

	// Check if the sync channel received any data

	success := &atomic.Bool{}
	go func() {
		select {
		case data := <-syncChan:
			assert.NotEmpty(t, data.FlagData, "Flag data should not be empty")
			assert.Equal(t, testURL, data.Source, "Source should match the test URL")
			assert.Equal(t, flagdsync.ALL, data.Type, "Type should be ALL for initial sync")
			// set success = true via atomic operation
			success.Store(true)
		}
	}()

	connector.Sync(context.Background(), syncChan)

	assert.Eventually(t, func() bool {
		return success.Load()
	}, 15*time.Second, 1*time.Second, "Sync channel should receive data within 15 seconds")
}

func TestGithubRawContentUsingCache(t *testing.T) {
	testURL := "https://raw.githubusercontent.com/open-feature/java-sdk-contrib/main/tools/flagd-http-connector/src/test/resources/testing-flags.json"

	zapLogger, err := logger.NewZapLogger(zapcore.LevelOf(zap.DebugLevel), "json")
	logger := logger.NewLogger(zapLogger, false)
	opts := HttpConnectorOptions{
		URL:                   testURL,
		PollIntervalSeconds:   5,
		ConnectTimeoutSeconds: 5,
		RequestTimeoutSeconds: 15,
		Log:                   logger,
		PayloadCache:          NewMockPayloadCache(),
		UsePollingCache:       true,
		PayloadCacheOptions: &PayloadCacheOptions{
			UpdateIntervalSeconds: 5,
		},
		Client: &http.Client{
			Transport: &mockRoundTripper{},
		},
	}

	connector, err := NewHttpConnector(opts)
	require.NoError(t, err)
	defer connector.Shutdown()

	connector.Init(context.Background())

	// second connector to simulate a different micro-service instance using same cache (e.g. Redis)

	connector2, err := NewHttpConnector(opts)
	require.NoError(t, err)
	defer connector2.Shutdown()

	connector2.Init(context.Background())

	syncChan := make(chan flagdsync.DataSync, 1)
	defer close(syncChan)

	// Check if the sync channel received any data
	success := &atomic.Bool{}

	go func() {
		for {
			select {
			case data := <-syncChan:
				slog.Info("Received data from sync channel",
					"source", data.Source,
					"testURL", testURL,
					"type", data.Type,
				)
				if data.FlagData == "" {
					slog.Info("Received empty flag data from sync channel")
					return
				}
				assert.NotEmpty(t, data.FlagData, "Flag data should not be empty")
				assert.Equal(t, testURL, data.Source, "Source should match the test URL")
				assert.Equal(t, flagdsync.ALL, data.Type, "Type should be ALL for initial sync")
				success.Store(true)
			}
		}
	}()

	slog.Info("Starting sync for first connector")
	connector.Sync(context.Background(), syncChan)

	// simulate start a bit later
	time.Sleep(200 * time.Millisecond)

	slog.Info("Starting sync for second connector")
	connector2.Sync(context.Background(), syncChan)
	slog.Info("Sync started for both connectors")

	assert.Eventually(t, func() bool {
		return opts.PayloadCache.(*MockPayloadCache).SuccessGetCount.Load() >= 2 && success.Load()
	}, 15*time.Second, 1*time.Second, "Sync channel should receive data within 15 seconds and cache should be hit once, "+
		"successGetCount: "+strconv.Itoa(int(opts.PayloadCache.(*MockPayloadCache).SuccessGetCount.Load()))+" success: "+strconv.FormatBool(success.Load()))

}

// MockPayloadCache PayloadCache implementation based on map
type MockFailSafeCache struct {
	cache           sync.Map
	SuccessGetCount atomic.Int32
	FailureGetCount atomic.Int32
}

func NewMockFailSafeCache() *MockFailSafeCache {
	return &MockFailSafeCache{}
}

// Get retrieves a payload from the cache by key.
func (m *MockFailSafeCache) Get(key string) (string, error) {
	value, ok := m.cache.Load(key)
	if !ok {
		m.FailureGetCount.Add(1)
		return "", errors.New("key not found")
	}
	payload, ok := value.(string)
	if !ok {
		return "", errors.New("invalid payload type")
	}
	m.SuccessGetCount.Add(1)
	return payload, nil
}

// Put adds or updates a payload in the cache by key.
func (m *MockFailSafeCache) Put(key, payload string) error {
	slog.Info("MockFailSafeCache.Put payload in cache", "key", key)
	m.PutWithTTL(key, payload, 1)
	return nil
}

// PutWithTTL adds a payload to the cache with a time-to-live (TTL).
func (m *MockFailSafeCache) PutWithTTL(key, payload string, ttlSeconds int) error {
	slog.Info("MockFailSafeCache.PutWithTTL payload in cache", "key", key)
	m.cache.Store(key, payload)

	// Start a goroutine to remove the key after the TTL expires.
	go func() {

		// the cache can be used as a distributed cache for other micro-service instances,
		sleepTime := time.Duration(ttlSeconds) * time.Second // + 1 // Add a small buffer to ensure the key is removed after the TTL

		time.Sleep(sleepTime)
		// log using slog
		slog.Info("Removing key from cache after TTL",
			"key", key)
		m.cache.Delete(key)
	}()
	return nil
}

func TestGithubRawContentUsingFailsafeCache(t *testing.T) {

	// non-existing url, simulating Github down
	invalidTestUrl := "https://raw.githubusercontent.com/open-feature/java-sdk-contrib/main/tools/flagd-http-connector/src/test/resources/non-existing-flags.json"

	zapLogger, err := logger.NewZapLogger(zapcore.LevelOf(zap.DebugLevel), "json")
	logger := logger.NewLogger(zapLogger, false)
	opts := HttpConnectorOptions{
		URL:                   invalidTestUrl,
		PollIntervalSeconds:   5,
		ConnectTimeoutSeconds: 5,
		RequestTimeoutSeconds: 15,
		Log:                   logger,
		PayloadCache:          NewMockFailSafeCache(),
		UsePollingCache:       true,
		UseFailsafeCache:      true,
		PayloadCacheOptions: &PayloadCacheOptions{
			UpdateIntervalSeconds: 5,
		},
		Client: &http.Client{
			Transport: &mockFailureRoundTripper{},
		},
	}

	connector, err := NewHttpConnector(opts)
	require.NoError(t, err)
	defer connector.Shutdown()

	connector.Init(context.Background())

	// simulate cache hit by pre-populating the fail-safe cache with a payload from previous micro-service run
	testPayload := "test-payload"

	connector.failSafeCache.payloadCache.Put(FailSafePayloadCacheKey, testPayload)

	syncChan := make(chan flagdsync.DataSync, 1)

	// Check if the sync channel received any data
	success := &atomic.Bool{}

	go func() {
		select {
		case data := <-syncChan:
			assert.NotEmpty(t, data.FlagData, "Flag data should not be empty")
			assert.Equal(t, invalidTestUrl, data.Source, "Source should match the test URL")
			assert.Equal(t, flagdsync.ALL, data.Type, "Type should be ALL for initial sync")
			assert.Equal(t, testPayload, data.FlagData, "Flag data should match the cached payload")
			success.Store(true)
		}
	}()

	connector.Sync(context.Background(), syncChan)

	assert.Eventually(t, func() bool {
		slog.Debug("Checking if sync channel received data",
			slog.Int("SuccessGetCount", int(connector.failSafeCache.payloadCache.(*MockFailSafeCache).SuccessGetCount.Load())),
			slog.Int("FailureGetCount", int(connector.failSafeCache.payloadCache.(*MockFailSafeCache).FailureGetCount.Load())),
		)
		return connector.failSafeCache.payloadCache.(*MockFailSafeCache).SuccessGetCount.Load() == 1 && // Ensure that the cache was hit once for the initial fetch
			opts.PayloadCache.(*MockFailSafeCache).FailureGetCount.Load() == 2 && // Ensure that the cache was hit once for the failure for payload cache
			success.Load()
	}, 3*time.Second, 1*time.Second, "Sync channel should receive data within 15 seconds "+
		"and cache should be hit once")
}

func TestGithubRawContentUsingHttpCache(t *testing.T) {
	testURL := "https://raw.githubusercontent.com/open-feature/java-sdk-contrib/main/tools/flagd-http-connector/src/test/resources/testing-flags.json"

	zapLogger, err := logger.NewZapLogger(zapcore.LevelOf(zap.DebugLevel), "json")
	logger := logger.NewLogger(zapLogger, false)
	opts := HttpConnectorOptions{
		URL:                   testURL,
		PollIntervalSeconds:   5,
		ConnectTimeoutSeconds: 5,
		RequestTimeoutSeconds: 15,
		Log:                   logger,
		UseHttpCache:          true,
		Client: &http.Client{
			Transport: &mock304RoundTripper{},
		},
	}

	connector, err := NewHttpConnector(opts)
	require.NoError(t, err)
	defer connector.Shutdown()

	connector.Init(context.Background())

	syncChan := make(chan flagdsync.DataSync, 1)

	dataCount := atomic.Int32{}

	go func() {
		select {
		case data := <-syncChan:
			assert.NotEmpty(t, data.FlagData, "Flag data should not be empty")
			assert.Equal(t, testURL, data.Source, "Source should match the test URL")
			assert.Equal(t, flagdsync.ALL, data.Type, "Type should be ALL for initial sync")
			dataCount.Add(1)
		}
	}()

	connector.Sync(context.Background(), syncChan)

	time.Sleep(16 * time.Second)

	assert.Equal(t, 1, int(dataCount.Load()), "Sync channel should receive data exactly once")
}
