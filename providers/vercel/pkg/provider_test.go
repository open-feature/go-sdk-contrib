package vercel

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/open-feature/go-sdk/openfeature"
)

func TestBooleanEvaluationResolvesPausedFlagAsStatic(t *testing.T) {
	t.Setenv("FLAGS", "")
	provider := newTestProvider(t)

	result := provider.BooleanEvaluation(t.Context(), "boolean-flag", false, openfeature.FlattenedContext{})

	if result.Value != true {
		t.Fatalf("expected true, got %v", result.Value)
	}
	if result.Reason != openfeature.StaticReason {
		t.Fatalf("expected reason %s, got %s", openfeature.StaticReason, result.Reason)
	}
}

func TestStringEvaluationResolvesFallthroughAsDefault(t *testing.T) {
	t.Setenv("FLAGS", "")
	provider := newTestProvider(t)

	result := provider.StringEvaluation(t.Context(), "active-flag", "default", openfeature.FlattenedContext{})

	if result.Value != "variant-b" {
		t.Fatalf("expected variant-b, got %q", result.Value)
	}
	if result.Reason != openfeature.DefaultReason {
		t.Fatalf("expected reason %s, got %s", openfeature.DefaultReason, result.Reason)
	}
}

func TestEvaluationUsesVercelEntityContext(t *testing.T) {
	t.Setenv("FLAGS", "")
	provider := newTestProvider(t)

	result := provider.StringEvaluation(t.Context(), "context-flag", "default", openfeature.FlattenedContext{
		"user": map[string]any{
			"id": "user-123",
		},
	})

	if result.Value != "variant-b" {
		t.Fatalf("expected variant-b, got %q", result.Value)
	}
	if result.Reason != openfeature.TargetingMatchReason {
		t.Fatalf("expected reason %s, got %s", openfeature.TargetingMatchReason, result.Reason)
	}
}

func TestMissingFlagReturnsDefaultWithFlagNotFoundError(t *testing.T) {
	t.Setenv("FLAGS", "")
	provider := newTestProvider(t)

	result := provider.BooleanEvaluation(t.Context(), "missing-flag", true, openfeature.FlattenedContext{})

	if result.Value != true {
		t.Fatalf("expected default true, got %v", result.Value)
	}
	if result.Reason != openfeature.ErrorReason {
		t.Fatalf("expected reason %s, got %s", openfeature.ErrorReason, result.Reason)
	}
	if got := result.ProviderResolutionDetail.ResolutionDetail().ErrorCode; got != openfeature.FlagNotFoundCode {
		t.Fatalf("expected error code %s, got %s", openfeature.FlagNotFoundCode, got)
	}
}

func TestTypeMismatchReturnsDefault(t *testing.T) {
	t.Setenv("FLAGS", "")
	provider := newTestProvider(t)

	result := provider.StringEvaluation(t.Context(), "boolean-flag", "default", openfeature.FlattenedContext{})

	if result.Value != "default" {
		t.Fatalf("expected default, got %q", result.Value)
	}
	if result.Reason != openfeature.ErrorReason {
		t.Fatalf("expected reason %s, got %s", openfeature.ErrorReason, result.Reason)
	}
	if got := result.ProviderResolutionDetail.ResolutionDetail().ErrorCode; got != openfeature.TypeMismatchCode {
		t.Fatalf("expected error code %s, got %s", openfeature.TypeMismatchCode, got)
	}
}

func TestProviderFetchesDatafile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/datafile" {
			t.Fatalf("expected /v1/datafile path, got %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer vf_server_test_key" {
			t.Fatalf("expected bearer token, got %q", got)
		}
		if err := json.NewEncoder(w).Encode(testDatafile()); err != nil {
			t.Fatal(err)
		}
	}))
	defer server.Close()

	provider, err := NewProvider(
		WithSDKKey("vf_server_test_key"),
		WithHost(server.URL),
		WithPollingDisabled(),
	)
	if err != nil {
		t.Fatal(err)
	}

	if err := provider.Init(openfeature.EvaluationContext{}); err != nil {
		t.Fatal(err)
	}

	result := provider.StringEvaluation(t.Context(), "active-flag", "default", openfeature.FlattenedContext{})
	if result.Value != "variant-b" {
		t.Fatalf("expected fetched variant, got %q", result.Value)
	}
	if provider.Status() != openfeature.ReadyState {
		t.Fatalf("expected ready state, got %s", provider.Status())
	}
}

func TestNewProviderAcceptsConnectionString(t *testing.T) {
	provider, err := NewProvider(
		WithConnectionString("flags:edgeConfigId=test&edgeConfigToken=test&sdkKey=vf_server_test_key"),
		WithDatafile(testDatafile()),
		WithPollingDisabled(),
	)
	if err != nil {
		t.Fatal(err)
	}

	if provider.sdkKey != "vf_server_test_key" {
		t.Fatalf("expected parsed sdk key, got %q", provider.sdkKey)
	}
}

func TestNewProviderRequiresSDKKeyWithoutDatafile(t *testing.T) {
	t.Setenv("FLAGS", "")

	_, err := NewProvider()
	if err == nil {
		t.Fatal("expected error")
	}
}

func newTestProvider(t *testing.T) *Provider {
	t.Helper()

	provider, err := NewProvider(WithDatafile(testDatafile()), WithPollingDisabled())
	if err != nil {
		t.Fatal(err)
	}
	return provider
}

func testDatafile() Datafile {
	return Datafile{
		ProjectID:   "test",
		Environment: "production",
		Definitions: map[string]FlagDefinition{
			"boolean-flag": {
				Environments: map[string]any{"production": 0},
				Variants:     []any{true},
			},
			"active-flag": {
				Environments: map[string]any{
					"production": map[string]any{
						"fallthrough": 1,
					},
				},
				Variants: []any{"variant-a", "variant-b"},
			},
			"context-flag": {
				Environments: map[string]any{
					"production": map[string]any{
						"targets": []any{
							map[string]any{},
							map[string]any{
								"user": map[string]any{
									"id": []any{"user-123"},
								},
							},
						},
						"fallthrough": 0,
					},
				},
				Variants: []any{"variant-a", "variant-b"},
			},
		},
		Segments: map[string]Segment{},
	}
}
