package logger

import (
	"context"
	"log/slog"
)

// ConditionalLogger Logger instance that may be empty so the caller does not need to worry about checking
// if logging is enabled or not. This type should be treated as immutable
type ConditionalLogger struct {
	l *slog.Logger
}

// NewConditionalLogger Creates a new ConditionalLogger. If a nil value is provided no logging will be performed and all
// methods will act as no-ops. The state of a ConditionalLogger should be treated as immutable
func NewConditionalLogger(l *slog.Logger) *ConditionalLogger {
	return &ConditionalLogger{l}
}

// enabled Checks to determine if logging should be performed. Also acts as an internal nil check
func (cl *ConditionalLogger) enabled() bool {
	return cl.l != nil
}

// LogError Log a message at the error level
func (cl *ConditionalLogger) LogError(ctx context.Context, msg string, attr ...slog.Attr) {
	if cl.enabled() {
		cl.l.LogAttrs(ctx, slog.LevelError, msg, attr...)
	}
}

// LogWarn Log a message at the warn level
func (cl *ConditionalLogger) LogWarn(ctx context.Context, msg string, attr ...slog.Attr) {
	if cl.enabled() {
		cl.l.LogAttrs(ctx, slog.LevelWarn, msg, attr...)
	}
}

// LogInfo Log a message at the info level (should be used sparingly)
func (cl *ConditionalLogger) LogInfo(ctx context.Context, msg string, attr ...slog.Attr) {
	if cl.enabled() {
		cl.l.LogAttrs(ctx, slog.LevelInfo, msg, attr...)
	}
}

// LogDebug Log a message at the debug level
func (cl *ConditionalLogger) LogDebug(ctx context.Context, msg string, attr ...slog.Attr) {
	if cl.enabled() {
		cl.l.LogAttrs(ctx, slog.LevelDebug, msg, attr...)
	}
}

// With Creates and returns a child logger with the provided attributes set. If the current logger is disabled by having
// the same disabled logger will be returned and this acts as a no-op.
func (cl *ConditionalLogger) With(attr ...any) *ConditionalLogger {
	if cl.enabled() {
		return &ConditionalLogger{l: cl.l.With(attr...)}
	}

	// Don't bother creating a child logger since there's no difference
	return cl
}
