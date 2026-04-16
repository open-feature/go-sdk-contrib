package evaluator

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

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
