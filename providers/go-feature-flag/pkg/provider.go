package gofeatureflag

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/open-feature/go-sdk-contrib/providers/go-feature-flag/pkg/model"
	of "github.com/open-feature/go-sdk/pkg/openfeature"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"time"
)

// Provider is the OpenFeature provider for GO Feature Flag.
type Provider struct {
	httpClient HTTPClient
	endpoint   string
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

func genericEvaluation[T model.JsonType](provider *Provider, ctx context.Context, flagName string, defaultValue T, evalCtx of.FlattenedContext) GenericResolutionDetail[T] {
	goffRequestBody, errConvert := model.NewEvalFlagRequest[T](evalCtx, defaultValue)
	if errConvert != nil {
		return GenericResolutionDetail[T]{
			Value: defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				ResolutionError: *errConvert,
				Reason:          of.ErrorReason,
			},
		}
	}

	goffRequestBodyStr, err := json.Marshal(goffRequestBody)
	if err != nil {
		return GenericResolutionDetail[T]{
			Value: defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				ResolutionError: of.NewGeneralResolutionError("impossible to marshal GO Feature Flag request"),
				Reason:          of.ErrorReason,
			},
		}
	}

	evalURL, err := url.Parse(provider.endpoint)
	if err != nil {
		return GenericResolutionDetail[T]{
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
		return GenericResolutionDetail[T]{
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
		return GenericResolutionDetail[T]{
			Value: defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				ResolutionError: of.NewGeneralResolutionError("impossible to contact GO Feature Flag relay proxy instance"),
				Reason:          of.ErrorReason,
			},
		}
	}
	responseStr, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return GenericResolutionDetail[T]{
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
		return GenericResolutionDetail[T]{
			Value: defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				ResolutionError: of.NewTypeMismatchResolutionError(fmt.Sprintf("unexpected type for flag %s", flagName)),
				Reason:          of.ErrorReason,
			},
		}
	}

	if evalResponse.ErrorCode == string(of.FlagNotFoundCode) {
		return GenericResolutionDetail[T]{
			Value: defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				ResolutionError: of.NewFlagNotFoundResolutionError(fmt.Sprintf("flag %s was not found in GO Feature Flag", flagName)),
				Reason:          of.ErrorReason,
			},
		}
	}

	if evalResponse.Reason == string(of.DisabledReason) {
		return GenericResolutionDetail[T]{
			Value: defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				Reason:  of.DisabledReason,
				Variant: "defaultSdk",
			},
		}
	}

	return GenericResolutionDetail[T]{
		Value: evalResponse.Value,
		ProviderResolutionDetail: of.ProviderResolutionDetail{
			Reason:  of.Reason(evalResponse.Reason),
			Variant: evalResponse.VariationType,
		},
	}
}

type GenericResolutionDetail[T model.JsonType] struct {
	Value T
	of.ProviderResolutionDetail
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
