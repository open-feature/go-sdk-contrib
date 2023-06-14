package configcat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	sdk "github.com/configcat/go-sdk/v7"
	"github.com/open-feature/go-sdk/pkg/openfeature"
)

var _ openfeature.FeatureProvider = (*Provider)(nil)

// Evaluation ctx keys that are mapped to ConfigCat user data.
const (
	IdentifierKey = openfeature.TargetingKey
	EmailKey      = "email"
	CountryKey    = "country"
)

type Client interface {
	GetBoolValueDetails(key string, defaultValue bool, user sdk.User) sdk.BoolEvaluationDetails
	GetStringValueDetails(key string, defaultValue string, user sdk.User) sdk.StringEvaluationDetails
	GetFloatValueDetails(key string, defaultValue float64, user sdk.User) sdk.FloatEvaluationDetails
	GetIntValueDetails(key string, defaultValue int, user sdk.User) sdk.IntEvaluationDetails
}

// NewProvider creates an OpenFeature provider backed by ConfigCat.
func NewProvider(client Client) *Provider {
	return &Provider{
		client: client,
	}
}

type Provider struct {
	client Client
}

// Metadata returns value of Metadata (name of current service, exposed to openfeature sdk)
func (p *Provider) Metadata() openfeature.Metadata {
	return openfeature.Metadata{
		Name: "ConfigCat",
	}
}

// Hooks are not currently implemented, an empty slice is returned.
func (p *Provider) Hooks() []openfeature.Hook {
	return nil
}

func (p *Provider) BooleanEvaluation(ctx context.Context, flag string, defaultValue bool, evalCtx openfeature.FlattenedContext) openfeature.BoolResolutionDetail {
	user, errDetails := toUserData(evalCtx)
	if errDetails != nil {
		return openfeature.BoolResolutionDetail{
			Value:                    defaultValue,
			ProviderResolutionDetail: *errDetails,
		}
	}

	evaluation := p.client.GetBoolValueDetails(flag, defaultValue, user)
	return openfeature.BoolResolutionDetail{
		Value:                    evaluation.Value,
		ProviderResolutionDetail: toResolutionDetail(evaluation.Data),
	}
}

func (p *Provider) StringEvaluation(ctx context.Context, flag string, defaultValue string, evalCtx openfeature.FlattenedContext) openfeature.StringResolutionDetail {
	user, errDetails := toUserData(evalCtx)
	if errDetails != nil {
		return openfeature.StringResolutionDetail{
			Value:                    defaultValue,
			ProviderResolutionDetail: *errDetails,
		}
	}

	evaluation := p.client.GetStringValueDetails(flag, defaultValue, user)
	return openfeature.StringResolutionDetail{
		Value:                    evaluation.Value,
		ProviderResolutionDetail: toResolutionDetail(evaluation.Data),
	}
}

func (p *Provider) FloatEvaluation(ctx context.Context, flag string, defaultValue float64, evalCtx openfeature.FlattenedContext) openfeature.FloatResolutionDetail {
	user, errDetails := toUserData(evalCtx)
	if errDetails != nil {
		return openfeature.FloatResolutionDetail{
			Value:                    defaultValue,
			ProviderResolutionDetail: *errDetails,
		}
	}

	evaluation := p.client.GetFloatValueDetails(flag, defaultValue, user)
	return openfeature.FloatResolutionDetail{
		Value:                    evaluation.Value,
		ProviderResolutionDetail: toResolutionDetail(evaluation.Data),
	}
}

func (p *Provider) IntEvaluation(ctx context.Context, flag string, defaultValue int64, evalCtx openfeature.FlattenedContext) openfeature.IntResolutionDetail {
	user, errDetails := toUserData(evalCtx)
	if errDetails != nil {
		return openfeature.IntResolutionDetail{
			Value:                    defaultValue,
			ProviderResolutionDetail: *errDetails,
		}
	}

	evaluation := p.client.GetIntValueDetails(flag, int(defaultValue), user)
	return openfeature.IntResolutionDetail{
		Value:                    int64(evaluation.Value),
		ProviderResolutionDetail: toResolutionDetail(evaluation.Data),
	}
}

