package hook

import (
	"context"

	"github.com/open-feature/go-sdk-contrib/providers/go-feature-flag/pkg/manager"
	"github.com/open-feature/go-sdk-contrib/providers/go-feature-flag/pkg/model"
	"github.com/open-feature/go-sdk/openfeature"
)

func NewDataCollectorHook(dataCollectorManager *manager.DataCollectorManager) openfeature.Hook {
	return &dataCollectorHook{dataCollectorManager: dataCollectorManager}
}

type dataCollectorHook struct {
	openfeature.UnimplementedHook
	dataCollectorManager *manager.DataCollectorManager
}

func (d *dataCollectorHook) After(_ context.Context, hookCtx openfeature.HookContext,
	evalDetails openfeature.InterfaceEvaluationDetails, hint openfeature.HookHints) error {
	event := model.NewFeatureEvent(
		hookCtx.EvaluationContext(),
		hookCtx.FlagKey(),
		evalDetails.Value,
		evalDetails.Variant,
		false,
		"",
		"PROVIDER_CACHE",
	)
	_ = d.dataCollectorManager.AddEvent(event)
	return nil
}

func (d *dataCollectorHook) Error(_ context.Context, hookCtx openfeature.HookContext,
	err error, hint openfeature.HookHints) {
	event := model.NewFeatureEvent(
		hookCtx.EvaluationContext(),
		hookCtx.FlagKey(),
		hookCtx.DefaultValue(),
		"SdkDefault",
		true,
		"",
		"PROVIDER_CACHE",
	)
	_ = d.dataCollectorManager.AddEvent(event)
}
