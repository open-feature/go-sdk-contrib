package hook

import (
	"context"

	"go.openfeature.dev/openfeature/v2"
)

var _ openfeature.Hook = (*evaluationEnrichmentHook)(nil)

func NewEvaluationEnrichmentHook(exporterMetadata map[string]any) openfeature.Hook {
	return &evaluationEnrichmentHook{exporterMetadata: exporterMetadata}
}

type evaluationEnrichmentHook struct {
	openfeature.UnimplementedHook
	exporterMetadata map[string]any
}

func (d *evaluationEnrichmentHook) Before(ctx context.Context, hookCtx openfeature.HookContext, _ openfeature.HookHints) (context.Context, error) {
	attributes := hookCtx.EvaluationContext().Attributes()
	if goffSpecific, ok := attributes["gofeatureflag"]; ok {
		switch typed := goffSpecific.(type) {
		case map[string]any:
			typed["exporterMetadata"] = d.exporterMetadata
		default:
			attributes["gofeatureflag"] = map[string]any{"exporterMetadata": d.exporterMetadata}
		}
	} else {
		attributes["gofeatureflag"] = map[string]any{"exporterMetadata": d.exporterMetadata}
	}
	newCtx := openfeature.NewEvaluationContext(hookCtx.EvaluationContext().TargetingKey(), attributes)
	return openfeature.WithTransactionContext(ctx, newCtx), nil
}
