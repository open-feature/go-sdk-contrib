package gofeatureflag

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/open-feature/go-sdk-contrib/providers/go-feature-flag/pkg/model"
	of "github.com/open-feature/go-sdk/pkg/openfeature"
	client "github.com/thomaspoignant/go-feature-flag"
	"github.com/thomaspoignant/go-feature-flag/ffcontext"
	"io"
	"net/http"
	"net/url"
	"path"
	"time"
)

const defaultDataCacheMaxEventInMemory = 500
const defaultDataCacheFlushInterval = 1 * time.Minute

// Provider is the OpenFeature provider for GO Feature Flag.
type Provider struct {
	httpClient            HTTPClient
	endpoint              string
	goFeatureFlagInstance *client.GoFeatureFlag
	apiKey                string
	cache                 Cache
	cacheTTL              time.Duration
	cacheDisable          bool
	dataCollectorHook     DataCollectorHook
}

// HTTPClient is a custom interface to be able to override it by any implementation
// of an HTTP client.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// defaultHTTPClient is the default HTTP client used to call GO Feature Flag.
// By default, we have a timeout of 10000 milliseconds.
func defaultHTTPClient() HTTPClient {
	netTransport := &http.Transport{
		TLSHandshakeTimeout: 10000 * time.Millisecond,
	}

	return &http.Client{
		Timeout:   10000 * time.Millisecond,
		Transport: netTransport,
	}
}

// NewProvider allows you to create a GO Feature Flag provider without any context.
// We recommend using the function NewProviderWithContext and provide your context when creating the provider.
func NewProvider(options ProviderOptions) (*Provider, error) {
	return NewProviderWithContext(context.Background(), options)
}

// NewProviderWithContext is the easiest way of creating a new GO Feature Flag provider.
func NewProviderWithContext(ctx context.Context, options ProviderOptions) (*Provider, error) {
	hook := NewDataCollectorHook(options)
	hook.Init(ctx)

	if options.GOFeatureFlagConfig != nil {
		goff, err := client.New(*options.GOFeatureFlagConfig)
		if err != nil {
			return nil, err
		}
		return &Provider{
			goFeatureFlagInstance: goff,
			dataCollectorHook:     *hook,
		}, nil
	}

	if options.Endpoint == "" {
		return nil, fmt.Errorf("invalid provider options, empty endpoint value")
	}

	// Set default values
	httpClient := options.HTTPClient
	if httpClient == nil {
		httpClient = defaultHTTPClient()
	}

	p := &Provider{
		apiKey:            options.APIKey,
		endpoint:          options.Endpoint,
		httpClient:        httpClient,
		cacheTTL:          options.FlagCacheTTL,
		cacheDisable:      options.DisableCache,
		cache:             *NewCache(options.FlagCacheSize, options.FlagCacheTTL),
		dataCollectorHook: *hook,
	}

	return p, nil
}

// Metadata returns the meta of the GO Feature Flag provider.
func (p *Provider) Metadata() of.Metadata {
	return of.Metadata{
		Name: "GO Feature Flag",
	}
}

func (p *Provider) BooleanEvaluation(ctx context.Context, flag string, defaultValue bool, evalCtx of.FlattenedContext) of.BoolResolutionDetail {
	res := genericEvaluation[bool](p, ctx, flag, defaultValue, evalCtx)
	return of.BoolResolutionDetail{
		Value:                    res.Value,
		ProviderResolutionDetail: res.ProviderResolutionDetail,
	}
}
func (p *Provider) StringEvaluation(ctx context.Context, flag string, defaultValue string, evalCtx of.FlattenedContext) of.StringResolutionDetail {
	res := genericEvaluation[string](p, ctx, flag, defaultValue, evalCtx)
	return of.StringResolutionDetail{
		Value:                    res.Value,
		ProviderResolutionDetail: res.ProviderResolutionDetail,
	}
}
func (p *Provider) FloatEvaluation(ctx context.Context, flag string, defaultValue float64, evalCtx of.FlattenedContext) of.FloatResolutionDetail {
	res := genericEvaluation[float64](p, ctx, flag, defaultValue, evalCtx)
	return of.FloatResolutionDetail{
		Value:                    res.Value,
		ProviderResolutionDetail: res.ProviderResolutionDetail,
	}
}
func (p *Provider) IntEvaluation(ctx context.Context, flag string, defaultValue int64, evalCtx of.FlattenedContext) of.IntResolutionDetail {
	res := genericEvaluation[int64](p, ctx, flag, defaultValue, evalCtx)
	return of.IntResolutionDetail{
		Value:                    res.Value,
		ProviderResolutionDetail: res.ProviderResolutionDetail,
	}
}
func (p *Provider) ObjectEvaluation(ctx context.Context, flag string, defaultValue interface{}, evalCtx of.FlattenedContext) of.InterfaceResolutionDetail {
	res := genericEvaluation[interface{}](p, ctx, flag, defaultValue, evalCtx)
	return of.InterfaceResolutionDetail{
		Value:                    res.Value,
		ProviderResolutionDetail: res.ProviderResolutionDetail,
	}
}

