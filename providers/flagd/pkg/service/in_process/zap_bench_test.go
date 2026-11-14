package process

import (
	"errors"
	"testing"

	"github.com/go-logr/logr"
	"github.com/open-feature/flagd/core/pkg/logger"
	"go.uber.org/zap"

	levels "github.com/open-feature/go-sdk-contrib/providers/flagd/internal/logger"
)

// nopSink accepts every level up to maxV and discards the output, so the benchmarks measure only
// the adapter and not a sink's encoding.
type nopSink struct{ maxV int }

func (nopSink) Init(logr.RuntimeInfo)            {}
func (s nopSink) Enabled(level int) bool         { return level <= s.maxV }
func (nopSink) Info(int, string, ...any)         {}
func (nopSink) Error(error, string, ...any)      {}
func (s nopSink) WithValues(...any) logr.LogSink { return s }
func (s nopSink) WithName(string) logr.LogSink   { return s }

func benchLogger(maxV int) *logger.Logger {
	return logger.NewLogger(zap.New(logrCore{l: logr.New(nopSink{maxV: maxV})}), false)
}

// BenchmarkLogrCoreInfo logs an info line shaped like flagd core's sync logs.
func BenchmarkLogrCoreInfo(b *testing.B) {
	l := benchLogger(levels.Debug)
	b.ReportAllocs()
	for b.Loop() {
		l.Info("filepath event", zap.String("component", "sync"), zap.String("sync", "filepath"),
			zap.String("path", "flags.json"), zap.String("op", "WRITE"))
	}
}

// BenchmarkLogrCoreError logs an error line with a zap.Error field and an int field.
func BenchmarkLogrCoreError(b *testing.B) {
	l := benchLogger(levels.Debug)
	err := errors.New("boom")
	b.ReportAllocs()
	for b.Loop() {
		l.Error("failed to sync", zap.String("component", "sync"), zap.Int("attempt", 3), zap.Error(err))
	}
}

// BenchmarkLogrCoreDisabled logs a debug line that the sink does not accept.
func BenchmarkLogrCoreDisabled(b *testing.B) {
	l := benchLogger(levels.Info)
	b.ReportAllocs()
	for b.Loop() {
		l.Debug("starting data sync process", zap.String("component", "sync"))
	}
}
