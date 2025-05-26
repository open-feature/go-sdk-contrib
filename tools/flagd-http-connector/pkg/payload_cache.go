package flagdhttpconnector

import "errors"

// PayloadCache defines the interface for a simple payload cache.
type PayloadCache interface {
	Get(key string) (string, error)
	Put(key, payload string) error

	// PutWithTTL puts a payload into the cache with a time-to-live (TTL).
	// This must be implemented if usePollingCache is true.
	PutWithTTL(key, payload string, ttlSeconds int) error
}

// ErrPutWithTTLNotSupported is returned if the cache doesn't support TTL operations.
var ErrPutWithTTLNotSupported = errors.New("PutWithTTL not supported")

type PayloadCacheOptions struct {
	UpdateIntervalSeconds int
	FailSafeTTLSeconds    int
}
