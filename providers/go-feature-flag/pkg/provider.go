package gofeatureflag

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/open-feature/go-sdk-contrib/providers/go-feature-flag/pkg/model"
	of "github.com/open-feature/go-sdk/pkg/openfeature"
	client "github.com/thomaspoignant/go-feature-flag"
	"github.com/thomaspoignant/go-feature-flag/ffuser"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"time"
)

// Provider is the OpenFeature provider for GO Feature Flag.
type Provider struct {
	httpClient            HTTPClient
	endpoint              string
	goFeatureFlagInstance *client.GoFeatureFlag
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

// NewProvider is the easiest way of creating a new GO Feature Flag provider.
func NewProvider(options ProviderOptions) (*Provider, error) {
	if options.GOFeatureFlagConfig != nil {
		goff, err := client.New(*options.GOFeatureFlagConfig)
		if err != nil {
			return nil, err
		}
		return &Provider{
			goFeatureFlagInstance: goff,
		}, nil
	}

	if options.Endpoint == "" {
		return nil, fmt.Errorf("invalid provider options, invalid endpoint value: %s", options.Endpoint)
	}

	httpClient := options.HTTPClient
	if httpClient == nil {
		httpClient = defaultHTTPClient()
	}

	return &Provider{
		endpoint:   options.Endpoint,
		httpClient: httpClient,
	}, nil
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

// Hooks is returning an empty array because GO Feature Flag does not use any hooks.
func (p *Provider) Hooks() []of.Hook {
	return []of.Hook{}
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
// it means that you don't need any rela proxy to make it works.
func evaluateLocally[T model.JsonType](provider *Provider, goffRequestBody model.EvalFlagRequest, flagName string, defaultValue T) model.GenericResolutionDetail[T] {
	// Construct user
	userBuilder := ffuser.NewUserBuilder(goffRequestBody.User.Key)
	userBuilder.Anonymous(goffRequestBody.User.Anonymous)
	for k, v := range goffRequestBody.User.Custom {
		userBuilder.AddCustom(k, v)
	}

	// Call GO Module
	rawResult, err := provider.goFeatureFlagInstance.RawVariation(flagName, userBuilder.Build(), defaultValue)
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

// evaluateWithRelayProxy is calling GO Feature Flag relay proxy to evaluate the file.
func evaluateWithRelayProxy[T model.JsonType](provider *Provider, ctx context.Context, goffRequestBody model.EvalFlagRequest, flagName string, defaultValue T) model.GenericResolutionDetail[T] {
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
	responseStr, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return model.GenericResolutionDetail[T]{
			Value: defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				ResolutionError: of.NewGeneralResolutionError("impossible to read API response from GO Feature Flag"),
				Reason:          of.ErrorReason,
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
		fmt.Println(err)
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
				Variant: "defaultSdk",
			},
		}
	}

	return model.GenericResolutionDetail[T]{
		Value: evalResponse.Value,
		ProviderResolutionDetail: of.ProviderResolutionDetail{
			Reason:  of.Reason(evalResponse.Reason),
			Variant: evalResponse.VariationType,
		},
	}
}
