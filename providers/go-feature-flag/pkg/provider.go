package gofeatureflag

import (
	"context"
	"fmt"

	"github.com/open-feature/go-sdk-contrib/providers/go-feature-flag/pkg/api"
	"github.com/open-feature/go-sdk-contrib/providers/go-feature-flag/pkg/evaluator"
	"github.com/open-feature/go-sdk-contrib/providers/go-feature-flag/pkg/hook"
	"github.com/open-feature/go-sdk-contrib/providers/go-feature-flag/pkg/model"
	"github.com/open-feature/go-sdk-contrib/providers/go-feature-flag/pkg/service"
	"github.com/open-feature/go-sdk/openfeature"
)

const providerName = "GO Feature Flag"

type Provider struct {
	// options are the provider options to use GO Feature Flag.
	options ProviderOptions

	// hooks are the hooks to use for the GO Feature Flag provider.
	hooks []openfeature.Hook

	// evaluator is the evaluator to use for the GO Feature Flag provider.
	// Depending on the evaluation type, it will be a different evaluator.
	// If the evaluation type is remote, it will be a remote evaluator.
	// If the evaluation type is in process, it will be an in process evaluator.
	// By default, it will be an in process evaluator.
	// The evaluator is used to evaluate the flags.
	evaluator evaluator.EvaluatorInterface

	// dataCollectorMngr is a service that is in charge of sending telemetry data to the relay-proxy.
	dataCollectorMngr *service.DataCollectorManager
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
	options = enrichExporterMetadataWithDefaults(options)
	evaluator, err := selectEvaluator(options)
	if err != nil {
		return nil, err
	}
	dataCollectorMngr := createDataCollectorManager(options)
	return &Provider{
		options:           options,
		evaluator:         evaluator,
		dataCollectorMngr: dataCollectorMngr,
		hooks: []openfeature.Hook{
			hook.NewEvaluationEnrichmentHook(options.ExporterMetadata),
		},
	}, nil
}

// Metadata returns the metadata of the GO Feature Flag provider.
func (p *Provider) Metadata() openfeature.Metadata {
	return openfeature.Metadata{
		Name: fmt.Sprintf("%s Provider", providerName),
	}
}

// Hooks returns a collection of openfeature.Hook defined by this provider
func (p *Provider) Hooks() []openfeature.Hook {
	return p.hooks
}

// Init holds initialization logic of the provider
func (p *Provider) Init(evaluationContext openfeature.EvaluationContext) error {
	if err := p.evaluator.Init(evaluationContext); err != nil {
		return err
	}
	if !p.options.DisableDataCollector {
		dcHook := hook.NewDataCollectorHook(p.dataCollectorMngr)
		p.hooks = append([]openfeature.Hook{dcHook}, p.hooks...)
		p.dataCollectorMngr.Start()
	}
	return nil
}

// Shutdown define the shutdown operation of the provider
func (p *Provider) Shutdown() {
	p.evaluator.Shutdown()
	if !p.options.DisableDataCollector {
		p.dataCollectorMngr.Stop()
	}
}

func (p *Provider) EventChannel() <-chan openfeature.Event {
	// panic("not implemented")
	return nil
}

// BooleanEvaluation returns a boolean flag
func (p *Provider) BooleanEvaluation(
	ctx context.Context,
	flag string, defaultValue bool,
	flatCtx openfeature.FlattenedContext,
) openfeature.BoolResolutionDetail {
	return p.evaluator.BooleanEvaluation(ctx, flag, defaultValue, flatCtx)
}

// StringEvaluation returns a string flag
func (p *Provider) StringEvaluation(
	ctx context.Context,
	flag string,
	defaultValue string,
	flatCtx openfeature.FlattenedContext,
) openfeature.StringResolutionDetail {
	return p.evaluator.StringEvaluation(ctx, flag, defaultValue, flatCtx)
}

// FloatEvaluation returns a float flag
func (p *Provider) FloatEvaluation(
	ctx context.Context,
	flag string,
	defaultValue float64,
	flatCtx openfeature.FlattenedContext,
) openfeature.FloatResolutionDetail {
	return p.evaluator.FloatEvaluation(ctx, flag, defaultValue, flatCtx)
}

// IntEvaluation returns an int flag
func (p *Provider) IntEvaluation(
	ctx context.Context,
	flag string,
	defaultValue int64,
	flatCtx openfeature.FlattenedContext,
) openfeature.IntResolutionDetail {
	return p.evaluator.IntEvaluation(ctx, flag, defaultValue, flatCtx)
}

// ObjectEvaluation returns an object flag
func (p *Provider) ObjectEvaluation(
	ctx context.Context,
	flag string,
	defaultValue any,
	flatCtx openfeature.FlattenedContext,
) openfeature.InterfaceResolutionDetail {
	return p.evaluator.ObjectEvaluation(ctx, flag, defaultValue, flatCtx)
}

// Track is used to track the usage of a flag.
// It will add a tracking event to the data collector manager.
// The tracking event will be sent to the relay-proxy periodically.
// The tracking event will contain the name of the flag, the evaluation context, and the details of the tracking event.
func (p *Provider) Track(
	ctx context.Context,
	trackingEventName string,
	evaluationContext openfeature.EvaluationContext,
	details openfeature.TrackingEventDetails,
) {
	p.dataCollectorMngr.AddEvent(
		model.NewTrackingEvent(evaluationContext, trackingEventName, details),
	)
}

// selectEvaluator gets the evaluator for the GO Feature Flag provider.
func selectEvaluator(options ProviderOptions) (evaluator.EvaluatorInterface, error) {
	switch options.EvaluationType {
	case EvaluationTypeRemote:
		return evaluator.NewRemoteEvaluator(evaluator.RemoteEvaluatorOptions{
			Endpoint:   options.Endpoint,
			APIKey:     options.APIKey,
			HTTPClient: options.HTTPClient,
		}), nil
	default:
		return nil, fmt.Errorf("invalid evaluation type: %s", options.EvaluationType)
	}
}

// createDataCollectorManager is preparing the data collector manager based on the provider options.
func createDataCollectorManager(options ProviderOptions) *service.DataCollectorManager {
	mngr := service.NewDataCollectorManager(
		api.NewGoffAPI(api.GoffAPIOptions{
			Endpoint:              options.Endpoint,
			DataCollectorEndpoint: options.DataCollectorEndpoint,
			HTTPClient:            options.HTTPClient,
			APIKey:                options.APIKey,
			ExporterMetadata:      options.ExporterMetadata,
		}),
		options.DataCollectorMaxEventStored,
		options.DataFlushInterval,
	)
	return &mngr
}

// enrichExporterMetadataWithDefaults sets the default exporter metadata if not provided.
func enrichExporterMetadataWithDefaults(options ProviderOptions) ProviderOptions {
	if options.ExporterMetadata == nil {
		options.ExporterMetadata = make(map[string]any)
	}
	options.ExporterMetadata["provider"] = "go"
	options.ExporterMetadata["openfeature"] = true
	return options
}
