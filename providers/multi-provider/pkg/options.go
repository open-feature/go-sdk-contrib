package multiprovider

import (
	"github.com/open-feature/go-sdk-contrib/providers/multi-provider/pkg/strategies"
	of "github.com/open-feature/go-sdk/openfeature"
	"log/slog"
	"time"
)

// WithLogger Sets a logger to be used with slog for internal logging
func WithLogger(l *slog.Logger) Option {
	return func(conf *Configuration) {
		conf.logger = l
	}
}

// WithLoggerDefault Uses the default [slog.Logger] (this is the default setting)
// use WithoutLogging to disable logging completely
func WithLoggerDefault() Option {
	return func(conf *Configuration) {
		conf.logger = slog.Default()
	}
}

// WithoutLogging Disables logging functionality
func WithoutLogging() Option {
	return func(conf *Configuration) {
		conf.logger = nil
	}
}

// WithTimeout Set a timeout for the total runtime for evaluation of parallel strategies
func WithTimeout(d time.Duration) Option {
	return func(conf *Configuration) {
		conf.timeout = d
	}
}

// WithFallbackProvider Sets a fallback provider when using the StrategyComparison
func WithFallbackProvider(p of.FeatureProvider) Option {
	return func(conf *Configuration) {
		conf.fallbackProvider = p
		conf.useFallback = true
	}
}

// WithCustomStrategy sets a custom strategy. This must be used in conjunction with StrategyCustom
func WithCustomStrategy(s strategies.Strategy) Option {
	return func(conf *Configuration) {
		conf.customStrategy = s
	}
}
