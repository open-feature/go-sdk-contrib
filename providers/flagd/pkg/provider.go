package flagd

import (
	"context"
	"errors"
	"fmt"
	lru "github.com/hashicorp/golang-lru/v2"
	flagdModels "github.com/open-feature/flagd/pkg/model"
	flagdService "github.com/open-feature/flagd/pkg/service"
	"github.com/open-feature/go-sdk-contrib/providers/flagd/pkg/cache"
	"github.com/open-feature/go-sdk-contrib/providers/flagd/pkg/constant"
	schemaV1 "go.buf.build/open-feature/flagd-connect/open-feature/flagd/schema/v1"
	"os"
	"strconv"

	"github.com/open-feature/go-sdk-contrib/providers/flagd/pkg/service"
	of "github.com/open-feature/go-sdk/pkg/openfeature"
	log "github.com/sirupsen/logrus"
)

const defaultLRUCacheSize int = 1000

type Provider struct {
	ctx                   context.Context
	Service               service.IService
	cacheEnabled          bool
	booleanCache          Cache[string, of.BoolResolutionDetail]
	stringCache           Cache[string, of.StringResolutionDetail]
	floatCache            Cache[string, of.FloatResolutionDetail]
	intCache              Cache[string, of.IntResolutionDetail]
	interfaceCache        Cache[string, of.InterfaceResolutionDetail]
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
		ctx: context.Background(),
		// providerConfiguration maintains its default values, to ensure that the FromEnv option does not overwrite any explicitly set
		// values (default values are then set after the options are run via applyDefaults())
		providerConfiguration: &ProviderConfiguration{},
	}
	WithLRUCache(defaultLRUCacheSize)(provider)
	for _, opt := range opts {
		opt(provider)
	}
	provider.applyDefaults()
	provider.Service = service.NewService(&service.Client{
		ServiceConfiguration: &service.ServiceConfiguration{
			Host:            provider.providerConfiguration.Host,
			Port:            provider.providerConfiguration.Port,
			CertificatePath: provider.providerConfiguration.CertificatePath,
			SocketPath:      provider.providerConfiguration.SocketPath,
		},
	})

	go func() {
		if err := provider.handleEvents(provider.ctx); err != nil {
			log.Error(fmt.Errorf("handle events: %w", err))
		}
	}()

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

