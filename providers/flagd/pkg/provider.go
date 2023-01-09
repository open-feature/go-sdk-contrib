package flagd

import (
	schemaV1 "buf.build/gen/go/open-feature/flagd/protocolbuffers/go/schema/v1"
	"context"
	"errors"
	"fmt"
	lru "github.com/hashicorp/golang-lru/v2"
	flagdModels "github.com/open-feature/flagd/pkg/model"
	flagdService "github.com/open-feature/flagd/pkg/service"
	"github.com/open-feature/go-sdk-contrib/providers/flagd/pkg/cache"
	"github.com/open-feature/go-sdk-contrib/providers/flagd/pkg/constant"
	"github.com/open-feature/go-sdk-contrib/providers/flagd/pkg/service"
	of "github.com/open-feature/go-sdk/pkg/openfeature"
	log "github.com/sirupsen/logrus"
	"os"
	"strconv"
)

const defaultLRUCacheSize int = 1000

type Provider struct {
	ctx                              context.Context
	Service                          service.IService
	cacheEnabled                     bool
	cache                            Cache[string, interface{}]
	providerConfiguration            *ProviderConfiguration
	eventStreamConnectionMaxAttempts int
	isReady                          chan struct{}
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
		providerConfiguration:            &ProviderConfiguration{},
		eventStreamConnectionMaxAttempts: 5,
		isReady:                          make(chan struct{}),
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
	}, nil)

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
		p.cache = nil
	}
}

// WithBasicInMemoryCache applies a basic in memory cache store (with no memory limits)
func WithBasicInMemoryCache() ProviderOption {
	return func(p *Provider) {
		p.cache = cache.NewInMemory[string, interface{}]()

		p.cacheEnabled = true
	}
}

// WithLRUCache applies least recently used caching (github.com/hashicorp/golang-lru).
// The provided size is the limit of the number of cached values. Once the limit is reached each new entry replaces the
// least recently used entry.
func WithLRUCache(size int) ProviderOption {
	return func(p *Provider) {
		c, err := lru.New[string, interface{}](size)
		if err != nil {
			log.Errorf("init cache: %v", err)
			return
		}
		p.cache = c

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

// WithEventStreamConnectionMaxAttempts sets the maximum number of attempts to connect to flagd's event stream.
// On successful connection the attempts are reset.
func WithEventStreamConnectionMaxAttempts(i int) ProviderOption {
	return func(p *Provider) {
		p.eventStreamConnectionMaxAttempts = i
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
		fromCache, ok := p.cache.Get(flagKey)
		if ok {
			fromCacheResDetail, ok := fromCache.(of.BoolResolutionDetail)
			if ok {
				fromCacheResDetail.Reason = constant.ReasonCached
				return fromCacheResDetail
			}
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
		p.cache.Add(flagKey, resDetail)
	}

	return resDetail
}

func (p *Provider) StringEvaluation(
	ctx context.Context, flagKey string, defaultValue string, evalCtx of.FlattenedContext,
) of.StringResolutionDetail {
	if p.isCacheAvailable() {
		fromCache, ok := p.cache.Get(flagKey)
		if ok {
			fromCacheResDetail, ok := fromCache.(of.StringResolutionDetail)
			if ok {
				fromCacheResDetail.Reason = constant.ReasonCached
				return fromCacheResDetail
			}
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
		p.cache.Add(flagKey, resDetail)
	}

	return resDetail
}

func (p *Provider) FloatEvaluation(
	ctx context.Context, flagKey string, defaultValue float64, evalCtx of.FlattenedContext,
) of.FloatResolutionDetail {
	if p.isCacheAvailable() {
		fromCache, ok := p.cache.Get(flagKey)
		if ok {
			fromCacheResDetail, ok := fromCache.(of.FloatResolutionDetail)
			if ok {
				fromCacheResDetail.Reason = constant.ReasonCached
				return fromCacheResDetail
			}
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
		p.cache.Add(flagKey, resDetail)
	}

	return resDetail
}

func (p *Provider) IntEvaluation(
	ctx context.Context, flagKey string, defaultValue int64, evalCtx of.FlattenedContext,
) of.IntResolutionDetail {
	if p.isCacheAvailable() {
		fromCache, ok := p.cache.Get(flagKey)
		if ok {
			fromCacheResDetail, ok := fromCache.(of.IntResolutionDetail)
			if ok {
				fromCacheResDetail.Reason = constant.ReasonCached
				return fromCacheResDetail
			}
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
		p.cache.Add(flagKey, resDetail)
	}

	return resDetail
}

func (p *Provider) ObjectEvaluation(
	ctx context.Context, flagKey string, defaultValue interface{}, evalCtx of.FlattenedContext,
) of.InterfaceResolutionDetail {
	if p.isCacheAvailable() {
		fromCache, ok := p.cache.Get(flagKey)
		if ok {
			fromCacheResDetail, ok := fromCache.(of.InterfaceResolutionDetail)
			if ok {
				fromCacheResDetail.Reason = constant.ReasonCached
				return fromCacheResDetail
			}
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
		p.cache.Add(flagKey, resDetail)
	}

	return resDetail
}

func (p *Provider) isCacheAvailable() bool {
	return p.cacheEnabled && p.Service.IsEventStreamAlive()
}

func (p *Provider) handleEvents(ctx context.Context) error {
	eventChan := make(chan *schemaV1.EventStreamResponse)
	errChan := make(chan error)

	go func() {
		p.Service.EventStream(ctx, eventChan, p.eventStreamConnectionMaxAttempts, errChan)
	}()

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
					// Purge the cache if we fail to handle the configuration change event
					p.cache.Purge()

					log.Warningf("handle configuration change event: %v", err)
				}
			case string(flagdService.ProviderReady): // signals that a new connection has been made
				p.handleProviderReadyEvent()
			}
		case err := <-errChan:
			if p.cacheEnabled { // disable cache
				WithoutCache()(p)
			}
			return err
		case <-ctx.Done():
			log.Info("Stop event handling with context done.")

			return nil
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

	flagsVal, ok := event.Data.AsMap()["flags"]
	if !ok {
		return errors.New("no flags field in event data")
	}

	flags, ok := flagsVal.(map[string]interface{})
	if !ok {
		return errors.New("flags isn't a map")
	}

	for flagKey := range flags {
		p.cache.Remove(flagKey)
	}

	return nil
}

func (p *Provider) handleProviderReadyEvent() {
	select {
	case <-p.isReady:
		// avoids panic from closing already closed channel
	default:
		close(p.isReady)
	}

	if !p.cacheEnabled {
		return
	}

	p.cache.Purge() // in case events were missed while the connection was down
}

// IsReady returns a non-blocking channel if the provider has received the provider_ready event from flagd
func (p *Provider) IsReady() <-chan struct{} {
	return p.isReady
}
