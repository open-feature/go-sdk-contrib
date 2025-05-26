//go:build integration
// +build integration

package flagdhttpconnector

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/open-feature/flagd/core/pkg/logger"
	flagdsync "github.com/open-feature/flagd/core/pkg/sync"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestGithubRawContent(t *testing.T) {

	testURL := "https://raw.githubusercontent.com/open-feature/java-sdk-contrib/main/tools/flagd-http-connector/src/test/resources/testing-flags.json"

	opts := HttpConnectorOptions{
		URL:                   testURL,
		PollIntervalSeconds:   10,
		ConnectTimeoutSeconds: 5,
		RequestTimeoutSeconds: 15,
		log:                   logger.NewLogger(NewRaw(), false),
	}

	connector, err := NewHttpConnector(opts)
	require.NoError(t, err)
	defer connector.Shutdown()

	connector.Init(context.Background())
	syncChan := make(chan flagdsync.DataSync, 1)
	connector.Sync(context.Background(), syncChan)

	// Check if the sync channel received any data
	success := false
	go func() {
		select {
		case data := <-syncChan:
			assert.NotEmpty(t, data.FlagData, "Flag data should not be empty")
			assert.Equal(t, testURL, data.Source, "Source should match the test URL")
			assert.Equal(t, flagdsync.ALL, data.Type, "Type should be ALL for initial sync")
			success = true
		}
	}()

	assert.Eventually(t, func() bool {
		return success
	}, 15*time.Second, 1*time.Second, "Sync channel should receive data within 15 seconds")
}

func TestGithubRawContentUsingCache(t *testing.T) {
	testURL := "https://raw.githubusercontent.com/open-feature/java-sdk-contrib/main/tools/flagd-http-connector/src/test/resources/testing-flags.json"

	opts := HttpConnectorOptions{
		URL:                   testURL,
		PollIntervalSeconds:   5,
		ConnectTimeoutSeconds: 5,
		RequestTimeoutSeconds: 15,
		log:                   logger.NewLogger(NewRaw(), false),
		PayloadCache:          NewMockPayloadCache(),
		UsePollingCache:       true,
		UseFailsafeCache:      true,
		PayloadCacheOptions: &PayloadCacheOptions{
			UpdateIntervalSeconds: 5,
		},
	}

	connector, err := NewHttpConnector(opts)
	require.NoError(t, err)
	defer connector.Shutdown()

	connector.Init(context.Background())

	syncChan := make(chan flagdsync.DataSync, 1)

	connector.Sync(context.Background(), syncChan)

	// Check if the sync channel received any data
	success := false

	go func() {
		select {
		case data := <-syncChan:
			assert.NotEmpty(t, data.FlagData, "Flag data should not be empty")
			assert.Equal(t, testURL, data.Source, "Source should match the test URL")
			assert.Equal(t, flagdsync.ALL, data.Type, "Type should be ALL for initial sync")
			success = true
		}
	}()

	assert.Eventually(t, func() bool {
		return opts.PayloadCache.(*MockPayloadCache).SuccessGetCount == 2 && success
	}, 15*time.Second, 1*time.Second, "Sync channel should receive data within 15 seconds and cache should be hit once")

}

// MockPayloadCache PayloadCache implementation based on map
type MockFailSafeCache struct {
	cache           sync.Map
	SuccessGetCount int
	FailureGetCount int
}

func NewMockFailSafeCache() *MockFailSafeCache {
	return &MockFailSafeCache{}
}

