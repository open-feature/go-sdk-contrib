package flagd

import (
	"github.com/open-feature/golang-sdk-contrib/providers/flagd/pkg/service"
	GRPCService "github.com/open-feature/golang-sdk-contrib/providers/flagd/pkg/service/grpc"
	HTTPService "github.com/open-feature/golang-sdk-contrib/providers/flagd/pkg/service/http"
	of "github.com/open-feature/golang-sdk/pkg/openfeature"
)

type Provider struct {
	Service               service.IService
	providerConfiguration *ProviderConfiguration
}

type ProviderConfiguration struct {
	Port        uint16
	Host        string
	ServiceName ServiceType
}

type ServiceType int

const (
	// HTTP argument for use in WithService, this is the default value
	HTTP ServiceType = iota
	// HTTPS argument for use in WithService, overrides the default value of http (NOT IMPLEMENTED BY FLAGD)
	HTTPS
	// GRPC argument for use in WithService, overrides the default value of http
	GRPC
)

type ProviderOption func(*Provider)

func NewProvider(opts ...ProviderOption) *Provider {
	provider := &Provider{
		providerConfiguration: &ProviderConfiguration{
			ServiceName: HTTP,
			Port:        8080,
			Host:        "localhost",
		},
	}
	for _, opt := range opts {
		opt(provider)
	}
	if provider.providerConfiguration.ServiceName == GRPC {
		provider.Service = GRPCService.NewGRPCService(
			GRPCService.WithPort(provider.providerConfiguration.Port),
			GRPCService.WithHost(provider.providerConfiguration.Host),
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

// WithPort specifies the port of the flagd server. Defaults to 8080
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
