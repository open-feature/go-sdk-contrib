package service

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/open-feature/go-sdk/openfeature"
)


type AWS struct{
	client	*ssm.Client
	decryption bool
}


func NewAWSService() (*AWS, error){

	cfg, err := config.LoadDefaultConfig(context.TODO())

    if err != nil {
    	return nil, fmt.Errorf("could not initialize aws config : %v+", err)
    }

    client := ssm.NewFromConfig(cfg)


    return &AWS{
   		client : client,
    }, nil
}

func (svc *AWS) ResolveBoolean(ctx context.Context, flag string, defaultValue bool, evalCtx openfeature.FlattenedContext) openfeature.BoolResolutionDetail {

}


func (svc *AWS) ResolveString(ctx context.Context, flag string, defaultValue string, evalCtx openfeature.FlattenedContext) openfeature.StringResolutionDetail {

	param := &ssm.GetParameterInput{
		Name: &flag,
		WithDecryption: &svc.decryption,
	}

	res, err := svc.client.GetParameter( ctx, param)

	if err != nil {
		return openfeature.StringResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
				Reason: openfeature.ErrorReason,
				ResolutionError: openfeature.NewGeneralResolutionError(err.Error()),
			},
		}
	}

	return nil


}


func (svc *AWS) ResolveInt(ctx context.Context, flag string, defaultValue int64, evalCtx openfeature.FlattenedContext) openfeature.IntResolutionDetail {

}


func (svc *AWS) ResolveFloat(ctx context.Context, flag string, defaultValue float64, evalCtx openfeature.FlattenedContext) openfeature.FloatResolutionDetail {

}



func (svc *AWS) ResolveObject(ctx context.Context, flag string, defaultValue interface{}, evalCtx openfeature.FlattenedContext) openfeature.InterfaceResolutionDetail {

}


func (svc *AWS) WithDecryption(decryption bool) *AWS {
	svc.decryption = decryption
	return svc
}
