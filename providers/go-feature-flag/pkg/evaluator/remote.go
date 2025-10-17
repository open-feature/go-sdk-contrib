package evaluator

import (
	"context"

	"net/http"

	"github.com/open-feature/go-sdk-contrib/providers/go-feature-flag/pkg/consts"
	"github.com/open-feature/go-sdk-contrib/providers/ofrep"
	"github.com/open-feature/go-sdk/openfeature"
)

type RemoteEvaluatorOptions struct {
	Endpoint   string
	APIKey     string
	HTTPClient *http.Client
}

type RemoteEvaluator struct {
	ofrepProvider *ofrep.Provider
}

func NewRemoteEvaluator(options RemoteEvaluatorOptions) *RemoteEvaluator {
	ofrepProvider := ofrep.NewProvider(options.Endpoint, prepareOfrepOptions(options)...)
	return &RemoteEvaluator{
		ofrepProvider: ofrepProvider,
	}
}

func (e *RemoteEvaluator) Init(evaluationContext openfeature.EvaluationContext) error {
	// we don't call the init function of the ofrep provider because it panics and nothing is done there.
	return nil
}

func (e *RemoteEvaluator) Shutdown() {
	// we don't call the shutdown function of the ofrep provider.
}

func (e *RemoteEvaluator) BooleanEvaluation(ctx context.Context, flag string, defaultValue bool, flatCtx openfeature.FlattenedContext) openfeature.BoolResolutionDetail {
	return e.ofrepProvider.BooleanEvaluation(ctx, flag, defaultValue, flatCtx)
}

func (e *RemoteEvaluator) StringEvaluation(ctx context.Context, flag string, defaultValue string, flatCtx openfeature.FlattenedContext) openfeature.StringResolutionDetail {
	return e.ofrepProvider.StringEvaluation(ctx, flag, defaultValue, flatCtx)
}

func (e *RemoteEvaluator) FloatEvaluation(ctx context.Context, flag string, defaultValue float64, flatCtx openfeature.FlattenedContext) openfeature.FloatResolutionDetail {
	return e.ofrepProvider.FloatEvaluation(ctx, flag, defaultValue, flatCtx)
}

func (e *RemoteEvaluator) IntEvaluation(ctx context.Context, flag string, defaultValue int64, flatCtx openfeature.FlattenedContext) openfeature.IntResolutionDetail {
	return e.ofrepProvider.IntEvaluation(ctx, flag, defaultValue, flatCtx)
}

func (e *RemoteEvaluator) ObjectEvaluation(ctx context.Context, flag string, defaultValue any, flatCtx openfeature.FlattenedContext) openfeature.InterfaceResolutionDetail {
	return e.ofrepProvider.ObjectEvaluation(ctx, flag, defaultValue, flatCtx)
}

// prepareOfrepOptions prepares the OFREP options for the given provider options.
func prepareOfrepOptions(options RemoteEvaluatorOptions) []ofrep.Option {
	ofrepOptions := make([]ofrep.Option, 0)
	if options.APIKey != "" {
		ofrepOptions = append(ofrepOptions, ofrep.WithBearerToken(options.APIKey))
	}
	if options.HTTPClient != nil {
		ofrepOptions = append(ofrepOptions, ofrep.WithClient(options.HTTPClient))
	}
	ofrepOptions = append(ofrepOptions, ofrep.WithHeaderProvider(func() (key string, value string) {
		return consts.ContentTypeHeader, consts.ApplicationJson
	}))
	return ofrepOptions
}
