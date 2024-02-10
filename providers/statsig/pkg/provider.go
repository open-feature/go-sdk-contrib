package statsig

import (
	"context"
	"fmt"

	of "github.com/open-feature/go-sdk/openfeature"
	statsig "github.com/statsig-io/go-sdk"
)

const providerNotReady = "Provider not ready"
const generalError = "general error"

const featureConfigKey = "feature_config"

type Provider struct {
	providerConfig ProviderConfig
	status         of.State
}

func NewProvider(providerConfig ProviderConfig) (*Provider, error) {
	provider := &Provider{
		status:         of.NotReadyState,
		providerConfig: providerConfig,
	}
	return provider, nil
}

func (p *Provider) Init(evaluationContext of.EvaluationContext) {
	statsig.InitializeWithOptions(p.providerConfig.SdkKey, &p.providerConfig.Options)
	p.status = of.ReadyState
}

func (p *Provider) Status() of.State {
	return p.status
}

func (p *Provider) Shutdown() {
	statsig.Shutdown()
	p.status = of.NotReadyState
}

// provider does not have any hooks, returns empty slice
func (p *Provider) Hooks() []of.Hook {
	return []of.Hook{}
}

// Metadata returns value of Metadata (name of current service, exposed to openfeature sdk)
func (p *Provider) Metadata() of.Metadata {
	return of.Metadata{
		Name: "Statsig",
	}
}

func (p *Provider) BooleanEvaluation(ctx context.Context, flag string, defaultValue bool, evalCtx of.FlattenedContext) of.BoolResolutionDetail {

	// TODO to be removed on new SDK version adoption which includes https://github.com/open-feature/spec/issues/238
	if p.status != of.ReadyState {
		if p.status == of.NotReadyState {
			return of.BoolResolutionDetail{
				Value: defaultValue,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					ResolutionError: of.NewProviderNotReadyResolutionError(providerNotReady),
					Reason:          of.ErrorReason,
				},
			}
		} else {
			return of.BoolResolutionDetail{
				Value: defaultValue,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					ResolutionError: of.NewGeneralResolutionError(generalError),
					Reason:          of.ErrorReason,
				},
			}
		}
	}

	statsigUser, err := toStatsigUser(evalCtx)
	if err != nil {
		return of.BoolResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				ResolutionError: of.NewInvalidContextResolutionError(err.Error()),
				Reason:          of.ErrorReason,
			},
		}
	}

	res := statsig.CheckGate(*statsigUser, flag)
	return of.BoolResolutionDetail{
		Value:                    res,
		ProviderResolutionDetail: of.ProviderResolutionDetail{},
	}
}

func (p *Provider) FloatEvaluation(ctx context.Context, flag string, defaultValue float64, evalCtx of.FlattenedContext) of.FloatResolutionDetail {
	res := p.ObjectEvaluation(ctx, flag, defaultValue, evalCtx)
	return of.FloatResolutionDetail{
		Value:                    res.Value.(float64),
		ProviderResolutionDetail: res.ProviderResolutionDetail,
	}
}

func (p *Provider) IntEvaluation(ctx context.Context, flag string, defaultValue int64, evalCtx of.FlattenedContext) of.IntResolutionDetail {
	res := p.ObjectEvaluation(ctx, flag, defaultValue, evalCtx)
	return of.IntResolutionDetail{
		Value:                    res.Value.(int64),
		ProviderResolutionDetail: res.ProviderResolutionDetail,
	}
}

func (p *Provider) StringEvaluation(ctx context.Context, flag string, defaultValue string, evalCtx of.FlattenedContext) of.StringResolutionDetail {
	res := p.ObjectEvaluation(ctx, flag, defaultValue, evalCtx)
	return of.StringResolutionDetail{
		Value: fmt.Sprint(res.Value),
		ProviderResolutionDetail: of.ProviderResolutionDetail{
			ResolutionError: res.ProviderResolutionDetail.ResolutionError,
			Reason:          res.ProviderResolutionDetail.Reason,
			Variant:         res.Variant,
			FlagMetadata:    res.FlagMetadata,
		},
	}
}

