package configcat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	configcat "github.com/configcat/go-sdk/v7"
	"github.com/open-feature/go-sdk/pkg/openfeature"
)

const (
	IdentifierKey = openfeature.TargetingKey
	EmailKey      = "email"
	CountryKey    = "country"
)

var _ openfeature.FeatureProvider = (*Provider)(nil)

type Provider struct {
	client *configcat.Client
}

func NewProvider(client *configcat.Client) *Provider {
	return &Provider{
		client: client,
	}
}

func (p *Provider) Metadata() openfeature.Metadata {
	return openfeature.Metadata{
		Name: "ConfigCat",
	}
}

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

func toUserData(evalCtx openfeature.FlattenedContext) (*configcat.UserData, *openfeature.ProviderResolutionDetail) {
	if len(evalCtx) == 0 {
		return nil, nil
	}

	userData := &configcat.UserData{}
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

func toResolutionDetail(details configcat.EvaluationDetailsData) openfeature.ProviderResolutionDetail {
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
	var errKeyNotFound configcat.ErrKeyNotFound
	if errors.As(err, &errKeyNotFound) {
		return openfeature.NewFlagNotFoundResolutionError(errKeyNotFound.Error())
	}

	return openfeature.NewGeneralResolutionError(err.Error())
}
