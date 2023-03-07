package posthog

import (
	"context"
	"errors"

	"github.com/open-feature/go-sdk/pkg/openfeature"
	"github.com/posthog/posthog-go"
)

const name = "posthog"

var (
	ErrApiKeyRequired = errors.New("api key is required")
)

// Force Provider to implement the FeatureProvider interface.
var _ openfeature.FeatureProvider = &Provider{}

// Provider implements the FeatureProvider interface and provides functions for evaluating flags.
type Provider struct {
	apiKey string
	config posthog.Config
	client posthog.Client
}

// NewProvider creates a new Posthog Provider.
// Either WithAPIKey or WithClient must be provided.
func NewProvider(opts ...ProviderOption) (*Provider, error) {
	p := &Provider{}

	for _, opt := range opts {
		opt(p)
	}

	// if client is already set, don't create a new one
	if p.client != nil {
		return p, nil
	}

	if p.apiKey == "" {
		return nil, ErrApiKeyRequired
	}

	if p.config.PersonalApiKey == "" {
		// see https://github.com/PostHog/posthog/issues/4849
		p.config.PersonalApiKey = p.apiKey
	}

	client, err := posthog.NewWithConfig(p.apiKey, p.config)
	if err != nil {
		return nil, err
	}

	p.client = client

	return p, err
}

// BooleanEvaluation implements openfeature.FeatureProvider.
func (p *Provider) BooleanEvaluation(ctx context.Context, flag string, defaultValue bool, evalCtx openfeature.FlattenedContext) openfeature.BoolResolutionDetail {
	val, detail := evaluate(p.client, flag, defaultValue, evalCtx)

	return openfeature.BoolResolutionDetail{
		Value:                    val,
		ProviderResolutionDetail: detail,
	}
}

// FloatEvaluation implements openfeature.FeatureProvider.
func (p *Provider) FloatEvaluation(ctx context.Context, flag string, defaultValue float64, evalCtx openfeature.FlattenedContext) openfeature.FloatResolutionDetail {
	val, detail := evaluate(p.client, flag, defaultValue, evalCtx)

	return openfeature.FloatResolutionDetail{
		Value:                    val,
		ProviderResolutionDetail: detail,
	}
}

// Hooks implements openfeature.FeatureProvider.
func (*Provider) Hooks() []openfeature.Hook {
	return nil
}

// IntEvaluation implements openfeature.FeatureProvider.
func (p *Provider) IntEvaluation(ctx context.Context, flag string, defaultValue int64, evalCtx openfeature.FlattenedContext) openfeature.IntResolutionDetail {
	val, detail := evaluate(p.client, flag, defaultValue, evalCtx)

	return openfeature.IntResolutionDetail{
		Value:                    val,
		ProviderResolutionDetail: detail,
	}
}

// Metadata implements openfeature.FeatureProvider.
func (*Provider) Metadata() openfeature.Metadata {
	return openfeature.Metadata{
		Name: name,
	}
}

// ObjectEvaluation implements openfeature.FeatureProvider.
func (p *Provider) ObjectEvaluation(ctx context.Context, flag string, defaultValue interface{}, evalCtx openfeature.FlattenedContext) openfeature.InterfaceResolutionDetail {
	val, detail := evaluate(p.client, flag, defaultValue, evalCtx)

	return openfeature.InterfaceResolutionDetail{
		Value:                    val,
		ProviderResolutionDetail: detail,
	}
}

// StringEvaluation implements openfeature.FeatureProvider.
func (p *Provider) StringEvaluation(ctx context.Context, flag string, defaultValue string, evalCtx openfeature.FlattenedContext) openfeature.StringResolutionDetail {
	val, detail := evaluate(p.client, flag, defaultValue, evalCtx)

	return openfeature.StringResolutionDetail{
		Value:                    val,
		ProviderResolutionDetail: detail,
	}
}
