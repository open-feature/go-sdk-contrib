package process

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
	"time"
)

// EncoderConfigOption is a function that can modify a `zapcore.EncoderConfig`.
type EncoderConfigOption func(*zapcore.EncoderConfig)

func newJSONEncoder(opts ...EncoderConfigOption) zapcore.Encoder {
	encoderConfig := zap.NewProductionEncoderConfig()
	for _, opt := range opts {
		opt(&encoderConfig)
	}
	return zapcore.NewJSONEncoder(encoderConfig)
}

// NewRaw returns a new zap.Logger configured with the passed Opts
// or their defaults. It uses KubeAwareEncoder which adds Type
// information and Namespace/Name to the log.
func NewRaw() *zap.Logger {
	level := zap.NewAtomicLevelAt(zap.InfoLevel)

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

	// this basically mimics New<type>Config, but with a custom sink
	sink := zapcore.AddSync(os.Stderr)
	zapOpts = append(zapOpts, zap.ErrorOutput(sink))
	log := zap.New(zapcore.NewCore(encoder, sink, level))
	log = log.WithOptions(zapOpts...)
	return log
}