// ObjectEvaluation attempts to parse a string feature flag value as JSON.
func (p *Provider) ObjectEvaluation(ctx context.Context, flag string, defaultValue interface{}, evalCtx openfeature.FlattenedContext) openfeature.InterfaceResolutionDetail {
	user, errDetails := toUserData(evalCtx)
	if errDetails != nil {
		return openfeature.InterfaceResolutionDetail{
			Value:                    defaultValue,
			ProviderResolutionDetail: *errDetails,
		}
	}

	evaluation := p.client.GetStringValueDetails(flag, "", user)
	if evaluation.Data.IsDefaultValue || evaluation.Data.Error != nil {
		// we evaludated with a fake default value, so we
		// need to use the one we were provided
		return openfeature.InterfaceResolutionDetail{
			Value:                    defaultValue,
			ProviderResolutionDetail: toResolutionDetail(evaluation.Data),
		}
	}

	// Attempt to unmarshal the string value as if it's JSON
	var object map[string]any
	err := json.Unmarshal([]byte(evaluation.Value), &object)
	if err != nil {
		return openfeature.InterfaceResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
				ResolutionError: openfeature.NewTypeMismatchResolutionError(
					fmt.Sprintf("failed to unmarshal string flag as json: %s", err),
				),
				Reason: openfeature.ErrorReason,
			},
		}
	}

	return openfeature.InterfaceResolutionDetail{
		Value:                    object,
		ProviderResolutionDetail: toResolutionDetail(evaluation.Data),
	}
}

func toUserData(evalCtx openfeature.FlattenedContext) (*sdk.UserData, *openfeature.ProviderResolutionDetail) {
	if len(evalCtx) == 0 {
		return nil, nil
	}

	userData := &sdk.UserData{}
	custom := make(map[string]string, len(evalCtx))
	for key, origVal := range evalCtx {
		val, ok := toStr(origVal)
		if !ok {
			return nil, &openfeature.ProviderResolutionDetail{
				ResolutionError: openfeature.NewInvalidContextResolutionError(
					fmt.Sprintf("key `%s` can not be converted to string", key),
				),
				Reason: openfeature.ErrorReason,
			}
		}

		switch key {
		case IdentifierKey:
			userData.Identifier = val
		case EmailKey:
			userData.Email = val
		case CountryKey:
			userData.Country = val
		default:
			custom[key] = val
		}
	}

	userData.Custom = custom
	return userData, nil
}

func toStr(val any) (string, bool) {
	switch v := val.(type) {
	case string:
		return v, true
	case int, int8, int16, int32, int64:
		return fmt.Sprintf("%d", v), true
	case float32, float64:
		return fmt.Sprintf("%.6f", v), true
	case bool:
		return fmt.Sprintf("%t", v), true
	default:
		return "", false
	}
}

func toResolutionDetail(details sdk.EvaluationDetailsData) openfeature.ProviderResolutionDetail {
	if details.Error != nil {
		return openfeature.ProviderResolutionDetail{
			ResolutionError: toResolutionError(details.Error),
			Reason:          openfeature.ErrorReason,
		}
	}

	reason := openfeature.DefaultReason
	if details.MatchedEvaluationRule != nil || details.MatchedEvaluationPercentageRule != nil {
		reason = openfeature.TargetingMatchReason
	}

	return openfeature.ProviderResolutionDetail{
		Reason:  reason,
		Variant: details.VariationID,
	}
}

func toResolutionError(err error) openfeature.ResolutionError {
	var errKeyNotFound sdk.ErrKeyNotFound
	if errors.As(err, &errKeyNotFound) {
		return openfeature.NewFlagNotFoundResolutionError(errKeyNotFound.Error())
	}

	return openfeature.NewGeneralResolutionError(err.Error())
}
