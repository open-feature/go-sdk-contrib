package awsssm

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/open-feature/go-sdk/openfeature"
)

type Provider struct {
	svc *awsService
}

type ProviderOption func(*Provider)


func NewProvider(cfg aws.Config, opts ...ProviderOption) (*Provider, error) {
	svc, err := newAWSService(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize AWS SSM provider: %w", err)
	}

	return &Provider{
		svc: svc,
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

func (p *Provider) BooleanEvaluation(ctx context.Context, flag string, defaultValue bool, flatCtx openfeature.FlattenedContext) openfeature.BoolResolutionDetail {
	return p.svc.ResolveBoolean(ctx, flag, defaultValue, flatCtx)
}

func (p *Provider) StringEvaluation(ctx context.Context, flag string, defaultValue string, flatCtx openfeature.FlattenedContext) openfeature.StringResolutionDetail {
	return p.svc.ResolveString(ctx, flag, defaultValue, flatCtx)
}

func (p *Provider) FloatEvaluation(ctx context.Context, flag string, defaultValue float64, flatCtx openfeature.FlattenedContext) openfeature.FloatResolutionDetail {
	return p.svc.ResolveFloat(ctx, flag, defaultValue, flatCtx)
}

func (p *Provider) IntEvaluation(ctx context.Context, flag string, defaultValue int64, flatCtx openfeature.FlattenedContext) openfeature.IntResolutionDetail {
	return p.svc.ResolveInt(ctx, flag, defaultValue, flatCtx)
}

func (p *Provider) ObjectEvaluation(ctx context.Context, flag string, defaultValue any, flatCtx openfeature.FlattenedContext) openfeature.InterfaceResolutionDetail {
	return p.svc.ResolveObject(ctx, flag, defaultValue, flatCtx)
}
