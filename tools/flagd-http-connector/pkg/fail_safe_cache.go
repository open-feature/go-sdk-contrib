package flagdhttpconnector

import (
	"errors"
	"time"
)

const FailSafePayloadCacheKey = "FailSafeCache.failsafe-payload"

// FailSafeCache is a non-thread-safe wrapper for managing a payload cache with an update interval.
type FailSafeCache struct {
	lastUpdateTime time.Time
	updateInterval time.Duration
	payloadCache   PayloadCache
}

// NewFailSafeCache constructs a new FailSafeCache.
func NewFailSafeCache(cache PayloadCache, opts *PayloadCacheOptions) (*FailSafeCache, error) {
	if opts == nil || opts.UpdateIntervalSeconds < 1 {
		return nil, errors.New("updateIntervalSeconds must be >= 1")
	}
	return &FailSafeCache{
		updateInterval: time.Duration(opts.UpdateIntervalSeconds) * time.Second,
		payloadCache:   cache,
	}, nil
}

// UpdatePayloadIfNeeded updates the cache if the update interval has elapsed.
func (f *FailSafeCache) UpdatePayloadIfNeeded(payload string) {
	if time.Since(f.lastUpdateTime) < f.updateInterval {
		return
	}

	err := f.payloadCache.Put(FailSafePayloadCacheKey, payload)
	if err != nil {
		return
	}
	f.lastUpdateTime = time.Now()
}

// Get retrieves the cached payload.
func (f *FailSafeCache) Get() string {
	val, err := f.payloadCache.Get(FailSafePayloadCacheKey)
	if err != nil {
		return ""
	}
	return val
}
