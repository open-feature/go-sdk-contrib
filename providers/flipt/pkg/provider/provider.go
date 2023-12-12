package flipt

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/open-feature/go-sdk-contrib/providers/flipt/pkg/service/transport"
	of "github.com/open-feature/go-sdk/openfeature"
	flipt "go.flipt.io/flipt/rpc/flipt"
	"go.flipt.io/flipt/rpc/flipt/evaluation"
	sdk "go.flipt.io/flipt/sdk/go"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

var _ of.FeatureProvider = (*Provider)(nil)

// Config is a configuration for the FliptProvider.
type Config struct {
	Address         string
	CertificatePath string
	TokenProvider   sdk.ClientTokenProvider
	Namespace       string
}

// Option is a configuration option for the provider.
type Option func(*Provider)

// WithAddress sets the address for the remote Flipt gRPC or HTTP API.
func WithAddress(address string) Option {
	return func(p *Provider) {
		p.config.Address = address
	}
}

// WithCertificatePath is an Option to set the certificate path (grpc only).
func WithCertificatePath(certificatePath string) Option {
	return func(p *Provider) {
		p.config.CertificatePath = certificatePath
	}
}

// WithConfig is an Option to set the entire configuration.
func WithConfig(config Config) Option {
	return func(p *Provider) {
		p.config = config
	}
}

// WithService is an Option to set the service for the Provider.
func WithService(svc Service) Option {
	return func(p *Provider) {
		p.svc = svc
	}
}

// WithClientTokenProvider sets the token provider for auth to support client
// auth needs.
func WithClientTokenProvider(tokenProvider sdk.ClientTokenProvider) Option {
	return func(p *Provider) {
		p.config.TokenProvider = tokenProvider
	}
}

// ForNamespace sets the namespace for flag lookup and evaluation in Flipt.
func ForNamespace(namespace string) Option {
	return func(p *Provider) {
		p.config.Namespace = namespace
	}
}

// NewProvider returns a new Flipt provider.
func NewProvider(opts ...Option) *Provider {
	p := &Provider{config: Config{
		Address:   "http://localhost:8080",
		Namespace: "default",
	}}

	for _, opt := range opts {
		opt(p)
	}

	if p.svc == nil {
		topts := []transport.Option{transport.WithAddress(p.config.Address), transport.WithCertificatePath(p.config.CertificatePath)}
		if p.config.TokenProvider != nil {
			topts = append(topts, transport.WithClientTokenProvider(p.config.TokenProvider))
		}

		p.svc = transport.New(topts...)
	}

	return p
}

//go:generate mockery --name=Service --structname=mockService --case=underscore --output=. --outpkg=flipt --filename=provider_support.go --testonly --with-expecter --disable-version-string
type Service interface {
	GetFlag(ctx context.Context, namespaceKey, flagKey string) (*flipt.Flag, error)
	Evaluate(ctx context.Context, namespaceKey, flagKey string, evalCtx map[string]interface{}) (*evaluation.VariantEvaluationResponse, error)
	Boolean(ctx context.Context, namespaceKey, flagKey string, evalCtx map[string]interface{}) (*evaluation.BooleanEvaluationResponse, error)
}

// Provider implements the FeatureProvider interface and provides functions for evaluating flags with Flipt.
type Provider struct {
	svc    Service
	config Config
}

// Metadata returns the metadata of the provider.
func (p Provider) Metadata() of.Metadata {
	return of.Metadata{Name: "flipt-provider"}
}

// BooleanEvaluation returns a boolean flag.
func (p Provider) BooleanEvaluation(ctx context.Context, flag string, defaultValue bool, evalCtx of.FlattenedContext) of.BoolResolutionDetail {
	resp, err := p.svc.Boolean(ctx, p.config.Namespace, flag, evalCtx)
	if err != nil {
		var (
			rerr   of.ResolutionError
			detail = of.BoolResolutionDetail{
				Value: defaultValue,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					Reason: of.DefaultReason,
				},
			}
		)

		if errors.As(err, &rerr) {
			detail.ProviderResolutionDetail.ResolutionError = rerr

			return detail
		}

		detail.ProviderResolutionDetail.ResolutionError = of.NewGeneralResolutionError(err.Error())

		return detail
	}

	return of.BoolResolutionDetail{
		Value: resp.Enabled,
		ProviderResolutionDetail: of.ProviderResolutionDetail{
			Reason: of.TargetingMatchReason,
		},
	}
}

// StringEvaluation returns a string flag.
func (p Provider) StringEvaluation(ctx context.Context, flag string, defaultValue string, evalCtx of.FlattenedContext) of.StringResolutionDetail {
	resp, err := p.svc.Evaluate(ctx, p.config.Namespace, flag, evalCtx)
	if err != nil {
		var (
			rerr   of.ResolutionError
			detail = of.StringResolutionDetail{
				Value: defaultValue,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					Reason: of.DefaultReason,
				},
			}
		)

		if errors.As(err, &rerr) {
			detail.ProviderResolutionDetail.ResolutionError = rerr

			return detail
		}

		detail.ProviderResolutionDetail.ResolutionError = of.NewGeneralResolutionError(err.Error())

		return detail
	}

	if resp.Reason == evaluation.EvaluationReason_FLAG_DISABLED_EVALUATION_REASON {
		return of.StringResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				Reason: of.DisabledReason,
			},
		}
	}

	if !resp.Match {
		return of.StringResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				Reason: of.DefaultReason,
			},
		}
	}

	return of.StringResolutionDetail{
		Value: resp.VariantKey,
		ProviderResolutionDetail: of.ProviderResolutionDetail{
			Reason: of.TargetingMatchReason,
		},
	}
}

