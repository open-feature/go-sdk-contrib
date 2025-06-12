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

// WithEventPublishing Enables event publishing (Not Yet Implemented)
func WithEventPublishing() Option {
	return func(conf *Configuration) {
		conf.publishEvents = true
	}
}

// WithoutEventPublishing Disables event publishing (this is the default, but included for explicit usage)
func WithoutEventPublishing() Option {
	return func(conf *Configuration) {
		conf.publishEvents = false
	}
}
