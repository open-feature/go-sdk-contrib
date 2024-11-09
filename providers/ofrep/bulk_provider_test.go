package ofrep_test

import (
	"context"
	"encoding/json"
	"io"
	"maps"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/open-feature/go-sdk-contrib/providers/ofrep"
	of "github.com/open-feature/go-sdk/openfeature"
)

var evalCtx = of.NewEvaluationContext("keyboard", map[string]any{
	"color": "red",
})

func TestBulkProviderEvaluationE2EBasic(t *testing.T) {
	of.SetEvaluationContext(evalCtx)
	baseUrl := setupTestServer(t)
	p := ofrep.NewBulkProvider(baseUrl, ofrep.WithBearerToken("api-key"))

	err := of.SetProviderAndWait(p)
	if err != nil {
		t.Errorf("expected ready provider, but got %v", err)
	}

	client := of.NewClient("app")
	ctx := context.Background()

	result := client.Boolean(ctx, "flag-bool", false, evalCtx)
	if !result {
		t.Errorf("expected %v, but got %v", true, result)
	}

	_, err = client.BooleanValueDetails(ctx, "flag-error", false, evalCtx)

	if err == nil {
		t.Errorf("expected error, but got nil")
	}

	if err.Error() != "error code: GENERAL: something wrong" {
		t.Errorf("expected error message '%v', but got '%v'", "error code: GENERAL: something wrong", err.Error())
	}

	of.Shutdown()

	if p.Status() != of.NotReadyState {
		t.Errorf("expected %v, but got %v", of.NotReadyState, p.Status())
	}
}

func TestBulkProviderEvaluationE2EPolling(t *testing.T) {
	of.SetEvaluationContext(evalCtx)
	baseUrl := setupTestServer(t)
	p := ofrep.NewBulkProvider(baseUrl, ofrep.WithBearerToken("api-key"), ofrep.WithPollingInterval(30*time.Millisecond))

	err := of.SetProviderAndWait(p)
	if err != nil {
		t.Errorf("expected ready provider, but got %v", err)
	}
	if p.Status() != of.ReadyState {
		t.Errorf("expected %v, but got %v", of.ReadyState, p.Status())
	}

	// let the provider poll for flags in background at least once
	time.Sleep(60 * time.Millisecond)

	of.Shutdown()
	if p.Status() != of.NotReadyState {
		t.Errorf("expected %v, but got %v", of.NotReadyState, p.Status())
	}
}

func setupTestServer(t testing.TB) string {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/ofrep/v1/evaluate/flags", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected post request, got: %v", r.Method)
		}

		if r.Header.Get("Authorization") != "Bearer api-key" {
			t.Errorf("expected Authorization header, got: %v", r.Header.Get("Authorization"))
		}

		requestData, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("error reading request data: %v", err)
		}

		evalData := struct {
			Context map[string]any `json:"context"`
		}{}

		err = json.Unmarshal(requestData, &evalData)
		if err != nil {
			t.Errorf("error parsing request data: %v", err)
		}

		flatCtx := ofrep.FlattenContext(evalCtx)
		if !maps.Equal(flatCtx, evalData.Context) {
			t.Errorf("expected request data with %v, but got %v", flatCtx, evalData.Context)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		data := `{"flags":[
      {"key":"flag-bool","reason":"DEFAULT","variant":"true","metadata":{},"value":true}, 
      {"key":"flag-error", "errorCode": "INVALID", "errorDetails": "something wrong" }
    ]}`
		_, err = w.Write([]byte(data))
		if err != nil {
			t.Errorf("error writing response: %v", err)
		}
	})

	s := httptest.NewServer(mux)
	t.Cleanup(s.Close)
	return s.URL
}
