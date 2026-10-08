package hook

import (
	"context"

	"github.com/open-feature/go-sdk-contrib/providers/go-feature-flag/pkg/manager"
	"github.com/open-feature/go-sdk-contrib/providers/go-feature-flag/pkg/model"
	"github.com/open-feature/go-sdk/openfeature"
)

const evaluationTypeRemote = "REMOTE"

// NewDataCollectorHook collects evaluation events; the optional isTrackable skips flags whose trackEvents is false.
func NewDataCollectorHook(dataCollectorManager *manager.DataCollectorManager, evaluationType string,
	isTrackable ...func(flagKey string) bool) openfeature.Hook {
	h := &dataCollectorHook{dataCollectorManager: dataCollectorManager, evaluationType: evaluationType}
	if len(isTrackable) > 0 {
		h.isTrackable = isTrackable[0]
	}
	return h
}

type dataCollectorHook struct {
	openfeature.UnimplementedHook
	dataCollectorManager *manager.DataCollectorManager
	evaluationType       string
	isTrackable          func(flagKey string) bool
}

func (d *dataCollectorHook) After(_ context.Context, hookCtx openfeature.HookContext,
	evalDetails openfeature.InterfaceEvaluationDetails, hint openfeature.HookHints) error {
	if d.evaluationType == evaluationTypeRemote &&
		evalDetails.Reason != openfeature.CachedReason {
		// only collect events for remote evaluation if the reason is cached
		return nil
	}
	if !d.trackable(hookCtx.FlagKey()) {
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
	if !d.trackable(hookCtx.FlagKey()) {
		return
	}
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

func (d *dataCollectorHook) trackable(flagKey string) bool {
	return d.isTrackable == nil || d.isTrackable(flagKey)
}

func getSource(evaluationType string) string {
	if evaluationType == evaluationTypeRemote {
		return "PROVIDER_CACHE"
	}
	return "INPROCESS"
}
