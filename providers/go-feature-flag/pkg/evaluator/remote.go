package evaluator

import (
	"context"
	"net/http"

	"github.com/open-feature/go-sdk-contrib/providers/ofrep"
	"github.com/open-feature/go-sdk/openfeature"
)

var _ Evaluator = &Remote{}

type Remote struct {
	ofrepProvider *ofrep.Provider
}

func NewRemoteEvaluator(baseUri string, httpClient *http.Client, apiKey string, headers map[string]string) *Remote {
	ofrepOptions := []ofrep.Option{}
	if httpClient != nil {
		ofrepOptions = append(ofrepOptions, ofrep.WithClient(httpClient))
	}
	if apiKey != "" {
		ofrepOptions = append(ofrepOptions, ofrep.WithApiKeyAuth(apiKey))
	}
	for k, v := range headers {
		ofrepOptions = append(ofrepOptions, ofrep.WithHeader(k, v))
	}
	return &Remote{
		ofrepProvider: ofrep.NewProvider(baseUri, ofrepOptions...),
	}
}

func (r *Remote) Init(_ context.Context) error {
	return nil
}

func (r *Remote) Shutdown(_ context.Context) error {
	return nil
}

func (r *Remote) BooleanEvaluation(ctx context.Context, flag string, defaultValue bool, flatCtx openfeature.FlattenedContext) openfeature.BoolResolutionDetail {
	return r.ofrepProvider.BooleanEvaluation(ctx, flag, defaultValue, flatCtx)
}

func (r *Remote) StringEvaluation(ctx context.Context, flag string, defaultValue string, flatCtx openfeature.FlattenedContext) openfeature.StringResolutionDetail {
	return r.ofrepProvider.StringEvaluation(ctx, flag, defaultValue, flatCtx)
}

func (r *Remote) FloatEvaluation(ctx context.Context, flag string, defaultValue float64, flatCtx openfeature.FlattenedContext) openfeature.FloatResolutionDetail {
	return r.ofrepProvider.FloatEvaluation(ctx, flag, defaultValue, flatCtx)
}

func (r *Remote) IntEvaluation(ctx context.Context, flag string, defaultValue int64, flatCtx openfeature.FlattenedContext) openfeature.IntResolutionDetail {
	return r.ofrepProvider.IntEvaluation(ctx, flag, defaultValue, flatCtx)
}

func (r *Remote) ObjectEvaluation(ctx context.Context, flag string, defaultValue any, flatCtx openfeature.FlattenedContext) openfeature.InterfaceResolutionDetail {
	return r.ofrepProvider.ObjectEvaluation(ctx, flag, defaultValue, flatCtx)
}
