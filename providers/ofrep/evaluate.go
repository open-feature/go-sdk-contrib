package ofrep

import (
	"context"

	of "go.openfeature.dev/openfeature/v2"
)

// Evaluator contract for flag evaluation
type Evaluator interface {
	ResolveBoolean(ctx context.Context, key string, defaultValue bool,
		evalCtx map[string]any) of.BoolResolutionDetail
	ResolveString(ctx context.Context, key string, defaultValue string,
		evalCtx map[string]any) of.StringResolutionDetail
	ResolveFloat(ctx context.Context, key string, defaultValue float64,
		evalCtx map[string]any) of.FloatResolutionDetail
	ResolveInt(ctx context.Context, key string, defaultValue int64,
		evalCtx map[string]any) of.IntResolutionDetail
	ResolveObject(ctx context.Context, key string, defaultValue any,
		evalCtx map[string]any) of.ObjectResolutionDetail
}
