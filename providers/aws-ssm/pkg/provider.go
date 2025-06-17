package awsssm

import (
	"context"
	"fmt"

	"github.com/open-feature/go-sdk-contrib/providers/aws-ssm/pkg/service"
	"github.com/open-feature/go-sdk/openfeature"
)


const providerName = "AWS SSM"

type Provider struct {
	svc *service.AWS
}


func NewProvider(opts ProviderOptions) (*Provider, error) {

	svc, err := service.NewAWSService()

	if err != nil {
		return nil, fmt.Errorf("could not inizialize provider: %v+", err)
	}

	return &Provider{
		svc : svc,
	}, nil

}


func (p *Provider) Metadata() openfeature.Metadata {
	return openfeature.Metadata{
		Name: "AWS System Manager Provider",
	}
}

func (p *Provider) Hooks() []openfeature.Hook {
	return []openfeature.Hook{}
}


func (p Provider) BooleanEvaluation(ctx context.Context, flag string, defaultValue bool, evalCtx openfeature.FlattenedContext) openfeature.BoolResolutionDetail {
	return p.svc.ResolveBoolean(ctx, flag, defaultValue, evalCtx)
}

func (p Provider) StringEvaluation(ctx context.Context, flag string, defaultValue string, evalCtx openfeature.FlattenedContext) openfeature.StringResolutionDetail {
	return p.svc.ResolveString(ctx, flag, defaultValue, evalCtx)
}

func (p Provider) FloatEvaluation(ctx context.Context, flag string, defaultValue float64, evalCtx openfeature.FlattenedContext) openfeature.FloatResolutionDetail {
	return p.svc.ResolveFloat(ctx, flag, defaultValue, evalCtx)
}

func (p Provider) IntEvaluation(ctx context.Context, flag string, defaultValue int64, evalCtx openfeature.FlattenedContext) openfeature.IntResolutionDetail {
	return p.svc.ResolveInt(ctx, flag, defaultValue, evalCtx)
}

func (p Provider) ObjectEvaluation(ctx context.Context, flag string, defaultValue interface{}, evalCtx openfeature.FlattenedContext) openfeature.InterfaceResolutionDetail {
	return p.svc.ResolveObject(ctx, flag, defaultValue, evalCtx)
}
