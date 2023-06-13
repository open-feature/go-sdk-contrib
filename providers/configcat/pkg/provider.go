package configcat

import (
	"context"
	"encoding/json"
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
		Name: "Configcat",
	}
}

func (p *Provider) Hooks() []openfeature.Hook {
	return nil
}

func (p *Provider) BooleanEvaluation(ctx context.Context, flag string, defaultValue bool, evalCtx openfeature.FlattenedContext) openfeature.BoolResolutionDetail {
	user, errDetails := p.toUserData(evalCtx)
	if errDetails != nil {
		return openfeature.BoolResolutionDetail{
			Value:                    defaultValue,
			ProviderResolutionDetail: *errDetails,
		}
	}

	evaluation := p.client.GetBoolValueDetails(flag, defaultValue, user)
	return openfeature.BoolResolutionDetail{
		Value:                    evaluation.Value,
		ProviderResolutionDetail: p.evalToResolutionDetail(evaluation.Data),
	}
}

func (p *Provider) StringEvaluation(ctx context.Context, flag string, defaultValue string, evalCtx openfeature.FlattenedContext) openfeature.StringResolutionDetail {
	user, errDetails := p.toUserData(evalCtx)
	if errDetails != nil {
		return openfeature.StringResolutionDetail{
			Value:                    defaultValue,
			ProviderResolutionDetail: *errDetails,
		}
	}

	evaluation := p.client.GetStringValueDetails(flag, defaultValue, user)
	return openfeature.StringResolutionDetail{
		Value:                    evaluation.Value,
		ProviderResolutionDetail: p.evalToResolutionDetail(evaluation.Data),
	}
}

func (p *Provider) FloatEvaluation(ctx context.Context, flag string, defaultValue float64, evalCtx openfeature.FlattenedContext) openfeature.FloatResolutionDetail {
	user, errDetails := p.toUserData(evalCtx)
	if errDetails != nil {
		return openfeature.FloatResolutionDetail{
			Value:                    defaultValue,
			ProviderResolutionDetail: *errDetails,
		}
	}

	evaluation := p.client.GetFloatValueDetails(flag, defaultValue, user)
	return openfeature.FloatResolutionDetail{
		Value:                    evaluation.Value,
		ProviderResolutionDetail: p.evalToResolutionDetail(evaluation.Data),
	}
}

func (p *Provider) IntEvaluation(ctx context.Context, flag string, defaultValue int64, evalCtx openfeature.FlattenedContext) openfeature.IntResolutionDetail {
	user, errDetails := p.toUserData(evalCtx)
	if errDetails != nil {
		return openfeature.IntResolutionDetail{
			Value:                    defaultValue,
			ProviderResolutionDetail: *errDetails,
		}
	}

	evaluation := p.client.GetIntValueDetails(flag, int(defaultValue), user)
	return openfeature.IntResolutionDetail{
		Value:                    int64(evaluation.Value),
		ProviderResolutionDetail: p.evalToResolutionDetail(evaluation.Data),
	}
}

func (p *Provider) ObjectEvaluation(ctx context.Context, flag string, defaultValue interface{}, evalCtx openfeature.FlattenedContext) openfeature.InterfaceResolutionDetail {
	user, errDetails := p.toUserData(evalCtx)
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
			ProviderResolutionDetail: p.evalToResolutionDetail(evaluation.Data),
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
		ProviderResolutionDetail: p.evalToResolutionDetail(evaluation.Data),
	}
}

func (p *Provider) toUserData(evalCtx openfeature.FlattenedContext) (*configcat.UserData, *openfeature.ProviderResolutionDetail) {
	if len(evalCtx) == 0 {
		return nil, nil
	}

	userData := &configcat.UserData{}
	errDetailFunc := func(key string) *openfeature.ProviderResolutionDetail {
		return &openfeature.ProviderResolutionDetail{
			ResolutionError: openfeature.NewInvalidContextResolutionError(
				fmt.Sprintf("key `%s` is not a string", key),
			),
			Reason: openfeature.ErrorReason,
		}
	}

	custom := make(map[string]string, len(evalCtx))
	for key, origVal := range evalCtx {
		val, ok := toStr(origVal)

		switch key {
		case IdentifierKey:
			if !ok {
				return nil, errDetailFunc(key)
			}
			userData.Identifier = val
		case EmailKey:
			if !ok {
				return nil, errDetailFunc(key)
			}
			userData.Email = val
		case CountryKey:
			if !ok {
				return nil, errDetailFunc(key)
			}
			userData.Country = val
		default:
			// custom
			// skip values we couldn't convert to string
			if ok {
				custom[key] = val
			}
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

func (p *Provider) evalToResolutionDetail(details configcat.EvaluationDetailsData) openfeature.ProviderResolutionDetail {
	if details.Error != nil {
		var resolutionErr openfeature.ResolutionError
		switch details.Error.(type) {
		case configcat.ErrKeyNotFound:
			resolutionErr = openfeature.NewFlagNotFoundResolutionError(details.Error.Error())
		default:
			resolutionErr = openfeature.NewGeneralResolutionError(details.Error.Error())
		}

		return openfeature.ProviderResolutionDetail{
			ResolutionError: resolutionErr,
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
