package gofeatureflag

import (
	"context"
	"fmt"
	"time"

	"github.com/open-feature/go-sdk-contrib/providers/go-feature-flag/pkg/controller"
	"github.com/open-feature/go-sdk-contrib/providers/go-feature-flag/pkg/hook"
	"github.com/open-feature/go-sdk-contrib/providers/go-feature-flag/pkg/util"
	"github.com/open-feature/go-sdk-contrib/providers/ofrep"
	of "github.com/open-feature/go-sdk/openfeature"
	ffclient "github.com/thomaspoignant/go-feature-flag"
	"github.com/thomaspoignant/go-feature-flag/ffcontext"
)

const providerName = "GO Feature Flag"
const cacheableMetadataKey = "gofeatureflag_cacheable"

type Provider struct {
	ofrepProvider         *ofrep.Provider
	cache                 *controller.Cache
	dataCollectorManager  controller.DataCollectorManager
	options               ProviderOptions
	status                of.State
	hooks                 []of.Hook
	goffAPI               controller.GoFeatureFlagAPI
	evaluationType        EvaluationType
	goFeatureFlagInstance *ffclient.GoFeatureFlag
	pollingInfo           struct {
		ticker  *time.Ticker
		channel chan bool
	}
	events chan of.Event
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

	if options.EvaluationType == "" {
		options.EvaluationType = EvaluationTypeInProcess
	}

	var ofrepProvider *ofrep.Provider
	var cacheCtrl *controller.Cache

	if options.EvaluationType == EvaluationTypeRemote {
		ofrepOptions := make([]ofrep.Option, 0)
		if options.APIKey != "" {
			ofrepOptions = append(ofrepOptions, ofrep.WithBearerToken(options.APIKey))
		}
		if options.HTTPClient != nil {
			ofrepOptions = append(ofrepOptions, ofrep.WithClient(options.HTTPClient))
		}
		ofrepOptions = append(ofrepOptions, ofrep.WithHeaderProvider(func() (key string, value string) {
			return controller.ContentTypeHeader, controller.ApplicationJson
		}))
		ofrepProvider = ofrep.NewProvider(options.Endpoint, ofrepOptions...)
		cacheCtrl = controller.NewCache(options.FlagCacheSize, options.FlagCacheTTL, options.DisableCache)
	}

	if options.ExporterMetadata == nil {
		options.ExporterMetadata = make(map[string]any)
	}
	options.ExporterMetadata["provider"] = "go"
	options.ExporterMetadata["openfeature"] = true

	goffAPI := controller.NewGoFeatureFlagAPI(controller.GoFeatureFlagApiOptions{
		Endpoint:         options.Endpoint,
		HTTPClient:       options.HTTPClient,
		APIKey:           options.APIKey,
		ExporterMetadata: options.ExporterMetadata,
	})
	dataCollectorManager := controller.NewDataCollectorManager(
		goffAPI,
		options.DataCollectorMaxEventStored,
		options.DataFlushInterval,
	)

	var goFeatureFlagInstance *ffclient.GoFeatureFlag
	if options.EvaluationType == EvaluationTypeInProcess {
		goff, err := ffclient.New(*options.GOFeatureFlagConfig)
		if err != nil {
			return nil, err
		}
		goFeatureFlagInstance = goff
	}

	return &Provider{
		ofrepProvider:         ofrepProvider,
		cache:                 cacheCtrl,
		dataCollectorManager:  dataCollectorManager,
		options:               options,
		goffAPI:               goffAPI,
		evaluationType:        options.EvaluationType,
		goFeatureFlagInstance: goFeatureFlagInstance,
		events:                make(chan of.Event, 5),
		hooks:                 []of.Hook{},
	}, nil
}

func (p *Provider) Metadata() of.Metadata {
	return of.Metadata{
		Name: fmt.Sprintf("%s Provider", providerName),
	}
}

func (p *Provider) BooleanEvaluation(ctx context.Context, flag string, defaultValue bool, evalCtx of.FlattenedContext) of.BoolResolutionDetail {
	if err := util.ValidateTargetingKey(evalCtx); err != nil {
		return of.BoolResolutionDetail{
			Value:                    defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{ResolutionError: *err, Reason: of.ErrorReason},
		}
	}

	if p.evaluationType == EvaluationTypeInProcess {
		return p.booleanEvaluationInProcess(ctx, flag, defaultValue, evalCtx)
	}

	if cacheValue, err := p.cache.GetBool(flag, evalCtx); err == nil && cacheValue != nil {
		cacheValue.Reason = of.CachedReason
		return *cacheValue
	}
	res := p.ofrepProvider.BooleanEvaluation(ctx, flag, defaultValue, evalCtx)
	if cachable, err := res.FlagMetadata.GetBool(cacheableMetadataKey); err == nil && cachable {
		_ = p.cache.Set(flag, evalCtx, res)
	}
	return res
}

