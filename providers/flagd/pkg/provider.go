package flagd

import (
	"context"
	"errors"
	"os"
	"strconv"

	"github.com/open-feature/go-sdk-contrib/providers/flagd/pkg/service"
	GRPCService "github.com/open-feature/go-sdk-contrib/providers/flagd/pkg/service/grpc"
	HTTPService "github.com/open-feature/go-sdk-contrib/providers/flagd/pkg/service/http"
	of "github.com/open-feature/go-sdk/pkg/openfeature"
	log "github.com/sirupsen/logrus"
)

type Provider struct {
	Service               service.IService
	providerConfiguration *ProviderConfiguration
}

type ProviderConfiguration struct {
	Port            uint16
	Host            string
	ServiceName     ServiceType
	CertificatePath string
	SocketPath      string
}

type ServiceType int

const (
	// HTTP argument for use in WithService, this is the default value
	HTTP ServiceType = iota + 1
	// HTTPS argument for use in WithService, overrides the default value of http
	HTTPS
	// GRPC argument for use in WithService, overrides the default value of http
	GRPC
)

type ProviderOption func(*Provider)

func NewProvider(opts ...ProviderOption) *Provider {
	provider := &Provider{
		// providerConfiguration maintains its default values, to ensure that the FromEnv option does not overwrite any explicitly set
		// values (default values are then set after the options are run via applyDefaults())
		providerConfiguration: &ProviderConfiguration{},
	}
	for _, opt := range opts {
		opt(provider)
	}
	provider.applyDefaults()
	if provider.providerConfiguration.ServiceName == GRPC {
		provider.Service = GRPCService.NewGRPCService(
			GRPCService.WithPort(provider.providerConfiguration.Port),
			GRPCService.WithHost(provider.providerConfiguration.Host),
			GRPCService.WithCertificatePath(provider.providerConfiguration.CertificatePath),
			GRPCService.WithSocketPath(provider.providerConfiguration.SocketPath),
		)
	} else if provider.providerConfiguration.ServiceName == HTTPS {
		provider.Service = HTTPService.NewHTTPService(
			HTTPService.WithPort(provider.providerConfiguration.Port),
			HTTPService.WithHost(provider.providerConfiguration.Host),
			HTTPService.WithProtocol(HTTPService.HTTPS),
		)
	} else {
		provider.Service = HTTPService.NewHTTPService(
			HTTPService.WithPort(provider.providerConfiguration.Port),
			HTTPService.WithHost(provider.providerConfiguration.Host),
		)
	}
	return provider
}

func (p *Provider) applyDefaults() {
	if p.providerConfiguration.Host == "" {
		p.providerConfiguration.Host = "localhost"
	}
	if p.providerConfiguration.Port == 0 {
		p.providerConfiguration.Port = 8013
	}
	if p.providerConfiguration.ServiceName == 0 {
		p.providerConfiguration.ServiceName = HTTP
	}
}

// WithSocketPath overrides the default hostname and port, a unix socket connection is made to flagd instead
func WithSocketPath(socketPath string) ProviderOption {
	return func(s *Provider) {
		s.providerConfiguration.SocketPath = socketPath
	}
}

// FromEnv sets the provider configuration from environemnt variables: FLAGD_HOST, FLAGD_PORT, FLAGD_SERVICE_PROVIDER, FLAGD_SERVER_CERT_PATH
func FromEnv() ProviderOption {
	return func(p *Provider) {

		if p.providerConfiguration.Port == 0 {
			portS := os.Getenv("FLAGD_PORT")
			if portS != "" {
				port, err := strconv.Atoi(portS)
				if err != nil {
					log.Error("invalid env config for FLAGD_PORT provided, using default value")
				} else {
					p.providerConfiguration.Port = uint16(port)
				}
			}
		}

		if p.providerConfiguration.ServiceName == 0 {
			serviceS := os.Getenv("FLAGD_SERVICE_PROVIDER")
			switch serviceS {
			case "http":
				p.providerConfiguration.ServiceName = HTTP
			case "https":
				p.providerConfiguration.ServiceName = HTTPS
			case "grpc":
				p.providerConfiguration.ServiceName = GRPC
			}
		}

		if p.providerConfiguration.CertificatePath == "" {
			certificatePath := os.Getenv("FLAGD_SERVER_CERT_PATH")
			if certificatePath != "" {
				p.providerConfiguration.CertificatePath = certificatePath
			}
		}

		if p.providerConfiguration.Host == "" {
			host := os.Getenv("FLAGD_HOST")
			if host != "" {
				p.providerConfiguration.Host = host
			}
		}

	}
}

// WithCertificatePath specifies the location of the certificate to be used in the gRPC dial credentials. If certificate loading fails insecure credentials will be used instead
func WithCertificatePath(path string) ProviderOption {
	return func(p *Provider) {
		p.providerConfiguration.CertificatePath = path
	}
}

// WithPort specifies the port of the flagd server. Defaults to 8013
func WithPort(port uint16) ProviderOption {
	return func(p *Provider) {
		p.providerConfiguration.Port = port
	}
}

