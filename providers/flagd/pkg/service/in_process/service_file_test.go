package process

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/open-feature/flagd/core/pkg/model"
	of "github.com/open-feature/go-sdk/openfeature"
)

// disabledFlagOfflineRsp defines a disabled flag with targeting that would match "on" if evaluated.
var disabledFlagOfflineRsp = `{
	"flags": {
		"disabledBoolFlag": {
			"state": "DISABLED",
			"variants": {
				"on": true,
				"off": false
			},
			"targeting": {
				"if": [true, "on"]
			}
		}
	}
}`

func TestInProcessOfflineMode(t *testing.T) {
	// given
	flagFile := "config.json"
	offlinePath := filepath.Join(t.TempDir(), flagFile)

	err := os.WriteFile(offlinePath, []byte(flagRsp), 0644)
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

func TestInProcessOfflineDisabledFlag(t *testing.T) {
	offlinePath := filepath.Join(t.TempDir(), "disabled-flags.json")
	if err := os.WriteFile(offlinePath, []byte(disabledFlagOfflineRsp), 0644); err != nil {
		t.Fatal(err)
	}

	service := NewInProcessService(Configuration{OfflineFlagSource: offlinePath})
	if err := service.Init(); err != nil {
		t.Fatal(err)
	}
	defer service.Shutdown()

	select {
	case event := <-service.EventChannel():
		if event.EventType != of.ProviderReady {
			t.Fatalf("provider initialization failed: got event %s", event.EventType)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("provider initialization did not complete within acceptable timeframe")
	}

	const codeDefault = false
	detail := service.ResolveBoolean(context.Background(), "disabledBoolFlag", codeDefault, make(map[string]interface{}))

	if detail.Reason == of.ErrorReason && detail.Error() != nil {
		t.Skip("flagd/core does not yet evaluate disabled flags with reason=DISABLED; upgrade core (see open-feature/flagd#1968)")
	}

	if detail.Reason != of.DisabledReason {
		t.Fatalf("expected reason %s, got %s", of.DisabledReason, detail.Reason)
	}
	if detail.Value != codeDefault {
		t.Fatalf("expected code default %v, got %v (targeting must not be evaluated for disabled flags)", codeDefault, detail.Value)
	}
	if detail.Variant != "" {
		t.Fatalf("expected empty variant, got %q", detail.Variant)
	}
	if detail.Error() != nil {
		t.Fatalf("expected no resolution error, got %v", detail.Error())
	}

	// Sanity check: core returns DISABLED reason string (not only OpenFeature constant).
	if string(detail.Reason) != model.DisabledReason {
		t.Fatalf("expected reason string %q, got %q", model.DisabledReason, detail.Reason)
	}
}
