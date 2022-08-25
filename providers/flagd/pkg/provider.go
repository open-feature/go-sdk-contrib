package flagd

import (
	"os"
	"strconv"

	"github.com/open-feature/golang-sdk-contrib/providers/flagd/pkg/service"
	GRPCService "github.com/open-feature/golang-sdk-contrib/providers/flagd/pkg/service/grpc"
	HTTPService "github.com/open-feature/golang-sdk-contrib/providers/flagd/pkg/service/http"
	of "github.com/open-feature/golang-sdk/pkg/openfeature"
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
}

type ServiceType int

const (
	// HTTP argument for use in WithService, this is the default value
	HTTP ServiceType = iota
	// HTTPS argument for use in WithService, overrides the default value of http
	HTTPS
	// GRPC argument for use in WithService, overrides the default value of http
	GRPC
)

type ProviderOption func(*Provider)

func NewProvider(opts ...ProviderOption) *Provider {
	provider := &Provider{
		// providerConfiguration maintains its default values, with the exception of ServiceName to ensure that the FromEnv
		// option does not overwrite any explicitly set values (default values are then set after the options are run via applyDefaults())
		providerConfiguration: &ProviderConfiguration{
			ServiceName: -1,
		},
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
	if p.providerConfiguration.ServiceName == -1 {
		p.providerConfiguration.ServiceName = HTTP
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

		if p.providerConfiguration.ServiceName == -1 {
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

func (p *Provider) BooleanEvaluation(flagKey string, defaultValue bool, evalCtx of.EvaluationContext, options of.EvaluationOptions) of.BoolResolutionDetail {
	res, err := p.Service.ResolveBoolean(flagKey, evalCtx)
	if err != nil {
		return of.BoolResolutionDetail{
			Value: defaultValue,
			ResolutionDetail: of.ResolutionDetail{
				Reason:    res.Reason,
				Value:     defaultValue,
				Variant:   res.Variant,
				ErrorCode: err.Error(),
			},
		}
	}
	return of.BoolResolutionDetail{
		Value: res.Value,
		ResolutionDetail: of.ResolutionDetail{
			Reason:  res.Reason,
			Value:   res.Value,
			Variant: res.Variant,
		},
	}
}

func (p *Provider) StringEvaluation(flagKey string, defaultValue string, evalCtx of.EvaluationContext, options of.EvaluationOptions) of.StringResolutionDetail {
	res, err := p.Service.ResolveString(flagKey, evalCtx)
	if err != nil {
		return of.StringResolutionDetail{
			Value: defaultValue,
			ResolutionDetail: of.ResolutionDetail{
				Reason:    res.Reason,
				Value:     defaultValue,
				Variant:   res.Variant,
				ErrorCode: err.Error(),
			},
		}
	}
	return of.StringResolutionDetail{
		Value: res.Value,
		ResolutionDetail: of.ResolutionDetail{
			Reason:  res.Reason,
			Value:   res.Value,
			Variant: res.Variant,
		},
	}
}

func (p *Provider) FloatEvaluation(flagKey string, defaultValue float64, evalCtx of.EvaluationContext, options of.EvaluationOptions) of.FloatResolutionDetail {
	res, err := p.Service.ResolveFloat(flagKey, evalCtx)
	if err != nil {
		return of.FloatResolutionDetail{
			Value: defaultValue,
			ResolutionDetail: of.ResolutionDetail{
				Reason:    res.Reason,
				Value:     defaultValue,
				Variant:   res.Variant,
				ErrorCode: err.Error(),
			},
		}
	}
	return of.FloatResolutionDetail{
		Value: res.Value,
		ResolutionDetail: of.ResolutionDetail{
			Reason:  res.Reason,
			Value:   res.Value,
			Variant: res.Variant,
		},
	}
}

func (p *Provider) IntEvaluation(flagKey string, defaultValue int64, evalCtx of.EvaluationContext, options of.EvaluationOptions) of.IntResolutionDetail {
	res, err := p.Service.ResolveInt(flagKey, evalCtx)
	if err != nil {
		return of.IntResolutionDetail{
			Value: defaultValue,
			ResolutionDetail: of.ResolutionDetail{
				Reason:    res.Reason,
				Value:     defaultValue,
				Variant:   res.Variant,
				ErrorCode: err.Error(),
			},
		}
	}
	return of.IntResolutionDetail{
		Value: res.Value,
		ResolutionDetail: of.ResolutionDetail{
			Reason:  res.Reason,
			Value:   res.Value,
			Variant: res.Variant,
		},
	}
}

func (p *Provider) ObjectEvaluation(flagKey string, defaultValue interface{}, evalCtx of.EvaluationContext, options of.EvaluationOptions) of.ResolutionDetail {
	res, err := p.Service.ResolveObject(flagKey, evalCtx)
	if err != nil {
		return of.ResolutionDetail{
			Reason:    res.Reason,
			Value:     defaultValue,
			Variant:   res.Variant,
			ErrorCode: err.Error(),
		}
	}
	return of.ResolutionDetail{
		Reason:  res.Reason,
		Value:   res.Value,
		Variant: res.Variant,
	}
}