// WithHost specifies the host name of the flagd server. Defaults to localhost
func WithHost(host string) ProviderOption {
	return func(p *Provider) {
		p.providerConfiguration.Host = host
	}
}

// WithService specifies the type of the service. Takes argument of type ServiceType. Defaults to http
func WithService(service ServiceType) ProviderOption {
	return func(p *Provider) {
		p.providerConfiguration.ServiceName = service
	}
}

// Hooks flagd provider does not have any hooks, returns empty slice
func (p *Provider) Hooks() []of.Hook {
	return []of.Hook{}
}

// Metadata returns value of Metadata (name of current service, exposed to openfeature sdk)
func (p *Provider) Metadata() of.Metadata {
	return of.Metadata{
		Name: "flagd",
	}
}

// Configuration returns the current configuration of the provider
func (p *Provider) Configuration() *ProviderConfiguration {
	return p.providerConfiguration
}

func (p *Provider) BooleanEvaluation(
	ctx context.Context, flagKey string, defaultValue bool, evalCtx of.FlattenedContext,
) of.BoolResolutionDetail {
	res, err := p.Service.ResolveBoolean(ctx, flagKey, evalCtx)
	if err != nil {
		var e of.ResolutionError
		if !errors.As(err, &e) {
			e = of.NewGeneralResolutionError(err.Error())
		}

		return of.BoolResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				ResolutionError: e,
				Reason:          of.Reason(res.Reason),
				Variant:         res.Variant,
			},
		}
	}

	return of.BoolResolutionDetail{
		Value: res.Value,
		ProviderResolutionDetail: of.ProviderResolutionDetail{
			Reason:  of.Reason(res.Reason),
			Variant: res.Variant,
		},
	}
}

func (p *Provider) StringEvaluation(
	ctx context.Context, flagKey string, defaultValue string, evalCtx of.FlattenedContext,
) of.StringResolutionDetail {
	res, err := p.Service.ResolveString(ctx, flagKey, evalCtx)
	if err != nil {
		var e of.ResolutionError
		if !errors.As(err, &e) {
			e = of.NewGeneralResolutionError(err.Error())
		}

		return of.StringResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				ResolutionError: e,
				Reason:          of.Reason(res.Reason),
				Variant:         res.Variant,
			},
		}
	}

	return of.StringResolutionDetail{
		Value: res.Value,
		ProviderResolutionDetail: of.ProviderResolutionDetail{
			Reason:  of.Reason(res.Reason),
			Variant: res.Variant,
		},
	}
}

func (p *Provider) FloatEvaluation(
	ctx context.Context, flagKey string, defaultValue float64, evalCtx of.FlattenedContext,
) of.FloatResolutionDetail {
	res, err := p.Service.ResolveFloat(ctx, flagKey, evalCtx)
	if err != nil {
		var e of.ResolutionError
		if !errors.As(err, &e) {
			e = of.NewGeneralResolutionError(err.Error())
		}

		return of.FloatResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				ResolutionError: e,
				Reason:          of.Reason(res.Reason),
				Variant:         res.Variant,
			},
		}
	}

	return of.FloatResolutionDetail{
		Value: res.Value,
		ProviderResolutionDetail: of.ProviderResolutionDetail{
			Reason:  of.Reason(res.Reason),
			Variant: res.Variant,
		},
	}
}

func (p *Provider) IntEvaluation(
	ctx context.Context, flagKey string, defaultValue int64, evalCtx of.FlattenedContext,
) of.IntResolutionDetail {
	res, err := p.Service.ResolveInt(ctx, flagKey, evalCtx)
	if err != nil {
		var e of.ResolutionError
		if !errors.As(err, &e) {
			e = of.NewGeneralResolutionError(err.Error())
		}

		return of.IntResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				ResolutionError: e,
				Reason:          of.Reason(res.Reason),
				Variant:         res.Variant,
			},
		}
	}

	return of.IntResolutionDetail{
		Value: res.Value,
		ProviderResolutionDetail: of.ProviderResolutionDetail{
			Reason:  of.Reason(res.Reason),
			Variant: res.Variant,
		},
	}
}

func (p *Provider) ObjectEvaluation(
	ctx context.Context, flagKey string, defaultValue interface{}, evalCtx of.FlattenedContext,
) of.InterfaceResolutionDetail {
	res, err := p.Service.ResolveObject(ctx, flagKey, evalCtx)
	if err != nil {
		var e of.ResolutionError
		if !errors.As(err, &e) {
			e = of.NewGeneralResolutionError(err.Error())
		}

		return of.InterfaceResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				ResolutionError: e,
				Reason:          of.Reason(res.Reason),
				Variant:         res.Variant,
			},
		}
	}

	return of.InterfaceResolutionDetail{
		Value: res.Value,
		ProviderResolutionDetail: of.ProviderResolutionDetail{
			Reason:  of.Reason(res.Reason),
			Variant: res.Variant,
		},
	}
}