// FromEnv sets the provider configuration from environment variables: FLAGD_HOST, FLAGD_PORT, FLAGD_SERVICE_PROVIDER, FLAGD_SERVER_CERT_PATH & FLAGD_CACHING_DISABLED
func FromEnv() ProviderOption {
	return func(p *Provider) {

		portS := os.Getenv("FLAGD_PORT")
		if portS != "" {
			port, err := strconv.Atoi(portS)
			if err != nil {
				log.Error("invalid env config for FLAGD_PORT provided, using default value")
			} else {
				p.providerConfiguration.Port = uint16(port)
			}
		}

		certificatePath := os.Getenv("FLAGD_SERVER_CERT_PATH")
		if certificatePath != "" {
			p.providerConfiguration.CertificatePath = certificatePath
		}

		host := os.Getenv("FLAGD_HOST")
		if host != "" {
			p.providerConfiguration.Host = host
		}

		cachingDisabled := os.Getenv("FLAGD_CACHING_DISABLED")
		if cachingDisabled == "true" {
			WithoutCache()(p)
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

// WithoutCache disables caching
func WithoutCache() ProviderOption {
	return func(p *Provider) {
		p.cacheEnabled = false
		p.booleanCache = nil
		p.stringCache = nil
		p.floatCache = nil
		p.interfaceCache = nil
		p.intCache = nil
	}
}

// WithBasicInMemoryCache applies a basic in memory cache store (with no memory limits)
func WithBasicInMemoryCache() ProviderOption {
	return func(p *Provider) {
		p.booleanCache = cache.NewInMemory[string, of.BoolResolutionDetail]()
		p.intCache = cache.NewInMemory[string, of.IntResolutionDetail]()
		p.stringCache = cache.NewInMemory[string, of.StringResolutionDetail]()
		p.floatCache = cache.NewInMemory[string, of.FloatResolutionDetail]()
		p.interfaceCache = cache.NewInMemory[string, of.InterfaceResolutionDetail]()

		p.cacheEnabled = true
	}
}

// WithLRUCache applies least recently used caching (github.com/hashicorp/golang-lru).
// The provided size is the limit of the number of cached values for each type of flag. Once the limit is reached each
// new entry replaces the least recently used entry.
func WithLRUCache(size int) ProviderOption {
	return func(p *Provider) {
		boolCache, err := lru.New[string, of.BoolResolutionDetail](size)
		if err != nil {
			log.Errorf("init boolean cache: %v", err)
			return
		}
		p.booleanCache = boolCache
		stringCache, err := lru.New[string, of.StringResolutionDetail](size)
		if err != nil {
			log.Errorf("init string cache: %v", err)
			return
		}
		p.stringCache = stringCache
		intCache, err := lru.New[string, of.IntResolutionDetail](size)
		if err != nil {
			log.Errorf("init int cache: %v", err)
			return
		}
		p.intCache = intCache
		floatCache, err := lru.New[string, of.FloatResolutionDetail](size)
		if err != nil {
			log.Errorf("init float cache: %v", err)
			return
		}
		p.floatCache = floatCache
		interfaceCache, err := lru.New[string, of.InterfaceResolutionDetail](size)
		if err != nil {
			log.Errorf("init interface cache: %v", err)
			return
		}
		p.interfaceCache = interfaceCache

		p.cacheEnabled = true
	}
}

// WithContext supplies the given context to the event stream. Not to be confused with the context used in individual
// flag evaluation requests.
func WithContext(ctx context.Context) ProviderOption {
	return func(p *Provider) {
		p.ctx = ctx
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
	if p.isCacheAvailable() {
		fromCache, ok := p.booleanCache.Get(flagKey)
		if ok {
			fromCache.Reason = constant.ReasonCached
			return fromCache
		}
	}

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

	resDetail := of.BoolResolutionDetail{
		Value: res.Value,
		ProviderResolutionDetail: of.ProviderResolutionDetail{
			Reason:  of.Reason(res.Reason),
			Variant: res.Variant,
		},
	}

	if p.isCacheAvailable() && res.Reason == flagdModels.StaticReason {
		p.booleanCache.Add(flagKey, resDetail)
	}

	return resDetail
}

func (p *Provider) StringEvaluation(
	ctx context.Context, flagKey string, defaultValue string, evalCtx of.FlattenedContext,
) of.StringResolutionDetail {
	if p.isCacheAvailable() {
		fromCache, ok := p.stringCache.Get(flagKey)
		if ok {
			fromCache.Reason = constant.ReasonCached
			return fromCache
		}
	}

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

	resDetail := of.StringResolutionDetail{
		Value: res.Value,
		ProviderResolutionDetail: of.ProviderResolutionDetail{
			Reason:  of.Reason(res.Reason),
			Variant: res.Variant,
		},
	}

	if p.isCacheAvailable() && res.Reason == flagdModels.StaticReason {
		p.stringCache.Add(flagKey, resDetail)
	}

	return resDetail
}

func (p *Provider) FloatEvaluation(
	ctx context.Context, flagKey string, defaultValue float64, evalCtx of.FlattenedContext,
) of.FloatResolutionDetail {
	if p.isCacheAvailable() {
		fromCache, ok := p.floatCache.Get(flagKey)
		if ok {
			fromCache.Reason = constant.ReasonCached
			return fromCache
		}
	}

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

	resDetail := of.FloatResolutionDetail{
		Value: res.Value,
		ProviderResolutionDetail: of.ProviderResolutionDetail{
			Reason:  of.Reason(res.Reason),
			Variant: res.Variant,
		},
	}

	if p.isCacheAvailable() && res.Reason == flagdModels.StaticReason {
		p.floatCache.Add(flagKey, resDetail)
	}

	return resDetail
}

func (p *Provider) IntEvaluation(
	ctx context.Context, flagKey string, defaultValue int64, evalCtx of.FlattenedContext,
) of.IntResolutionDetail {
	if p.isCacheAvailable() {
		fromCache, ok := p.intCache.Get(flagKey)
		if ok {
			fromCache.Reason = constant.ReasonCached
			return fromCache
		}
	}

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

	resDetail := of.IntResolutionDetail{
		Value: res.Value,
		ProviderResolutionDetail: of.ProviderResolutionDetail{
			Reason:  of.Reason(res.Reason),
			Variant: res.Variant,
		},
	}

	if p.isCacheAvailable() && res.Reason == flagdModels.StaticReason {
		p.intCache.Add(flagKey, resDetail)
	}

	return resDetail
}

func (p *Provider) ObjectEvaluation(
	ctx context.Context, flagKey string, defaultValue interface{}, evalCtx of.FlattenedContext,
) of.InterfaceResolutionDetail {
	if p.isCacheAvailable() {
		fromCache, ok := p.interfaceCache.Get(flagKey)
		if ok {
			fromCache.Reason = constant.ReasonCached
			return fromCache
		}
	}

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

	resDetail := of.InterfaceResolutionDetail{
		Value: res.Value,
		ProviderResolutionDetail: of.ProviderResolutionDetail{
			Reason:  of.Reason(res.Reason),
			Variant: res.Variant,
		},
	}

	if p.isCacheAvailable() && res.Reason == flagdModels.StaticReason {
		p.interfaceCache.Add(flagKey, resDetail)
	}

	return resDetail
}

func (p *Provider) isCacheAvailable() bool {
	return p.cacheEnabled && p.Service.IsEventStreamAlive()
}

func (p *Provider) handleEvents(ctx context.Context) error {
	eventChan := make(chan *schemaV1.EventStreamResponse)
	errChan := make(chan error)

	go p.Service.EventStream(ctx, eventChan, errChan)

	for {
		select {
		case event, ok := <-eventChan:
			if !ok {
				if p.cacheEnabled { // disable cache
					WithoutCache()(p)
				}
				return nil
			}

			switch event.Type {
			case string(flagdService.ConfigurationChange):
				if err := p.handleConfigurationChangeEvent(ctx, event); err != nil {
					log.Errorf("handle configuration change event: %v", err)
				}
			}
		case err := <-errChan:
			if p.cacheEnabled { // disable cache
				WithoutCache()(p)
			}
			return err
		}
	}
}

func (p *Provider) handleConfigurationChangeEvent(ctx context.Context, event *schemaV1.EventStreamResponse) error {
	if !p.cacheEnabled {
		return nil
	}

	if event.Data == nil {
		return errors.New("no data in event")
	}

	flagKeyVal, ok := event.Data.AsMap()["flagKey"]
	if !ok {
		return errors.New("no flagKey field in event data")
	}

	flagKey, ok := flagKeyVal.(string)
	if !ok {
		return errors.New("flagKey is not a string")
	}

	// TODO: consider sending the flag type in the configuration change event
	p.booleanCache.Remove(flagKey)
	p.intCache.Remove(flagKey)
	p.floatCache.Remove(flagKey)
	p.interfaceCache.Remove(flagKey)
	p.stringCache.Remove(flagKey)

	return nil
}
