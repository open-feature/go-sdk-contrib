package flagdhttpconnector

// generate full unit tests
import (
	context "context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/open-feature/flagd/core/pkg/logger"
	flagdsync "github.com/open-feature/flagd/core/pkg/sync"
	flagd "github.com/open-feature/go-sdk-contrib/providers/flagd/pkg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// MockPayloadCache PayloadCache implementation based on map
type MockPayloadCache struct {
	cache           sync.Map
	SuccessGetCount int
}

func NewMockPayloadCache() *MockPayloadCache {
	return &MockPayloadCache{}
}

// Get retrieves a payload from the cache by key.
func (m *MockPayloadCache) Get(key string) (string, error) {
	value, ok := m.cache.Load(key)
	if !ok {
		return "", errors.New("key not found")
	}
	payload, ok := value.(string)
	if !ok {
		return "", errors.New("invalid payload type")
	}
	m.SuccessGetCount++
	return payload, nil
}

// Put adds or updates a payload in the cache by key.
func (m *MockPayloadCache) Put(key, payload string) error {
	m.cache.Store(key, payload)
	return nil
}

// PutWithTTL adds a payload to the cache with a time-to-live (TTL).
func (m *MockPayloadCache) PutWithTTL(key, payload string, ttlSeconds int) error {
	m.cache.Store(key, payload)

	// Start a goroutine to remove the key after the TTL expires.
	go func() {
		time.Sleep(time.Duration(ttlSeconds) * time.Second)
		m.cache.Delete(key)
	}()
	return nil
}

func TestNewHttpConnectorOptions(t *testing.T) {
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
	}
	err := Validate(opts)
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
	}
	err := Validate(opts)
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
	}
	err := Validate(opts)
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
		ProxyPort:             0, // Invalid
		PayloadCacheOptions:   nil,
		PayloadCache:          nil,
		UseHttpCache:          true,
		UseFailsafeCache:      false,
		UsePollingCache:       false,
		URL:                   "http://example.com",
	}
	err := Validate(opts)
	assert.Error(t, err)
}
func TestValidateHttpConnectorOptions_ValidProxyConfig(t *testing.T) {
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
	}
	err := Validate(opts)
	assert.NoError(t, err)
}

func TestValidateHttpConnectorOptions_ValidPayloadCacheConfig(t *testing.T) {
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
	}
	err := Validate(opts)
	assert.NoError(t, err)
}
func TestValidateHttpConnectorOptions_ValidPayloadCacheWithPolling(t *testing.T) {
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
	}
	err := Validate(opts)
	assert.NoError(t, err)
}

// test using flagd provider
func TestWithFlagdProvider(t *testing.T) {
	zapLogger, err := logger.NewZapLogger(zapcore.LevelOf(zap.DebugLevel), "json")
	logger := logger.NewLogger(zapLogger, false)
	opts := &HttpConnectorOptions{
		PollIntervalSeconds:   10,
		ConnectTimeoutSeconds: 5,
		RequestTimeoutSeconds: 15,
		URL:                   "http://example.com",
		log:                   logger,
	}

	connector, err := NewHttpConnector(*opts)
	require.NoError(t, err)
	assert.NotNil(t, connector)

	provider := flagd.NewProvider(
		flagd.WithInProcessResolver(),
		flagd.WithCustomSyncProvider(connector),
	)
	require.NoError(t, err)
	assert.NotNil(t, provider)
}

func TestShutdownHttpConnector(t *testing.T) {
	zapLogger, err := logger.NewZapLogger(zapcore.LevelOf(zap.DebugLevel), "json")
	logger := logger.NewLogger(zapLogger, false)
	opts := &HttpConnectorOptions{
		log:                   logger,
		PollIntervalSeconds:   10,
		ConnectTimeoutSeconds: 5,
		RequestTimeoutSeconds: 15,
		URL:                   "http://example.com",
	}

	connector, err := NewHttpConnector(*opts)
	require.NoError(t, err)
	assert.NotNil(t, connector)

	connector.Init(context.Background())
	syncChan := make(chan flagdsync.DataSync, 1)
	connector.Sync(context.Background(), syncChan)

	connector.Shutdown()
	assert.NotPanics(t, func() { connector.Shutdown() }) // Ensure shutdown is idempotent
}