func (p *Provider) StringEvaluation(ctx context.Context, flag string, defaultValue string, evalCtx of.FlattenedContext) of.StringResolutionDetail {
	if err := util.ValidateTargetingKey(evalCtx); err != nil {
		return of.StringResolutionDetail{
			Value:                    defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{ResolutionError: *err, Reason: of.ErrorReason},
		}
	}

	if p.evaluationType == EvaluationTypeInProcess {
		return p.stringEvaluationInProcess(ctx, flag, defaultValue, evalCtx)
	}

	if cacheValue, err := p.cache.GetString(flag, evalCtx); err == nil && cacheValue != nil {
		cacheValue.Reason = of.CachedReason
		return *cacheValue
	}
	res := p.ofrepProvider.StringEvaluation(ctx, flag, defaultValue, evalCtx)
	if cachable, err := res.FlagMetadata.GetBool(cacheableMetadataKey); err == nil && cachable {
		_ = p.cache.Set(flag, evalCtx, res)
	}
	return res
}

func (p *Provider) FloatEvaluation(ctx context.Context, flag string, defaultValue float64, evalCtx of.FlattenedContext) of.FloatResolutionDetail {
	if err := util.ValidateTargetingKey(evalCtx); err != nil {
		return of.FloatResolutionDetail{
			Value:                    defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{ResolutionError: *err, Reason: of.ErrorReason},
		}
	}

	if p.evaluationType == EvaluationTypeInProcess {
		return p.floatEvaluationInProcess(ctx, flag, defaultValue, evalCtx)
	}

	if cacheValue, err := p.cache.GetFloat(flag, evalCtx); err == nil && cacheValue != nil {
		cacheValue.Reason = of.CachedReason
		return *cacheValue
	}
	res := p.ofrepProvider.FloatEvaluation(ctx, flag, defaultValue, evalCtx)
	if cachable, err := res.FlagMetadata.GetBool(cacheableMetadataKey); err == nil && cachable {
		_ = p.cache.Set(flag, evalCtx, res)
	}
	return res
}

func (p *Provider) IntEvaluation(ctx context.Context, flag string, defaultValue int64, evalCtx of.FlattenedContext) of.IntResolutionDetail {
	if err := util.ValidateTargetingKey(evalCtx); err != nil {
		return of.IntResolutionDetail{
			Value:                    defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{ResolutionError: *err, Reason: of.ErrorReason},
		}
	}

	if p.evaluationType == EvaluationTypeInProcess {
		return p.intEvaluationInProcess(ctx, flag, defaultValue, evalCtx)
	}

	if cacheValue, err := p.cache.GetInt(flag, evalCtx); err == nil && cacheValue != nil {
		cacheValue.Reason = of.CachedReason
		return *cacheValue
	}
	res := p.ofrepProvider.IntEvaluation(ctx, flag, defaultValue, evalCtx)
	if cachable, err := res.FlagMetadata.GetBool(cacheableMetadataKey); err == nil && cachable {
		_ = p.cache.Set(flag, evalCtx, res)
	}
	return res
}

func (p *Provider) ObjectEvaluation(ctx context.Context, flag string, defaultValue any, evalCtx of.FlattenedContext) of.InterfaceResolutionDetail {
	if err := util.ValidateTargetingKey(evalCtx); err != nil {
		return of.InterfaceResolutionDetail{
			Value:                    defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{ResolutionError: *err, Reason: of.ErrorReason},
		}
	}

	if p.evaluationType == EvaluationTypeInProcess {
		return p.objectEvaluationInProcess(ctx, flag, defaultValue, evalCtx)
	}

	if cacheValue, err := p.cache.GetInterface(flag, evalCtx); err == nil && cacheValue != nil {
		cacheValue.Reason = of.CachedReason
		return *cacheValue
	}
	res := p.ofrepProvider.ObjectEvaluation(ctx, flag, defaultValue, evalCtx)
	if cachable, err := res.FlagMetadata.GetBool(cacheableMetadataKey); err == nil && cachable {
		_ = p.cache.Set(flag, evalCtx, res)
	}
	return res
}

func (p *Provider) Hooks() []of.Hook {
	return p.hooks
}

