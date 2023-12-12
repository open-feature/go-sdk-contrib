package pkg

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/open-feature/flagd/core/pkg/eval"
	"github.com/open-feature/flagd/core/pkg/logger"
	"github.com/open-feature/flagd/core/pkg/model"
	"github.com/open-feature/flagd/core/pkg/runtime"
	"github.com/open-feature/flagd/core/pkg/store"
	ofsync "github.com/open-feature/flagd/core/pkg/sync"

	"os"
	"strconv"

	of "github.com/open-feature/go-sdk/openfeature"
)

type ProviderType string

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

	SourceTypeGrpc       ProviderType = "grpc"
	SourceTypeKubernetes ProviderType = "kubernetes"
)

const (
	stateNotReady connectionState = iota
	stateReady
	stateError
	stateStale
)

const defaultInitBackoffDuration = 2 * time.Second
const defaultMaxBackoffDuration = 120 * time.Second

type connectionInfo struct {
	state              connectionState
	retries            int
	maxSyncRetries     int
	backoffDuration    time.Duration
	maxBackoffDuration time.Duration
}

type connectionState int

// Deprecated: Please use flagd with WithInProcessResolver option instead of this dedicated provider
type Provider struct {
	ctx                   context.Context
	cancelFunc            context.CancelFunc
	cacheEnabled          bool
	providerConfiguration *ProviderConfiguration
	isReady               chan struct{}
	connectionInfo        connectionInfo
	logger                *logger.Logger
	evaluator             eval.IEvaluator
	syncSource            ofsync.ISync
	mu                    sync.Mutex
	ofEventChannel        chan of.Event
}
type ProviderConfiguration struct {
	CertificatePath string
	TLSEnabled      bool
	SourceConfig    runtime.SourceConfig
}

type ProviderOption func(*Provider)