func TestShutdownWithoutSyncHttpConnector(t *testing.T) {
	zapLogger, err := logger.NewZapLogger(zapcore.LevelOf(zap.DebugLevel), "json")
	logger := logger.NewLogger(zapLogger, false)
	opts := &HttpConnectorOptions{
		log:                   logger,
		PollIntervalSeconds:   10,
		ConnectTimeoutSeconds: 5,
		RequestTimeoutSeconds: 15,
		URL:                   "http://example.com",
	}

	connector, err := NewHttpConnector(*opts)
	require.NoError(t, err)
	assert.NotNil(t, connector)

	connector.Shutdown()
	assert.NotPanics(t, func() { connector.Shutdown() }) // Ensure shutdown is idempotent
}
func TestHttpConnector_Init_IsReady(t *testing.T) {
	zapLogger, _ := logger.NewZapLogger(zapcore.LevelOf(zap.DebugLevel), "json")
	l := logger.NewLogger(zapLogger, false)
	opts := HttpConnectorOptions{
		log:                   l,
		PollIntervalSeconds:   1,
		ConnectTimeoutSeconds: 1,
		RequestTimeoutSeconds: 1,
		URL:                   "http://example.com",
	}
	conn, err := NewHttpConnector(opts)
	require.NoError(t, err)
	assert.NoError(t, conn.Init(context.Background()))
	assert.True(t, conn.IsReady())
}

// type DummyFailSafeCache struct {
// 	UpdateCalled bool
// 	Payload      string
// }

// // Ensure DummyFailSafeCache implements the FailSafeCache interface
// var _ FailSafeCache = (*DummyFailSafeCache)(nil)

// func (d *DummyFailSafeCache) UpdatePayloadIfNeeded(payload string) {
// 	d.UpdateCalled = true
// 	d.Payload = payload
// }
// func (d *DummyFailSafeCache) Get() string {
// 	return d.Payload
// }

// DummyPayloadCache is a simple in-memory implementation of PayloadCache.
// type DummyPayloadCache struct {
// 	store map[string]string
// 	mu    sync.RWMutex
// }

// // NewDummyPayloadCache creates a new DummyPayloadCache.
// func NewDummyPayloadCache() *DummyPayloadCache {
// 	return &DummyPayloadCache{
// 		store: make(map[string]string),
// 	}
// }

// // Get retrieves a value by key.
// func (d *DummyPayloadCache) Get(key string) (string, error) {
// 	d.mu.RLock()
// 	defer d.mu.RUnlock()

// 	val, ok := d.store[key]
// 	if !ok {
// 		return "", errors.New("key not found")
// 	}
// 	return val, nil
// }

// // Put sets a value by key.
// func (d *DummyPayloadCache) Put(key string, value string) error {
// 	d.mu.Lock()
// 	defer d.mu.Unlock()

// 	d.store[key] = value
// 	return nil
// }

func TestHttpConnector_updateCache(t *testing.T) {
	zapLogger, _ := logger.NewZapLogger(zapcore.LevelOf(zap.DebugLevel), "json")
	l := logger.NewLogger(zapLogger, false)
	mockCache := NewMockPayloadCache()
	// failSafe := &DummyFailSafeCache{}
	// cache := NewDummyPayloadCache()
	// opts := &PayloadCacheOptions{UpdateIntervalSeconds: 10}
	cache := &MockPayloadCache{}
	opts := &PayloadCacheOptions{UpdateIntervalSeconds: 10}
	failSafeCache, _ := NewFailSafeCache(cache, opts)
	conn := &HttpConnector{
		options: HttpConnectorOptions{
			log:          l,
			PayloadCache: mockCache,
		},
		failSafeCache:              failSafeCache,
		payloadCachePollTtlSeconds: 1,
	}
	conn.updateCache("payload1")
	time.Sleep(10 * time.Millisecond)
	val, err := mockCache.Get(PollingPayloadCacheKey)
	assert.NoError(t, err)
	assert.Equal(t, "payload1", val)
	// assert.True(t, failSafeCache.UpdateCalled)
	assert.Equal(t, "payload1", failSafeCache.Get())
}