// Init holds initialization logic of the provider
func (p *Provider) Init(_ of.EvaluationContext) error {
	p.hooks = append(p.hooks, hook.NewEvaluationEnrichmentHook(p.options.ExporterMetadata))
	if !p.options.DisableDataCollector {
		dataCollectorHook := hook.NewDataCollectorHook(&p.dataCollectorManager)
		p.hooks = append(p.hooks, dataCollectorHook)
		p.dataCollectorManager.Start()
	}

	// Start polling to check if there is any flag change in order to invalidate the cache.
	if p.options.FlagChangePollingInterval >= 0 {
		if p.evaluationType == EvaluationTypeRemote && !p.options.DisableCache {
			p.startPolling(p.options.FlagChangePollingInterval)
		} else if p.evaluationType == EvaluationTypeInProcess {
			p.startInProcessPolling(p.options.FlagChangePollingInterval)
		}
	}

	p.status = of.ReadyState
	p.events <- of.Event{
		ProviderName: providerName, EventType: of.ProviderReady,
		ProviderEventDetails: of.ProviderEventDetails{Message: "Provider is ready"}}
	return nil
}

// Status exposes the status of the provider
func (p *Provider) Status() of.State {
	return p.status
}

// Shutdown defines the shutdown operation of the provider
func (p *Provider) Shutdown() {
	if !p.options.DisableDataCollector {
		p.hooks = []of.Hook{}
		p.dataCollectorManager.Stop()
	}
	p.stopPolling()
	if p.evaluationType == EvaluationTypeInProcess && p.goFeatureFlagInstance != nil {
		p.goFeatureFlagInstance.Close()
	}
}

// EventChannel returns the event channel of this provider
func (p *Provider) EventChannel() <-chan of.Event {
	return p.events
}

// startPolling starts the polling mechanism that checks if the configuration has changed.
func (p *Provider) startPolling(pollingInterval time.Duration) {
	if pollingInterval == 0 {
		pollingInterval = 120000 * time.Millisecond
	}
	p.pollingInfo = struct {
		ticker  *time.Ticker
		channel chan bool
	}{
		ticker:  time.NewTicker(pollingInterval),
		channel: make(chan bool),
	}
	go func() {
		for {
			select {
			case <-p.pollingInfo.channel:
				return
			case <-p.pollingInfo.ticker.C:
				changeStatus, err := p.goffAPI.ConfigurationHasChanged()
				switch changeStatus {
				case controller.FlagConfigurationInitialized,
					controller.FlagConfigurationNotChanged:
					// do nothing

				case controller.FlagConfigurationUpdated:
					// Clearing the cache when the configuration is updated
					p.cache.Purge()
					p.events <- of.Event{
						ProviderName: providerName, EventType: of.ProviderConfigChange,
						ProviderEventDetails: of.ProviderEventDetails{Message: "Configuration has changed"}}
				case controller.ErrorConfigurationChange:
					p.events <- of.Event{
						ProviderName: providerName, EventType: of.ProviderStale,
						ProviderEventDetails: of.ProviderEventDetails{
							Message: fmt.Sprintf("Impossible to check configuration change: %s", err),
						},
					}
				}
			}
		}
	}()
}

// stopPolling stops the polling mechanism that check if the configuration has changed.
func (p *Provider) stopPolling() {
	if p.pollingInfo.channel != nil {
		p.pollingInfo.channel <- true
	}
	if p.pollingInfo.ticker != nil {
		p.pollingInfo.ticker.Stop()
	}
}

// startInProcessPolling starts the polling mechanism for in-process mode to check if the configuration has changed.
func (p *Provider) startInProcessPolling(pollingInterval time.Duration) {
	if pollingInterval == 0 {
		pollingInterval = 120000 * time.Millisecond
	}
	p.pollingInfo = struct {
		ticker  *time.Ticker
		channel chan bool
	}{
		ticker:  time.NewTicker(pollingInterval),
		channel: make(chan bool),
	}
	go func() {
		for {
			select {
			case <-p.pollingInfo.channel:
				return
			case <-p.pollingInfo.ticker.C:
				// For in-process mode, we rely on the GO Feature Flag module's internal polling
				// We emit a configuration change event to notify users of potential updates
				if p.goFeatureFlagInstance != nil {
					p.events <- of.Event{
						ProviderName: providerName, EventType: of.ProviderConfigChange,
						ProviderEventDetails: of.ProviderEventDetails{Message: "Configuration check completed (in-process mode)"}}
				}
			}
		}
	}()
}

func (p *Provider) booleanEvaluationInProcess(ctx context.Context, flagName string, defaultValue bool, evalCtx of.FlattenedContext) of.BoolResolutionDetail {
	res := evaluateLocally[bool](p, flagName, defaultValue, evalCtx)
	return of.BoolResolutionDetail{
		Value:                    res.Value,
		ProviderResolutionDetail: res.ProviderResolutionDetail,
	}
}

func (p *Provider) stringEvaluationInProcess(ctx context.Context, flagName string, defaultValue string, evalCtx of.FlattenedContext) of.StringResolutionDetail {
	res := evaluateLocally[string](p, flagName, defaultValue, evalCtx)
	return of.StringResolutionDetail{
		Value:                    res.Value,
		ProviderResolutionDetail: res.ProviderResolutionDetail,
	}
}