func (p *Provider) ObjectEvaluation(ctx context.Context, flag string, defaultValue interface{}, evalCtx of.FlattenedContext) of.InterfaceResolutionDetail {

	// TODO to be removed on new SDK version adoption which includes https://github.com/open-feature/spec/issues/238
	if p.status != of.ReadyState {
		if p.status == of.NotReadyState {
			return of.InterfaceResolutionDetail{
				Value: defaultValue,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					ResolutionError: of.NewProviderNotReadyResolutionError(providerNotReady),
					Reason:          of.ErrorReason,
				},
			}
		} else {
			return of.InterfaceResolutionDetail{
				Value: defaultValue,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					ResolutionError: of.NewGeneralResolutionError(generalError),
					Reason:          of.ErrorReason,
				},
			}
		}
	}

	statsigUser, err := toStatsigUser(evalCtx)
	if err != nil {
		return of.InterfaceResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				ResolutionError: of.NewGeneralResolutionError(err.Error()),
				Reason:          of.ErrorReason,
			},
		}
	}

	featureConfig, err := toFeatureConfig(evalCtx)
	if err != nil {
		return of.InterfaceResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				ResolutionError: of.NewGeneralResolutionError(err.Error()),
				Reason:          of.ErrorReason,
			},
		}
	}
	if featureConfig.FeatureConfigType == CONFIG {
		config := statsig.GetConfig(*statsigUser, featureConfig.Name)
		flagMetadata := make(map[string]interface{})
		flagMetadata["GroupName"] = config.GroupName
		flagMetadata["LogExposure"] = config.LogExposure
		flagMetadata["Name"] = config.Name
		flagMetadata["RuleID"] = config.RuleID
		return of.InterfaceResolutionDetail{
			Value: config.Value[flag],
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				FlagMetadata: flagMetadata,
			},
		}
	} else if featureConfig.FeatureConfigType == LAYER {
		layer := statsig.GetLayer(*statsigUser, featureConfig.Name)
		flagMetadata := make(map[string]interface{})
		flagMetadata["GroupName"] = layer.GroupName
		flagMetadata["LogExposure"] = layer.LogExposure
		flagMetadata["Name"] = layer.Name
		flagMetadata["RuleID"] = layer.RuleID
		return of.InterfaceResolutionDetail{
			Value: layer.Value[flag],
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				FlagMetadata: flagMetadata,
			},
		}
	} else {
		return of.InterfaceResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				ResolutionError: of.NewGeneralResolutionError(fmt.Sprintf("not implemented FeatureConfigType: %s", featureConfig.FeatureConfigType)),
				Reason:          of.ErrorReason,
			},
		}
	}
}

func toStatsigUser(evalCtx of.FlattenedContext) (*statsig.User, error) {
	if len(evalCtx) == 0 {
		return &statsig.User{}, nil
	}

	statsigUser := statsig.User{}
	for key, origVal := range evalCtx {
		switch key {
		case "UserID":
			val, ok := toStr(origVal)
			if !ok {
				return nil, fmt.Errorf("key `%s` can not be converted to string", key)
			}
			statsigUser.UserID = val
		case "Email":
			val, ok := toStr(origVal)
			if !ok {
				return nil, fmt.Errorf("key `%s` can not be converted to string", key)
			}
			statsigUser.Email = val
		case "IpAddress":
			val, ok := toStr(origVal)
			if !ok {
				return nil, fmt.Errorf("key `%s` can not be converted to string", key)
			}
			statsigUser.IpAddress = val
		case "UserAgent":
			val, ok := toStr(origVal)
			if !ok {
				return nil, fmt.Errorf("key `%s` can not be converted to string", key)
			}
			statsigUser.UserAgent = val
		case "Country":
			val, ok := toStr(origVal)
			if !ok {
				return nil, fmt.Errorf("key `%s` can not be converted to string", key)
			}
			statsigUser.Country = val
		case "Locale":
			val, ok := toStr(origVal)
			if !ok {
				return nil, fmt.Errorf("key `%s` can not be converted to string", key)
			}
			statsigUser.Locale = val
		case "AppVersion":
			val, ok := toStr(origVal)
			if !ok {
				return nil, fmt.Errorf("key `%s` can not be converted to string", key)
			}
			statsigUser.AppVersion = val
		case "Custom":
			if val, ok := origVal.(map[string]interface{}); ok {
				statsigUser.Custom = val
			} else {
				return nil, fmt.Errorf("key `%s` can not be converted to map", key)
			}
		case "PrivateAttributes":
			if val, ok := origVal.(map[string]interface{}); ok {
				statsigUser.PrivateAttributes = val
			} else {
				return nil, fmt.Errorf("key `%s` can not be converted to map", key)
			}
		case "StatsigEnvironment":
			if val, ok := origVal.(map[string]string); ok {
				statsigUser.StatsigEnvironment = val
			} else {
				return nil, fmt.Errorf("key `%s` can not be converted to map", key)
			}
		case "CustomIDs":
			if val, ok := origVal.(map[string]string); ok {
				statsigUser.CustomIDs = val
			} else {
				return nil, fmt.Errorf("key `%s` can not be converted to map", key)
			}
		case featureConfigKey:
		default:
			return nil, fmt.Errorf("key `%s` is not mapped", key)
		}
	}

	return &statsigUser, nil
}

func toStr(val interface{}) (string, bool) {
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

type FeatureConfigType string

const (
	CONFIG FeatureConfigType = "CONFIG" // Dynamic Config
	LAYER  FeatureConfigType = "LAYER"  // Layer
)

type FeatureConfig struct {
	FeatureConfigType FeatureConfigType
	Name              string
}

func toFeatureConfig(evalCtx of.FlattenedContext) (*FeatureConfig, error) {
	if len(evalCtx) == 0 {
		return &FeatureConfig{}, nil
	}

	// featureConfig := &FeatureConfig{}
	featureConfig, ok := evalCtx[featureConfigKey].(FeatureConfig)
	if !ok {
		return nil, fmt.Errorf("`%s` not found at evaluation context.", featureConfigKey)
	}

	return &featureConfig, nil
}
