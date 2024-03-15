package ofrep

import (
	"context"

	"github.com/open-feature/go-sdk-contrib/providers/ofrep/internal/evaluate"
	"github.com/open-feature/go-sdk-contrib/providers/ofrep/internal/outbound"
	"github.com/open-feature/go-sdk/openfeature"
)

// Configuration of the OFREP provider
type Configuration struct {
	BasePath           string
	AuthHeaderProvider outbound.AuthCallback
}

// Provider implementation for OFREP
type Provider struct {
	evaluator Evaluator
}

// NewProvider returns a provider configured with provided Configuration
func NewProvider(cfg Configuration) *Provider {
	provider := &Provider{
		evaluator: evaluate.NewFlagsEvaluator(cfg.BasePath, cfg.AuthHeaderProvider),
	}

	return provider
}

func (p Provider) Metadata() openfeature.Metadata {

	return openfeature.Metadata{
		Name: "OFREP provider",
	}
}

func (p Provider) BooleanEvaluation(ctx context.Context, flag string, defaultValue bool, evalCtx openfeature.FlattenedContext) openfeature.BoolResolutionDetail {
	return p.evaluator.ResolveBoolean(ctx, flag, defaultValue, evalCtx)
}

func (p Provider) StringEvaluation(ctx context.Context, flag string, defaultValue string, evalCtx openfeature.FlattenedContext) openfeature.StringResolutionDetail {
	return p.evaluator.ResolveString(ctx, flag, defaultValue, evalCtx)
}

func (p Provider) FloatEvaluation(ctx context.Context, flag string, defaultValue float64, evalCtx openfeature.FlattenedContext) openfeature.FloatResolutionDetail {
	return p.evaluator.ResolveFloat(ctx, flag, defaultValue, evalCtx)
}

func (p Provider) IntEvaluation(ctx context.Context, flag string, defaultValue int64, evalCtx openfeature.FlattenedContext) openfeature.IntResolutionDetail {
	return p.evaluator.ResolveInt(ctx, flag, defaultValue, evalCtx)
}

func (p Provider) ObjectEvaluation(ctx context.Context, flag string, defaultValue interface{}, evalCtx openfeature.FlattenedContext) openfeature.InterfaceResolutionDetail {
	return p.evaluator.ResolveObject(ctx, flag, defaultValue, evalCtx)
}

func (p Provider) Hooks() []openfeature.Hook {
	return []openfeature.Hook{}
}
