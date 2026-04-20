package evaluator

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/open-feature/go-sdk/openfeature"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ofrepFlagResponse returns a minimal valid OFREP flag evaluation response.
func ofrepFlagResponse(value any) []byte {
	b, _ := json.Marshal(map[string]any{
		"key":      "test-flag",
		"reason":   "STATIC",
		"variant":  "default",
		"value":    value,
		"metadata": map[string]any{},
	})
	return b
}

// ofrepFlagResponseCacheable returns an OFREP response with gofeatureflag_cacheable=true.
func ofrepFlagResponseCacheable(value any) []byte {
	b, _ := json.Marshal(map[string]any{
		"key":     "test-flag",
		"reason":  "TARGETING_MATCH",
		"variant": "default",
		"value":   value,
		"metadata": map[string]any{
			"gofeatureflag_cacheable": true,
		},
	})
	return b
}

func TestNewRemoteEvaluator_WithAPIKey(t *testing.T) {
	var capturedHeader string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedHeader = r.Header.Get("X-API-Key")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(ofrepFlagResponse(true))
	}))
	defer srv.Close()

	evaluator := NewRemoteEvaluator(srv.URL, nil, "my-secret-key", nil, 0, 0, true, 0, nil)
	result := evaluator.BooleanEvaluation(context.Background(), "test-flag", false, openfeature.FlattenedContext{})

	require.NotEqual(t, openfeature.ErrorReason, result.Reason)
	assert.Equal(t, "my-secret-key", capturedHeader)
}

func TestNewRemoteEvaluator_WithCustomHeaders(t *testing.T) {
	capturedHeaders := http.Header{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedHeaders = r.Header.Clone()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(ofrepFlagResponse("hello"))
	}))
	defer srv.Close()

	headers := map[string]string{
		"X-Custom-Header": "custom-value",
		"X-Tenant-ID":     "tenant-123",
	}
	evaluator := NewRemoteEvaluator(srv.URL, nil, "", headers, 0, 0, true, 0, nil)
	result := evaluator.StringEvaluation(context.Background(), "test-flag", "default", openfeature.FlattenedContext{})

	require.NotEqual(t, openfeature.ErrorReason, result.Reason)
	assert.Equal(t, "custom-value", capturedHeaders.Get("X-Custom-Header"))
	assert.Equal(t, "tenant-123", capturedHeaders.Get("X-Tenant-ID"))
}

func TestNewRemoteEvaluator_WithoutOptionalParams(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(ofrepFlagResponse(true))
	}))
	defer srv.Close()

	evaluator := NewRemoteEvaluator(srv.URL, nil, "", nil, 0, 0, true, 0, nil)
	require.NotNil(t, evaluator)

	result := evaluator.BooleanEvaluation(context.Background(), "test-flag", false, openfeature.FlattenedContext{})
	assert.NotEqual(t, openfeature.ErrorReason, result.Reason)
}

func TestRemote_InitShutdown(t *testing.T) {
	evaluator := NewRemoteEvaluator("http://localhost", nil, "", nil, 0, 0, true, 0, nil)
	require.NotNil(t, evaluator)

	assert.NoError(t, evaluator.Init(context.Background()))
	assert.NoError(t, evaluator.Shutdown(context.Background()))
}

// ---------------------------------------------------------------------------
// Cache tests
// ---------------------------------------------------------------------------