func (p *Provider) Shutdown() {
	p.dataCollectorHook.Shutdown()
}

// Hooks is returning an empty array because GO Feature Flag does not use any hooks.
func (p *Provider) Hooks() []of.Hook {
	return []of.Hook{&p.dataCollectorHook}
}

// genericEvaluation is doing evaluation for all types using generics.
func genericEvaluation[T model.JsonType](provider *Provider, ctx context.Context, flagName string, defaultValue T, evalCtx of.FlattenedContext) model.GenericResolutionDetail[T] {
	goffRequestBody, errConvert := model.NewEvalFlagRequest[T](evalCtx, defaultValue)
	if errConvert != nil {
		return model.GenericResolutionDetail[T]{
			Value: defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				ResolutionError: *errConvert,
				Reason:          of.ErrorReason,
			},
		}
	}

	// if we have a GO Feature Flag instance instantiate we evaluate the flag locally,
	// using the GO module directly. We will not send any remote calls to the relay proxy.
	if provider.goFeatureFlagInstance != nil {
		return evaluateLocally(provider, goffRequestBody, flagName, defaultValue)
	}
	return evaluateWithRelayProxy(provider, ctx, goffRequestBody, flagName, defaultValue)
}

// evaluateLocally is using the GO Feature Flag module to evaluate your flag.
// it means that you don't need any relay proxy to make it work.
func evaluateLocally[T model.JsonType](provider *Provider, goffRequestBody model.EvalFlagRequest, flagName string, defaultValue T) model.GenericResolutionDetail[T] {
	// Construct user
	ctxBuilder := ffcontext.NewEvaluationContextBuilder(goffRequestBody.EvaluationContext.Key)
	for k, v := range goffRequestBody.EvaluationContext.Custom {
		ctxBuilder.AddCustom(k, v)
	}

	// Call GO Module
	rawResult, err := provider.goFeatureFlagInstance.RawVariation(flagName, ctxBuilder.Build(), defaultValue)
	if err != nil {
		switch rawResult.ErrorCode {
		case string(of.FlagNotFoundCode):
			return model.GenericResolutionDetail[T]{
				Value: defaultValue,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					ResolutionError: of.NewFlagNotFoundResolutionError(fmt.Sprintf("flag %s was not found in GO Feature Flag", flagName)),
					Reason:          of.ErrorReason,
				},
			}
		case string(of.ProviderNotReadyCode):
			return model.GenericResolutionDetail[T]{
				Value: defaultValue,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					ResolutionError: of.NewProviderNotReadyResolutionError(
						fmt.Sprintf("provider not ready for evaluation of flag %s", flagName)),
					Reason: of.ErrorReason,
				},
			}
		case string(of.ParseErrorCode):
			return model.GenericResolutionDetail[T]{
				Value: defaultValue,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					ResolutionError: of.NewParseErrorResolutionError(
						fmt.Sprintf("parse error during evaluation of flag %s", flagName)),
					Reason: of.ErrorReason,
				},
			}
		case string(of.TypeMismatchCode):
			return model.GenericResolutionDetail[T]{
				Value: defaultValue,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					ResolutionError: of.NewTypeMismatchResolutionError(
						fmt.Sprintf("unexpected type for flag %s", flagName)),
					Reason: of.ErrorReason,
				},
			}
		case string(of.GeneralCode):
			return model.GenericResolutionDetail[T]{
				Value: defaultValue,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					ResolutionError: of.NewGeneralResolutionError(
						fmt.Sprintf("unexpected error during evaluation of the flag %s", flagName)),
					Reason: of.ErrorReason,
				},
			}
		}
	}

	// This part convert the int received by the module to int64 to be compatible with
	// the types expect by Open-feature.
	var v model.JsonType
	switch value := rawResult.Value.(type) {
	case int:
		v = int64(value)
	default:
		v = value
	}

	switch value := v.(type) {
	case nil:
		return model.GenericResolutionDetail[T]{
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				Reason:  of.Reason(rawResult.Reason),
				Variant: rawResult.VariationType,
			},
		}
	case T:
		return model.GenericResolutionDetail[T]{
			Value: value,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				Reason:  of.Reason(rawResult.Reason),
				Variant: rawResult.VariationType,
			},
		}
	default:
		return model.GenericResolutionDetail[T]{
			Value: defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				ResolutionError: of.NewTypeMismatchResolutionError(fmt.Sprintf("unexpected type for flag %s", flagName)),
				Reason:          of.ErrorReason,
			},
		}
	}
}

func convertCache[T model.JsonType](value interface{}) (model.GenericResolutionDetail[T], error) {
	switch v := value.(type) {
	case model.GenericResolutionDetail[T]:
		return v, nil
	default:
		return model.GenericResolutionDetail[T]{}, fmt.Errorf("impossible to convert into the cache")
	}
}

