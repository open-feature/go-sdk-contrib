package gofeatureflag

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/open-feature/go-sdk-contrib/providers/go-feature-flag/pkg/api"
	"github.com/open-feature/go-sdk-contrib/providers/go-feature-flag/pkg/evaluator"
	"github.com/open-feature/go-sdk-contrib/providers/go-feature-flag/pkg/hook"
	controller "github.com/open-feature/go-sdk-contrib/providers/go-feature-flag/pkg/manager"
	"github.com/open-feature/go-sdk-contrib/providers/go-feature-flag/pkg/model"
	"github.com/open-feature/go-sdk/openfeature"
)

var _ openfeature.FeatureProvider = &Provider{}
var _ openfeature.Tracker = &Provider{}
var _ openfeature.ContextAwareStateHandler = &Provider{}
var _ openfeature.EventHandler = &Provider{}
var eventChannelBuffer = 5

const providerName = "GO Feature Flag"

type Provider struct {
	options          ProviderOptions
	dataCollectorMgr controller.DataCollectorManager
	eventStream      chan openfeature.Event
	evaluator        evaluator.Evaluator
	hooks            []openfeature.Hook
	logger           *slog.Logger
}

// NewProvider allows you to create a GO Feature Flag provider without any context.
// We recommend using the function NewProviderWithContext and provide your context when creating the provider.
func NewProvider(options ProviderOptions) (*Provider, error) {
	return NewProviderWithContext(context.Background(), options)
}

// NewProviderWithContext is the easiest way of creating a new GO Feature Flag provider.
func NewProviderWithContext(ctx context.Context, options ProviderOptions) (*Provider, error) {
	if err := options.Validation(); err != nil {
		return nil, err
	}
	if options.ExporterMetadata == nil {
		options.ExporterMetadata = make(map[string]any)
	}
	options.ExporterMetadata["provider"] = "go"
	options.ExporterMetadata["openfeature"] = true

	goffAPI := api.NewGoFeatureFlagAPI(api.GoFeatureFlagAPIOptions{
		Endpoint:             options.Endpoint,
		DataCollectorBaseURL: options.DataCollectorBaseURL,
		HTTPClient:           options.HTTPClient,
		APIKey:               options.APIKey,
		Headers:              options.Headers,
		ExporterMetadata:     options.ExporterMetadata,
	})

	if options.Logger == nil {
		options.Logger = slog.Default()
	}

	var dcm controller.DataCollectorManager
	if !options.DataCollectorDisabled {
		dcm = controller.NewDataCollectorManager(*goffAPI, options.DataCollectorMaxEventStored, options.DataCollectorCollectInterval)
	}

	eventStream := make(chan openfeature.Event, eventChannelBuffer)
	ev := selectEvaluator(options, goffAPI, eventStream)
	p := &Provider{
		options:          options,
		dataCollectorMgr: dcm,
		eventStream:      eventStream,
		evaluator:        ev,
		logger:           options.Logger,
	}
	p.hooks = buildHooks(options, &p.dataCollectorMgr)
	return p, nil
}

// EventChannel implements [openfeature.EventHandler].
func (p *Provider) EventChannel() <-chan openfeature.Event {
	return p.eventStream
}

// InitWithContext implements [openfeature.ContextAwareStateHandler].
func (p *Provider) InitWithContext(ctx context.Context, evaluationContext openfeature.EvaluationContext) error {
	if !p.options.DataCollectorDisabled {
		p.dataCollectorMgr.Start()
	}
	return p.evaluator.Init(ctx)
}

// ShutdownWithContext implements [openfeature.ContextAwareStateHandler].
func (p *Provider) ShutdownWithContext(ctx context.Context) error {
	if !p.options.DataCollectorDisabled {
		p.dataCollectorMgr.Stop(ctx)
	}
	return p.evaluator.Shutdown(ctx)
}

// BooleanEvaluation implements [openfeature.FeatureProvider].
func (p *Provider) BooleanEvaluation(ctx context.Context, flag string, defaultValue bool, flatCtx openfeature.FlattenedContext) openfeature.BoolResolutionDetail {
	return p.evaluator.BooleanEvaluation(ctx, flag, defaultValue, flatCtx)
}