// newCachingServer creates a test server that counts how many times it is called
// and responds with an OFREP response for the given value.
// When cacheable is true the response includes gofeatureflag_cacheable=true.
func newCachingServer(t *testing.T, value any, cacheable bool, callCount *atomic.Int32) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if cacheable {
			_, _ = w.Write(ofrepFlagResponseCacheable(value))
		} else {
			_, _ = w.Write(ofrepFlagResponse(value))
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestRemote_Cache_HitOnSecondCall(t *testing.T) {
	var calls atomic.Int32
	srv := newCachingServer(t, true, true, &calls)

	e := NewRemoteEvaluator(srv.URL, nil, "", nil, 100, 5*time.Minute, false, 24*time.Hour, nil)
	ctx := context.Background()
	flatCtx := openfeature.FlattenedContext{"targetingKey": "user-1"}

	r1 := e.BooleanEvaluation(ctx, "test-flag", false, flatCtx)
	assert.NotEqual(t, openfeature.CachedReason, r1.Reason)

	r2 := e.BooleanEvaluation(ctx, "test-flag", false, flatCtx)
	assert.Equal(t, openfeature.CachedReason, r2.Reason)

	assert.Equal(t, int32(1), calls.Load(), "server should be called only once")
}

func TestRemote_Cache_MissForDifferentContexts(t *testing.T) {
	var calls atomic.Int32
	srv := newCachingServer(t, true, true, &calls)

	e := NewRemoteEvaluator(srv.URL, nil, "", nil, 100, 5*time.Minute, false, 24*time.Hour, nil)
	ctx := context.Background()

	for i := range 3 {
		flatCtx := openfeature.FlattenedContext{"targetingKey": string(rune('a' + i))}
		r := e.BooleanEvaluation(ctx, "test-flag", false, flatCtx)
		assert.NotEqual(t, openfeature.CachedReason, r.Reason)
	}

	assert.Equal(t, int32(3), calls.Load(), "each unique context should call the server")
}

func TestRemote_Cache_NonCacheableFlag_NeverCached(t *testing.T) {
	var calls atomic.Int32
	srv := newCachingServer(t, true, false, &calls) // cacheable=false

	e := NewRemoteEvaluator(srv.URL, nil, "", nil, 100, 5*time.Minute, false, 24*time.Hour, nil)
	ctx := context.Background()
	flatCtx := openfeature.FlattenedContext{"targetingKey": "user-1"}

	r1 := e.BooleanEvaluation(ctx, "test-flag", false, flatCtx)
	assert.NotEqual(t, openfeature.CachedReason, r1.Reason)

	r2 := e.BooleanEvaluation(ctx, "test-flag", false, flatCtx)
	assert.NotEqual(t, openfeature.CachedReason, r2.Reason)

	assert.Equal(t, int32(2), calls.Load(), "non-cacheable flag must always call the server")
}

func TestRemote_Cache_Disabled_AlwaysCallsServer(t *testing.T) {
	var calls atomic.Int32
	srv := newCachingServer(t, true, true, &calls)

	e := NewRemoteEvaluator(srv.URL, nil, "", nil, 100, 5*time.Minute, true, 24*time.Hour, nil) // disableCache=true
	ctx := context.Background()
	flatCtx := openfeature.FlattenedContext{"targetingKey": "user-1"}

	e.BooleanEvaluation(ctx, "test-flag", false, flatCtx)
	e.BooleanEvaluation(ctx, "test-flag", false, flatCtx)

	assert.Equal(t, int32(2), calls.Load(), "disabled cache must always call the server")
}

func TestRemote_Cache_TTLExpiry_RefetchesAfterExpiry(t *testing.T) {
	var calls atomic.Int32
	srv := newCachingServer(t, true, true, &calls)

	e := NewRemoteEvaluator(srv.URL, nil, "", nil, 100, 200*time.Millisecond, false, 24*time.Hour, nil)
	ctx := context.Background()
	flatCtx := openfeature.FlattenedContext{"targetingKey": "user-1"}

	e.BooleanEvaluation(ctx, "test-flag", false, flatCtx)
	assert.Equal(t, int32(1), calls.Load())

	// Second call within TTL → cache hit
	r := e.BooleanEvaluation(ctx, "test-flag", false, flatCtx)
	assert.Equal(t, openfeature.CachedReason, r.Reason)
	assert.Equal(t, int32(1), calls.Load())

	time.Sleep(300 * time.Millisecond) // wait for TTL to expire

	r = e.BooleanEvaluation(ctx, "test-flag", false, flatCtx)
	assert.NotEqual(t, openfeature.CachedReason, r.Reason)
	assert.Equal(t, int32(2), calls.Load(), "expired entry must trigger a server call")
}

func TestRemote_Cache_AllTypes(t *testing.T) {
	t.Run("string", func(t *testing.T) {
		var calls atomic.Int32
		srv := newCachingServer(t, "hello", true, &calls)
		e := NewRemoteEvaluator(srv.URL, nil, "", nil, 100, 5*time.Minute, false, 24*time.Hour, nil)
		ctx := context.Background()
		flatCtx := openfeature.FlattenedContext{"targetingKey": "user-1"}

		e.StringEvaluation(ctx, "test-flag", "", flatCtx)
		r := e.StringEvaluation(ctx, "test-flag", "", flatCtx)
		assert.Equal(t, openfeature.CachedReason, r.Reason)
		assert.Equal(t, int32(1), calls.Load())
	})

	t.Run("float", func(t *testing.T) {
		var calls atomic.Int32
		srv := newCachingServer(t, 3.14, true, &calls)
		e := NewRemoteEvaluator(srv.URL, nil, "", nil, 100, 5*time.Minute, false, 24*time.Hour, nil)
		ctx := context.Background()
		flatCtx := openfeature.FlattenedContext{"targetingKey": "user-1"}

		e.FloatEvaluation(ctx, "test-flag", 0, flatCtx)
		r := e.FloatEvaluation(ctx, "test-flag", 0, flatCtx)
		assert.Equal(t, openfeature.CachedReason, r.Reason)
		assert.Equal(t, int32(1), calls.Load())
	})

	t.Run("int", func(t *testing.T) {
		var calls atomic.Int32
		srv := newCachingServer(t, 42, true, &calls)
		e := NewRemoteEvaluator(srv.URL, nil, "", nil, 100, 5*time.Minute, false, 24*time.Hour, nil)
		ctx := context.Background()
		flatCtx := openfeature.FlattenedContext{"targetingKey": "user-1"}

		e.IntEvaluation(ctx, "test-flag", 0, flatCtx)
		r := e.IntEvaluation(ctx, "test-flag", 0, flatCtx)
		assert.Equal(t, openfeature.CachedReason, r.Reason)
		assert.Equal(t, int32(1), calls.Load())
	})

	t.Run("object", func(t *testing.T) {
		var calls atomic.Int32
		srv := newCachingServer(t, map[string]any{"k": "v"}, true, &calls)
		e := NewRemoteEvaluator(srv.URL, nil, "", nil, 100, 5*time.Minute, false, 24*time.Hour, nil)
		ctx := context.Background()
		flatCtx := openfeature.FlattenedContext{"targetingKey": "user-1"}

		e.ObjectEvaluation(ctx, "test-flag", nil, flatCtx)
		r := e.ObjectEvaluation(ctx, "test-flag", nil, flatCtx)
		assert.Equal(t, openfeature.CachedReason, r.Reason)
		assert.Equal(t, int32(1), calls.Load())
	})
}
