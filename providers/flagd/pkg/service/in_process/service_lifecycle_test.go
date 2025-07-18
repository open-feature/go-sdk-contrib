package process

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	of "github.com/open-feature/go-sdk/openfeature"
)

func TestInProcessServiceShutdownCleansUpGoroutines(t *testing.T) {
	checkGoroutineLeaks(t)

	flagFile := "config.json"
	offlinePath := filepath.Join(t.TempDir(), flagFile)
	if err := os.WriteFile(offlinePath, []byte(flagRsp), 0644); err != nil {
		t.Fatal(err)
	}
	service := NewInProcessService(Configuration{OfflineFlagSource: offlinePath})
	if err := service.Init(); err != nil {
		t.Fatal(err)
	}

	// Wait for provider to be ready.
	channel := service.EventChannel()
	select {
	case event := <-channel:
		if event.EventType != of.ProviderReady {
			t.Fatalf("Provider initialization failed. Got event type %s with message %s", event.EventType, event.Message)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Provider initialization did not complete within acceptable timeframe")
	}

	// Service is ready - now shut down.
	service.Shutdown()
}

func TestInProcessServiceImmediateShutdownCleansUpGoroutines(t *testing.T) {
	checkGoroutineLeaks(t)

	flagFile := "config.json"
	offlinePath := filepath.Join(t.TempDir(), flagFile)
	if err := os.WriteFile(offlinePath, []byte(flagRsp), 0644); err != nil {
		t.Fatal(err)
	}
	service := NewInProcessService(Configuration{OfflineFlagSource: offlinePath})
	if err := service.Init(); err != nil {
		t.Fatal(err)
	}

	// Immediately shut down - don't wait for the provider to be ready.
	service.Shutdown()
}

// At the end of the test, if no other failures have occurred, check for
// goroutine leaks, and fail the test if any were found.
func checkGoroutineLeaks(t *testing.T) {
	startingGoroutineCount := runtime.NumGoroutine()
	t.Cleanup(func() {
		if t.Failed() {
			return
		}
		buf := make([]byte, 1<<20)
		stacklen := runtime.Stack(buf, true)
		if numGoroutinesAfter := runtime.NumGoroutine(); numGoroutinesAfter > startingGoroutineCount {
			t.Errorf("Goroutines leaked: %d goroutines before, %d goroutines after", startingGoroutineCount, numGoroutinesAfter)
			fmt.Fprintf(os.Stderr, "%s\n", buf[:stacklen])
		}
	})
}
