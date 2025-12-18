package hook

import (
	"context"
	"time"

	"go.openfeature.dev/contrib/providers/go-feature-flag/v2/pkg/controller"
	"go.openfeature.dev/contrib/providers/go-feature-flag/v2/pkg/model"
	"go.openfeature.dev/openfeature/v2"
)

var _ openfeature.Hook

func NewDataCollectorHook(dataCollectorManager *controller.DataCollectorManager) openfeature.Hook {
	return &dataCollectorHook{dataCollectorManager: dataCollectorManager}
}

type dataCollectorHook struct {
	openfeature.UnimplementedHook
	dataCollectorManager *controller.DataCollectorManager
}

func (d *dataCollectorHook) After(_ context.Context, hookCtx openfeature.HookContext,
	evalDetails openfeature.HookEvaluationDetails, hint openfeature.HookHints,
) error {
	if evalDetails.Reason != openfeature.CachedReason {
		// we send it only when cached because the evaluation will be collected directly in the relay-proxy
		return nil
	}
	event := model.FeatureEvent{
		Kind:         "feature",
		ContextKind:  "user",
		UserKey:      hookCtx.EvaluationContext().TargetingKey(),
		CreationDate: time.Now().Unix(),
		Key:          hookCtx.FlagKey(),
		Variation:    evalDetails.Variant,
		Value:        evalDetails.Value,
		Default:      false,
		Source:       "PROVIDER_CACHE",
	}
	_ = d.dataCollectorManager.AddEvent(event)
	return nil
}

func (d *dataCollectorHook) Error(_ context.Context, hookCtx openfeature.HookContext,
	err error, hint openfeature.HookHints,
) {
	event := model.FeatureEvent{
		Kind:         "feature",
		ContextKind:  "user",
		UserKey:      hookCtx.EvaluationContext().TargetingKey(),
		CreationDate: time.Now().Unix(),
		Key:          hookCtx.FlagKey(),
		Variation:    "SdkDefault",
		Value:        hookCtx.DefaultValue(),
		Default:      true,
		Source:       "PROVIDER_CACHE",
	}
	_ = d.dataCollectorManager.AddEvent(event)
}
