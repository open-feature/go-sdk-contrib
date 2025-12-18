package configcat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	sdk "github.com/configcat/go-sdk/v9"
	"go.openfeature.dev/openfeature/v2"
)

var (
	_ openfeature.FeatureProvider = (*Provider)(nil)
	_ sdk.UserAttributes          = (*userAttributes)(nil)
)

// Evaluation ctx keys that are mapped to ConfigCat user data.
const (
	IdentifierKey = openfeature.TargetingKey
	EmailKey      = "email"
	CountryKey    = "country"
)

type userAttributes struct {
	attributes map[string]any
}

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
	evaluation := p.client.GetBoolValueDetails(flag, defaultValue, toUserData(evalCtx))
	return openfeature.BoolResolutionDetail{
		Value:                    evaluation.Value,
		ProviderResolutionDetail: toResolutionDetail(evaluation.Data),
	}
}

func (p *Provider) StringEvaluation(ctx context.Context, flag string, defaultValue string, evalCtx openfeature.FlattenedContext) openfeature.StringResolutionDetail {
	evaluation := p.client.GetStringValueDetails(flag, defaultValue, toUserData(evalCtx))
	return openfeature.StringResolutionDetail{
		Value:                    evaluation.Value,
		ProviderResolutionDetail: toResolutionDetail(evaluation.Data),
	}
}

func (p *Provider) FloatEvaluation(ctx context.Context, flag string, defaultValue float64, evalCtx openfeature.FlattenedContext) openfeature.FloatResolutionDetail {
	evaluation := p.client.GetFloatValueDetails(flag, defaultValue, toUserData(evalCtx))
	return openfeature.FloatResolutionDetail{
		Value:                    evaluation.Value,
		ProviderResolutionDetail: toResolutionDetail(evaluation.Data),
	}
}

func (p *Provider) IntEvaluation(ctx context.Context, flag string, defaultValue int64, evalCtx openfeature.FlattenedContext) openfeature.IntResolutionDetail {
	evaluation := p.client.GetIntValueDetails(flag, int(defaultValue), toUserData(evalCtx))
	return openfeature.IntResolutionDetail{
		Value:                    int64(evaluation.Value),
		ProviderResolutionDetail: toResolutionDetail(evaluation.Data),
	}
}

// ObjectEvaluation attempts to parse a string feature flag value as JSON.
func (p *Provider) ObjectEvaluation(ctx context.Context, flag string, defaultValue any, evalCtx openfeature.FlattenedContext) openfeature.ObjectResolutionDetail {
	evaluation := p.client.GetStringValueDetails(flag, "", toUserData(evalCtx))
	if evaluation.Data.IsDefaultValue || evaluation.Data.Error != nil {
		// we evaluated with a fake default value, so we
		// need to use the one we were provided
		return openfeature.ObjectResolutionDetail{
			Value:                    defaultValue,
			ProviderResolutionDetail: toResolutionDetail(evaluation.Data),
		}
	}

	// Attempt to unmarshal the string value as if it's JSON
	var object map[string]any
	err := json.Unmarshal([]byte(evaluation.Value), &object)
	if err != nil {
		return openfeature.ObjectResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
				ResolutionError: openfeature.NewTypeMismatchResolutionError(
					fmt.Sprintf("failed to unmarshal string flag as json: %s", err),
				),
				Reason: openfeature.ErrorReason,
			},
		}
	}

	return openfeature.ObjectResolutionDetail{
		Value:                    object,
		ProviderResolutionDetail: toResolutionDetail(evaluation.Data),
	}
}

func (u *userAttributes) GetAttribute(key string) any {
	return u.attributes[key]
}

func toUserData(evalCtx openfeature.FlattenedContext) sdk.User {
	if len(evalCtx) == 0 {
		return nil
	}

	attributes := make(map[string]any, len(evalCtx))
	for key, val := range evalCtx {
		switch key {
		case IdentifierKey:
			attributes["Identifier"] = val
		case EmailKey:
			attributes["Email"] = val
		case CountryKey:
			attributes["Country"] = val
		default:
			attributes[key] = val
		}
	}
	return &userAttributes{attributes: attributes}
}

func toResolutionDetail(details sdk.EvaluationDetailsData) openfeature.ProviderResolutionDetail {
	if details.Error != nil {
		return openfeature.ProviderResolutionDetail{
			ResolutionError: toResolutionError(details.Error),
			Reason:          openfeature.ErrorReason,
		}
	}

	reason := openfeature.DefaultReason
	if details.MatchedTargetingRule != nil || details.MatchedPercentageOption != nil {
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

	var errTypeMismatch sdk.ErrSettingTypeMismatch
	if errors.As(err, &errTypeMismatch) {
		return openfeature.NewTypeMismatchResolutionError(errTypeMismatch.Error())
	}

	var errConfigJsonMissing sdk.ErrConfigJsonMissing
	if errors.As(err, &errConfigJsonMissing) {
		return openfeature.NewParseErrorResolutionError(errConfigJsonMissing.Error())
	}

	return openfeature.NewGeneralResolutionError(err.Error())
}
