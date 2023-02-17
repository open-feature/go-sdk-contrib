package flagd

import (
	schemaV1 "buf.build/gen/go/open-feature/flagd/protocolbuffers/go/schema/v1"
	"context"
	"errors"
	"fmt"
	"github.com/go-logr/logr"
	lru "github.com/hashicorp/golang-lru/v2"
	flagdModels "github.com/open-feature/flagd/pkg/model"
	flagdService "github.com/open-feature/flagd/pkg/service"
	"github.com/open-feature/go-sdk-contrib/providers/flagd/internal/logger"
	"github.com/open-feature/go-sdk-contrib/providers/flagd/pkg/cache"
	"github.com/open-feature/go-sdk-contrib/providers/flagd/pkg/constant"
	"github.com/open-feature/go-sdk-contrib/providers/flagd/pkg/service"
	of "github.com/open-feature/go-sdk/pkg/openfeature"
	"os"
	"strconv"
)

// naming and defaults follow https://github.com/open-feature/flagd/blob/main/docs/other_resources/creating_providers.md?plain=1#L117
const (
	defaultMaxCacheSize                               int    = 1000
	defaultPort                                              = 8013
	defaultMaxEventStreamRetries                             = 5
	defaultTLS                                        bool   = false
	defaultCache                                      string = "lru"
	defaultHost                                              = "localhost"
	flagdHostEnvironmentVariableName                         = "FLAGD_HOST"
	flagdPortEnvironmentVariableName                         = "FLAGD_PORT"
	flagdTLSEnvironmentVariableName                          = "FLAGD_TLS"
	flagdSocketPathEnvironmentVariableName                   = "FLAGD_SOCKET_PATH"
	flagdServerCertPathEnvironmentVariableName               = "FLAGD_SERVER_CERT_PATH"
	flagdCacheEnvironmentVariableName                        = "FLAGD_CACHE"
	flagdMaxCacheSizeEnvironmentVariableName                 = "FLAGD_MAX_CACHE_SIZE"
	flagdMaxEventStreamRetriesEnvironmentVariableName        = "FLAGD_MAX_EVENT_STREAM_RETRIES"
	cacheDisabledValue                                       = "disabled"
	cacheLRUValue                                            = "lru"
)

type Provider struct {
	ctx                              context.Context
	service                          service.IService
	cacheEnabled                     bool
	cache                            Cache[string, interface{}]
	maxCacheSize                     int
	providerConfiguration            *ProviderConfiguration
	eventStreamConnectionMaxAttempts int
	isReady                          chan struct{}
	logger                           logr.Logger
}
type ProviderConfiguration struct {
	Port            uint16
	Host            string
	CertificatePath string
	SocketPath      string
	TLSEnabled      bool
}

type ProviderOption func(*Provider)

func NewProvider(opts ...ProviderOption) *Provider {
	provider := &Provider{
		ctx: context.Background(),
		// providerConfiguration maintains its default values, to ensure that the FromEnv option does not overwrite any explicitly set
		// values (default values are then set after the options are run via applyDefaults())
		providerConfiguration: &ProviderConfiguration{},
		isReady:               make(chan struct{}),
		logger:                logr.New(logger.Logger{}),
	}
	provider.applyDefaults()   // defaults have the lowest priority
	FromEnv()(provider)        // env variables have higher priority than defaults
	for _, opt := range opts { // explicitly declared options have the highest priority
		opt(provider)
	}
	provider.service = service.NewService(&service.Client{
		ServiceConfiguration: &service.ServiceConfiguration{
			Host:            provider.providerConfiguration.Host,
			Port:            provider.providerConfiguration.Port,
			CertificatePath: provider.providerConfiguration.CertificatePath,
			SocketPath:      provider.providerConfiguration.SocketPath,
			TLSEnabled:      provider.providerConfiguration.TLSEnabled,
		},
	}, provider.logger, nil)

	go func() {
		if err := provider.handleEvents(provider.ctx); err != nil {
			provider.logger.Error(err, "handle events")
		}
	}()

	return provider
}

