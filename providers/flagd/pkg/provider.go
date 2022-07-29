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
	serviceName string
	protocol    string
}

type ProviderOption func(*Provider)

func NewProvider(opts ...ProviderOption) *Provider {
	provider := &Provider{
		serviceName: "http",
		port:        8080,
		host:        "localhost",
		protocol:    "http",
	}
	for _, opt := range opts {
		opt(provider)
	}
	if provider.serviceName == "grpc" {
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

func WithPort(port int32) ProviderOption {
	return func(p *Provider) {
		p.port = port
	}
}

func WithHost(host string) ProviderOption {
	return func(p *Provider) {
		p.host = host
	}
}

// service should be one of "http" or "grpc", if not the default "http" will be used
func WithService(service string) ProviderOption {
	return func(p *Provider) {
		p.serviceName = service
	}
}

// service should be one of "http" or "https", if not the default "http" will be used, https is not currently supported
func WithProtocol(protocol string) ProviderOption {
	return func(p *Provider) {
		p.protocol = protocol
	}
}

func (p *Provider) Metadata() of.Metadata {
	return of.Metadata{
		Name: "flagd",
	}
}

func (p *Provider) GetBooleanEvaluation(flagKey string, defaultValue bool, evalCtx of.EvaluationContext, options ...of.EvaluationOption) of.BoolResolutionDetail {
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

func (p *Provider) GetStringEvaluation(flagKey string, defaultValue string, evalCtx of.EvaluationContext, options ...of.EvaluationOption) of.StringResolutionDetail {
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

func (p *Provider) GetNumberEvaluation(flagKey string, defaultValue float64, evalCtx of.EvaluationContext, options ...of.EvaluationOption) of.NumberResolutionDetail {
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
		Value: float64(res.Value), // todo - update flagd to output float64 (proto file change)
		ResolutionDetail: of.ResolutionDetail{
			Reason:  res.Reason,
			Value:   res.Value,
			Variant: res.Variant,
		},
	}
}

func (p *Provider) GetObjectEvaluation(flagKey string, defaultValue interface{}, evalCtx of.EvaluationContext, options ...of.EvaluationOption) of.ResolutionDetail {
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
