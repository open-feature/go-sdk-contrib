package ofrep

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
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
// The only mandatory configuration is the baseURI, which is the base path of the OFREP service implementation.
func NewProvider(baseURI string, options ...Option) *Provider {
	cfg := outbound.Configuration{
		Timeout: 10 * time.Second,
	}

	// Apply configuration from OFREP environment variables
	WithFromEnv()(&cfg)
	// Follow the spec - 3. Allow programmatic configuration to override environment variables
	if baseURI != "" {
		cfg.BaseURI = baseURI
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

// WithFromEnv configures the provider using environment variables.
//
// Supported environment variables:
//
//   - OFREP_ENDPOINT: Base URL of the OFREP service.
//   - OFREP_TIMEOUT_MS: Request timeout in milliseconds (e.g., "5000").
//   - OFREP_HEADERS: Comma-separated list of custom headers
//     (e.g., "Key1=Value1,Key2=Value2").
//
// When provided, environment variables take precedence over values
// set programmatically via previous options.
//
// Example:
//
//	provider := NewProvider(
//	    "https://ofrep.localhost/",
//	    WithTimeout(time.Minute),
//	    WithFromEnv(),
//	)
//
// In this example, if OFREP_ENDPOINT or OFREP_TIMEOUT_MS are set,
// their values override the ones passed via programmatic options.
func WithFromEnv() func(*outbound.Configuration) {
	envHandlers := map[string]func(*outbound.Configuration, string){
		"OFREP_ENDPOINT": func(c *outbound.Configuration, v string) {
			WithBaseURI(v)(c)
		},
		"OFREP_TIMEOUT_MS": func(c *outbound.Configuration, v string) {
			t, err := strconv.Atoi(v)
			if err == nil && t > 0 {
				WithTimeout(time.Duration(t) * time.Millisecond)(c)
			}
		},
		"OFREP_HEADERS": func(c *outbound.Configuration, v string) {
			v, err := url.PathUnescape(v)
			if err != nil {
				// skip invalid value
				return
			}
			for pair := range strings.SplitSeq(v, ",") {
				kv := strings.SplitN(pair, "=", 2)
				if len(kv) != 2 {
					// skip invalid value
					continue
				}
				k := strings.TrimSpace(kv[0])
				v := strings.TrimSpace(kv[1])
				WithHeader(k, v)(c)
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
