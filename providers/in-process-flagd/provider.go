package flagd

import (
	"context"
	"errors"
	"fmt"
	"github.com/open-feature/flagd/core/pkg/eval"
	"github.com/open-feature/flagd/core/pkg/logger"
	"github.com/open-feature/flagd/core/pkg/runtime"
	"github.com/open-feature/flagd/core/pkg/store"
	"github.com/open-feature/flagd/core/pkg/sync"
	"log"
	sync2 "sync"

	of "github.com/open-feature/go-sdk/pkg/openfeature"
	"os"
	"strconv"
)

const (
	defaultMaxSyncRetries      = 5
	defaultTLS            bool = false

	flagdProxyTLSEnvironmentVariableName          = "FLAGD_PROXY_TLS"
	flagdProxyCertPathEnvironmentVariableName     = "FLAGD_PROXY_CERT_PATH"
	flagdMaxSyncRetriesEnvironmentVariableName    = "FLAGD_MAX_SYNC_RETRIES"
	flagdSyncRetryIntervalEnvironmentVariableName = "FLAGD_MAX_SYNC_RETRY_INTERVAL"
	flagdSourceURIEnvironmentVariableName         = "FLAGD_SOURCE_URI"
	flagdSourceProviderEnvironmentVariableName    = "FLAGD_SOURCE_PROVIDER_TYPE"
	flagdSourceSelectorEnvironmentVariableName    = "FLAGD_SOURCE_SELECTOR"

	syncProviderGrpc       = "grpc"
	syncProviderKubernetes = "kubernetes"
)

type Provider struct {
	ctx                       context.Context
	cacheEnabled              bool
	providerConfiguration     *ProviderConfiguration
	isReady                   chan struct{}
	logger                    *logger.Logger
	otelIntercept             bool
	evaluator                 eval.IEvaluator
	syncSource                sync.ISync
	mu                        sync2.Mutex
	syncConnectionMaxAttempts int
}
type ProviderConfiguration struct {
	Port            uint16
	Host            string
	CertificatePath string
	SocketPath      string
	TLSEnabled      bool
	SourceConfig    *runtime.SourceConfig
}

type ProviderOption func(*Provider)

func NewProvider(ctx context.Context, opts ...ProviderOption) *Provider {
	provider := &Provider{
		ctx: ctx,
		// providerConfiguration maintains its default values, to ensure that the FromEnv option does not overwrite any explicitly set
		// values (default values are then set after the options are run via applyDefaults())
		providerConfiguration: &ProviderConfiguration{},
		isReady:               make(chan struct{}),
		logger:                logger.NewLogger(nil, false),
	}
	provider.applyDefaults()   // defaults have the lowest priority
	FromEnv()(provider)        // env variables have higher priority than defaults
	for _, opt := range opts { // explicitly declared options have the highest priority
		opt(provider)
	}

	if provider.providerConfiguration.SourceConfig == nil {
		log.Fatal(errors.New("no sync source configuration provided"))
	}

	s := store.NewFlags()

	s.FlagSources = append(s.FlagSources, provider.providerConfiguration.SourceConfig.URI)
	s.SourceMetadata[provider.providerConfiguration.SourceConfig.URI] = store.SourceDetails{
		Source:   provider.providerConfiguration.SourceConfig.URI,
		Selector: provider.providerConfiguration.SourceConfig.Selector,
	}

	var err error
	provider.syncSource, err = syncProviderFromConfig(provider.logger, *provider.providerConfiguration.SourceConfig)
	if err != nil {
		log.Fatal(err)
	}

	provider.evaluator = eval.NewJSONEvaluator(
		provider.logger,
		s,
		eval.WithEvaluator(
			"fractionalEvaluation",
			eval.NewFractionalEvaluator(provider.logger).FractionalEvaluation,
		),
		eval.WithEvaluator(
			"starts_with",
			eval.NewStringComparisonEvaluator(provider.logger).StartsWithEvaluation,
		),
		eval.WithEvaluator(
			"ends_with",
			eval.NewStringComparisonEvaluator(provider.logger).EndsWithEvaluation,
		),
		eval.WithEvaluator(
			"sem_ver",
			eval.NewSemVerComparisonEvaluator(provider.logger).SemVerEvaluation,
		),
	)

	dataSync := make(chan sync.DataSync, 1)

	go provider.watchForUpdates(dataSync)

	if err := provider.syncSource.Init(provider.ctx); err != nil {
		log.Fatal("sync provider Init returned error: %w", err)
	}

	// Start sync provider
	go provider.startSyncSources(dataSync)

	return provider
}

