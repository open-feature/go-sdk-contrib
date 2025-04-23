package multiprovider

import (
	of "github.com/open-feature/go-sdk/openfeature"
	"log/slog"
	"time"
)

func WithLogger(l *slog.Logger) Option {
	return func(conf *Configuration) {
		conf.logger = l
	}
}

func WithTimeout(d time.Duration) Option {
	return func(conf *Configuration) {
		conf.timeout = d
	}
}

func WithFallbackProvider(p of.FeatureProvider) Option {
	return func(conf *Configuration) {
		conf.fallbackProvider = p
		conf.useFallback = true
	}
}

func WithEventPublishing() Option {
	return func(conf *Configuration) {
		conf.publishEvents = true
	}
}

func WithoutEventPublishing() Option {
	return func(conf *Configuration) {
		conf.publishEvents = false
	}
}
