package flagd

import (
	schemaV1 "buf.build/gen/go/open-feature/flagd/protocolbuffers/go/schema/v1"
	"context"
	"errors"
	"github.com/go-logr/logr"
	lru "github.com/hashicorp/golang-lru/v2"
	flagdModels "github.com/open-feature/flagd/core/pkg/model"
	flagdService "github.com/open-feature/flagd/core/pkg/service"
	"github.com/open-feature/go-sdk-contrib/providers/flagd/internal/logger"
	"github.com/open-feature/go-sdk-contrib/providers/flagd/pkg/cache"
	"github.com/open-feature/go-sdk-contrib/providers/flagd/pkg/constant"
	"github.com/open-feature/go-sdk-contrib/providers/flagd/pkg/service"
	of "github.com/open-feature/go-sdk/pkg/openfeature"
)

type Provider struct {
	cache                 Cache[string, interface{}]
	ctx                   context.Context
	isReady               chan struct{}
	logger                logr.Logger
	providerConfiguration *providerConfiguration
	service               service.IService
}

func NewProvider(opts ...ProviderOption) *Provider {
	log := logr.New(logger.Logger{})

	// initialize with default configurations
	configuration := newDefaultConfiguration()

	// env variables have higher priority than defaults
	configuration.updateFromEnvVar(log)

	provider := &Provider{
		ctx:                   context.Background(),
		isReady:               make(chan struct{}),
		logger:                log,
		providerConfiguration: configuration,
	}

	// explicitly declared options have the highest priority
	for _, opt := range opts {
		opt(provider)
	}

	setupCache(provider)

	provider.service = service.NewService(service.NewClient(
		&service.Configuration{
			Host:            provider.providerConfiguration.Host,
			Port:            provider.providerConfiguration.Port,
			CertificatePath: provider.providerConfiguration.CertificatePath,
			SocketPath:      provider.providerConfiguration.SocketPath,
			TLSEnabled:      provider.providerConfiguration.TLSEnabled,
			OtelInterceptor: provider.providerConfiguration.otelIntercept,
		},
	), provider.logger, nil)

	go func() {
		if err := provider.handleEvents(provider.ctx); err != nil {
			provider.logger.Error(err, "handle events")
		}
	}()

	return provider
}

// setupCache helper to setup cache
func setupCache(provider *Provider) {
	var c Cache[string, interface{}]
	var err error

	// setup cache
	switch provider.providerConfiguration.CacheType {
	case cacheLRUValue:
		c, err = lru.New[string, interface{}](provider.providerConfiguration.MaxCacheSize)
		if err != nil {
			provider.logger.Error(err, "init lru cache")
		} else {
			provider.providerConfiguration.CacheEnabled = true
		}
	case cacheInMemValue:
		c = cache.NewInMemory[string, interface{}]()
		provider.providerConfiguration.CacheEnabled = true
	case cacheDisabledValue:
	default:
		provider.providerConfiguration.CacheEnabled = false
		c = nil
	}

	provider.cache = c
}

// ProviderOptions

type ProviderOption func(*Provider)

// WithSocketPath overrides the default hostname and port, a unix socket connection is made to flagd instead
func WithSocketPath(socketPath string) ProviderOption {
	return func(p *Provider) {
		p.providerConfiguration.SocketPath = socketPath
	}
}

// WithCertificatePath specifies the location of the certificate to be used in the gRPC dial credentials.
// If certificate loading fails insecure credentials will be used instead
func WithCertificatePath(path string) ProviderOption {
	return func(p *Provider) {
		p.providerConfiguration.CertificatePath = path
		p.providerConfiguration.TLSEnabled = true
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
		p.providerConfiguration.CacheType = cacheDisabledValue
	}
}

// WithBasicInMemoryCache applies a basic in memory cache store (with no memory limits)
func WithBasicInMemoryCache() ProviderOption {
	return func(p *Provider) {
		p.providerConfiguration.CacheType = cacheInMemValue
	}
}

// WithLRUCache applies least recently used caching (github.com/hashicorp/golang-lru).
// The provided size is the limit of the number of cached values. Once the limit is reached each new entry replaces the
// least recently used entry.
func WithLRUCache(size int) ProviderOption {
	return func(p *Provider) {
		if size > 0 {
			p.providerConfiguration.MaxCacheSize = size
		}
		p.providerConfiguration.CacheType = cacheLRUValue
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
		p.providerConfiguration.EventStreamConnectionMaxAttempts = i
	}
}

// WithLogger sets the logger used by the provider.
func WithLogger(l logr.Logger) ProviderOption {
	return func(p *Provider) {
		p.logger = l
	}
}

// WithTLS enables TLS. If certPath is not given, system certs are used.
func WithTLS(certPath string) ProviderOption {
	return func(p *Provider) {
		p.providerConfiguration.TLSEnabled = true
		p.providerConfiguration.CertificatePath = certPath
	}
}

