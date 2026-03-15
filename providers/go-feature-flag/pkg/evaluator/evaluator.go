package evaluator

import (
	"context"

	"github.com/open-feature/go-sdk/openfeature"
)

type Evaluator interface {
	Init(ctx context.Context) error
	Shutdown(ctx context.Context) error
	BooleanEvaluation(ctx context.Context, flag string, defaultValue bool, flatCtx openfeature.FlattenedContext) openfeature.BoolResolutionDetail
	StringEvaluation(ctx context.Context, flag string, defaultValue string, flatCtx openfeature.FlattenedContext) openfeature.StringResolutionDetail
	FloatEvaluation(ctx context.Context, flag string, defaultValue float64, flatCtx openfeature.FlattenedContext) openfeature.FloatResolutionDetail
	IntEvaluation(ctx context.Context, flag string, defaultValue int64, flatCtx openfeature.FlattenedContext) openfeature.IntResolutionDetail
	ObjectEvaluation(ctx context.Context, flag string, defaultValue any, flatCtx openfeature.FlattenedContext) openfeature.InterfaceResolutionDetail
}
