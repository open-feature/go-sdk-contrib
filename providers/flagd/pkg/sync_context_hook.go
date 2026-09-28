package flagd

import (
	"context"

	of "github.com/open-feature/go-sdk/openfeature"
)

// ContextEnricher builds an EvaluationContext from the flagd sync-context; nil disables enrichment.
type ContextEnricher func(map[string]any) *of.EvaluationContext

// SyncContextHook mixes the enriched sync-context into each evaluation (mirrors Java SyncMetadataHook).
type SyncContextHook struct {
	of.UnimplementedHook
	contextEnricher func() *of.EvaluationContext
}

// NewSyncContextHook returns a hook reading the current enriched context via accessor (may return nil).
func NewSyncContextHook(contextEnricher func() *of.EvaluationContext) SyncContextHook {
	return SyncContextHook{contextEnricher: contextEnricher}
}

// Before merges sync-context over earlier hook context; merge is manual as go-sdk <=v1.18.0 replaces before-hook results (see open-feature/go-sdk#569).
func (hook SyncContextHook) Before(
	_ context.Context, hookContext of.HookContext, _ of.HookHints,
) (*of.EvaluationContext, error) {
	enriched := hook.contextEnricher()
	if enriched == nil {
		return nil, nil
	}

	merged := mergeEvaluationContexts(*enriched, hookContext.EvaluationContext())

	return &merged, nil
}

// mergeEvaluationContexts merges the given contexts, earlier ones taking precedence.
func mergeEvaluationContexts(contexts ...of.EvaluationContext) of.EvaluationContext {
	targetingKey := ""
	attributes := map[string]any{}

	for _, evalCtx := range contexts {
		if targetingKey == "" {
			targetingKey = evalCtx.TargetingKey()
		}

		for key, value := range evalCtx.Attributes() {
			if _, ok := attributes[key]; !ok {
				attributes[key] = value
			}
		}
	}

	return of.NewEvaluationContext(targetingKey, attributes)
}