// WithOtelInterceptor enable/disable otel interceptor for flagd communication
func WithOtelInterceptor(intercept bool) ProviderOption {
	return func(p *Provider) {
		p.providerConfiguration.otelIntercept = intercept
	}
}

// FromEnv sets the provider configuration from environment variables (if set)
func FromEnv() ProviderOption {
	return func(p *Provider) {
		p.providerConfiguration.updateFromEnvVar(p.logger)
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

	res, err := p.service.ResolveBoolean(ctx, flagKey, evalCtx)
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
				FlagMetadata:    res.Metadata.AsMap(),
			},
		}
	}

	resDetail := of.BoolResolutionDetail{
		Value: res.Value,
		ProviderResolutionDetail: of.ProviderResolutionDetail{
			Reason:       of.Reason(res.Reason),
			Variant:      res.Variant,
			FlagMetadata: res.Metadata.AsMap(),
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

	res, err := p.service.ResolveString(ctx, flagKey, evalCtx)
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
				FlagMetadata:    res.Metadata.AsMap(),
			},
		}
	}

	resDetail := of.StringResolutionDetail{
		Value: res.Value,
		ProviderResolutionDetail: of.ProviderResolutionDetail{
			Reason:       of.Reason(res.Reason),
			Variant:      res.Variant,
			FlagMetadata: res.Metadata.AsMap(),
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

	res, err := p.service.ResolveFloat(ctx, flagKey, evalCtx)
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
				FlagMetadata:    res.Metadata.AsMap(),
			},
		}
	}

	resDetail := of.FloatResolutionDetail{
		Value: res.Value,
		ProviderResolutionDetail: of.ProviderResolutionDetail{
			Reason:       of.Reason(res.Reason),
			Variant:      res.Variant,
			FlagMetadata: res.Metadata.AsMap(),
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

	res, err := p.service.ResolveInt(ctx, flagKey, evalCtx)
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
				FlagMetadata:    res.Metadata.AsMap(),
			},
		}
	}

	resDetail := of.IntResolutionDetail{
		Value: res.Value,
		ProviderResolutionDetail: of.ProviderResolutionDetail{
			Reason:       of.Reason(res.Reason),
			Variant:      res.Variant,
			FlagMetadata: res.Metadata.AsMap(),
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

	res, err := p.service.ResolveObject(ctx, flagKey, evalCtx)
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
				FlagMetadata:    res.Metadata.AsMap(),
			},
		}
	}

	resDetail := of.InterfaceResolutionDetail{
		Value: res.Value,
		ProviderResolutionDetail: of.ProviderResolutionDetail{
			Reason:       of.Reason(res.Reason),
			Variant:      res.Variant,
			FlagMetadata: res.Metadata.AsMap(),
		},
	}

	if p.isCacheAvailable() && res.Reason == flagdModels.StaticReason {
		p.cache.Add(flagKey, resDetail)
	}

	return resDetail
}

func (p *Provider) isCacheAvailable() bool {
	return p.providerConfiguration.CacheEnabled && p.service.IsEventStreamAlive()
}

// todo: Event handling to be isolated to own component

func (p *Provider) handleEvents(ctx context.Context) error {
	eventChan := make(chan *schemaV1.EventStreamResponse)
	errChan := make(chan error)

	go func() {
		p.service.EventStream(ctx, eventChan, p.providerConfiguration.EventStreamConnectionMaxAttempts, errChan)
	}()

	for {
		select {
		case event, ok := <-eventChan:
			if !ok {
				if p.providerConfiguration.CacheEnabled { // disable cache
					p.providerConfiguration.CacheEnabled = false
				}
				return nil
			}

			switch event.Type {
			case string(flagdService.ConfigurationChange):
				if err := p.handleConfigurationChangeEvent(ctx, event); err != nil {
					// Purge the cache if we fail to handle the configuration change event
					p.cache.Purge()

					p.logger.V(logger.Warn).Info("handle configuration change event", "err", err)
				}
			case string(flagdService.ProviderReady): // signals that a new connection has been made
				p.handleProviderReadyEvent()
			}
		case err := <-errChan:
			if p.providerConfiguration.CacheEnabled { // disable cache
				p.providerConfiguration.CacheEnabled = false
			}
			return err
		case <-ctx.Done():
			p.logger.V(logger.Info).Info("stop event handling with context done")

			return nil
		}
	}
}

func (p *Provider) handleConfigurationChangeEvent(ctx context.Context, event *schemaV1.EventStreamResponse) error {
	if !p.providerConfiguration.CacheEnabled {
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

	if !p.providerConfiguration.CacheEnabled {
		return
	}

	p.cache.Purge() // in case events were missed while the connection was down
}

// IsReady returns a non-blocking channel if the provider has received the provider_ready event from flagd
func (p *Provider) IsReady() <-chan struct{} {
	return p.isReady
}
