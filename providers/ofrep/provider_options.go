package ofrep

import (
	"time"

	"github.com/open-feature/go-sdk-contrib/providers/ofrep/internal/outbound"
)

// WithPollingInterval allows to set the polling interval for the OFREP bulk provider
func WithPollingInterval(interval time.Duration) func(*outbound.Configuration) {
	return func(c *outbound.Configuration) {
		c.ClientPollingInterval = interval
	}
}
