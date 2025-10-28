package evaluator

import (
	"context"

	"github.com/open-feature/go-sdk-contrib/providers/go-feature-flag/pkg/api"
	"github.com/open-feature/go-sdk/openfeature"
)

type InProcessEvaluator struct {
}

func NewInProcessEvaluator(api api.GoffAPI) *InProcessEvaluator {
	return &InProcessEvaluator{}
}

// Init initializes the evaluation context
func (e *InProcessEvaluator) Init(evaluationContext openfeature.EvaluationContext) error {
	return nil
}

// Shutdown shuts down the evaluation context
func (e *InProcessEvaluator) Shutdown() {
}

// BooleanEvaluation evaluates a boolean flag
func (e *InProcessEvaluator) BooleanEvaluation(ctx context.Context, flag string, defaultValue bool, flatCtx openfeature.FlattenedContext) openfeature.BoolResolutionDetail {
	return openfeature.BoolResolutionDetail{
		Value: defaultValue,
	}
}

// FloatEvaluation evaluates a float flag
func (e *InProcessEvaluator) FloatEvaluation(ctx context.Context, flag string, defaultValue float64, flatCtx openfeature.FlattenedContext) openfeature.FloatResolutionDetail {
	return openfeature.FloatResolutionDetail{
		Value: defaultValue,
	}
}

// IntEvaluation evaluates an int flag
func (e *InProcessEvaluator) IntEvaluation(ctx context.Context, flag string, defaultValue int64, flatCtx openfeature.FlattenedContext) openfeature.IntResolutionDetail {
	return openfeature.IntResolutionDetail{
		Value: defaultValue,
	}
}

// ObjectEvaluation evaluates an object flag
func (e *InProcessEvaluator) ObjectEvaluation(ctx context.Context, flag string, defaultValue any, flatCtx openfeature.FlattenedContext) openfeature.InterfaceResolutionDetail {
	return openfeature.InterfaceResolutionDetail{
		Value: defaultValue,
	}
}
