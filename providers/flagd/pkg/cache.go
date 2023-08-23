package flagd

import (
	"github.com/go-logr/logr"
	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/open-feature/go-sdk-contrib/providers/flagd/pkg/cache"
)

// Cache is the contract of the cache implementation
type Cache[K comparable, V any] interface {
	Add(K, V) (evicted bool)
	Purge()
	Get(K) (value V, ok bool)
	Remove(K) (present bool)
}

type cacheService struct {
	cacheEnabled bool
	cache        Cache[string, interface{}]
}

func newCacheService(cacheType string, maxCacheSize int, log logr.Logger) *cacheService {
	var c Cache[string, interface{}]
	var err error
	var cacheEnabled bool

	// setup cache
	switch cacheType {
	case cacheLRUValue:
		c, err = lru.New[string, interface{}](maxCacheSize)
		if err != nil {
			log.Error(err, "init lru cache")
		} else {
			cacheEnabled = true
		}
	case cacheInMemValue:
		c = cache.NewInMemory[string, interface{}]()
		cacheEnabled = true
	case cacheDisabledValue:
	default:
		cacheEnabled = false
		c = nil
	}

	return &cacheService{
		cacheEnabled: cacheEnabled,
		cache:        c,
	}
}

func (s *cacheService) getCache() Cache[string, interface{}] {
	return s.cache
}

func (s *cacheService) isEnabled() bool {
	return s.cacheEnabled
}

func (s *cacheService) disable() {
	s.cacheEnabled = false
}
