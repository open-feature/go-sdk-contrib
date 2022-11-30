package cache

import (
	"context"
	"sync"
)

type InMemory struct {
	values map[string]interface{}
	rwMux  *sync.RWMutex
}

func NewInMemory() *InMemory {
	return &InMemory{
		values: make(map[string]interface{}),
		rwMux:  &sync.RWMutex{},
	}
}

func (m *InMemory) Set(ctx context.Context, flagKey string, value interface{}) error {
	m.rwMux.Lock()
	defer m.rwMux.Unlock()
	m.values[flagKey] = value

	return nil
}

func (m *InMemory) Get(ctx context.Context, flagKey string) (interface{}, error) {
	m.rwMux.RLock()
	defer m.rwMux.RUnlock()

	return m.values[flagKey], nil
}

func (m *InMemory) Delete(ctx context.Context, flagKey string) error {
	m.rwMux.Lock()
	defer m.rwMux.Unlock()

	delete(m.values, flagKey)
	return nil
}

func (m *InMemory) DeleteAll(ctx context.Context) error {
	m.rwMux.Lock()
	defer m.rwMux.Unlock()

	m.values = make(map[string]interface{})

	return nil
}
