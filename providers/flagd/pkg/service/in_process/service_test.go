package process_test

import (
	"context"
	"sync"
	"testing"
	"time"

	isync "github.com/open-feature/flagd/core/pkg/sync"
	of "github.com/open-feature/go-sdk/openfeature"
	process "github.com/open-feature/go-sdk-contrib/providers/flagd/pkg/service/in_process"
)

type mockSync struct {
	events   chan process.SyncEvent
	dataChan chan chan<- isync.DataSync
}

func (m *mockSync) Init(ctx context.Context) error { return nil }
func (m *mockSync) IsReady() bool                  { return true }
func (m *mockSync) ReSync(ctx context.Context, data chan<- isync.DataSync) error { return nil }
func (m *mockSync) Sync(ctx context.Context, data chan<- isync.DataSync) error {
	m.dataChan <- data
	<-ctx.Done()
	return nil
}
func (m *mockSync) Events() chan process.SyncEvent {
	return m.events
}

func TestInProcessServiceDataRace(t *testing.T) {
	m := &mockSync{
		events:   make(chan process.SyncEvent, 100),
		dataChan: make(chan chan<- isync.DataSync, 1),
	}
	service := process.NewInProcessService(process.Configuration{
		CustomSyncProvider:    m,
		CustomSyncProviderUri: "test-source",
	})

	// Start Init in a goroutine because it blocks until first DataSync
	go service.Init()
	defer service.Shutdown()

	// Wait for data channel to be passed to Sync
	var dataChan chan<- isync.DataSync
	select {
	case dataChan = <-m.dataChan:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for Sync to be called")
	}

	done := make(chan struct{})
	go func() {
		for {
			select {
			case <-service.EventChannel():
			case <-done:
				return
			}
		}
	}()
	defer close(done)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			m.events <- process.SyncEvent{Event: of.ProviderError}
			time.Sleep(time.Millisecond)
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			dataChan <- isync.DataSync{FlagData: "{\"flags\":{}}", Source: "test-source"}
			time.Sleep(time.Millisecond)
		}
	}()

	wg.Wait()
}
