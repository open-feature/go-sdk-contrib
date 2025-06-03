package flagdhttpconnector

// generate full unit tests
import (
	context "context"
	"errors"
	"io/ioutil"
	"net/http"
	"strings"
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

type mockRoundTripper struct{}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// Return a fake response
	body := ioutil.NopCloser(strings.NewReader(`{"status": "ok"}`))
	return &http.Response{
		StatusCode: 200,
		Body:       body,
		Header:     make(http.Header),
	}, nil
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
		Client: &http.Client{
			Transport: &mockRoundTripper{},
		},
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
		Client: &http.Client{
			Transport: &mockRoundTripper{},
		},
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
		Client: &http.Client{
			Transport: &mockRoundTripper{},
		},
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
		Client: &http.Client{
			Transport: &mockRoundTripper{},
		},
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
		ProxyPort:             0, // Invalid
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
		Client: &http.Client{
			Transport: &mockRoundTripper{},
		},
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
		Client: &http.Client{
			Transport: &mockRoundTripper{},
		},
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
		Client: &http.Client{
			Transport: &mockRoundTripper{},
		},
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
		Client: &http.Client{
			Transport: &mockRoundTripper{},
		},
		Log: logger,
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

// TODO add more for test coverage
