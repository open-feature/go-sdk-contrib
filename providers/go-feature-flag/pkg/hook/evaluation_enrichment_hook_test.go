package hook_test

import (
	"context"
	"testing"

	"github.com/open-feature/go-sdk-contrib/providers/go-feature-flag/pkg/hook"
	"github.com/open-feature/go-sdk/openfeature"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newHookContext(targetingKey string, attributes map[string]any) openfeature.HookContext {
	return openfeature.NewHookContext(
		"test-flag",
		openfeature.Boolean,
		false,
		openfeature.NewClientMetadata(""),
		openfeature.Metadata{Name: "test-provider"},
		openfeature.NewEvaluationContext(targetingKey, attributes),
	)
}

func Test_NewEvaluationEnrichmentHook(t *testing.T) {
	exporterMetadata := map[string]any{"env": "production", "provider": "go"}
	h := hook.NewEvaluationEnrichmentHook(exporterMetadata)
	require.NotNil(t, h)
}

func Test_EvaluationEnrichmentHook_Before(t *testing.T) {
	ctx := context.Background()

	t.Run("adds gofeatureflag with exporterMetadata when attributes are empty", func(t *testing.T) {
		exporterMetadata := map[string]any{"env": "production"}
		h := hook.NewEvaluationEnrichmentHook(exporterMetadata)
		hookCtx := newHookContext("user-123", map[string]any{})

		newCtx, err := h.Before(ctx, hookCtx, openfeature.HookHints{})
		require.NoError(t, err)
		require.NotNil(t, newCtx)
		assert.Equal(t, "user-123", newCtx.TargetingKey())

		goff := newCtx.Attribute("gofeatureflag")
		require.NotNil(t, goff)
		goffMap, ok := goff.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, exporterMetadata, goffMap["exporterMetadata"])
	})

	t.Run("adds exporterMetadata to existing gofeatureflag map", func(t *testing.T) {
		exporterMetadata := map[string]any{"provider": "go"}
		h := hook.NewEvaluationEnrichmentHook(exporterMetadata)
		hookCtx := newHookContext("user-456", map[string]any{
			"gofeatureflag": map[string]any{"flags": []string{"flag1", "flag2"}},
		})

		newCtx, err := h.Before(ctx, hookCtx, openfeature.HookHints{})
		require.NoError(t, err)
		require.NotNil(t, newCtx)

		goff := newCtx.Attribute("gofeatureflag")
		require.NotNil(t, goff)
		goffMap, ok := goff.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, exporterMetadata, goffMap["exporterMetadata"])
		assert.Equal(t, []string{"flag1", "flag2"}, goffMap["flags"])
	})

	t.Run("replaces non-map gofeatureflag with new map containing exporterMetadata", func(t *testing.T) {
		exporterMetadata := map[string]any{"env": "test"}
		h := hook.NewEvaluationEnrichmentHook(exporterMetadata)
		hookCtx := newHookContext("user-789", map[string]any{
			"gofeatureflag": "invalid-type",
		})

		newCtx, err := h.Before(ctx, hookCtx, openfeature.HookHints{})
		require.NoError(t, err)
		require.NotNil(t, newCtx)

		goff := newCtx.Attribute("gofeatureflag")
		require.NotNil(t, goff)
		goffMap, ok := goff.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, exporterMetadata, goffMap["exporterMetadata"])
	})

	t.Run("preserves other attributes", func(t *testing.T) {
		exporterMetadata := map[string]any{}
		h := hook.NewEvaluationEnrichmentHook(exporterMetadata)
		hookCtx := newHookContext("user-abc", map[string]any{
			"email": "user@example.com",
			"age":   30,
		})

		newCtx, err := h.Before(ctx, hookCtx, openfeature.HookHints{})
		require.NoError(t, err)
		require.NotNil(t, newCtx)
		assert.Equal(t, "user@example.com", newCtx.Attribute("email"))
		assert.Equal(t, 30, newCtx.Attribute("age"))
	})
}
