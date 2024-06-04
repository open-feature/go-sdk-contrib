package statsig

import (
	"context"
	"fmt"
	"reflect"

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

	statsigUser, err := ToStatsigUser(evalCtx)
	if err != nil {
		return of.BoolResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				ResolutionError: of.NewInvalidContextResolutionError(err.Error()),
				Reason:          of.ErrorReason,
			},
		}
	}

	_, err = toFeatureConfig(evalCtx)
	if err != nil {
		res := statsig.GetGate(*statsigUser, flag)
		return of.BoolResolutionDetail{
			Value:                    res.Value,
			ProviderResolutionDetail: of.ProviderResolutionDetail{},
		}
	}

	res := p.ObjectEvaluation(ctx, flag, defaultValue, evalCtx)
	if v, ok := res.Value.(bool); ok {
		return of.BoolResolutionDetail{
			Value:                    v,
			ProviderResolutionDetail: res.ProviderResolutionDetail,
		}
	}
	return of.BoolResolutionDetail{
		Value: defaultValue,
		ProviderResolutionDetail: of.ProviderResolutionDetail{
			ResolutionError: of.NewGeneralResolutionError("evaluated value is from incompatible type"),
			Reason:          of.ErrorReason,
		},
	}
}

func (p *Provider) FloatEvaluation(ctx context.Context, flag string, defaultValue float64, evalCtx of.FlattenedContext) of.FloatResolutionDetail {
	res := p.ObjectEvaluation(ctx, flag, defaultValue, evalCtx)
	if v, ok := res.Value.(float64); ok {
		return of.FloatResolutionDetail{
			Value:                    v,
			ProviderResolutionDetail: res.ProviderResolutionDetail,
		}
	}
	return of.FloatResolutionDetail{
		Value: defaultValue,
		ProviderResolutionDetail: of.ProviderResolutionDetail{
			ResolutionError: of.NewGeneralResolutionError("evaluated value is from incompatible type"),
			Reason:          of.ErrorReason,
		},
	}
}

