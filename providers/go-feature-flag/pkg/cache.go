package gofeatureflag

import (
	"github.com/bluele/gcache"
	"time"
)

const defaultCacheSize = 10000
const defaultCacheTTL = 1 * time.Minute

type Cache struct {
	// CacheSize (optional) is the maximum number of flag events we keep in memory to cache your flags.
	// default: 10000
	CacheSize int

	// CacheTTL (optional) is the time we keep the evaluation in the cache before we consider it as obsolete.
	// If you want to keep the value forever you can set the CacheTTL field to -1
	// default: 1 minute
	CacheTTL time.Duration

	// cache is the internal representation of the cache.
	cache gcache.Cache
}

func NewCache(cacheSize int, cacheTTL time.Duration) *Cache {
	if cacheSize == 0 {
		cacheSize = defaultCacheSize
	}
	if cacheTTL == 0 {
		cacheTTL = defaultCacheTTL
	}

	return &Cache{
		CacheSize: cacheSize,
		CacheTTL:  cacheTTL,
		cache:     gcache.New(cacheSize).LRU().Build(),
	}
}

func (c *Cache) Get(key interface{}) (interface{}, error) {
	return c.cache.Get(key)
}
func (c *Cache) Set(key, value interface{}) error {
	if c.cache == nil {
		return nil
	}

	if c.CacheTTL == -1 {
		_ = c.cache.Set(key, value)
	} else {
		_ = c.cache.SetWithExpire(key, value, c.CacheTTL)
	}
	return c.cache.Set(key, value)
}

func (c *Cache) Remove(key interface{}) bool {
	return c.cache.Remove(key)
}

func (c *Cache) Intialize() {
	// Nothing here for now
}

func (c *Cache) Shutdown() {
	// TODO: clean + stop websocket
	c.cache.Purge()
}
