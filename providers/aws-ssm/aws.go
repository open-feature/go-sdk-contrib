package awsssm

import (
	"context"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/open-feature/go-sdk/openfeature"
)

type AWS struct {
	client     SSMClient
	decryption bool
}

type SSMClient interface {
	GetParameter(ctx context.Context, params *ssm.GetParameterInput, opts ...func(*ssm.Options)) (*ssm.GetParameterOutput, error)
}

func NewAWSService(cfg aws.Config) (*AWS, error) {

	client := ssm.NewFromConfig(cfg)

	return &AWS{
		client: client,
	}, nil
}

func (svc *AWS) ResolveBoolean(ctx context.Context, flag string, defaultValue bool, evalCtx openfeature.FlattenedContext) openfeature.BoolResolutionDetail {

	res, err := svc.getValueFromSSM(ctx, flag)

	if err != nil {
		return openfeature.BoolResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
				Reason:          openfeature.ErrorReason,
				ResolutionError: openfeature.NewGeneralResolutionError(err.Error()),
			},
		}
	}

	b, err := strconv.ParseBool(*res.Parameter.Value)

	if err != nil {
		return openfeature.BoolResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
				Reason:          openfeature.ErrorReason,
				ResolutionError: openfeature.NewGeneralResolutionError(err.Error()),
			},
		}
	}

	return openfeature.BoolResolutionDetail{
		Value: b,
		ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
			Reason: openfeature.StaticReason,
			FlagMetadata: openfeature.FlagMetadata{
				"SSMMetadata": res.ResultMetadata,
			},
		},
	}

}

func (svc *AWS) ResolveString(ctx context.Context, flag string, defaultValue string, evalCtx openfeature.FlattenedContext) openfeature.StringResolutionDetail {

	res, err := svc.getValueFromSSM(ctx, flag)

	if err != nil {
		return openfeature.StringResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
				Reason:          openfeature.ErrorReason,
				ResolutionError: openfeature.NewGeneralResolutionError(err.Error()),
			},
		}
	}

	return openfeature.StringResolutionDetail{
		Value: *res.Parameter.Value,
		ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
			Reason: openfeature.StaticReason,
			FlagMetadata: openfeature.FlagMetadata{
				"SSMMetadata": res.ResultMetadata,
			},
		},
	}

}

func (svc *AWS) ResolveInt(ctx context.Context, flag string, defaultValue int64, evalCtx openfeature.FlattenedContext) openfeature.IntResolutionDetail {
	res, err := svc.getValueFromSSM(ctx, flag)

	if err != nil {
		return openfeature.IntResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
				Reason:          openfeature.ErrorReason,
				ResolutionError: openfeature.NewGeneralResolutionError(err.Error()),
			},
		}
	}

	i, err := strconv.ParseInt(*res.Parameter.Value, 10, 64)

	if err != nil {
		return openfeature.IntResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
				Reason:          openfeature.ErrorReason,
				ResolutionError: openfeature.NewGeneralResolutionError(err.Error()),
			},
		}
	}

	return openfeature.IntResolutionDetail{
		Value: i,
		ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
			Reason: openfeature.StaticReason,
			FlagMetadata: openfeature.FlagMetadata{
				"SSMMetadata": res.ResultMetadata,
			},
		},
	}
}

func (svc *AWS) ResolveFloat(ctx context.Context, flag string, defaultValue float64, evalCtx openfeature.FlattenedContext) openfeature.FloatResolutionDetail {
	res, err := svc.getValueFromSSM(ctx, flag)

	if err != nil {
		return openfeature.FloatResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
				Reason:          openfeature.ErrorReason,
				ResolutionError: openfeature.NewGeneralResolutionError(err.Error()),
			},
		}
	}

	f, err := strconv.ParseFloat(*res.Parameter.Value, 64)

	if err != nil {
		return openfeature.FloatResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
				Reason:          openfeature.ErrorReason,
				ResolutionError: openfeature.NewGeneralResolutionError(err.Error()),
			},
		}
	}

	return openfeature.FloatResolutionDetail{
		Value: f,
		ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
			Reason: openfeature.StaticReason,
			FlagMetadata: openfeature.FlagMetadata{
				"SSMMetadata": res.ResultMetadata,
			},
		},
	}
}

func (svc *AWS) ResolveObject(ctx context.Context, flag string, defaultValue any, evalCtx openfeature.FlattenedContext) openfeature.InterfaceResolutionDetail {
	res, err := svc.getValueFromSSM(ctx, flag)

	if err != nil {
		return openfeature.InterfaceResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
				Reason:          openfeature.ErrorReason,
				ResolutionError: openfeature.NewGeneralResolutionError(err.Error()),
			},
		}
	}

	return openfeature.InterfaceResolutionDetail{
		Value: defaultValue,
		ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
			Reason: openfeature.StaticReason,
			FlagMetadata: openfeature.FlagMetadata{
				"SSMMetadata": res.ResultMetadata,
			},
		},
	}
}

func (svc *AWS) WithDecryption(decryption bool) *AWS {
	svc.decryption = decryption
	return svc
}

func (svc *AWS) getValueFromSSM(ctx context.Context, flag string) (*ssm.GetParameterOutput, error) {

	param := &ssm.GetParameterInput{
		Name:           &flag,
		WithDecryption: &svc.decryption,
	}

	res, err := svc.client.GetParameter(ctx, param)

	if err != nil {
		return nil, err
	}

	return res, nil
}
