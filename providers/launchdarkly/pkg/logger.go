package launchdarkly

// Logger defines a minimal interface for the provider's logger.
type Logger interface {
	Debug(msg string, args ...any)
	Error(msg string, args ...any)
	Warn(msg string, args ...any)
}

var _ Logger = (*NoOpLogger)(nil)

type NoOpLogger struct{}

func (l *NoOpLogger) Debug(msg string, args ...any) {}
func (l *NoOpLogger) Info(msg string, args ...any)  {}
func (l *NoOpLogger) Error(msg string, args ...any) {}
func (l *NoOpLogger) Warn(msg string, args ...any)  {}
