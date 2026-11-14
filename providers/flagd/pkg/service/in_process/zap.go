package process

import (
	"fmt"
	"os"
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

// vLevel maps a zap level below Error to the provider's logr V-level.
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

// Enabled reports whether the logr logger accepts entries at lvl. Error and above are always on.
func (c logrCore) Enabled(lvl zapcore.Level) bool {
	return lvl >= zapcore.ErrorLevel || c.l.V(vLevel(lvl)).Enabled()
}

// With returns a core whose logr logger carries fields as key/value pairs.
func (c logrCore) With(fields []zapcore.Field) zapcore.Core {
	return logrCore{l: c.l.WithValues(keysAndValues(fields)...)}
}

// Check adds this core to ce when the entry's level is enabled.
func (c logrCore) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(ent.Level) {
		return ce.AddCore(ent, c)
	}
	return ce
}

// Write sends the entry to logr: Error and above to Error, lower levels to V(level).Info.
func (c logrCore) Write(ent zapcore.Entry, fields []zapcore.Field) error {
	// skip logrCore.Write, zapcore.CheckedEntry.Write and the zap/flagd logger method,
	// so sinks that record the caller report the real call site
	l := c.l.WithCallDepth(3)
	if ent.Level < zapcore.ErrorLevel {
		l.V(vLevel(ent.Level)).Info(ent.Message, keysAndValues(fields)...)
		return nil
	}
	// hand the first zap.Error field to logr as err instead of a duplicate "error" value
	var err error
	skip := -1
	for i, f := range fields {
		if e, ok := f.Interface.(error); ok && f.Type == zapcore.ErrorType {
			err, skip = e, i
			break
		}
	}
	kv := make([]any, 0, 2*len(fields))
	for i, f := range fields {
		if i != skip {
			kv = appendField(kv, f)
		}
	}
	l.Error(err, ent.Message, kv...)
	return nil
}

// Sync is a no-op; logr has no flush.
func (c logrCore) Sync() error { return nil }

// keysAndValues encodes zap fields into logr key/value pairs, in field order.
func keysAndValues(fields []zapcore.Field) []any {
	if len(fields) == 0 {
		return nil
	}
	kv := make([]any, 0, 2*len(fields))
	for _, f := range fields {
		kv = appendField(kv, f)
	}
	return kv
}

// appendField appends one zap field as a key/value pair. The scalar types flagd core logs are
// read straight from the field; other types go through a MapObjectEncoder.
func appendField(kv []any, f zapcore.Field) []any {
	switch f.Type {
	case zapcore.StringType:
		return append(kv, f.Key, f.String)
	case zapcore.Int64Type, zapcore.Int32Type, zapcore.Int16Type, zapcore.Int8Type:
		return append(kv, f.Key, f.Integer)
	case zapcore.BoolType:
		return append(kv, f.Key, f.Integer == 1)
	case zapcore.DurationType:
		return append(kv, f.Key, time.Duration(f.Integer))
	case zapcore.ErrorType:
		return append(kv, f.Key, f.Interface)
	case zapcore.StringerType:
		return append(kv, f.Key, f.Interface.(fmt.Stringer).String())
	case zapcore.SkipType:
		return kv
	}
	enc := zapcore.NewMapObjectEncoder()
	f.AddTo(enc)
	for k, v := range enc.Fields {
		kv = append(kv, k, v)
	}
	return kv
}
