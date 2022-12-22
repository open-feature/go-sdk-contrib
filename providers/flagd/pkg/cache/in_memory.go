package cache

import (
	"sync"
)

type InMemory[K comparable, V any] struct {
	values map[K]V
	rwMux  *sync.RWMutex
}

func NewInMemory[K comparable, V any]() *InMemory[K, V] {
	return &InMemory[K, V]{
		values: make(map[K]V),
		rwMux:  &sync.RWMutex{},
	}
}

func (m *InMemory[K, V]) Add(flagKey K, value V) (evicted bool) {
	m.rwMux.Lock()
	defer m.rwMux.Unlock()
	m.values[flagKey] = value

	return false
}

func (m *InMemory[K, V]) Get(flagKey K) (value V, ok bool) {
	m.rwMux.RLock()
	defer m.rwMux.RUnlock()

	val, ok := m.values[flagKey]

	return val, ok
}

func (m *InMemory[K, V]) Remove(flagKey K) (present bool) {
	m.rwMux.Lock()
	defer m.rwMux.Unlock()

	_, ok := m.values[flagKey]
	delete(m.values, flagKey)
	return ok
}

func (m *InMemory[K, V]) Purge() {
	m.rwMux.Lock()
	defer m.rwMux.Unlock()

	m.values = make(map[K]V)
}