func TestHttpConnector_updateFromCache_PayloadCache(t *testing.T) {
	zapLogger, _ := logger.NewZapLogger(zapcore.LevelOf(zap.DebugLevel), "json")
	l := logger.NewLogger(zapLogger, false)
	mockCache := NewMockPayloadCache()
	mockCache.Put(PollingPayloadCacheKey, "payload2")
	conn := &HttpConnector{
		options: HttpConnectorOptions{
			log:          l,
			PayloadCache: mockCache,
			URL:          "http://example.com",
		},
	}
	ch := make(chan flagdsync.DataSync, 1)
	conn.updateFromCache(ch)
	select {
	case ds := <-ch:
		assert.Equal(t, "payload2", ds.FlagData)
		assert.Equal(t, "http://example.com", ds.Source)
	default:
		t.Fatal("expected data sync")
	}
}

func TestHttpConnector_updateFromCache_FailSafeCache(t *testing.T) {
	zapLogger, _ := logger.NewZapLogger(zapcore.LevelOf(zap.DebugLevel), "json")
	l := logger.NewLogger(zapLogger, false)
	cache := &MockPayloadCache{}
	// opts := &PayloadCacheOptions{UpdateIntervalSeconds: 10}
	// failSafeCache, _ := NewFailSafeCache(cache, opts)
	conn, err := NewHttpConnector(HttpConnectorOptions{
		log:              l,
		URL:              "http://example.com",
		PayloadCache:     cache,
		UseFailsafeCache: true,
	})
	require.NoError(t, err)
	ch := make(chan flagdsync.DataSync, 1)
	err = conn.Sync(context.Background(), ch)
	require.NoError(t, err)
	conn.updateFromCache(ch)
	select {
	case ds := <-ch:
		assert.Equal(t, "failsafe", ds.FlagData)
		assert.Equal(t, "http://example.com", ds.Source)
	default:
		t.Fatal("expected data sync")
	}
}

func TestHttpConnector_updateFromCache_NoCache(t *testing.T) {
	zapLogger, _ := logger.NewZapLogger(zapcore.LevelOf(zap.DebugLevel), "json")
	l := logger.NewLogger(zapLogger, false)
	conn, err := NewHttpConnector(HttpConnectorOptions{
		log: l,
		URL: "http://example.com",
	})
	require.NoError(t, err)
	ch := make(chan flagdsync.DataSync, 1)

	err = conn.Sync(context.Background(), ch)
	require.NoError(t, err)
	conn.updateFromCache(ch)
	select {
	case <-ch:
		t.Fatal("should not send data sync when no cache")
	default:
	}
}

func TestHttpConnector_Shutdown_Idempotent(t *testing.T) {
	zapLogger, _ := logger.NewZapLogger(zapcore.LevelOf(zap.DebugLevel), "json")
	l := logger.NewLogger(zapLogger, false)
	conn, err := NewHttpConnector(HttpConnectorOptions{
		log: l,
		URL: "http://example.com",
	})
	require.NoError(t, err)
	ch := make(chan flagdsync.DataSync, 1)
	err = conn.Sync(context.Background(), ch)
	require.NoError(t, err)
	conn.Shutdown()
	assert.NotPanics(t, func() { conn.Shutdown() })
}

func TestWaitWithTimeout_Completes(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)
	done := make(chan struct{})
	go func() {
		defer wg.Done()
		time.Sleep(10 * time.Millisecond)
		close(done)
	}()
	ok := waitWithTimeout(&wg, 1*time.Second)
	assert.True(t, ok)
}

func TestWaitWithTimeout_TimesOut(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)
	ok := waitWithTimeout(&wg, 10*time.Millisecond)
	assert.False(t, ok)
}

// TODO add more for test coverage
