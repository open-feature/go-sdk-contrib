package cache

import (
	"github.com/go-logr/logr"
	lru "github.com/hashicorp/golang-lru/v2"
)

type Type string

const (
	LRUValue      Type = "lru"
	InMemValue    Type = "mem"
	DisabledValue Type = "disabled"
)

// Cache is the contract of the cache implementation
type Cache[K comparable, V any] interface {
	Add(K, V) (evicted bool)
	Purge()
	Get(K) (value V, ok bool)
	Remove(K) (present bool)
}

type Service struct {
	cacheEnabled bool
	cache        Cache[string, interface{}]
}

func NewCacheService(cacheType Type, maxCacheSize int, log logr.Logger) *Service {
	var c Cache[string, interface{}]
	var err error
	var cacheEnabled bool

	// setup cache
	switch cacheType {
	case LRUValue:
		c, err = lru.New[string, interface{}](maxCacheSize)
		if err != nil {
			log.Error(err, "init lru cache")
		} else {
			cacheEnabled = true
		}
	case InMemValue:
		c = NewInMemory[string, interface{}]()
		cacheEnabled = true
	case DisabledValue:
	default:
		cacheEnabled = false
		c = nil
	}

	return &Service{
		cacheEnabled: cacheEnabled,
		cache:        c,
	}
}

func (s *Service) GetCache() Cache[string, interface{}] {
	return s.cache
}

func (s *Service) IsEnabled() bool {
	return s.cacheEnabled
}

func (s *Service) Disable() {
	if s.IsEnabled() {
		s.cacheEnabled = false
		s.cache.Purge()
	}
}
