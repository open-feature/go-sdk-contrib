package flagd

import (
	"context"

	of "github.com/open-feature/go-sdk/openfeature"
)

// ContextEnricher turns the sync-context flagd sends alongside the flag
// configuration into an EvaluationContext that is mixed into every evaluation.
// It is applied once per received sync payload, not once per evaluation.
// Returning nil disables enrichment.
type ContextEnricher func(map[string]any) *of.EvaluationContext

// SyncContextHook mixes the enriched sync-context into each evaluation.
// It mirrors the SyncMetadataHook of the Java reference implementation.
type SyncContextHook struct {
	of.UnimplementedHook
	contextEnricher func() *of.EvaluationContext
}

// NewSyncContextHook returns a hook that pulls the current enriched context from
// the supplied accessor. The accessor may return nil, e.g. before the provider
// received its first sync payload or when the resolver has no sync-context at all.
func NewSyncContextHook(contextEnricher func() *of.EvaluationContext) SyncContextHook {
	return SyncContextHook{contextEnricher: contextEnricher}
}

// Before returns the sync-context merged over the context accumulated by the hooks
// that ran before this one.
//
// Merging is done here deliberately. Up to and including go-sdk v1.18.0 - the version
// this module pins - Client.beforeHooks replaces, rather than merges, the HookContext's
// evaluation context with each before-hook result, so returning the sync-context on its
// own would silently drop whatever earlier hooks contributed. Fixed upstream by
// open-feature/go-sdk#569, which is not in a release yet; once this module bumps past it
// the merge below becomes redundant (the SDK merges with the same precedence) and can go.
// The resulting precedence matches the spec and the Java implementation:
// sync-context > earlier before-hooks > invocation > client > transaction > global.
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