// evaluateWithRelayProxy is calling GO Feature Flag relay proxy to evaluate the file.
func evaluateWithRelayProxy[T model.JsonType](provider *Provider, ctx context.Context, goffRequestBody model.EvalFlagRequest, flagName string, defaultValue T) model.GenericResolutionDetail[T] {
	cacheKey := fmt.Sprintf("%s-%+v", flagName, goffRequestBody.EvaluationContext)
	// check if flag is available in the cache
	cacheResInterface, err := provider.cache.Get(cacheKey)
	if err == nil {
		// we have retrieve something from the cache.
		cacheValue, err := convertCache[T](cacheResInterface)
		if err != nil {
			// impossible to convert the cache, we remove the entry from the cache assuming the next
			// call to convertCache wouldn't result in the same error on the next call.
			provider.cache.Remove(cacheKey)
		} else {
			cacheValue.Reason = of.CachedReason
			return cacheValue
		}
	}

	goffRequestBodyStr, err := json.Marshal(goffRequestBody)
	if err != nil {
		return model.GenericResolutionDetail[T]{
			Value: defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				ResolutionError: of.NewGeneralResolutionError("impossible to marshal GO Feature Flag request"),
				Reason:          of.ErrorReason,
			},
		}
	}

	evalURL, err := url.Parse(provider.endpoint)
	if err != nil {
		return model.GenericResolutionDetail[T]{
			Value: defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				ResolutionError: of.NewGeneralResolutionError("impossible to parse GO Feature Flag endpoint option"),
				Reason:          of.ErrorReason,
			},
		}
	}
	evalURL.Path = path.Join(evalURL.Path, "v1", "/")
	evalURL.Path = path.Join(evalURL.Path, "feature", "/")
	evalURL.Path = path.Join(evalURL.Path, flagName, "/")
	evalURL.Path = path.Join(evalURL.Path, "eval", "/")

	goffRequest, err :=
		http.NewRequestWithContext(ctx, http.MethodPost, evalURL.String(), bytes.NewBuffer(goffRequestBodyStr))
	if err != nil {
		return model.GenericResolutionDetail[T]{
			Value: defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				ResolutionError: of.NewGeneralResolutionError("error while building GO Feature Flag relay proxy request"),
				Reason:          of.ErrorReason,
			},
		}
	}
	goffRequest.Header.Set("Content-Type", "application/json")
	if provider.apiKey != "" {
		goffRequest.Header.Set("Authorization", fmt.Sprintf("Bearer %s", provider.apiKey))
	}

	response, err := provider.httpClient.Do(goffRequest)
	if err != nil {
		return model.GenericResolutionDetail[T]{
			Value: defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				ResolutionError: of.NewGeneralResolutionError("impossible to contact GO Feature Flag relay proxy instance"),
				Reason:          of.ErrorReason,
			},
		}
	}
	responseStr, err := io.ReadAll(response.Body)
	if err != nil {
		return model.GenericResolutionDetail[T]{
			Value: defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				ResolutionError: of.NewGeneralResolutionError("impossible to read API response from GO Feature Flag"),
				Reason:          of.ErrorReason,
			},
		}
	}

	if response.StatusCode == http.StatusUnauthorized {
		return model.GenericResolutionDetail[T]{
			Value: defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				ResolutionError: of.NewGeneralResolutionError(
					"invalid token used to contact GO Feature Flag relay proxy instance"),
				Reason: of.ErrorReason,
			},
		}
	}
	if response.StatusCode >= http.StatusBadRequest {
		return model.GenericResolutionDetail[T]{
			Value: defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				ResolutionError: of.NewGeneralResolutionError(
					"unexpected answer from the relay proxy"),
				Reason: of.ErrorReason,
			},
		}
	}

	var evalResponse model.EvalResponse[T]
	err = json.Unmarshal(responseStr, &evalResponse)
	if err != nil {
		if err.Error() == "unexpected end of JSON input" {
			return model.GenericResolutionDetail[T]{
				Value: defaultValue,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					ResolutionError: of.NewParseErrorResolutionError(
						fmt.Sprintf("impossible to parse response for flag %s: %s", flagName, responseStr)),
					Reason: of.ErrorReason,
				},
			}
		}
		return model.GenericResolutionDetail[T]{
			Value: defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				ResolutionError: of.NewTypeMismatchResolutionError(fmt.Sprintf("unexpected type for flag %s", flagName)),
				Reason:          of.ErrorReason,
			},
		}
	}

	if evalResponse.ErrorCode == string(of.FlagNotFoundCode) {
		return model.GenericResolutionDetail[T]{
			Value: defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				ResolutionError: of.NewFlagNotFoundResolutionError(fmt.Sprintf("flag %s was not found in GO Feature Flag", flagName)),
				Reason:          of.ErrorReason,
			},
		}
	}

	if evalResponse.Reason == string(of.DisabledReason) {
		return model.GenericResolutionDetail[T]{
			Value: defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				Reason:  of.DisabledReason,
				Variant: "SdkDefault",
			},
		}
	}

	resDetail := model.GenericResolutionDetail[T]{
		Value: evalResponse.Value,
		ProviderResolutionDetail: of.ProviderResolutionDetail{
			Reason:  of.Reason(evalResponse.Reason),
			Variant: evalResponse.VariationType,
		},
	}

	if !provider.cacheDisable && evalResponse.Cacheable {
		provider.cache.Set(cacheKey, resDetail)
	}
	return resDetail
}
