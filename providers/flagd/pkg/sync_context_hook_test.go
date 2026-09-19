package flagd

import (
	"context"
	"reflect"
	"testing"

	of "github.com/open-feature/go-sdk/openfeature"
)

func TestSyncContextHook_Before(t *testing.T) {
	tests := []struct {
		name          string
		enriched      *of.EvaluationContext
		incoming      of.EvaluationContext
		expectNil     bool
		expectedAttrs map[string]any
		expectedKey   string
	}{
		{
			name:      "no sync context yet returns nil, leaving the context untouched",
			enriched:  nil,
			incoming:  of.NewTargetlessEvaluationContext(map[string]any{"from": "invocation"}),
			expectNil: true,
		},
		{
			name:          "sync context is added to the incoming context",
			enriched:      ptr(of.NewTargetlessEvaluationContext(map[string]any{"scope": "dev"})),
			incoming:      of.NewTargetlessEvaluationContext(map[string]any{"from": "invocation"}),
			expectedAttrs: map[string]any{"scope": "dev", "from": "invocation"},
		},
		{
			name:          "context accumulated by earlier hooks is preserved",
			enriched:      ptr(of.NewTargetlessEvaluationContext(map[string]any{"scope": "dev"})),
			incoming:      of.NewTargetlessEvaluationContext(map[string]any{"fromEarlierHook": "yes"}),
			expectedAttrs: map[string]any{"scope": "dev", "fromEarlierHook": "yes"},
		},
		{
			name:          "sync context takes precedence on conflicting keys",
			enriched:      ptr(of.NewTargetlessEvaluationContext(map[string]any{"scope": "sync"})),
			incoming:      of.NewTargetlessEvaluationContext(map[string]any{"scope": "invocation"}),
			expectedAttrs: map[string]any{"scope": "sync"},
		},
		{
			name:          "targeting key of the incoming context survives a targetless sync context",
			enriched:      ptr(of.NewTargetlessEvaluationContext(map[string]any{"scope": "dev"})),
			incoming:      of.NewEvaluationContext("user-1", map[string]any{}),
			expectedAttrs: map[string]any{"scope": "dev"},
			expectedKey:   "user-1",
		},
		{
			name:          "targeting key from the enricher wins",
			enriched:      ptr(of.NewEvaluationContext("sync-key", map[string]any{})),
			incoming:      of.NewEvaluationContext("user-1", map[string]any{}),
			expectedAttrs: map[string]any{},
			expectedKey:   "sync-key",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			hook := NewSyncContextHook(func() *of.EvaluationContext { return test.enriched })

			result, err := hook.Before(context.Background(), newHookContext(test.incoming), of.HookHints{})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if test.expectNil {
				if result != nil {
					t.Fatalf("expected nil evaluation context, got %v", result)
				}
				return
			}

			if result == nil {
				t.Fatal("expected an evaluation context, got nil")
			}
			if !reflect.DeepEqual(test.expectedAttrs, result.Attributes()) {
				t.Errorf("expected attributes %v, got %v", test.expectedAttrs, result.Attributes())
			}
			if test.expectedKey != result.TargetingKey() {
				t.Errorf("expected targeting key %q, got %q", test.expectedKey, result.TargetingKey())
			}
		})
	}
}

// TestProviderRegistersSyncContextHook guards the wiring between the provider and the hook.
func TestProviderRegistersSyncContextHook(t *testing.T) {
	for _, opts := range [][]ProviderOption{
		{WithInProcessResolver()},
		{WithRPCResolver()},
	} {
		provider, err := NewProvider(opts...)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		hooks := provider.Hooks()
		if len(hooks) != 1 {
			t.Fatalf("expected exactly one provider hook, got %d", len(hooks))
		}
		if _, ok := hooks[0].(SyncContextHook); !ok {
			t.Fatalf("expected a SyncContextHook, got %T", hooks[0])
		}

		// Without a sync payload the hook must not alter the evaluation context.
		result, err := hooks[0].Before(context.Background(),
			newHookContext(of.NewTargetlessEvaluationContext(map[string]any{})), of.HookHints{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != nil {
			t.Fatalf("expected nil evaluation context before the first sync, got %v", result)
		}
	}
}

func ptr(ctx of.EvaluationContext) *of.EvaluationContext {
	return &ctx
}

func newHookContext(evalCtx of.EvaluationContext) of.HookContext {
	return of.NewHookContext("flag", of.Boolean, false, of.ClientMetadata{}, of.Metadata{}, evalCtx)
}
