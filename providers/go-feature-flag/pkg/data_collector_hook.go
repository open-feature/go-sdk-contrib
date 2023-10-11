package gofeatureflag

import (
	"context"
	"fmt"
	of "github.com/open-feature/go-sdk/pkg/openfeature"
	"github.com/thomaspoignant/go-feature-flag/exporter"
	"github.com/thomaspoignant/go-feature-flag/exporter/webhookexporter"
	"github.com/thomaspoignant/go-feature-flag/ffcontext"
	"net/url"
	"path"
)

type DataCollectorHook struct {
	dataCollectorScheduler *exporter.Scheduler
	options                ProviderOptions
	isDisabled             bool
}

func NewDataCollectorHook(options ProviderOptions) *DataCollectorHook {
	if options.DataMaxEventInMemory == 0 {
		options.DataMaxEventInMemory = defaultDataCacheMaxEventInMemory
	}
	if options.DataFlushInterval == 0 {
		options.DataFlushInterval = defaultDataCacheFlushInterval
	}
	return &DataCollectorHook{
		options: options,
		// We are not collecting data when using the lib or  if the cache is disabled.
		isDisabled: options.GOFeatureFlagConfig != nil || options.DisableCache,
	}
}

func (d *DataCollectorHook) After(
	_ context.Context,
	hookContext of.HookContext,
	flagEvaluationDetails of.InterfaceEvaluationDetails,
	_ of.HookHints) error {
	if d.isDisabled || flagEvaluationDetails.Reason != of.CachedReason {
		return nil
	}
	event := exporter.NewFeatureEvent(
		ffcontext.NewEvaluationContext(hookContext.EvaluationContext().TargetingKey()),
		hookContext.FlagKey(),
		flagEvaluationDetails.Value,
		flagEvaluationDetails.Variant,
		flagEvaluationDetails.Reason == of.ErrorReason,
		"",
	)

	d.dataCollectorScheduler.AddEvent(event)
	return nil
}

func (d *DataCollectorHook) Error(_ context.Context, hookContext of.HookContext, _ error, _ of.HookHints) {
	if d.isDisabled {
		return
	}
	event := exporter.NewFeatureEvent(
		ffcontext.NewEvaluationContext(hookContext.EvaluationContext().TargetingKey()),
		hookContext.FlagKey(),
		hookContext.DefaultValue(),
		"SdkDefault",
		true,
		"",
	)

	d.dataCollectorScheduler.AddEvent(event)
}

func (d *DataCollectorHook) Init(ctx context.Context) {
	if d.options.DisableCache {
		return
	}

	u, _ := url.Parse(d.options.Endpoint)
	u.Path = path.Join(u.Path, "v1", "/")
	u.Path = path.Join(u.Path, "data", "/")
	u.Path = path.Join(u.Path, "collector", "/")

	webhook := webhookexporter.Exporter{
		EndpointURL: u.String(),
		Meta: map[string]string{
			"provider":    "go",
			"openfeature": "true",
		},
	}
	if d.options.APIKey != "" {
		webhook.Headers = map[string][]string{
			"Authorization": {fmt.Sprintf("Bearer %s", d.options.APIKey)},
		}
	}
	if ctx == nil {
		ctx = context.Background()
	}
	d.dataCollectorScheduler = exporter.NewScheduler(ctx,
		d.options.DataFlushInterval, d.options.DataMaxEventInMemory, &webhook, nil)
	go d.dataCollectorScheduler.StartDaemon()
}

func (d *DataCollectorHook) Before(ctx context.Context, hookContext of.HookContext, hookHints of.HookHints) (*of.EvaluationContext, error) {
	// nothing to do
	return nil, nil
}

func (d *DataCollectorHook) Finally(ctx context.Context, hookContext of.HookContext, hookHints of.HookHints) {
	// nothing to do
}

func (d *DataCollectorHook) Shutdown() {
	if !d.isDisabled {
		d.dataCollectorScheduler.Close()
	}
}
