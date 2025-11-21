package flipt

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/open-feature/go-sdk-contrib/providers/flipt/pkg/service/transport"
	of "github.com/open-feature/go-sdk/openfeature"
	flipt "go.flipt.io/flipt/rpc/flipt"
	"go.flipt.io/flipt/rpc/flipt/evaluation"
	sdk "go.flipt.io/flipt/sdk/go"
	"google.golang.org/grpc"
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
	GRPCDialOptions []grpc.DialOption
	httpClient      *http.Client
}

// Option is a configuration option for the provider.
type Option func(*Provider)

// WithHTTPClient returns an [Option] that specifies the HTTP client to use as the basis of communications.
func WithHTTPClient(client *http.Client) Option {
	return func(p *Provider) {
		p.config.httpClient = client
	}
}

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

// WithGRPCDialOptions sets the options for the underlying gRPC transport.
func WithGRPCDialOptions(dialOptions ...grpc.DialOption) Option {
	return func(p *Provider) {
		p.config.GRPCDialOptions = append(p.config.GRPCDialOptions, dialOptions...)
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
		Address:         "http://localhost:8080",
		Namespace:       "default",
		GRPCDialOptions: []grpc.DialOption{},
		httpClient:      transport.DefaultClient,
	}}

	for _, opt := range opts {
		opt(p)
	}

	if p.svc == nil {
		topts := []transport.Option{
			transport.WithAddress(p.config.Address),
			transport.WithHTTPClient(p.config.httpClient),
			transport.WithCertificatePath(p.config.CertificatePath),
		}
		if p.config.TokenProvider != nil {
			topts = append(topts, transport.WithClientTokenProvider(p.config.TokenProvider))
		}
		if len(p.config.GRPCDialOptions) != 0 {
			topts = append(topts, transport.WithGRPCDialOptions(p.config.GRPCDialOptions...))
		}

		p.svc = transport.New(topts...)
	}

	return p
}

//go:generate mockery --name=Service --structname=mockService --case=underscore --output=. --outpkg=flipt --filename=provider_support.go --testonly --with-expecter --disable-version-string
type Service interface {
	GetFlag(ctx context.Context, namespaceKey, flagKey string) (*flipt.Flag, error)
	Evaluate(ctx context.Context, namespaceKey, flagKey string, evalCtx map[string]any) (*evaluation.VariantEvaluationResponse, error)
	Boolean(ctx context.Context, namespaceKey, flagKey string, evalCtx map[string]any) (*evaluation.BooleanEvaluationResponse, error)
}

// Provider implements the FeatureProvider interface and provides functions for evaluating flags with Flipt.
type Provider struct {
	svc    Service
	config Config
}

// Metadata returns the metadata of the provider.
func (p *Provider) Metadata() of.Metadata {
	return of.Metadata{Name: "flipt-provider"}
}

// BooleanEvaluation returns a boolean flag.
func (p *Provider) BooleanEvaluation(ctx context.Context, flag string, defaultValue bool, evalCtx of.FlattenedContext) of.BoolResolutionDetail {
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
			detail.ResolutionError = rerr

			return detail
		}

		detail.ResolutionError = of.NewGeneralResolutionError(err.Error())

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
func (p *Provider) StringEvaluation(ctx context.Context, flag string, defaultValue string, evalCtx of.FlattenedContext) of.StringResolutionDetail {
	value, detail := evaluateVariantFlag(ctx, p.svc, p.config.Namespace, flag, defaultValue, evalCtx, transformToString)
	return of.StringResolutionDetail{
		Value:                    value,
		ProviderResolutionDetail: detail,
	}
}

// FloatEvaluation returns a float flag.
func (p Provider) FloatEvaluation(ctx context.Context, flag string, defaultValue float64, evalCtx of.FlattenedContext) of.FloatResolutionDetail {
	value, detail := evaluateVariantFlag(ctx, p.svc, p.config.Namespace, flag, defaultValue, evalCtx, transformToFloat64)
	return of.FloatResolutionDetail{
		Value:                    value,
		ProviderResolutionDetail: detail,
	}
}