// FloatEvaluation implements [openfeature.FeatureProvider].
func (p *Provider) FloatEvaluation(ctx context.Context, flag string, defaultValue float64, flatCtx openfeature.FlattenedContext) openfeature.FloatResolutionDetail {
	return p.evaluator.FloatEvaluation(ctx, flag, defaultValue, flatCtx)
}

// Hooks implements [openfeature.FeatureProvider].
func (p *Provider) Hooks() []openfeature.Hook { return p.hooks }

// IntEvaluation implements [openfeature.FeatureProvider].
func (p *Provider) IntEvaluation(ctx context.Context, flag string, defaultValue int64, flatCtx openfeature.FlattenedContext) openfeature.IntResolutionDetail {
	return p.evaluator.IntEvaluation(ctx, flag, defaultValue, flatCtx)
}

// Metadata implements [openfeature.FeatureProvider].
func (p *Provider) Metadata() openfeature.Metadata {
	return openfeature.Metadata{
		Name: fmt.Sprintf("%s Provider", providerName),
	}
}

// ObjectEvaluation implements [openfeature.FeatureProvider].
func (p *Provider) ObjectEvaluation(ctx context.Context, flag string, defaultValue any, flatCtx openfeature.FlattenedContext) openfeature.InterfaceResolutionDetail {
	return p.evaluator.ObjectEvaluation(ctx, flag, defaultValue, flatCtx)
}

// StringEvaluation implements [openfeature.FeatureProvider].
func (p *Provider) StringEvaluation(ctx context.Context, flag string, defaultValue string, flatCtx openfeature.FlattenedContext) openfeature.StringResolutionDetail {
	return p.evaluator.StringEvaluation(ctx, flag, defaultValue, flatCtx)
}

// Track implements [openfeature.Tracker].
func (p *Provider) Track(ctx context.Context, trackingEventName string, evaluationContext openfeature.EvaluationContext, details openfeature.TrackingEventDetails) {
	if p.options.DataCollectorDisabled {
		p.logger.Warn("Data collector is disabled, skipping tracking event", "trackingEventName", trackingEventName)
		return
	}

	contextKind := "user"
	if isAnonymous, ok := evaluationContext.Attribute("anonymous").(bool); ok && isAnonymous {
		contextKind = "anonymousUser"
	}
	userKey := evaluationContext.TargetingKey()
	if userKey == "" {
		userKey = "undefined"
	}

	evalCtxMap := map[string]any{}
	for k, v := range evaluationContext.Attributes() {
		evalCtxMap[k] = v
	}

	event := model.TrackingEvent{
		Kind:              "tracking",
		ContextKind:       contextKind,
		UserKey:           userKey,
		CreationDate:      time.Now().Unix(),
		Key:               trackingEventName,
		EvaluationContext: evalCtxMap,
		TrackingDetails:   details.Attributes(),
	}
	if err := p.dataCollectorMgr.AddEvent(event); err != nil {
		p.logger.Error("Failed to add tracking event to data collector", "error", err)
	}
}

// selectEvaluator selects the evaluator based on the evaluation type
func selectEvaluator(options ProviderOptions, goffAPI *api.GoFeatureFlagAPI, eventStream chan openfeature.Event) evaluator.Evaluator {
	if options.EvaluationType == EvaluationTypeRemote {
		return evaluator.NewRemoteEvaluator(options.Endpoint, options.HTTPClient, options.APIKey, options.Headers)
	}
	return evaluator.NewInprocessEvaluator(options.FlagChangePollingInterval, goffAPI, eventStream)
}

// buildHooks constructs the list of hooks for the provider.
func buildHooks(options ProviderOptions, dcm *controller.DataCollectorManager) []openfeature.Hook {
	hooks := []openfeature.Hook{
		hook.NewEvaluationEnrichmentHook(options.ExporterMetadata),
	}
	if options.EvaluationType != EvaluationTypeRemote && !options.DataCollectorDisabled {
		hooks = append(hooks, hook.NewDataCollectorHook(dcm))
	}
	return hooks
}

// Init implements [openfeature.ContextAwareStateHandler].
func (p *Provider) Init(evaluationContext openfeature.EvaluationContext) error {
	// nothing to do here since we are using the context aware state handler
	return nil
}

// Shutdown implements [openfeature.ContextAwareStateHandler].
func (p *Provider) Shutdown() {
	// nothing to do here since we are using the context aware state handler
}
