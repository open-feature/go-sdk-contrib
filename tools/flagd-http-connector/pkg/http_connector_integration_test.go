//go:build integration
// +build integration

package flagdhttpconnector

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/open-feature/flagd/core/pkg/logger"
	flagdsync "github.com/open-feature/flagd/core/pkg/sync"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestIntegrationGithubRawContent(t *testing.T) {

	testURL := "https://raw.githubusercontent.com/open-feature/java-sdk-contrib/main/tools/flagd-http-connector/src/test/resources/testing-flags.json"

	log := NewRaw()
	defer func() {
		if err := log.Sync(); err != nil {
			slog.Error("Failed to sync logger", "error", err)
		}
	}()
	opts := HttpConnectorOptions{
		URL:                   testURL,
		PollIntervalSeconds:   10,
		ConnectTimeoutSeconds: 5,
		RequestTimeoutSeconds: 15,
		Log:                   logger.NewLogger(log, false),
	}

	log.Info("Starting integration test for HTTP connector")
	connector, err := NewHttpConnector(opts)

	require.NoError(t, err)
	defer connector.Shutdown()

	connector.Init(context.Background())
	syncChan := make(chan flagdsync.DataSync, 1)

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

	connector.Sync(context.Background(), syncChan)

	assert.Eventually(t, func() bool {
		return success
	}, 15*time.Second, 1*time.Second, "Sync channel should receive data within 15 seconds")
}

func TestIntegrationGithubRawContentUsingCache(t *testing.T) {
	testURL := "https://raw.githubusercontent.com/open-feature/java-sdk-contrib/main/tools/flagd-http-connector/src/test/resources/testing-flags.json"

	opts := HttpConnectorOptions{
		URL:                   testURL,
		PollIntervalSeconds:   5,
		ConnectTimeoutSeconds: 5,
		RequestTimeoutSeconds: 15,
		Log:                   logger.NewLogger(NewRaw(), false),
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

	// second connector to simulate a different micro-service instance using same cache (e.g. Redis)

	// simulate start a bit later
	time.Sleep(200 * time.Millisecond)

	connector2, err := NewHttpConnector(opts)
	require.NoError(t, err)
	defer connector2.Shutdown()

	connector2.Init(context.Background())

	syncChan := make(chan flagdsync.DataSync, 1)
	defer close(syncChan)

	slog.Info("Starting sync for first connector")
	connector.Sync(context.Background(), syncChan)
	slog.Info("Starting sync for second connector")
	connector2.Sync(context.Background(), syncChan)
	slog.Info("Sync started for both connectors")

	// Check if the sync channel received any data
	success := false

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
				success = true
			}
		}
	}()

	assert.Eventually(t, func() bool {
		return opts.PayloadCache.(*MockPayloadCache).SuccessGetCount >= 2 && success
	}, 15*time.Second, 1*time.Second, "Sync channel should receive data within 15 seconds and cache should be hit once")

}

func TestIntegrationGithubRawContentUsingFailsafeCache(t *testing.T) {

	// non-existing url, simulating Github down
	invalidTestUrl := "https://raw.githubusercontent.com/open-feature/java-sdk-contrib/main/tools/flagd-http-connector/src/test/resources/non-existing-flags.json"

	opts := HttpConnectorOptions{
		URL:                   invalidTestUrl,
		PollIntervalSeconds:   5,
		ConnectTimeoutSeconds: 5,
		RequestTimeoutSeconds: 15,
		Log:                   logger.NewLogger(NewRaw(), false),
		PayloadCache:          NewMockFailSafeCache(),
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

	// simulate cache hit by pre-populating the fail-safe cache with a payload from previous micro-service run
	testPayload := "test-payload"

	connector.failSafeCache.payloadCache.Put(FailSafePayloadCacheKey, testPayload)

	syncChan := make(chan flagdsync.DataSync, 1)

	// Check if the sync channel received any data
	success := false

	go func() {
		for {
			select {
			case data := <-syncChan:
				if data.FlagData == "" {
					slog.Info("Received empty flag data from sync channel")
					return
				}
				assert.NotEmpty(t, data.FlagData, "Flag data should not be empty")
				assert.Equal(t, invalidTestUrl, data.Source, "Source should match the test URL")
				assert.Equal(t, flagdsync.ALL, data.Type, "Type should be ALL for initial sync")
				assert.Equal(t, testPayload, data.FlagData, "Flag data should match the cached payload")
				success = true
			}
		}
	}()

	connector.Sync(context.Background(), syncChan)

	assert.Eventually(t, func() bool {
		slog.Debug("Checking if sync channel received data",
			slog.Int("SuccessGetCount", connector.failSafeCache.payloadCache.(*MockFailSafeCache).SuccessGetCount),
			slog.Int("FailureGetCount", connector.failSafeCache.payloadCache.(*MockFailSafeCache).FailureGetCount),
		)
		return connector.failSafeCache.payloadCache.(*MockFailSafeCache).SuccessGetCount == 1 && // Ensure that the cache was hit once for the initial fetch
			opts.PayloadCache.(*MockFailSafeCache).FailureGetCount == 2 && // Ensure that the cache was hit once for the failure for payload cache
			success
	}, 3*time.Second, 1*time.Second, "Sync channel should receive data within 15 seconds "+
		"and cache should be hit once")
}

func TestIntegrationGithubRawContentUsingHttpCache(t *testing.T) {
	testURL := "https://raw.githubusercontent.com/open-feature/java-sdk-contrib/main/tools/flagd-http-connector/src/test/resources/testing-flags.json"

	opts := HttpConnectorOptions{
		URL:                   testURL,
		PollIntervalSeconds:   5,
		ConnectTimeoutSeconds: 5,
		RequestTimeoutSeconds: 15,
		Log:                   logger.NewLogger(NewRaw(), false),
		UseHttpCache:          true,
	}

	connector, err := NewHttpConnector(opts)
	require.NoError(t, err)
	defer connector.Shutdown()

	connector.Init(context.Background())

	syncChan := make(chan flagdsync.DataSync, 1)

	connector.Sync(context.Background(), syncChan)

	dataCount := 0

	go func() {
		select {
		case data := <-syncChan:
			assert.NotEmpty(t, data.FlagData, "Flag data should not be empty")
			assert.Equal(t, testURL, data.Source, "Source should match the test URL")
			assert.Equal(t, flagdsync.ALL, data.Type, "Type should be ALL for initial sync")
			dataCount++
		}
	}()

	time.Sleep(16 * time.Second)

	assert.Equal(t, 1, dataCount, "Sync channel should receive data exactly once")
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
	zapOpts = append(zapOpts, zap.AddStacktrace(zap.NewAtomicLevelAt(zap.ErrorLevel)))

	f := func(ecfg *zapcore.EncoderConfig) {
		ecfg.EncodeTime = zapcore.RFC3339TimeEncoder
	}

	// add struct memory address to the encoder config

	encoder := newJSONEncoder(f)

	sink := zapcore.AddSync(zapcore.Lock(os.Stderr))
	zapOpts = append(zapOpts, zap.ErrorOutput(sink))
	log := zap.New(zapcore.NewCore(encoder, sink, level))
	log = log.WithOptions(zapOpts...)
	return log
}