// Deprecated : Please use flagd with WithInProcessResolver option instead of this dedicated provider
func NewProvider(ctx context.Context, opts ...ProviderOption) *Provider {
	ctx, cancel := context.WithCancel(ctx)
	provider := &Provider{
		ctx:        ctx,
		cancelFunc: cancel,
		// providerConfiguration maintains its default values, to ensure that the FromEnv option does not overwrite any explicitly set
		// values (default values are then set after the options are run via applyDefaults())
		providerConfiguration: &ProviderConfiguration{},
		isReady:               make(chan struct{}),
		connectionInfo: connectionInfo{
			state:              stateNotReady,
			retries:            0,
			maxSyncRetries:     defaultMaxSyncRetries,
			backoffDuration:    defaultInitBackoffDuration,
			maxBackoffDuration: defaultMaxBackoffDuration,
		},
		ofEventChannel: make(chan of.Event),
		logger:         logger.NewLogger(nil, false),
	}
	provider.applyDefaults()   // defaults have the lowest priority
	FromEnv()(provider)        // env variables have higher priority than defaults
	for _, opt := range opts { // explicitly declared options have the highest priority
		opt(provider)
	}

	if provider.providerConfiguration.SourceConfig.URI == "" {
		log.Fatal(errors.New("no sync source configuration provided"))
	}

	s := store.NewFlags()

	s.FlagSources = append(s.FlagSources, provider.providerConfiguration.SourceConfig.URI)
	s.SourceMetadata[provider.providerConfiguration.SourceConfig.URI] = store.SourceDetails{
		Source:   provider.providerConfiguration.SourceConfig.URI,
		Selector: provider.providerConfiguration.SourceConfig.Selector,
	}

	var err error
	provider.syncSource, err = syncProviderFromConfig(provider.logger, provider.providerConfiguration.SourceConfig)
	if err != nil {
		log.Fatal(err)
	}

	provider.evaluator = eval.NewJSONEvaluator(
		provider.logger,
		s,
		eval.WithEvaluator(
			"fractional",
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

	dataSync := make(chan ofsync.DataSync, 1)

	go provider.watchForUpdates(dataSync)

	if err := provider.syncSource.Init(provider.ctx); err != nil {
		log.Fatal("sync provider Init returned error: %w", err)
	}

	// Start sync provider
	go provider.startSyncSource(dataSync)

	return provider
}

func (p *Provider) startSyncSource(dataSync chan ofsync.DataSync) {
	for {
		if err := p.syncSource.Sync(p.ctx, dataSync); err != nil {
			p.handleConnectionErr(fmt.Errorf("error during source sync: %w", err))
		}
	}

}

func (p *Provider) watchForUpdates(dataSync chan ofsync.DataSync) error {
	for {
		select {
		case data := <-dataSync:
			// resync events are triggered when a delete occurs during flag merges in the store
			// resync events may trigger further resync events, however for a flag to be deleted from the store
			// its source must match, preventing the opportunity for resync events to snowball
			if resyncRequired := p.updateWithNotify(data); resyncRequired {
				go func() {
					p.tryReSync(dataSync)
				}()
			}
			if data.Type == ofsync.ALL {
				p.handleProviderReady()
			} else {
				p.logger.Warn(fmt.Sprintf("Received unexpected message type: %d", data.Type))
			}
			p.sendProviderEvent(of.Event{
				EventType: of.ProviderConfigChange,
			})
		case <-p.ctx.Done():
			return nil
		}
	}
}

func (p *Provider) tryReSync(dataSync chan ofsync.DataSync) {
	for {
		err := p.syncSource.ReSync(p.ctx, dataSync)
		if err != nil {
			p.handleConnectionErr(fmt.Errorf("error resyncing source: %w", err))
		} else {
			p.handleProviderReady()
			continue
		}
	}

}

func (p *Provider) handleConnectionErr(err error) {
	p.mu.Lock()
	p.logger.Error(fmt.Sprintf("Encountered unexpected sync error: %v", err))
	if p.connectionInfo.retries >= p.connectionInfo.maxSyncRetries && p.connectionInfo.state != stateError {
		p.logger.Error("Number of maximum retry attempts has been exceeded. Going into ERROR state.")
		p.connectionInfo.state = stateError
		p.sendProviderEvent(of.Event{
			EventType: of.ProviderError,
			ProviderEventDetails: of.ProviderEventDetails{
				Message: err.Error(),
			},
		})
	}
	// go to STALE state, if we have been ready previously; otherwise
	// we will stay in NOT_READY
	if p.connectionInfo.state == stateReady {
		p.logger.Warn("Going into STALE state")
		p.connectionInfo.state = stateStale
		p.sendProviderEvent(of.Event{
			EventType: of.ProviderStale,
			ProviderEventDetails: of.ProviderEventDetails{
				Message: err.Error(),
			},
		})
	}
	p.connectionInfo.retries++
	if newBackoffDuration := p.connectionInfo.backoffDuration * 2; newBackoffDuration < p.connectionInfo.maxBackoffDuration {
		p.connectionInfo.backoffDuration = newBackoffDuration
	} else {
		p.connectionInfo.backoffDuration = p.connectionInfo.maxBackoffDuration
	}
	p.mu.Unlock()
	<-time.After(p.connectionInfo.backoffDuration)
}

func (p *Provider) handleProviderReady() {
	p.mu.Lock()
	oldState := p.connectionInfo.state
	p.connectionInfo.retries = 0
	p.connectionInfo.state = stateReady
	p.connectionInfo.backoffDuration = defaultInitBackoffDuration
	p.mu.Unlock()
	// notify event channel listeners that we are now ready
	if oldState != stateReady {
		p.sendProviderEvent(of.Event{
			EventType: of.ProviderReady,
		})
	}
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

		maxSyncRetriesStr := os.Getenv(flagdMaxSyncRetriesEnvironmentVariableName)
		if maxSyncRetriesStr != "" {
			maxSyncRetries, err := strconv.Atoi(maxSyncRetriesStr)
			if err != nil {
				p.logger.Error(
					fmt.Sprintf("invalid env config for %s provided, using default value: %d",
						flagdMaxSyncRetriesEnvironmentVariableName, defaultMaxSyncRetries))
			} else {
				p.connectionInfo.maxSyncRetries = maxSyncRetries
			}
		}

		maxSyncRetryIntervalStr := os.Getenv(flagdSyncRetryIntervalEnvironmentVariableName)
		if maxSyncRetryIntervalStr != "" {
			maxSyncRetryInterval, err := time.ParseDuration(maxSyncRetryIntervalStr)
			if err != nil {
				p.logger.Error(
					fmt.Sprintf(
						"Invalid env config for %s provided, using default value: %s",
						flagdSyncRetryIntervalEnvironmentVariableName,
						defaultMaxBackoffDuration.String(),
					),
				)
			} else {
				p.connectionInfo.maxBackoffDuration = maxSyncRetryInterval
			}
		}

		sourceURI := os.Getenv(flagdSourceURIEnvironmentVariableName)
		sourceProvider := os.Getenv(flagdSourceProviderEnvironmentVariableName)
		selector := os.Getenv(flagdSourceSelectorEnvironmentVariableName)

		if sourceURI != "" && sourceProvider != "" {
			p.providerConfiguration.SourceConfig = runtime.SourceConfig{
				URI:      sourceURI,
				Provider: sourceProvider,
				Selector: selector,
				CertPath: p.providerConfiguration.CertificatePath,
				TLS:      p.providerConfiguration.TLSEnabled,
			}
		}
	}
}

// WithSourceURI sets the URI of the sync source
func WithSourceURI(uri string) ProviderOption {
	return func(p *Provider) {
		p.providerConfiguration.SourceConfig.URI = uri
	}
}

func WithSourceType(providerType ProviderType) ProviderOption {
	return func(p *Provider) {
		p.providerConfiguration.SourceConfig.Provider = string(providerType)
	}
}

// WithContext supplies the given context to the event stream. Not to be confused with the context used in individual
// flag evaluation requests.
func WithContext(ctx context.Context) ProviderOption {
	return func(p *Provider) {
		p.ctx = ctx
	}
}

// WithSyncStreamConnectionMaxAttempts sets the maximum number of attempts to connect to flagd's event stream.
// On successful connection the attempts are reset.
func WithSyncStreamConnectionMaxAttempts(i int) ProviderOption {
	return func(p *Provider) {
		p.connectionInfo.maxSyncRetries = i
	}
}

// WithSyncStreamConnectionBackoff sets the backoff duration between reattempts of connecting to the sync source.
func WithSyncStreamConnectionBackoff(duration time.Duration) ProviderOption {
	return func(p *Provider) {
		p.connectionInfo.maxBackoffDuration = duration
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

// WithSelector sets the selector for the sync source
func WithSelector(selector string) ProviderOption {
	return func(p *Provider) {
		p.providerConfiguration.SourceConfig.Selector = selector
	}
}

// Status returns the current provider status
func (p *Provider) Status() of.State {
	switch p.connectionInfo.state {
	case stateNotReady:
		return of.NotReadyState
	case stateReady:
		return of.ReadyState
	case stateStale:
		// currently there is no STALE state in the of API, so we also return ERROR for STALE
		return of.ErrorState
	case stateError:
		return of.ErrorState
	default:
		return of.NotReadyState
	}
}

//////////////////////////////////////////////////
// OF API StateHandler interface implementation //
//////////////////////////////////////////////////

// Shutdown implements the shutdown logic for this provider
func (p *Provider) Shutdown() {
	p.connectionInfo.state = stateNotReady
	p.cancelFunc()
}

// Init implements the Init method required for the OF API StateHandler interface
func (p *Provider) Init(of.EvaluationContext) {

}

// Hooks in-processflagd provider does not have any hooks, returns empty slice
func (p *Provider) Hooks() []of.Hook {
	return []of.Hook{}
}

// Metadata returns value of Metadata (name of current service, exposed to openfeature sdk)
func (p *Provider) Metadata() of.Metadata {
	return of.Metadata{
		Name: "flagd-in-process-provider",
	}
}

//////////////////////////////////////////////////
// OF API EventHandler interface implementation //
//////////////////////////////////////////////////

func (p *Provider) EventChannel() <-chan of.Event {
	return p.ofEventChannel
}

//////////////////////////////////////////////
// OF API Provider interface implementation //
//////////////////////////////////////////////

func (p *Provider) BooleanEvaluation(
	ctx context.Context, flagKey string, defaultValue bool, evalCtx of.FlattenedContext,
) of.BoolResolutionDetail {

	value, variant, reason, metadata, err := p.evaluator.ResolveBooleanValue(ctx, "", flagKey, evalCtx)

	if err != nil {
		return of.BoolResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				ResolutionError: mapError(err),
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
		return of.StringResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				ResolutionError: mapError(err),
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
		return of.FloatResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				ResolutionError: mapError(err),
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
		return of.IntResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				ResolutionError: mapError(err),
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
		return of.InterfaceResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				ResolutionError: mapError(err),
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
func (p *Provider) updateWithNotify(payload ofsync.DataSync) bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	_, resyncRequired, err := p.evaluator.SetState(payload)
	if err != nil {
		p.logger.Error(err.Error())
		return false
	}

	return resyncRequired
}

func (p *Provider) sendProviderEvent(event of.Event) {
	go func() {
		if p.ofEventChannel == nil {
			return
		}
		event.ProviderName = p.Metadata().Name
		p.ofEventChannel <- event
	}()
}

// syncProviderFromConfig is a helper to build ISync implementations from SourceConfig
func syncProviderFromConfig(logger *logger.Logger, sourceConfig runtime.SourceConfig) (ofsync.ISync, error) {
	switch sourceConfig.Provider {
	case string(SourceTypeKubernetes):
		k8sSync, err := runtime.NewK8s(sourceConfig.URI, logger)
		if err != nil {
			return nil, err
		}
		logger.Debug(fmt.Sprintf("using kubernetes sync-provider for: %s", sourceConfig.URI))
		return k8sSync, nil
	case string(SourceTypeGrpc):
		logger.Debug(fmt.Sprintf("using grpc sync-provider for: %s", sourceConfig.URI))
		return runtime.NewGRPC(sourceConfig, logger), nil

	default:
		return nil, fmt.Errorf("invalid sync provider: %s, must be one of with '%s' or '%s'",
			sourceConfig.Provider, SourceTypeGrpc, SourceTypeKubernetes)
	}
}

func mapError(err error) of.ResolutionError {
	switch err.Error() {
	case model.FlagNotFoundErrorCode, model.FlagDisabledErrorCode:
		return of.NewFlagNotFoundResolutionError(string(of.FlagNotFoundCode))
	case model.TypeMismatchErrorCode:
		return of.NewTypeMismatchResolutionError(string(of.TypeMismatchCode))
	case model.ParseErrorCode:
		return of.NewParseErrorResolutionError(string(of.ParseErrorCode))
	default:
		return of.NewGeneralResolutionError(string(of.GeneralCode))
	}
}
