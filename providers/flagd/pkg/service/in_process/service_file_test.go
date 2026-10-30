package process

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	of "github.com/open-feature/go-sdk/openfeature"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

type recordingTracerProvider struct {
	noop.TracerProvider
	tracer *recordingTracer
}

func (p *recordingTracerProvider) Tracer(_ string, _ ...trace.TracerOption) trace.Tracer {
	return p.tracer
}

type recordingTracer struct {
	trace.Tracer
	started bool
}

func (t *recordingTracer) Start(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	t.started = true
	return t.Tracer.Start(ctx, name, opts...)
}

func TestInProcessUsesTracerProvider(t *testing.T) {
	previousTracerProvider := otel.GetTracerProvider()
	exporter := tracetest.NewInMemoryExporter()
	testTracerProvider := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	otel.SetTracerProvider(testTracerProvider)
	t.Cleanup(func() {
		otel.SetTracerProvider(previousTracerProvider)
		if err := testTracerProvider.Shutdown(context.Background()); err != nil {
			t.Errorf("failed to shut down test tracer provider: %v", err)
		}
	})

	offlinePath := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(offlinePath, []byte(flagRsp), 0644); err != nil {
		t.Fatal(err)
	}

	tracer := &recordingTracer{Tracer: noop.NewTracerProvider().Tracer("test")}
	service := NewInProcessService(Configuration{
		OfflineFlagSource: offlinePath,
		TracerProvider:    &recordingTracerProvider{tracer: tracer},
	})
	if err := service.Init(); err != nil {
		t.Fatal(err)
	}
	defer service.Shutdown()

	exporter.Reset()
	if detail := service.ResolveBoolean(t.Context(), "myBoolFlag", false, map[string]any{}); !detail.Value {
		t.Fatal("expected true from offline flag configuration")
	}
	if !tracer.started {
		t.Fatal("expected the configured tracer to start an evaluation span")
	}
	if spans := exporter.GetSpans(); len(spans) != 0 {
		t.Fatalf("expected no spans from the global tracer provider, got %d", len(spans))
	}
}

func TestInProcessNilTracerProvider(t *testing.T) {
	previousTracerProvider := otel.GetTracerProvider()
	exporter := tracetest.NewInMemoryExporter()
	testTracerProvider := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	otel.SetTracerProvider(testTracerProvider)
	t.Cleanup(func() {
		otel.SetTracerProvider(previousTracerProvider)
		if err := testTracerProvider.Shutdown(context.Background()); err != nil {
			t.Errorf("failed to shut down test tracer provider: %v", err)
		}
	})

	offlinePath := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(offlinePath, []byte(flagRsp), 0644); err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		name     string
		provider trace.TracerProvider
	}{
		{name: "nil interface"},
		{name: "typed nil pointer", provider: (*recordingTracerProvider)(nil)},
	} {
		t.Run(test.name, func(t *testing.T) {
			service := NewInProcessService(Configuration{
				OfflineFlagSource: offlinePath,
				TracerProvider:    test.provider,
			})
			if err := service.Init(); err != nil {
				t.Fatal(err)
			}
			defer service.Shutdown()

			exporter.Reset()
			if detail := service.ResolveBoolean(t.Context(), "myBoolFlag", false, map[string]any{}); !detail.Value {
				t.Fatal("expected true from offline flag configuration")
			}
			if spans := exporter.GetSpans(); len(spans) == 0 {
				t.Fatal("expected an evaluation span from the global tracer provider")
			}
		})
	}
}

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

// TestInProcessOfflineModePolling verifies that the configured OfflinePollMs interval is used to
// watch the offline flag source, so changes to the file are picked up.
func TestInProcessOfflineModePolling(t *testing.T) {
	// given
	offlinePath := filepath.Join(t.TempDir(), "config.json")

	if err := os.WriteFile(offlinePath, []byte(flagRsp), 0644); err != nil {
		t.Fatal(err)
	}

	service := NewInProcessService(Configuration{OfflineFlagSource: offlinePath, OfflinePollMs: 100})

	if err := service.Init(); err != nil {
		t.Fatal(err)
	}
	defer service.Shutdown()

	if detail := service.ResolveBoolean(t.Context(), "myBoolFlag", false, map[string]any{}); !detail.Value {
		t.Fatal("Expected true from the initial flag configuration, but got false")
	}

	// when - the flag configuration on disk changes
	updated := `{
		"flags": {
		  "myBoolFlag": {
			"state": "ENABLED",
			"variants": {
			  "on": true,
			  "off": false
			},
			"defaultVariant": "off"
		  }
		}
	}`
	if err := os.WriteFile(offlinePath, []byte(updated), 0644); err != nil {
		t.Fatal(err)
	}

	// then - the change is detected within a few poll intervals
	deadline := time.Now().Add(2 * time.Second)
	for {
		detail := service.ResolveBoolean(t.Context(), "myBoolFlag", true, map[string]any{})
		if !detail.Value {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("Flag configuration change was not detected within acceptable timeframe")
		}
		time.Sleep(50 * time.Millisecond)
	}
}