func (p *Provider) startSyncSources(dataSync chan sync.DataSync) {
	if err := p.syncSource.Sync(p.ctx, dataSync); err != nil {
		p.logger.Error(fmt.Sprintf("Error during source sync: %v", err))
	}
}

func (p *Provider) watchForUpdates(dataSync chan sync.DataSync) error {
	for {
		select {
		case data := <-dataSync:
			// resync events are triggered when a delete occurs during flag merges in the store
			// resync events may trigger further resync events, however for a flag to be deleted from the store
			// its source must match, preventing the opportunity for resync events to snowball
			if resyncRequired := p.updateWithNotify(data); resyncRequired {
				go func() {
					err := p.syncSource.ReSync(p.ctx, dataSync)
					if err != nil {
						// TODO put provider in ERROR state
						p.logger.Error(fmt.Sprintf("error resyncing sources: %v", err))
					}
				}()
			}
			if data.Type == sync.ALL {
				p.handleProviderReady()
			}
		case <-p.ctx.Done():
			return nil
		}
	}
}

func (p *Provider) handleProviderReady() {
	select {
	case <-p.isReady:
		// avoids panic from closing already closed channel
	default:
		close(p.isReady)
	}
}

func (p *Provider) applyDefaults() {
	p.providerConfiguration.TLSEnabled = defaultTLS
}

// FromEnv sets the provider configuration from environment variables (if set) as defined https://github.com/open-feature/flagd/blob/main/docs/other_resources/creating_providers.md?plain=1#L117
func FromEnv() ProviderOption {
	return func(p *Provider) {
		certificatePath := os.Getenv(flagdProxyCertPathEnvironmentVariableName)
		if certificatePath != "" || os.Getenv(flagdProxyTLSEnvironmentVariableName) == "true" {
			WithTLS(certificatePath)(p)
		}

		maxSynccRetriesS := os.Getenv(flagdMaxSyncRetriesEnvironmentVariableName)
		if maxSynccRetriesS != "" {
			maxSyncRetries, err := strconv.Atoi(maxSynccRetriesS)
			if err != nil {
				p.logger.Error(
					fmt.Sprintf("invalid env config for %s provided, using default value: %d",
						flagdMaxSyncRetriesEnvironmentVariableName, defaultMaxSyncRetries))
			} else {
				p.syncConnectionMaxAttempts = maxSyncRetries
			}
		}

		sourceURI := os.Getenv(flagdSourceURIEnvironmentVariableName)
		sourceProvider := os.Getenv(flagdSourceProviderEnvironmentVariableName)
		selector := os.Getenv(flagdSourceSelectorEnvironmentVariableName)

		if sourceURI != "" && sourceProvider != "" {
			p.providerConfiguration.SourceConfig = &runtime.SourceConfig{
				URI:      sourceURI,
				Provider: sourceProvider,
				Selector: selector,
				CertPath: p.providerConfiguration.CertificatePath,
				TLS:      p.providerConfiguration.TLSEnabled,
			}
		}
	}
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
		p.syncConnectionMaxAttempts = i
	}
}

// WithLogger sets the logger used by the provider.
func WithLogger(l *logger.Logger) ProviderOption {
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
		p.otelIntercept = intercept
	}
}

// WithSyncSource sets the sync source config of the provider
func WithSyncSource(sourceConfig *runtime.SourceConfig) ProviderOption {
	return func(p *Provider) {
		p.providerConfiguration.SourceConfig = sourceConfig
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

	value, variant, reason, metadata, err := p.evaluator.ResolveBooleanValue(ctx, "", flagKey, evalCtx)

	if err != nil {
		var e of.ResolutionError
		if !errors.As(err, &e) {
			e = of.NewGeneralResolutionError(err.Error())
		}

		return of.BoolResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				ResolutionError: e,
				Reason:          of.Reason(reason),
				Variant:         variant,
				FlagMetadata:    metadata,
			},
		}
	}

	resDetail := of.BoolResolutionDetail{
		Value: value,
		ProviderResolutionDetail: of.ProviderResolutionDetail{
			Reason:       of.Reason(reason),
			Variant:      variant,
			FlagMetadata: metadata,
		},
	}

	return resDetail
}

func (p *Provider) StringEvaluation(
	ctx context.Context, flagKey string, defaultValue string, evalCtx of.FlattenedContext,
) of.StringResolutionDetail {
	value, variant, reason, metadata, err := p.evaluator.ResolveStringValue(ctx, "", flagKey, evalCtx)

	if err != nil {
		var e of.ResolutionError
		if !errors.As(err, &e) {
			e = of.NewGeneralResolutionError(err.Error())
		}

		return of.StringResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				ResolutionError: e,
				Reason:          of.Reason(reason),
				Variant:         variant,
				FlagMetadata:    metadata,
			},
		}
	}

	resDetail := of.StringResolutionDetail{
		Value: value,
		ProviderResolutionDetail: of.ProviderResolutionDetail{
			Reason:       of.Reason(reason),
			Variant:      variant,
			FlagMetadata: metadata,
		},
	}

	return resDetail
}