func (p *Provider) IntEvaluation(ctx context.Context, flag string, defaultValue int64, evalCtx of.FlattenedContext) of.IntResolutionDetail {
	res := p.ObjectEvaluation(ctx, flag, defaultValue, evalCtx)

	// statsig evaluator treats int as float64
	// https://github.com/statsig-io/go-sdk/blob/5af41eea1f4729a0571147f9ea188378e47c1d42/evaluator.go#L614
	if v, ok := res.Value.(float64); ok {
		return of.IntResolutionDetail{
			Value:                    int64(v),
			ProviderResolutionDetail: res.ProviderResolutionDetail,
		}
	}
	if v, ok := res.Value.(int64); ok {
		return of.IntResolutionDetail{
			Value:                    v,
			ProviderResolutionDetail: res.ProviderResolutionDetail,
		}
	}
	return of.IntResolutionDetail{
		Value: defaultValue,
		ProviderResolutionDetail: of.ProviderResolutionDetail{
			ResolutionError: of.NewGeneralResolutionError("evaluated value is from incompatible type"),
			Reason:          of.ErrorReason,
		},
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

	statsigUser, err := ToStatsigUser(evalCtx)
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
	var value interface{}
	flagMetadata := make(map[string]interface{})
	if featureConfig.FeatureConfigType == CONFIG {
		config := statsig.GetConfig(*statsigUser, featureConfig.Name)
		defaultValueV := reflect.ValueOf(defaultValue)
		switch defaultValueV.Kind() {
		case reflect.Bool:
			value = config.GetBool(flag, defaultValueV.Bool())
		case reflect.Int, reflect.Int8, reflect.Int32, reflect.Int64:
			value = config.GetNumber(flag, float64(defaultValueV.Int()))
		case reflect.Float32, reflect.Float64:
			value = config.GetNumber(flag, defaultValueV.Float())
		case reflect.String:
			value = config.GetString(flag, defaultValueV.String())
		case reflect.Array, reflect.Slice:
			sliceDefaultValue, _ := defaultValueV.Interface().([]interface{})
			value = config.GetSlice(flag, sliceDefaultValue)
		case reflect.Map:
			mapValue, ok := defaultValueV.Interface().(map[string]interface{})
			if !ok {
				return of.InterfaceResolutionDetail{
					Value: defaultValue,
					ProviderResolutionDetail: of.ProviderResolutionDetail{
						ResolutionError: of.NewGeneralResolutionError(fmt.Sprintf("default value is from unexpected type: %s", defaultValueV.Kind())),
						Reason:          of.ErrorReason,
					},
				}
			}
			value = config.GetMap(flag, mapValue)
		default:
			return of.InterfaceResolutionDetail{
				Value: defaultValue,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					ResolutionError: of.NewGeneralResolutionError(fmt.Sprintf("not implemented default value type: %s", defaultValueV.Kind())),
					Reason:          of.ErrorReason,
				},
			}
		}
		flagMetadata["GroupName"] = config.GroupName
		flagMetadata["LogExposure"] = config.LogExposure
		flagMetadata["Name"] = config.Name
		flagMetadata["RuleID"] = config.RuleID
	} else if featureConfig.FeatureConfigType == LAYER {
		layer := statsig.GetLayer(*statsigUser, featureConfig.Name)
		defaultValueV := reflect.ValueOf(defaultValue)
		switch defaultValueV.Kind() {
		case reflect.Bool:
			value = layer.GetBool(flag, defaultValueV.Bool())
		case reflect.Int, reflect.Int8, reflect.Int32, reflect.Int64:
			value = layer.GetNumber(flag, float64(defaultValueV.Int()))
		case reflect.Float32, reflect.Float64:
			value = layer.GetNumber(flag, defaultValueV.Float())
		case reflect.String:
			value = layer.GetString(flag, defaultValueV.String())
		case reflect.Array, reflect.Slice:
			sliceDefaultValue, _ := defaultValueV.Interface().([]interface{})
			value = layer.GetSlice(flag, sliceDefaultValue)
		case reflect.Map:
			mapValue, ok := defaultValueV.Interface().(map[string]interface{})
			if !ok {
				return of.InterfaceResolutionDetail{
					Value: defaultValue,
					ProviderResolutionDetail: of.ProviderResolutionDetail{
						ResolutionError: of.NewGeneralResolutionError(fmt.Sprintf("default value is from unexpected type: %s", defaultValueV.Kind())),
						Reason:          of.ErrorReason,
					},
				}
			}
			value = layer.GetMap(flag, mapValue)
		default:
			return of.InterfaceResolutionDetail{
				Value: defaultValue,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					ResolutionError: of.NewGeneralResolutionError(fmt.Sprintf("not implemented default value type: %s", defaultValueV.Kind())),
					Reason:          of.ErrorReason,
				},
			}
		}
		flagMetadata["GroupName"] = layer.GroupName
		flagMetadata["LogExposure"] = layer.LogExposure
		flagMetadata["Name"] = layer.Name
		flagMetadata["RuleID"] = layer.RuleID
	} else {
		return of.InterfaceResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				ResolutionError: of.NewGeneralResolutionError(fmt.Sprintf("not implemented FeatureConfigType: %s", featureConfig.FeatureConfigType)),
				Reason:          of.ErrorReason,
			},
		}
	}

	return of.InterfaceResolutionDetail{
		Value: value,
		ProviderResolutionDetail: of.ProviderResolutionDetail{
			FlagMetadata: flagMetadata,
		},
	}
}

func ToStatsigUser(evalCtx of.FlattenedContext) (*statsig.User, error) {
	if len(evalCtx) == 0 {
		return &statsig.User{}, nil
	}

	statsigUser := statsig.User{}
	for key, origVal := range evalCtx {
		switch key {
		case of.TargetingKey, "UserID":
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
				if statsigUser.Custom == nil {
					statsigUser.Custom = val
				} else {
					for k, v := range val {
						statsigUser.Custom[k] = v
					}
				}
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
			if statsigUser.Custom == nil {
				statsigUser.Custom = make(map[string]interface{})
			}
			statsigUser.Custom[key] = origVal
		}
	}
	if statsigUser.UserID == "" {
		return nil, of.NewTargetingKeyMissingResolutionError("UserID/targetingKey is missing")
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
		return nil, fmt.Errorf("`%s` not found at evaluation context.", featureConfigKey)
	}

	featureConfig, ok := evalCtx[featureConfigKey].(FeatureConfig)
	if !ok {
		return nil, fmt.Errorf("`%s` not found at evaluation context.", featureConfigKey)
	}

	return &featureConfig, nil
}
