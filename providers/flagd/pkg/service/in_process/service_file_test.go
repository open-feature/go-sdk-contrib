package process

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	of "go.openfeature.dev/openfeature/v2"
)

func TestInProcessOfflineMode(t *testing.T) {
	// given
	flagFile := "config.json"
	offlinePath := filepath.Join(t.TempDir(), flagFile)

	err := os.WriteFile(offlinePath, []byte(flagRsp), 0o644)
	if err != nil {
		t.Fatal(err)
	}

	// when
	service := NewInProcessService(Configuration{OfflineFlagSource: offlinePath})

	err = service.Init()
	if err != nil {
		t.Fatal(err)
	}

	// then
	channel := service.EventChannel()

	select {
	case event := <-channel:
		if event.EventType != of.ProviderReady {
			t.Fatalf("Provider initialization failed. Got event type %s with message %s", event.EventType, event.Message)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Provider initialization did not complete within acceptable timeframe ")
	}

	// provider must evaluate flag from the grpc source data
	detail := service.ResolveBoolean(context.Background(), "myBoolFlag", false, make(map[string]interface{}))

	if !detail.Value {
		t.Fatal("Expected true, but got false")
	}

	// check for metadata - scope from grpc sync
	if len(detail.FlagMetadata) == 0 && detail.FlagMetadata["scope"] == "" {
		t.Fatal("Expected scope to be present, but got none")
	}
}