// IntEvaluation returns an int flag.
func (p *Provider) IntEvaluation(ctx context.Context, flag string, defaultValue int64, evalCtx of.FlattenedContext) of.IntResolutionDetail {
	value, detail := evaluateVariantFlag(ctx, p.svc, p.config.Namespace, flag, defaultValue, evalCtx, transformToInt64)
	return of.IntResolutionDetail{
		Value:                    value,
		ProviderResolutionDetail: detail,
	}
}

// ObjectEvaluation returns an object flag with attachment if any. Value is a map of key/value pairs ([string]any).
func (p *Provider) ObjectEvaluation(ctx context.Context, flag string, defaultValue any, evalCtx of.FlattenedContext) of.InterfaceResolutionDetail {
	value, detail := evaluateVariantFlag(ctx, p.svc, p.config.Namespace, flag, defaultValue, evalCtx, transformToObject)
	return of.InterfaceResolutionDetail{
		Value:                    value,
		ProviderResolutionDetail: detail,
	}
}

// Hooks returns hooks.
func (p *Provider) Hooks() []of.Hook {
	// code to retrieve hooks
	return []of.Hook{}
}

// evaluateVariantFlag is a helper which evaluates a variant flag and returns the value and resolution detail.
func evaluateVariantFlag[T any](ctx context.Context, svc Service, namespace string, flag string, defaultValue T, evalCtx of.FlattenedContext, transform transformFunc[T]) (T, of.ProviderResolutionDetail) {
	detail := of.ProviderResolutionDetail{
		Reason: of.DefaultReason,
	}
	value := defaultValue
	resp, err := svc.Evaluate(ctx, namespace, flag, evalCtx)
	if err != nil {
		var rerr of.ResolutionError

		if errors.As(err, &rerr) {
			detail.ResolutionError = rerr
		} else {
			detail.ResolutionError = of.NewGeneralResolutionError(err.Error())
		}

		return value, detail
	}

	if resp.Reason == evaluation.EvaluationReason_FLAG_DISABLED_EVALUATION_REASON {
		detail.Reason = of.DisabledReason
		return value, detail
	}

	if resp.Match || resp.Reason == evaluation.EvaluationReason_DEFAULT_EVALUATION_REASON {
		value, err = transform(resp, defaultValue)
		if err != nil {
			detail.ResolutionError = of.NewTypeMismatchResolutionError(err.Error())
			detail.Reason = of.ErrorReason
			return value, detail
		}
	}

	if resp.Match {
		detail.Reason = of.TargetingMatchReason
	}

	return value, detail
}

type transformFunc[T any] func(*evaluation.VariantEvaluationResponse, T) (T, error)

func transformToString(resp *evaluation.VariantEvaluationResponse, _ string) (string, error) {
	return resp.VariantKey, nil
}

func transformToFloat64(resp *evaluation.VariantEvaluationResponse, defaultValue float64) (float64, error) {
	fv, err := strconv.ParseFloat(resp.VariantKey, 64)
	if err != nil {
		return defaultValue, errors.New("value is not a float")
	}
	return fv, nil
}

func transformToInt64(resp *evaluation.VariantEvaluationResponse, defaultValue int64) (int64, error) {
	iv, err := strconv.ParseInt(resp.VariantKey, 10, 64)
	if err != nil {
		return defaultValue, errors.New("value is not an integer")
	}
	return iv, nil
}

func transformToObject(resp *evaluation.VariantEvaluationResponse, defaultValue any) (any, error) {
	if resp.VariantAttachment == "" {
		return defaultValue, nil
	}
	out := new(structpb.Struct)
	if err := protojson.Unmarshal([]byte(resp.VariantAttachment), out); err != nil {
		return defaultValue, fmt.Errorf("value is not an object: %q", resp.VariantAttachment)
	}
	return out.AsMap(), nil
}
