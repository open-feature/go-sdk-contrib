//go:build integration
// +build integration

package flagdhttpconnector

// generate full unit tests
import (
	"context"
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
