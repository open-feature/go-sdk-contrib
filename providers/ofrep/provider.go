package ofrep

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/open-feature/go-sdk-contrib/providers/ofrep/internal/evaluate"
	"github.com/open-feature/go-sdk-contrib/providers/ofrep/internal/outbound"
	"github.com/open-feature/go-sdk/openfeature"
)

var _ openfeature.FeatureProvider = &Provider{}

// Provider implementation for OFREP
type Provider struct {
	evaluator Evaluator
}

type Option func(*outbound.Configuration)

// NewProvider returns an OFREP provider configured with provided configuration.
// The only mandatory configuration is the baseUri, which is the base path of the OFREP service implementation.
func NewProvider(baseUri string, options ...Option) *Provider {
	cfg := outbound.Configuration{
		BaseURI: baseUri,
		Timeout: 10 * time.Second,
	}

	for _, option := range options {
		option(&cfg)
	}

	provider := &Provider{
		evaluator: evaluate.NewFlagsEvaluator(cfg),
	}

	return provider
}

func (p Provider) Metadata() openfeature.Metadata {
	return openfeature.Metadata{
		Name: "OpenFeature Remote Evaluation Protocol Provider",
	}
}

func (p Provider) BooleanEvaluation(ctx context.Context, flag string, defaultValue bool, evalCtx openfeature.FlattenedContext) openfeature.BoolResolutionDetail {
	return p.evaluator.ResolveBoolean(ctx, flag, defaultValue, evalCtx)
}

func (p Provider) StringEvaluation(ctx context.Context, flag string, defaultValue string, evalCtx openfeature.FlattenedContext) openfeature.StringResolutionDetail {
	return p.evaluator.ResolveString(ctx, flag, defaultValue, evalCtx)
}

func (p Provider) FloatEvaluation(ctx context.Context, flag string, defaultValue float64, evalCtx openfeature.FlattenedContext) openfeature.FloatResolutionDetail {
	return p.evaluator.ResolveFloat(ctx, flag, defaultValue, evalCtx)
}

func (p Provider) IntEvaluation(ctx context.Context, flag string, defaultValue int64, evalCtx openfeature.FlattenedContext) openfeature.IntResolutionDetail {
	return p.evaluator.ResolveInt(ctx, flag, defaultValue, evalCtx)
}

func (p Provider) ObjectEvaluation(ctx context.Context, flag string, defaultValue any, evalCtx openfeature.FlattenedContext) openfeature.InterfaceResolutionDetail {
	return p.evaluator.ResolveObject(ctx, flag, defaultValue, evalCtx)
}

func (p Provider) Hooks() []openfeature.Hook {
	return []openfeature.Hook{}
}

// options of the OFREP provider

// WithHeaderProvider allows to configure a custom header callback to set a custom authorization header
func WithHeaderProvider(callback outbound.HeaderCallback) func(*outbound.Configuration) {
	return func(c *outbound.Configuration) {
		c.Callbacks = append(c.Callbacks, callback)
	}
}

// WithBearerToken allows to set token to be used for bearer token authorization
func WithBearerToken(token string) func(*outbound.Configuration) {
	return func(c *outbound.Configuration) {
		c.Callbacks = append(c.Callbacks, func() (string, string) {
			return "Authorization", fmt.Sprintf("Bearer %s", token)
		})
	}
}

// WithApiKeyAuth allows to set token to be used for api key authorization
func WithApiKeyAuth(token string) func(*outbound.Configuration) {
	return func(c *outbound.Configuration) {
		c.Callbacks = append(c.Callbacks, func() (string, string) {
			return "X-API-Key", token
		})
	}
}

// WithClient allows to provide a pre-configured http.Client for the communication with the OFREP service
func WithClient(client *http.Client) func(configuration *outbound.Configuration) {
	return func(configuration *outbound.Configuration) {
		configuration.Client = client
	}
}

// WithHeader allows to set a custom header
func WithHeader(key, value string) func(*outbound.Configuration) {
	return func(c *outbound.Configuration) {
		c.Callbacks = append(c.Callbacks, func() (string, string) {
			return key, value
		})
	}
}

// WithBaseURI allows to override the base URI of the OFREP service
func WithBaseURI(baseURI string) func(*outbound.Configuration) {
	return func(c *outbound.Configuration) {
		c.BaseURI = baseURI
	}
}

// WithTimeout allows to configure the timeout for the http client used for communication with the OFREP service.
// This option is ignored if a custom client is provided via WithClient.
func WithTimeout(timeout time.Duration) func(*outbound.Configuration) {
	return func(c *outbound.Configuration) {
		c.Timeout = timeout
	}
}

// WithFromEnv uses environment variables to configure the provider.
//
// Experimental: This feature is experimental and may change in future versions.
//
// Supported environment variables:
//   - OFREP_ENDPOINT: base URI for the OFREP service
//   - OFREP_TIMEOUT: timeout duration (e.g., "30s", "1m" or raw "5000" in milliseconds )
//   - OFREP_HEADERS: comma-separated custom headers (e.g., "Key1=Value1,Key2=Value2")
func WithFromEnv() func(*outbound.Configuration) {
	envHandlers := map[string]func(*outbound.Configuration, string){
		"OFREP_ENDPOINT": func(c *outbound.Configuration, v string) {
			WithBaseURI(v)(c)
		},
		"OFREP_TIMEOUT": func(c *outbound.Configuration, v string) {
			if t, err := time.ParseDuration(v); err == nil && t > 0 {
				WithTimeout(t)(c)
				return
			}
			// as the specification is not finalized, also support raw milliseconds
			t, err := strconv.Atoi(v)
			if err == nil && t > 0 {
				WithTimeout(time.Duration(t) * time.Millisecond)(c)
			}
		},
		"OFREP_HEADERS": func(c *outbound.Configuration, v string) {
			for pair := range strings.SplitSeq(v, ",") {
				kv := strings.SplitN(strings.TrimSpace(pair), "=", 2)
				if len(kv) == 2 {
					WithHeader(kv[0], kv[1])(c)
				}
			}
		},
	}
	return func(c *outbound.Configuration) {
		for key, handler := range envHandlers {
			if v := os.Getenv(key); v != "" {
				handler(c, v)
			}
		}
	}
}
