package flagd

import (
	"context"
	"github.com/open-feature/go-sdk/openfeature"
)

type Supplier func() *openfeature.EvaluationContext

type SyncContextHook struct {
	openfeature.UnimplementedHook
	contextSupplier Supplier
}

func NewSyncContextHook(contextSupplier Supplier) SyncContextHook {
	return SyncContextHook{contextSupplier: contextSupplier}
}

func (hook SyncContextHook) Before(ctx context.Context, hookContext openfeature.HookContext, hookHints openfeature.HookHints) (*openfeature.EvaluationContext, error) {
	return hook.contextSupplier(), nil
}
