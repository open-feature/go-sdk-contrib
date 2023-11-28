package rpc

import "time"

const (
	defaultDelay = time.Second
	factor       = 2
)

type retryCounter struct {
	baseRetryDelay time.Duration
	maxRetries     int

	currentDelay   time.Duration
	currentRetries int
}

func newRetryCounter(maxRetries int) retryCounter {
	return retryCounter{
		baseRetryDelay: defaultDelay,
		maxRetries:     maxRetries,
	}
}

// reset the retry counter and sleep delay
func (c *retryCounter) reset() {
	c.currentDelay = defaultDelay
	c.currentRetries = 0
}

// retry increments current retry attempts, check and return a boolean stating retry is allowed
func (c *retryCounter) retry() bool {
	c.currentRetries++
	return c.currentRetries <= c.maxRetries
}

// sleep returns the current sleep delay and increment the next sleep value
func (c *retryCounter) sleep() time.Duration {
	var value = c.currentDelay
	c.currentDelay = factor * c.currentDelay

	return value
}
