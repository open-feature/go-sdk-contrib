package logger

import (
	"log"

	"github.com/go-logr/logr"
)

const (
	Warn  = 0
	Info  = 1
	Debug = 2
)

// Logger is the provider's default logger
// logs using the standard log package on error, all other logs are no-ops
type Logger struct{}

func (l Logger) Init(info logr.RuntimeInfo) {}

func (l Logger) Enabled(level int) bool { return true }

func (l Logger) Info(level int, msg string, keysAndValues ...interface{}) {}

func (l Logger) Error(err error, msg string, keysAndValues ...interface{}) {
	log.Println("flagd-provider:", err)
}

func (l Logger) WithValues(keysAndValues ...interface{}) logr.LogSink { return l }

func (l Logger) WithName(name string) logr.LogSink { return l }
