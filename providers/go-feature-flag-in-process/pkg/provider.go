package gofeatureflaginprocess

import (
	"context"
	"fmt"
	of "github.com/open-feature/go-sdk/openfeature"
	ff "github.com/thomaspoignant/go-feature-flag"
	"github.com/thomaspoignant/go-feature-flag/ffcontext"
)

// Provider is the OpenFeature provider for GO Feature Flag.
type Provider struct {
	// goFeatureFlagInstance is the instance of the GO Feature Flag module.
	goFeatureFlagInstance *ff.GoFeatureFlag
}

// NewProvider allows you to create a GO Feature Flag provider without any context.
// We recommend using the function NewProviderWithContext and provide your context when creating the provider.
func NewProvider(options ProviderOptions) (*Provider, error) {
	return NewProviderWithContext(context.Background(), options)
}

// NewProviderWithContext is the easiest way of creating a new GO Feature Flag provider.
func NewProviderWithContext(ctx context.Context, options ProviderOptions) (*Provider, error) {
	if options.GOFeatureFlagConfig == nil {
		return nil, fmt.Errorf("invalid provider options, empty GOFeatureFlagConfig value")
	}

	goff, err := ff.New(*options.GOFeatureFlagConfig)
	if err != nil {
		return nil, err
	}
	return &Provider{
		goFeatureFlagInstance: goff,
	}, nil
}

// Metadata returns the meta of the GO Feature Flag provider.
func (p *Provider) Metadata() of.Metadata {
	return of.Metadata{
		Name: "GO Feature Flag In Process Provider",
	}
}

func (p *Provider) BooleanEvaluation(_ context.Context, flag string, defaultValue bool, evalCtx of.FlattenedContext) of.BoolResolutionDetail {
	res := evaluateLocally[bool](p, flag, defaultValue, evalCtx)
	return of.BoolResolutionDetail{
		Value:                    res.Value,
		ProviderResolutionDetail: res.ProviderResolutionDetail,
	}
}
func (p *Provider) StringEvaluation(_ context.Context, flag string, defaultValue string, evalCtx of.FlattenedContext) of.StringResolutionDetail {
	res := evaluateLocally[string](p, flag, defaultValue, evalCtx)
	return of.StringResolutionDetail{
		Value:                    res.Value,
		ProviderResolutionDetail: res.ProviderResolutionDetail,
	}
}
func (p *Provider) FloatEvaluation(_ context.Context, flag string, defaultValue float64, evalCtx of.FlattenedContext) of.FloatResolutionDetail {
	res := evaluateLocally[float64](p, flag, defaultValue, evalCtx)
	return of.FloatResolutionDetail{
		Value:                    res.Value,
		ProviderResolutionDetail: res.ProviderResolutionDetail,
	}
}
func (p *Provider) IntEvaluation(_ context.Context, flag string, defaultValue int64, evalCtx of.FlattenedContext) of.IntResolutionDetail {
	res := evaluateLocally[int64](p, flag, defaultValue, evalCtx)
	return of.IntResolutionDetail{
		Value:                    res.Value,
		ProviderResolutionDetail: res.ProviderResolutionDetail,
	}
}
func (p *Provider) ObjectEvaluation(_ context.Context, flag string, defaultValue interface{}, evalCtx of.FlattenedContext) of.InterfaceResolutionDetail {
	res := evaluateLocally[interface{}](p, flag, defaultValue, evalCtx)
	return of.InterfaceResolutionDetail{
		Value:                    res.Value,
		ProviderResolutionDetail: res.ProviderResolutionDetail,
	}
}
func (p *Provider) Init(_ of.EvaluationContext) error {
	return nil
}
func (p *Provider) Shutdown() {
	p.goFeatureFlagInstance.Close()
}

// Hooks is returning an empty array because GO Feature Flag does not use any hooks.
func (p *Provider) Hooks() []of.Hook {
	return []of.Hook{}
}

// evaluateLocally is using the GO Feature Flag module to evaluate your flag.
// it means that you don't need any relay proxy to make it work.
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

	// This part convert the int received by the module to int64 to be compatible with
	// the types expect by Open-feature.
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