// FloatEvaluation returns a float flag.
func (p Provider) FloatEvaluation(ctx context.Context, flag string, defaultValue float64, evalCtx of.FlattenedContext) of.FloatResolutionDetail {
	resp, err := p.svc.Evaluate(ctx, p.config.Namespace, flag, evalCtx)
	if err != nil {
		var (
			rerr   of.ResolutionError
			detail = of.FloatResolutionDetail{
				Value: defaultValue,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					Reason: of.DefaultReason,
				},
			}
		)

		if errors.As(err, &rerr) {
			detail.ProviderResolutionDetail.ResolutionError = rerr

			return detail
		}

		detail.ProviderResolutionDetail.ResolutionError = of.NewGeneralResolutionError(err.Error())

		return detail
	}

	if resp.Reason == evaluation.EvaluationReason_FLAG_DISABLED_EVALUATION_REASON {
		return of.FloatResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				Reason: of.DisabledReason,
			},
		}
	}

	if !resp.Match {
		return of.FloatResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				Reason: of.DefaultReason,
			},
		}
	}

	fv, err := strconv.ParseFloat(resp.VariantKey, 64)
	if err != nil {
		return of.FloatResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				ResolutionError: of.NewTypeMismatchResolutionError("value is not a float"),
				Reason:          of.ErrorReason,
			},
		}
	}

	return of.FloatResolutionDetail{
		Value: fv,
		ProviderResolutionDetail: of.ProviderResolutionDetail{
			Reason: of.TargetingMatchReason,
		},
	}
}

// IntEvaluation returns an int flag.
func (p Provider) IntEvaluation(ctx context.Context, flag string, defaultValue int64, evalCtx of.FlattenedContext) of.IntResolutionDetail {
	resp, err := p.svc.Evaluate(ctx, p.config.Namespace, flag, evalCtx)
	if err != nil {
		var (
			rerr   of.ResolutionError
			detail = of.IntResolutionDetail{
				Value: defaultValue,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					Reason: of.DefaultReason,
				},
			}
		)

		if errors.As(err, &rerr) {
			detail.ProviderResolutionDetail.ResolutionError = rerr

			return detail
		}

		detail.ProviderResolutionDetail.ResolutionError = of.NewGeneralResolutionError(err.Error())

		return detail
	}

	if resp.Reason == evaluation.EvaluationReason_FLAG_DISABLED_EVALUATION_REASON {
		return of.IntResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				Reason: of.DisabledReason,
			},
		}
	}

	if !resp.Match {
		return of.IntResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				Reason: of.DefaultReason,
			},
		}
	}

	iv, err := strconv.ParseInt(resp.VariantKey, 10, 64)
	if err != nil {
		return of.IntResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				ResolutionError: of.NewTypeMismatchResolutionError("value is not an integer"),
				Reason:          of.ErrorReason,
			},
		}
	}

	return of.IntResolutionDetail{
		Value: iv,
		ProviderResolutionDetail: of.ProviderResolutionDetail{
			Reason: of.TargetingMatchReason,
		},
	}
}

// ObjectEvaluation returns an object flag with attachment if any. Value is a map of key/value pairs ([string]interface{}).
func (p Provider) ObjectEvaluation(ctx context.Context, flag string, defaultValue interface{}, evalCtx of.FlattenedContext) of.InterfaceResolutionDetail {
	resp, err := p.svc.Evaluate(ctx, p.config.Namespace, flag, evalCtx)
	if err != nil {
		var (
			rerr   of.ResolutionError
			detail = of.InterfaceResolutionDetail{
				Value: defaultValue,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					Reason: of.DefaultReason,
				},
			}
		)

		if errors.As(err, &rerr) {
			detail.ProviderResolutionDetail.ResolutionError = rerr

			return detail
		}

		detail.ProviderResolutionDetail.ResolutionError = of.NewGeneralResolutionError(err.Error())

		return detail
	}

	if resp.Reason == evaluation.EvaluationReason_FLAG_DISABLED_EVALUATION_REASON {
		return of.InterfaceResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				Reason: of.DisabledReason,
			},
		}
	}

	if !resp.Match {
		return of.InterfaceResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				Reason: of.DefaultReason,
			},
		}
	}

	if resp.VariantAttachment == "" {
		return of.InterfaceResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				Reason:  of.DefaultReason,
				Variant: resp.VariantKey,
			},
		}
	}

	out := new(structpb.Struct)
	if err := protojson.Unmarshal([]byte(resp.VariantAttachment), out); err != nil {
		return of.InterfaceResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				ResolutionError: of.NewTypeMismatchResolutionError(fmt.Sprintf("value is not an object: %q", resp.VariantAttachment)),
				Reason:          of.ErrorReason,
			},
		}
	}

	return of.InterfaceResolutionDetail{
		Value: out.AsMap(),
		ProviderResolutionDetail: of.ProviderResolutionDetail{
			Reason:  of.TargetingMatchReason,
			Variant: resp.VariantKey,
		},
	}
}

// Hooks returns hooks.
func (p Provider) Hooks() []of.Hook {
	// code to retrieve hooks
	return []of.Hook{}
}