func (p *Provider) floatEvaluationInProcess(ctx context.Context, flagName string, defaultValue float64, evalCtx of.FlattenedContext) of.FloatResolutionDetail {
	res := evaluateLocally[float64](p, flagName, defaultValue, evalCtx)
	return of.FloatResolutionDetail{
		Value:                    res.Value,
		ProviderResolutionDetail: res.ProviderResolutionDetail,
	}
}

func (p *Provider) intEvaluationInProcess(ctx context.Context, flagName string, defaultValue int64, evalCtx of.FlattenedContext) of.IntResolutionDetail {
	res := evaluateLocally[int64](p, flagName, defaultValue, evalCtx)
	return of.IntResolutionDetail{
		Value:                    res.Value,
		ProviderResolutionDetail: res.ProviderResolutionDetail,
	}
}

func (p *Provider) objectEvaluationInProcess(ctx context.Context, flagName string, defaultValue any, evalCtx of.FlattenedContext) of.InterfaceResolutionDetail {
	res := evaluateLocally[any](p, flagName, defaultValue, evalCtx)
	return of.InterfaceResolutionDetail{
		Value:                    res.Value,
		ProviderResolutionDetail: res.ProviderResolutionDetail,
	}
}

func evaluateLocally[T JsonType](provider *Provider, flagName string, defaultValue T, evalCtx of.FlattenedContext) GenericResolutionDetail[T] {
	goffRequestBody, errConvert := NewEvalFlagRequest[T](evalCtx, defaultValue)
	if errConvert != nil {
		return GenericResolutionDetail[T]{
			Value: defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				ResolutionError: *errConvert,
				Reason:          of.ErrorReason,
			},
		}
	}

	ctxBuilder := ffcontext.NewEvaluationContextBuilder(goffRequestBody.EvaluationContext.Key)
	for k, v := range goffRequestBody.EvaluationContext.Custom {
		ctxBuilder.AddCustom(k, v)
	}

	rawResult, err := provider.goFeatureFlagInstance.RawVariation(flagName, ctxBuilder.Build(), defaultValue)
	if err != nil {
		return GenericResolutionDetail[T]{
			Value: defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				ResolutionError: of.NewGeneralResolutionError(err.Error()),
				Reason:          of.ErrorReason,
			},
		}
	}

	if rawResult.ErrorCode != "" {
		switch rawResult.ErrorCode {
		case string(of.FlagNotFoundCode):
			return GenericResolutionDetail[T]{
				Value: defaultValue,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					ResolutionError: of.NewFlagNotFoundResolutionError(fmt.Sprintf("flag %s was not found in GO Feature Flag", flagName)),
					Reason:          of.ErrorReason,
				},
			}
		case string(of.ProviderNotReadyCode):
			return GenericResolutionDetail[T]{
				Value: defaultValue,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					ResolutionError: of.NewProviderNotReadyResolutionError(
						fmt.Sprintf("provider not ready for evaluation of flag %s", flagName)),
					Reason: of.ErrorReason,
				},
			}
		case string(of.ParseErrorCode):
			return GenericResolutionDetail[T]{
				Value: defaultValue,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					ResolutionError: of.NewParseErrorResolutionError(
						fmt.Sprintf("parse error during evaluation of flag %s", flagName)),
					Reason: of.ErrorReason,
				},
			}
		case string(of.TypeMismatchCode):
			return GenericResolutionDetail[T]{
				Value: defaultValue,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					ResolutionError: of.NewTypeMismatchResolutionError(
						fmt.Sprintf("unexpected type for flag %s", flagName)),
					Reason: of.ErrorReason,
				},
			}
		case string(of.GeneralCode):
			return GenericResolutionDetail[T]{
				Value: defaultValue,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					ResolutionError: of.NewGeneralResolutionError(
						fmt.Sprintf("unexpected error during evaluation of the flag %s", flagName)),
					Reason: of.ErrorReason,
				},
			}
		}
	}

	var v JsonType
	switch value := rawResult.Value.(type) {
	case int:
		v = int64(value)
	default:
		v = value
	}

	switch value := v.(type) {
	case nil:
		return GenericResolutionDetail[T]{
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				Reason:  of.Reason(rawResult.Reason),
				Variant: rawResult.VariationType,
			},
		}
	case T:
		return GenericResolutionDetail[T]{
			Value: value,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				Reason:  of.Reason(rawResult.Reason),
				Variant: rawResult.VariationType,
			},
		}
	default:
		return GenericResolutionDetail[T]{
			Value: defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				ResolutionError: of.NewTypeMismatchResolutionError(fmt.Sprintf("unexpected type for flag %s", flagName)),
				Reason:          of.ErrorReason,
			},
		}
	}
}