func (p *Provider) FloatEvaluation(
	ctx context.Context, flagKey string, defaultValue float64, evalCtx of.FlattenedContext,
) of.FloatResolutionDetail {
	value, variant, reason, metadata, err := p.evaluator.ResolveFloatValue(ctx, "", flagKey, evalCtx)
	if err != nil {
		var e of.ResolutionError
		if !errors.As(err, &e) {
			e = of.NewGeneralResolutionError(err.Error())
		}

		return of.FloatResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				ResolutionError: e,
				Reason:          of.Reason(reason),
				Variant:         variant,
				FlagMetadata:    metadata,
			},
		}
	}

	resDetail := of.FloatResolutionDetail{
		Value: value,
		ProviderResolutionDetail: of.ProviderResolutionDetail{
			Reason:       of.Reason(reason),
			Variant:      variant,
			FlagMetadata: metadata,
		},
	}

	return resDetail
}

func (p *Provider) IntEvaluation(
	ctx context.Context, flagKey string, defaultValue int64, evalCtx of.FlattenedContext,
) of.IntResolutionDetail {
	value, variant, reason, metadata, err := p.evaluator.ResolveIntValue(ctx, "", flagKey, evalCtx)
	if err != nil {
		var e of.ResolutionError
		if !errors.As(err, &e) {
			e = of.NewGeneralResolutionError(err.Error())
		}

		return of.IntResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				ResolutionError: e,
				Reason:          of.Reason(reason),
				Variant:         variant,
				FlagMetadata:    metadata,
			},
		}
	}

	resDetail := of.IntResolutionDetail{
		Value: value,
		ProviderResolutionDetail: of.ProviderResolutionDetail{
			Reason:       of.Reason(reason),
			Variant:      variant,
			FlagMetadata: metadata,
		},
	}

	return resDetail
}

func (p *Provider) ObjectEvaluation(
	ctx context.Context, flagKey string, defaultValue interface{}, evalCtx of.FlattenedContext,
) of.InterfaceResolutionDetail {

	value, variant, reason, metadata, err := p.evaluator.ResolveObjectValue(ctx, "", flagKey, evalCtx)
	if err != nil {
		var e of.ResolutionError
		if !errors.As(err, &e) {
			e = of.NewGeneralResolutionError(err.Error())
		}

		return of.InterfaceResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				ResolutionError: e,
				Reason:          of.Reason(reason),
				Variant:         variant,
				FlagMetadata:    metadata,
			},
		}
	}

	resDetail := of.InterfaceResolutionDetail{
		Value: value,
		ProviderResolutionDetail: of.ProviderResolutionDetail{
			Reason:       of.Reason(reason),
			Variant:      variant,
			FlagMetadata: metadata,
		},
	}

	return resDetail
}

// IsReady returns a non-blocking channel if the provider has completed the initial flag sync
func (p *Provider) IsReady() <-chan struct{} {
	return p.isReady
}

// updateWithNotify helps to update state and notify listeners
func (p *Provider) updateWithNotify(payload sync.DataSync) bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	_, resyncRequired, err := p.evaluator.SetState(payload)
	if err != nil {
		p.logger.Error(err.Error())
		return false
	}

	return resyncRequired
}

// syncProviderFromConfig is a helper to build ISync implementations from SourceConfig
func syncProviderFromConfig(logger *logger.Logger, sourceConfig runtime.SourceConfig) (sync.ISync, error) {
	switch sourceConfig.Provider {
	case syncProviderKubernetes:
		k8sSync, err := runtime.NewK8s(sourceConfig.URI, logger)
		if err != nil {
			return nil, err
		}
		logger.Debug(fmt.Sprintf("using kubernetes sync-provider for: %s", sourceConfig.URI))
		return k8sSync, nil
	case syncProviderGrpc:
		logger.Debug(fmt.Sprintf("using grpc sync-provider for: %s", sourceConfig.URI))
		return runtime.NewGRPC(sourceConfig, logger), nil

	default:
		return nil, fmt.Errorf("invalid sync provider: %s, must be one of with '%s' or '%s'",
			sourceConfig.Provider, syncProviderGrpc, syncProviderKubernetes)
	}
}
