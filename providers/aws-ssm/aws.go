package awsssm

import (
	"context"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/open-feature/go-sdk/openfeature"
)

// SSMMetadataKey is the key used in flag metadata to store SSM result metadata
const SSMMetadataKey = "SSMMetadata"

type awsService struct {
	client     ssmClient
	decryption bool
}

type ssmClient interface {
	GetParameter(ctx context.Context, params *ssm.GetParameterInput, opts ...func(*ssm.Options)) (*ssm.GetParameterOutput, error)
}

func newAWSService(cfg aws.Config) (*awsService) {

	client := ssm.NewFromConfig(cfg)

	return &awsService{
		client: client,
	}
}

func (svc *awsService) ResolveBoolean(ctx context.Context, flag string, defaultValue bool, evalCtx openfeature.FlattenedContext) openfeature.BoolResolutionDetail {

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
				SSMMetadataKey: res.ResultMetadata,
			},
		},
	}

}

func (svc *awsService) ResolveString(ctx context.Context, flag string, defaultValue string, evalCtx openfeature.FlattenedContext) openfeature.StringResolutionDetail {

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
				SSMMetadataKey: res.ResultMetadata,
			},
		},
	}

}

func (svc *awsService) ResolveInt(ctx context.Context, flag string, defaultValue int64, evalCtx openfeature.FlattenedContext) openfeature.IntResolutionDetail {
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

func (svc *awsService) ResolveFloat(ctx context.Context, flag string, defaultValue float64, evalCtx openfeature.FlattenedContext) openfeature.FloatResolutionDetail {
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
				SSMMetadataKey: res.ResultMetadata,
			},
		},
	}
}

func (svc *awsService) ResolveObject(ctx context.Context, flag string, defaultValue any, evalCtx openfeature.FlattenedContext) openfeature.InterfaceResolutionDetail {
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
				SSMMetadataKey: res.ResultMetadata,
			},
		},
	}
}

func (svc *awsService) getValueFromSSM(ctx context.Context, flag string) (*ssm.GetParameterOutput, error) {

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