// Get retrieves a payload from the cache by key.
func (m *MockFailSafeCache) Get(key string) (string, error) {
	value, ok := m.cache.Load(key)
	if !ok {
		m.FailureGetCount++
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
func (m *MockFailSafeCache) Put(key, payload string) error {
	m.PutWithTTL(key, payload, 1)
	return nil
}

// PutWithTTL adds a payload to the cache with a time-to-live (TTL).
func (m *MockFailSafeCache) PutWithTTL(key, payload string, ttlSeconds int) error {
	m.cache.Store(key, payload)

	// Start a goroutine to remove the key after the TTL expires.
	go func() {
		time.Sleep(time.Duration(ttlSeconds) * time.Second)
		m.cache.Delete(key)
	}()
	return nil
}

func TestGithubRawContentUsingFailsafeCache(t *testing.T) {

	// non-existing url, simulating Github down
	invalidTestUrl := "https://raw.githubusercontent.com/open-feature/java-sdk-contrib/main/tools/flagd-http-connector/src/test/resources/non-existing-flags.json"

	opts := HttpConnectorOptions{
		URL:                   invalidTestUrl,
		PollIntervalSeconds:   5,
		ConnectTimeoutSeconds: 5,
		RequestTimeoutSeconds: 15,
		log:                   logger.NewLogger(NewRaw(), false),
		PayloadCache:          NewMockFailSafeCache(),
		UsePollingCache:       true,
		UseFailsafeCache:      true,
		PayloadCacheOptions: &PayloadCacheOptions{
			UpdateIntervalSeconds: 5,
			FailSafeTTLSeconds:    10,
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

	connector.Sync(context.Background(), syncChan)

	// Check if the sync channel received any data
	success := false

	go func() {
		select {
		case data := <-syncChan:
			assert.NotEmpty(t, data.FlagData, "Flag data should not be empty")
			assert.Equal(t, invalidTestUrl, data.Source, "Source should match the test URL")
			assert.Equal(t, flagdsync.ALL, data.Type, "Type should be ALL for initial sync")
			assert.Equal(t, testPayload, data.FlagData, "Flag data should match the cached payload")
			success = true
		}
	}()

	assert.Eventually(t, func() bool {
		connector.options.log.Logger.Debug("Checking if sync channel received data",
			zap.Int("SuccessGetCount", connector.failSafeCache.payloadCache.(*MockFailSafeCache).SuccessGetCount),
			zap.Int("FailureGetCount", connector.failSafeCache.payloadCache.(*MockFailSafeCache).FailureGetCount),
		)
		return connector.failSafeCache.payloadCache.(*MockFailSafeCache).SuccessGetCount == 1 && // Ensure that the cache was hit once for the initial fetch
			opts.PayloadCache.(*MockFailSafeCache).FailureGetCount == 1 && // Ensure that the cache was hit once for the failure for payload cache
			success
	}, 3*time.Second, 1*time.Second, "Sync channel should receive data within 15 seconds "+
		"and cache should be hit once")
}

type EncoderConfigOption func(*zapcore.EncoderConfig)

func newJSONEncoder(opts ...EncoderConfigOption) zapcore.Encoder {
	encoderConfig := zap.NewProductionEncoderConfig()
	for _, opt := range opts {
		opt(&encoderConfig)
	}
	return zapcore.NewJSONEncoder(encoderConfig)
}

func NewRaw() *zap.Logger {
	level := zap.NewAtomicLevelAt(zap.DebugLevel)

	var zapOpts []zap.Option
	if level.Enabled(zapcore.Level(-2)) {
		zapOpts = append(zapOpts,
			zap.WrapCore(func(core zapcore.Core) zapcore.Core {
				return zapcore.NewSamplerWithOptions(core, time.Second, 100, 100)
			}))
	}
	zapOpts = append(zapOpts, zap.AddStacktrace(zap.NewAtomicLevelAt(zap.ErrorLevel)))

	f := func(ecfg *zapcore.EncoderConfig) {
		ecfg.EncodeTime = zapcore.RFC3339TimeEncoder
	}
	encoder := newJSONEncoder(f)

	sink := zapcore.AddSync(os.Stderr)
	zapOpts = append(zapOpts, zap.ErrorOutput(sink))
	log := zap.New(zapcore.NewCore(encoder, sink, level))
	log = log.WithOptions(zapOpts...)
	return log
}

// TODO http cache fetcher test
