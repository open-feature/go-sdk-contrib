package flipt

import (
	"context"

	"go.flipt.io/flipt/rpc/flipt/evaluation"
)

//go:generate mockery --name=Client --case=underscore --inpackage --filename=service_mock.go --testonly --with-expecter --disable-version-string
type Client interface {
	Variant(ctx context.Context, v *evaluation.EvaluationRequest) (*evaluation.VariantEvaluationResponse, error)
	Boolean(ctx context.Context, v *evaluation.EvaluationRequest) (*evaluation.BooleanEvaluationResponse, error)
}
