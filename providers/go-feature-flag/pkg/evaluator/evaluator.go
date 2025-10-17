package evaluator

import (
	"context"

	"github.com/open-feature/go-sdk/openfeature"
)

type EvaluatorInterface interface {
	// Init initializes the evaluation context
	Init(evaluationContext openfeature.EvaluationContext) error

	// Shutdown shuts down the evaluation context
	Shutdown()

	// BooleanEvaluation evaluates a boolean flag
	BooleanEvaluation(
		ctx context.Context,
		flag string, defaultValue bool,
		flatCtx openfeature.FlattenedContext,
	) openfeature.BoolResolutionDetail

	// StringEvaluation evaluates a string flag
	StringEvaluation(
		ctx context.Context,
		flag string,
		defaultValue string,
		flatCtx openfeature.FlattenedContext,
	) openfeature.StringResolutionDetail

	// FloatEvaluation evaluates a float flag
	FloatEvaluation(
		ctx context.Context,
		flag string,
		defaultValue float64,
		flatCtx openfeature.FlattenedContext,
	) openfeature.FloatResolutionDetail

	// IntEvaluation evaluates an int flag
	IntEvaluation(
		ctx context.Context,
		flag string,
		defaultValue int64,
		flatCtx openfeature.FlattenedContext,
	) openfeature.IntResolutionDetail

	// ObjectEvaluation evaluates an object flag
	ObjectEvaluation(
		ctx context.Context,
		flag string,
		defaultValue any,
		flatCtx openfeature.FlattenedContext,
	) openfeature.InterfaceResolutionDetail
}
