package flagd

import (
	"github.com/open-feature/golang-sdk-contrib/providers/flagd/pkg/service"
	GRPCService "github.com/open-feature/golang-sdk-contrib/providers/flagd/pkg/service/grpc"
	HTTPService "github.com/open-feature/golang-sdk-contrib/providers/flagd/pkg/service/http"
	of "github.com/open-feature/golang-sdk/pkg/openfeature"
)

type Provider struct {
	service     service.IService
	port        int32
	host        string
	serviceName ServiceType
	protocol    HTTPService.ProtocolType
}

type ServiceType int

const (
	HTTP ServiceType = iota
	GRPC
)

type ProviderOption func(*Provider)

func NewProvider(opts ...ProviderOption) *Provider {
	provider := &Provider{
		serviceName: HTTP,
		port:        8080,
		host:        "localhost",
		protocol:    HTTPService.HTTPS,
	}
	for _, opt := range opts {
		opt(provider)
	}
	if provider.serviceName == GRPC {
		provider.service = GRPCService.NewGRPCService(
			GRPCService.WithPort(provider.port),
			GRPCService.WithHost(provider.host),
		)
	} else {
		provider.service = HTTPService.NewHTTPService(
			HTTPService.WithPort(provider.port),
			HTTPService.WithHost(provider.host),
			HTTPService.WithProtocol(provider.protocol),
		)
	}
	return provider
}

// WithHost specifies the port of the flagd server. Defaults to 8080
func WithPort(port int32) ProviderOption {
	return func(p *Provider) {
		p.port = port
	}
}

// WithHost specifies the host name of the flagd server. Defaults to localhost
func WithHost(host string) ProviderOption {
	return func(p *Provider) {
		p.host = host
	}
}

// WithService specifies the type of the service. service should be one of "http" or "grpc", if not the default "http" will be used
func WithService(service ServiceType) ProviderOption {
	return func(p *Provider) {
		p.serviceName = service
	}
}

// WithProtocol specifies the protocol used by the http service. Should be one of "http" or "https", if not the default "http" will be used, https is not currently supported
func WithProtocol(protocol HTTPService.ProtocolType) ProviderOption {
	return func(p *Provider) {
		p.protocol = protocol
	}
}

func (p *Provider) Metadata() of.Metadata {
	return of.Metadata{
		Name: "flagd",
	}
}

func (p *Provider) BooleanEvaluation(flagKey string, defaultValue bool, evalCtx of.EvaluationContext, options of.EvaluationOptions) of.BoolResolutionDetail {
	res, err := p.service.ResolveBoolean(flagKey, evalCtx)
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
	res, err := p.service.ResolveString(flagKey, evalCtx)
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

func (p *Provider) NumberEvaluation(flagKey string, defaultValue float64, evalCtx of.EvaluationContext, options of.EvaluationOptions) of.NumberResolutionDetail {
	res, err := p.service.ResolveNumber(flagKey, evalCtx)
	if err != nil {
		return of.NumberResolutionDetail{
			Value: defaultValue,
			ResolutionDetail: of.ResolutionDetail{
				Reason:    res.Reason,
				Value:     defaultValue,
				Variant:   res.Variant,
				ErrorCode: err.Error(),
			},
		}
	}
	return of.NumberResolutionDetail{
		Value: float64(res.Value),
		ResolutionDetail: of.ResolutionDetail{
			Reason:  res.Reason,
			Value:   res.Value,
			Variant: res.Variant,
		},
	}
}

func (p *Provider) ObjectEvaluation(flagKey string, defaultValue interface{}, evalCtx of.EvaluationContext, options of.EvaluationOptions) of.ResolutionDetail {
	res, err := p.service.ResolveObject(flagKey, evalCtx)
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
