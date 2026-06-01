package hook

import (
	"context"

	"github.com/open-feature/go-sdk-contrib/providers/go-feature-flag/pkg/manager"
	"github.com/open-feature/go-sdk-contrib/providers/go-feature-flag/pkg/model"
	"github.com/open-feature/go-sdk/openfeature"
)

const evaluationTypeRemote = "REMOTE"

func NewDataCollectorHook(dataCollectorManager *manager.DataCollectorManager, evaluationType string) openfeature.Hook {
	return &dataCollectorHook{dataCollectorManager: dataCollectorManager, evaluationType: evaluationType}
}

type dataCollectorHook struct {
	openfeature.UnimplementedHook
	dataCollectorManager *manager.DataCollectorManager
	evaluationType       string
}

func (d *dataCollectorHook) After(_ context.Context, hookCtx openfeature.HookContext,
	evalDetails openfeature.InterfaceEvaluationDetails, hint openfeature.HookHints) error {
	if d.evaluationType == evaluationTypeRemote &&
		evalDetails.Reason != openfeature.CachedReason {
		// only collect events for remote evaluation if the reason is cached
		return nil
	}

	event := model.NewFeatureEvent(
		hookCtx.EvaluationContext(),
		hookCtx.FlagKey(),
		evalDetails.Value,
		evalDetails.Variant,
		false,
		"",
		getSource(d.evaluationType),
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
		getSource(d.evaluationType),
	)
	_ = d.dataCollectorManager.AddEvent(event)
}

func getSource(evaluationType string) string {
	if evaluationType == evaluationTypeRemote {
		return "PROVIDER_CACHE"
	}
	return "INPROCESS"
}
