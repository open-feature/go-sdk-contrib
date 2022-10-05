package flagd

import (
	"context"
	"errors"
	"os"
	"strconv"

	"github.com/open-feature/go-sdk-contrib/providers/flagd/pkg/service"
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
	CertificatePath string
	SocketPath      string
}

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
	provider.Service = &service.Service{
		Client: &service.Client{
			ServiceConfiguration: &service.ServiceConfiguration{
				Host:            provider.providerConfiguration.Host,
				Port:            provider.providerConfiguration.Port,
				CertificatePath: provider.providerConfiguration.CertificatePath,
				SocketPath:      provider.providerConfiguration.SocketPath,
			},
		},
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
