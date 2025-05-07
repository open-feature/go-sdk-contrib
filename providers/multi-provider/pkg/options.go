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

// WithGlobalHooks sets the global hooks for the provider. These are hooks that affect ALL providers. For hooks that
// target specific providers make sure to attach them to that provider directly, or use the WithProviderHook Option if
// that provider does not provide its own hook functionality
func WithGlobalHooks(hooks ...of.Hook) Option {
	return func(conf *Configuration) {
		conf.hooks = hooks
	}
}

// WithProviderHooks sets hooks that execute only for a specific provider. The providerName must match the unique provider
// name set during MultiProvider creation. This should only be used if you need hooks that execute around a specific
// provider, but that provider does not currently accept a way to set hooks. This option can be used multiple times using
// unique provider names. Using a provider name that is not known will cause an error.
func WithProviderHooks(providerName string, hooks ...of.Hook) Option {
	return func(conf *Configuration) {
		conf.providerHooks[providerName] = hooks
	}
}