func (p *Provider) applyDefaults() {
	p.providerConfiguration.Host = defaultHost
	p.providerConfiguration.Port = defaultPort
	p.providerConfiguration.TLSEnabled = defaultTLS
	p.eventStreamConnectionMaxAttempts = defaultMaxEventStreamRetries
	p.maxCacheSize = defaultMaxCacheSize
	p.withCache(defaultCache)
}

// WithSocketPath overrides the default hostname and port, a unix socket connection is made to flagd instead
func WithSocketPath(socketPath string) ProviderOption {
	return func(s *Provider) {
		s.providerConfiguration.SocketPath = socketPath
	}
}

// FromEnv sets the provider configuration from environment variables (if set) as defined https://github.com/open-feature/flagd/blob/main/docs/other_resources/creating_providers.md?plain=1#L117
func FromEnv() ProviderOption {
	return func(p *Provider) {
		portS := os.Getenv(flagdPortEnvironmentVariableName)
		if portS != "" {
			port, err := strconv.Atoi(portS)
			if err != nil {
				p.logger.Error(err, fmt.Sprintf("invalid env config for %s provided, using default value", flagdPortEnvironmentVariableName))
			} else {
				p.providerConfiguration.Port = uint16(port)
			}
		}

		host := os.Getenv(flagdHostEnvironmentVariableName)
		if host != "" {
			p.providerConfiguration.Host = host
		}

		socketPath := os.Getenv(flagdSocketPathEnvironmentVariableName)
		if socketPath != "" {
			p.providerConfiguration.SocketPath = socketPath
		}

		certificatePath := os.Getenv(flagdServerCertPathEnvironmentVariableName)
		if certificatePath != "" || os.Getenv(flagdTLSEnvironmentVariableName) == "true" {
			WithTLS(certificatePath)(p)
		}

		maxCacheSizeS := os.Getenv(flagdMaxCacheSizeEnvironmentVariableName)
		if maxCacheSizeS != "" {
			maxCacheSizeFromEnv, err := strconv.Atoi(maxCacheSizeS)
			if err != nil {
				p.logger.Error(err, fmt.Sprintf("invalid env config for %s provided, using default value", flagdMaxCacheSizeEnvironmentVariableName))
			} else {
				p.maxCacheSize = maxCacheSizeFromEnv
			}
		}

		if cacheValue := os.Getenv(flagdCacheEnvironmentVariableName); cacheValue != "" {
			if ok := p.withCache(cacheValue); !ok {
				p.logger.Error(fmt.Errorf("%s is invalid", cacheValue), fmt.Sprintf("invalid env config for %s provided, using default value", flagdCacheEnvironmentVariableName))
			}
		}

		maxEventStreamRetriesS := os.Getenv(flagdMaxEventStreamRetriesEnvironmentVariableName)
		if maxEventStreamRetriesS != "" {
			maxEventStreamRetries, err := strconv.Atoi(maxEventStreamRetriesS)
			if err != nil {
				p.logger.Error(err, fmt.Sprintf("invalid env config for %s provided, using default value", flagdMaxEventStreamRetriesEnvironmentVariableName))
			} else {
				p.eventStreamConnectionMaxAttempts = maxEventStreamRetries
			}
		}
	}
}

func (p *Provider) withCache(cache string) bool {
	switch cache {
	case cacheDisabledValue:
		WithoutCache()(p)
	case cacheLRUValue:
		WithLRUCache(p.maxCacheSize)
	default:
		return false
	}

	return true
}

// WithCertificatePath specifies the location of the certificate to be used in the gRPC dial credentials. If certificate loading fails insecure credentials will be used instead
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
		if size != 0 {
			p.maxCacheSize = size
		}
		c, err := lru.New[string, interface{}](p.maxCacheSize)
		if err != nil {
			p.logger.Error(err, "init lru cache")
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
	return p.cacheEnabled && p.service.IsEventStreamAlive()
}

func (p *Provider) handleEvents(ctx context.Context) error {
	eventChan := make(chan *schemaV1.EventStreamResponse)
	errChan := make(chan error)

	go func() {
		p.service.EventStream(ctx, eventChan, p.eventStreamConnectionMaxAttempts, errChan)
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

					p.logger.V(logger.Warn).Info("handle configuration change event", "err", err)
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
			p.logger.V(logger.Info).Info("stop event handling with context done")

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
