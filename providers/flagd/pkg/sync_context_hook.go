package flagd

import (
	"context"
	"github.com/open-feature/go-sdk/openfeature"
)

type ContextEnricher func(map[string]any) *openfeature.EvaluationContext

type SyncContextHook struct {
	openfeature.UnimplementedHook
	contextEnricher func() *openfeature.EvaluationContext
}

func NewSyncContextHook(contextEnricher func() *openfeature.EvaluationContext) SyncContextHook {
	return SyncContextHook{contextEnricher: contextEnricher}
}

func (hook SyncContextHook) Before(ctx context.Context, hookContext openfeature.HookContext, hookHints openfeature.HookHints) (*openfeature.EvaluationContext, error) {
	return hook.contextEnricher(), nil
}
