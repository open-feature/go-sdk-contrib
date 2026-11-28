package process

import (
	"os"
	"reflect"
	"slices"
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
	// ns holds the fields from the first zap.Namespace onward that With received. logr cannot
	// reopen a map it already got, so they are encoded together with each entry's fields.
	ns []zapcore.Field
}

// newLogrCore returns a logrCore for l. The call depth skips logrCore.Write,
// zapcore.CheckedEntry.Write and the zap/flagd logger method, so sinks that record the caller
// report the real call site; WithValues keeps it for cores built by With.
func newLogrCore(l logr.Logger) *logrCore {
	return &logrCore{l: l.WithCallDepth(3)}
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
func (c *logrCore) Enabled(lvl zapcore.Level) bool {
	return lvl >= zapcore.ErrorLevel || c.l.V(vLevel(lvl)).Enabled()
}

// With returns a core whose logr logger carries fields as key/value pairs. Fields from the first
// zap.Namespace onward are kept on the core instead, so later fields nest under that namespace.
func (c *logrCore) With(fields []zapcore.Field) zapcore.Core {
	if len(c.ns) > 0 {
		// Clip forces append to copy, so sibling cores never share a backing array
		return &logrCore{l: c.l, ns: append(slices.Clip(c.ns), fields...)}
	}
	i := nsIndex(fields)
	return &logrCore{l: c.l.WithValues(keysAndValues(fields[:i], -1, nil)...), ns: slices.Clone(fields[i:])}
}

// Check adds this core to ce when the entry's level is enabled.
func (c *logrCore) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(ent.Level) {
		return ce.AddCore(ent, c)
	}
	return ce
}

// Write sends the entry to logr: Error and above to Error, lower levels to V(level).Info.
func (c *logrCore) Write(ent zapcore.Entry, fields []zapcore.Field) error {
	// top holds the fields before any namespace; nested holds the namespace and what follows it
	top, nested := fields, c.ns
	if len(nested) > 0 {
		top, nested = nil, append(slices.Clip(nested), fields...)
	} else {
		i := nsIndex(fields)
		top, nested = fields[:i], fields[i:]
	}
	if ent.Level < zapcore.ErrorLevel {
		c.l.V(vLevel(ent.Level)).Info(ent.Message, keysAndValues(top, -1, nested)...)
		return nil
	}
	// hand the first top-level zap.Error field to logr as err instead of a duplicate "error"
	// value; an error inside a namespace stays nested, as zap's JSON encoder writes it. A
	// zap.NamedError keeps its key, and a nil pointer error stays a field so it logs as "<nil>"
	// instead of panicking in a sink that calls Error on it.
	var err error
	skip := -1
	for i, f := range top {
		if e, ok := f.Interface.(error); ok && f.Type == zapcore.ErrorType && f.Key == "error" && !nilPtr(e) {
			err, skip = e, i
			break
		}
	}
	c.l.Error(err, ent.Message, keysAndValues(top, skip, nested)...)
	return nil
}

// Sync is a no-op; logr has no flush.
func (c *logrCore) Sync() error { return nil }

// nsIndex returns the index of the first zap.Namespace field, or len(fields) if there is none.
func nsIndex(fields []zapcore.Field) int {
	for i, f := range fields {
		if f.Type == zapcore.NamespaceType {
			return i
		}
	}
	return len(fields)
}

// keysAndValues encodes zap fields into logr key/value pairs. The top fields come first, in order,
// without the one at index skip. The nested fields, which start with a zap.Namespace, go through one
// shared MapObjectEncoder so every field after a namespace lands inside it.
func keysAndValues(top []zapcore.Field, skip int, nested []zapcore.Field) []any {
	if len(top) == 0 && len(nested) == 0 {
		return nil
	}
	kv := make([]any, 0, 2*len(top))
	for i, f := range top {
		if i != skip {
			kv = appendField(kv, f)
		}
	}
	if len(nested) > 0 {
		enc := zapcore.NewMapObjectEncoder()
		for _, f := range nested {
			f.AddTo(enc)
		}
		for k, v := range enc.Fields {
			kv = append(kv, k, v)
		}
	}
	return kv
}

// appendField appends one zap field as a key/value pair. The scalar types flagd core logs are
// read straight from the field; other types, and an error that is a nil pointer, go through a
// MapObjectEncoder, which recovers a panic from String or Error and writes "<nil>" for a nil pointer.
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
		if !nilPtr(f.Interface) {
			return append(kv, f.Key, f.Interface)
		}
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

// nilPtr reports whether v is a nil pointer.
func nilPtr(v any) bool {
	rv := reflect.ValueOf(v)
	return rv.Kind() == reflect.Pointer && rv.IsNil()
}
