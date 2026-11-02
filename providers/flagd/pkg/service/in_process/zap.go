package process

import (
	"os"
	"sort"
	"time"

	"github.com/go-logr/logr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	levels "github.com/open-feature/go-sdk-contrib/providers/flagd/internal/logger"
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

// logrCore is a zapcore.Core that forwards entries to a logr.Logger, so flagd core's zap-based
// logging reaches the logger passed via WithLogger. Debug, Info and Warn map to the provider's
// V-levels; Error and above go to logr's Error.
type logrCore struct {
	l logr.Logger
}

func vLevel(lvl zapcore.Level) int {
	switch {
	case lvl <= zapcore.DebugLevel:
		return levels.Debug
	case lvl == zapcore.InfoLevel:
		return levels.Info
	default:
		return levels.Warn
	}
}

func (c logrCore) Enabled(lvl zapcore.Level) bool {
	return lvl >= zapcore.ErrorLevel || c.l.V(vLevel(lvl)).Enabled()
}

func (c logrCore) With(fields []zapcore.Field) zapcore.Core {
	return logrCore{l: c.l.WithValues(keysAndValues(fields)...)}
}

func (c logrCore) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(ent.Level) {
		return ce.AddCore(ent, c)
	}
	return ce
}

func (c logrCore) Write(ent zapcore.Entry, fields []zapcore.Field) error {
	kv := keysAndValues(fields)
	if ent.Level >= zapcore.ErrorLevel {
		c.l.Error(nil, ent.Message, kv...)
	} else {
		c.l.V(vLevel(ent.Level)).Info(ent.Message, kv...)
	}
	return nil
}

func (c logrCore) Sync() error { return nil }

// keysAndValues encodes zap fields into logr key/value pairs, sorted by key.
func keysAndValues(fields []zapcore.Field) []any {
	if len(fields) == 0 {
		return nil
	}
	enc := zapcore.NewMapObjectEncoder()
	for _, f := range fields {
		f.AddTo(enc)
	}
	keys := make([]string, 0, len(enc.Fields))
	for k := range enc.Fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	kv := make([]any, 0, 2*len(keys))
	for _, k := range keys {
		kv = append(kv, k, enc.Fields[k])
	}
	return kv
}
