package posthog

import "github.com/posthog/posthog-go"

type ProviderOption func(*Provider)

// WithAPIKey sets the API key to use for authentication
func WithAPIKey(apiKey string) ProviderOption {
	return func(p *Provider) {
		p.apiKey = apiKey
	}
}

// WithConfig sets the posthog.Config to use for the client
func WithConfig(config posthog.Config) ProviderOption {
	return func(p *Provider) {
		p.config = config
	}
}

// WithClient sets the posthog.Client to use for evaluation.
// This should be used instead of WithConfig and WithAPIKey.
func WithClient(client posthog.Client) ProviderOption {
	return func(p *Provider) {
		p.client = client
	}
}
