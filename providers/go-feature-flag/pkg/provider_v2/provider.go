package provider_v2

import (
	"context"
	"github.com/open-feature/go-sdk-contrib/providers/go-feature-flag/pkg/provider_v2/controller"
	"github.com/open-feature/go-sdk-contrib/providers/go-feature-flag/pkg/provider_v2/util"
	"github.com/open-feature/go-sdk-contrib/providers/ofrep"
	of "github.com/open-feature/go-sdk/openfeature"
)

type Provider struct {
	ofrepProvider *ofrep.Provider
	cache         *controller.Cache
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
	ofrepOptions := make([]ofrep.Option, 0)
	if options.APIKey != "" {
		ofrepOptions = append(ofrepOptions, ofrep.WithBearerToken(options.APIKey))
	}
	if options.HTTPClient != nil {
		ofrepOptions = append(ofrepOptions, ofrep.WithClient(options.HTTPClient))
	}

	ofrepProvider := ofrep.NewProvider(options.Endpoint, ofrepOptions...)
	cacheCtrl := controller.NewCache(options.FlagCacheSize, options.FlagCacheTTL, options.DisableCache)
	return &Provider{ofrepProvider: ofrepProvider, cache: cacheCtrl}, nil
}

func (p Provider) Metadata() of.Metadata {
	return of.Metadata{
		Name: "GO Feature Flag Provider",
	}
}

func (p Provider) BooleanEvaluation(ctx context.Context, flag string, defaultValue bool, evalCtx of.FlattenedContext) of.BoolResolutionDetail {
	if err := util.ValidateTargetingKey(evalCtx); err != nil {
		return of.BoolResolutionDetail{
			Value:                    defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{ResolutionError: *err, Reason: of.ErrorReason},
		}
	}
	if cacheValue, err := p.cache.GetBool(flag, evalCtx); err == nil && cacheValue != nil {
		cacheValue.Reason = of.CachedReason
		return *cacheValue
	}
	res := p.ofrepProvider.BooleanEvaluation(ctx, flag, defaultValue, evalCtx)
	if cachable, err := res.FlagMetadata.GetBool("gofeatureflag_cacheable"); err == nil && cachable {
		_ = p.cache.Set(flag, evalCtx, res)
	}
	return res
}

func (p Provider) StringEvaluation(ctx context.Context, flag string, defaultValue string, evalCtx of.FlattenedContext) of.StringResolutionDetail {
	if err := util.ValidateTargetingKey(evalCtx); err != nil {
		return of.StringResolutionDetail{
			Value:                    defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{ResolutionError: *err, Reason: of.ErrorReason},
		}
	}
	if cacheValue, err := p.cache.GetString(flag, evalCtx); err == nil && cacheValue != nil {
		cacheValue.Reason = of.CachedReason
		return *cacheValue
	}
	res := p.ofrepProvider.StringEvaluation(ctx, flag, defaultValue, evalCtx)
	if cachable, err := res.FlagMetadata.GetBool("gofeatureflag_cacheable"); err == nil && cachable {
		_ = p.cache.Set(flag, evalCtx, res)
	}
	return res
}

func (p Provider) FloatEvaluation(ctx context.Context, flag string, defaultValue float64, evalCtx of.FlattenedContext) of.FloatResolutionDetail {
	if err := util.ValidateTargetingKey(evalCtx); err != nil {
		return of.FloatResolutionDetail{
			Value:                    defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{ResolutionError: *err, Reason: of.ErrorReason},
		}
	}
	if cacheValue, err := p.cache.GetFloat(flag, evalCtx); err == nil && cacheValue != nil {
		cacheValue.Reason = of.CachedReason
		return *cacheValue
	}
	res := p.ofrepProvider.FloatEvaluation(ctx, flag, defaultValue, evalCtx)
	if cachable, err := res.FlagMetadata.GetBool("gofeatureflag_cacheable"); err == nil && cachable {
		_ = p.cache.Set(flag, evalCtx, res)
	}
	return res
}

func (p Provider) IntEvaluation(ctx context.Context, flag string, defaultValue int64, evalCtx of.FlattenedContext) of.IntResolutionDetail {
	if err := util.ValidateTargetingKey(evalCtx); err != nil {
		return of.IntResolutionDetail{
			Value:                    defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{ResolutionError: *err, Reason: of.ErrorReason},
		}
	}
	if cacheValue, err := p.cache.GetInt(flag, evalCtx); err == nil && cacheValue != nil {
		cacheValue.Reason = of.CachedReason
		return *cacheValue
	}
	res := p.ofrepProvider.IntEvaluation(ctx, flag, defaultValue, evalCtx)
	if cachable, err := res.FlagMetadata.GetBool("gofeatureflag_cacheable"); err == nil && cachable {
		_ = p.cache.Set(flag, evalCtx, res)
	}
	return res
}

func (p Provider) ObjectEvaluation(ctx context.Context, flag string, defaultValue interface{}, evalCtx of.FlattenedContext) of.InterfaceResolutionDetail {
	if err := util.ValidateTargetingKey(evalCtx); err != nil {
		return of.InterfaceResolutionDetail{
			Value:                    defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{ResolutionError: *err, Reason: of.ErrorReason},
		}
	}
	if cacheValue, err := p.cache.GetInterface(flag, evalCtx); err == nil && cacheValue != nil {
		cacheValue.Reason = of.CachedReason
		return *cacheValue
	}
	res := p.ofrepProvider.ObjectEvaluation(ctx, flag, defaultValue, evalCtx)
	if cachable, err := res.FlagMetadata.GetBool("gofeatureflag_cacheable"); err == nil && cachable {
		_ = p.cache.Set(flag, evalCtx, res)
	}
	return res
}

func (p Provider) Hooks() []of.Hook {
	return []of.Hook{}
}

// Init holds initialization logic of the provider
func (p Provider) Init(_ of.EvaluationContext) error {
	// code to initialize your provider
	return nil
}

// Status expose the status of the provider
func (p Provider) Status() of.State {
	// The state is typically set during initialization.
	return of.ReadyState
}

// Shutdown define the shutdown operation of the provider
func (p Provider) Shutdown() {
	// code to shutdown your provider
}
