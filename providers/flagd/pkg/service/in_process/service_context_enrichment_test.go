package process

import (
	"testing"
	"time"

	isync "github.com/open-feature/flagd/core/pkg/sync"
	of "github.com/open-feature/go-sdk/openfeature"
	"google.golang.org/protobuf/types/known/structpb"
)

const enrichmentFlags = `{"flags":{}}`

func syncContextOf(t *testing.T, values map[string]any) *structpb.Struct {
	t.Helper()

	syncContext, err := structpb.NewStruct(values)
	if err != nil {
		t.Fatalf("could not build sync context: %v", err)
	}

	return syncContext
}

// startEnrichmentService boots an in-process service against a mock sync provider, returns its payload channel
func startEnrichmentService(t *testing.T, cfg Configuration) (*InProcess, chan<- isync.DataSync) {
	t.Helper()

	mock := &mockSync{
		events:   make(chan SyncEvent, 100),
		dataChan: make(chan chan<- isync.DataSync, 1),
	}
	cfg.CustomSyncProvider = mock
	cfg.CustomSyncProviderUri = "test-source"

	service := NewInProcessService(cfg)

	errChan := make(chan error, 1)
	go func() { errChan <- service.Init() }()
	t.Cleanup(service.Shutdown)

	var dataChan chan<- isync.DataSync
	select {
	case dataChan = <-mock.dataChan:
	case err := <-errChan:
		t.Fatalf("Init failed: %v", err)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for Sync to be called")
	}

	// Drain events so the service never blocks on its event channel.
	done := make(chan struct{})
	t.Cleanup(func() { close(done) })
	go func() {
		for {
			select {
			case <-service.EventChannel():
			case <-done:
				return
			}
		}
	}()

	return service, dataChan
}

// awaitContextValues polls until the enriched context satisfies want (payloads process asynchronously)
func awaitContextValues(t *testing.T, service *InProcess, want func(*of.EvaluationContext) bool) {
	t.Helper()

	deadline := time.After(2 * time.Second)
	for {
		if want(service.ContextValues()) {
			return
		}
		select {
		case <-deadline:
			t.Fatalf("timed out; enriched context was %v", service.ContextValues())
		case <-time.After(5 * time.Millisecond):
		}
	}
}

func hasScope(scope string) func(*of.EvaluationContext) bool {
	return func(evalCtx *of.EvaluationContext) bool {
		return evalCtx != nil && evalCtx.Attributes()["scope"] == scope
	}
}

func enrichmentConfiguration() Configuration {
	return Configuration{
		ContextEnricher: func(values map[string]any) *of.EvaluationContext {
			evaluationContext := of.NewTargetlessEvaluationContext(values)
			return &evaluationContext
		},
	}
}

// TestContextEnrichmentIsRefreshedPerPayload covers flagd's sync context: one value per stream, repeated per payload, empty but never nil
func TestContextEnrichmentIsRefreshedPerPayload(t *testing.T) {
	service, dataChan := startEnrichmentService(t, enrichmentConfiguration())

	dataChan <- isync.DataSync{
		FlagData:    enrichmentFlags,
		Source:      "test-source",
		SyncContext: syncContextOf(t, map[string]any{"scope": "dev"}),
	}
	awaitContextValues(t, service, hasScope("dev"))

	// A reconnect to a flagd with different context values replaces the enrichment.
	dataChan <- isync.DataSync{
		FlagData:    enrichmentFlags,
		Source:      "test-source",
		SyncContext: syncContextOf(t, map[string]any{"scope": "prod"}),
	}
	awaitContextValues(t, service, hasScope("prod"))

	// flagd sends an empty struct (not nil) when no context values are configured; this clears a stale enrichment
	dataChan <- isync.DataSync{
		FlagData:    enrichmentFlags,
		Source:      "test-source",
		SyncContext: syncContextOf(t, map[string]any{}),
	}
	awaitContextValues(t, service, func(evalCtx *of.EvaluationContext) bool {
		return evalCtx != nil && len(evalCtx.Attributes()) == 0
	})
}

// TestContextEnrichmentSurvivesPayloadWithoutSyncContext documents keeping previous enrichment when a payload has no sync context (matches Java)
func TestContextEnrichmentSurvivesPayloadWithoutSyncContext(t *testing.T) {
	service, dataChan := startEnrichmentService(t, enrichmentConfiguration())

	dataChan <- isync.DataSync{
		FlagData:    enrichmentFlags,
		Source:      "test-source",
		SyncContext: syncContextOf(t, map[string]any{"scope": "dev"}),
	}
	awaitContextValues(t, service, hasScope("dev"))

	dataChan <- isync.DataSync{FlagData: enrichmentFlags, Source: "test-source"}

	// Give the payload time to be processed, then confirm the enrichment is still there.
	time.Sleep(100 * time.Millisecond)
	awaitContextValues(t, service, hasScope("dev"))
}

// TestNoContextEnricherLeavesContextUntouched covers rpc and file resolvers that pass no enricher; hook stays a no-op
func TestNoContextEnricherLeavesContextUntouched(t *testing.T) {
	service, dataChan := startEnrichmentService(t, Configuration{})

	dataChan <- isync.DataSync{
		FlagData:    enrichmentFlags,
		Source:      "test-source",
		SyncContext: syncContextOf(t, map[string]any{"scope": "dev"}),
	}

	time.Sleep(100 * time.Millisecond)
	if service.ContextValues() != nil {
		t.Fatalf("expected no enriched context without an enricher, got %v", service.ContextValues())
	}
}
