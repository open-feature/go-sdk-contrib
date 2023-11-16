package flipt

import (
	"context"

	flipt "go.flipt.io/flipt/rpc/flipt"
	"go.flipt.io/flipt/rpc/flipt/evaluation"
)

//go:generate mockery --name=Client --case=underscore --inpackage --filename=service_support.go --testonly --with-expecter --disable-version-string
type Client interface {
	GetFlag(ctx context.Context, c *flipt.GetFlagRequest) (*flipt.Flag, error)
	Variant(ctx context.Context, v *evaluation.EvaluationRequest) (*evaluation.VariantEvaluationResponse, error)
	Boolean(ctx context.Context, v *evaluation.EvaluationRequest) (*evaluation.BooleanEvaluationResponse, error)
}
