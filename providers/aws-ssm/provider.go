package awsssm

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"go.openfeature.dev/openfeature/v2"
)

type Provider struct {
	svc *awsService
}

var _ openfeature.FeatureProvider = (*Provider)(nil)

type ProviderOption func(*Provider)

func NewProvider(cfg aws.Config, opts ...ProviderOption) (*Provider, error) {
	p := &Provider{
		svc: newAWSService(cfg),
	}

	for _, opt := range opts {
		opt(p)
	}

	return p, nil
}

func (p *Provider) Metadata() openfeature.Metadata {
	return openfeature.Metadata{
		Name: "AWS System Manager Provider",
	}
}

func (p *Provider) Hooks() []openfeature.Hook {
	return []openfeature.Hook{}
}

func WithDecryption() ProviderOption {
	return func(p *Provider) {
		p.svc.decryption = true
	}
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

func (p *Provider) ObjectEvaluation(ctx context.Context, flag string, defaultValue any, flatCtx openfeature.FlattenedContext) openfeature.ObjectResolutionDetail {
	return p.svc.ResolveObject(ctx, flag, defaultValue, flatCtx)
}
