package process

import (
	"context"
	"fmt"
	"regexp"
	"sync"
	"time"

	"go.uber.org/zap"
	googlegrpc "google.golang.org/grpc"

	"github.com/open-feature/flagd/core/pkg/evaluator"
	"github.com/open-feature/flagd/core/pkg/logger"
	"github.com/open-feature/flagd/core/pkg/model"
	"github.com/open-feature/flagd/core/pkg/store"
	isync "github.com/open-feature/flagd/core/pkg/sync"
	"github.com/open-feature/flagd/core/pkg/sync/file"
	"github.com/open-feature/flagd/core/pkg/sync/grpc"
	"github.com/open-feature/flagd/core/pkg/sync/grpc/credentials"
	of "github.com/open-feature/go-sdk/openfeature"
	"golang.org/x/exp/maps"
)

const (
	// Channel buffer sizes
	eventChannelBuffer = 5
	syncChannelBuffer  = 1

	// Provider name for events
	providerName = "flagd"
)

// InProcess service implements flagd flag evaluation in-process.
// Flag configurations are obtained from supported sources.
type InProcess struct {
	// Core components
	evaluator       evaluator.IEvaluator
	syncProvider    isync.ISync
	logger          *logger.Logger
	configuration   Configuration
	serviceMetadata model.Metadata

	// Event handling
	events    chan of.Event
	eventSync EventSync

	// Shutdown coordination
	ctx              context.Context
	cancelFunc       context.CancelFunc
	shutdownChannels *shutdownChannels
	wg               sync.WaitGroup
	shutdownOnce     sync.Once

	// Stateless coordination using sync.Once
	initOnce            sync.Once
	sendReadyOnNextData sync.Once
	staleTimer          *staleTimer
}

// shutdownChannels groups all shutdown-related channels
type shutdownChannels struct {
	listenerShutdown chan struct{}
	syncData         chan isync.DataSync
	initSuccess      chan struct{}
	initError        chan error
}

// staleTimer manages the stale connection timer with thread safety
type staleTimer struct {
	timer *time.Timer
	mu    sync.Mutex
}

// newStaleTimer creates a new thread-safe stale timer
func newStaleTimer() *staleTimer {
	return &staleTimer{}
}

// start starts or restarts the stale timer
func (st *staleTimer) start(duration time.Duration, callback func()) {
	st.mu.Lock()
	defer st.mu.Unlock()

	if st.timer == nil {
		st.timer = time.AfterFunc(duration, callback)
	}
}

// stop stops the stale timer
func (st *staleTimer) stop() {
	st.mu.Lock()
	defer st.mu.Unlock()

	if st.timer != nil {
		st.timer.Stop()
		st.timer = nil
	}
}

// Configuration holds all configuration for the InProcess service
type Configuration struct {
	Host                    any
	Port                    any
	TargetUri               string
	ProviderID              string
	Selector                string
	TLSEnabled              bool
	OfflineFlagSource       string
	CustomSyncProvider      isync.ISync
	CustomSyncProviderUri   string
	GrpcDialOptionsOverride []googlegrpc.DialOption
	CertificatePath         string
	RetryGracePeriod        int
	RetryBackOffMs          int
	RetryBackOffMaxMs       int
	FatalStatusCodes 	    []string
}

// EventSync interface for sync providers that support events
type EventSync interface {
	isync.ISync
	Events() chan SyncEvent
}

// SyncEvent represents an event from the sync provider
type SyncEvent struct {
	event of.EventType
}

// Shutdowner interface for graceful shutdown
type Shutdowner interface {
	Shutdown() error
}

// NewInProcessService creates a new InProcess service with the given configuration
func NewInProcessService(cfg Configuration) *InProcess {
	log := logger.NewLogger(NewRaw(), false)
	syncProvider, uri := createSyncProvider(cfg, log)

	flagStore := store.NewFlags()
	flagStore.FlagSources = append(flagStore.FlagSources, uri)

	return &InProcess{
		evaluator:           evaluator.NewJSON(log, flagStore),
		syncProvider:        syncProvider,
		logger:              log,
		configuration:       cfg,
		serviceMetadata:     createServiceMetadata(cfg),
		events:              make(chan of.Event, eventChannelBuffer),
		staleTimer:          newStaleTimer(),
		sendReadyOnNextData: sync.Once{}, // Armed and ready to fire on first data
	}
}

// createServiceMetadata builds the service metadata from configuration
func createServiceMetadata(cfg Configuration) model.Metadata {
	metadata := make(model.Metadata, 2)
	if cfg.Selector != "" {
		metadata["scope"] = cfg.Selector
	}
	if cfg.ProviderID != "" {
		metadata["providerID"] = cfg.ProviderID
	}
	return metadata
}

// Init initializes the service and starts all background processes
func (i *InProcess) Init() error {
	i.logger.Info("initializing InProcess service")

	// Setup context and shutdown channels
	i.setupShutdownInfrastructure()

	// Initialize sync provider
	if err := i.syncProvider.Init(i.ctx); err != nil {
		return fmt.Errorf("failed to initialize sync provider: %w", err)
	}

	// Start background processes
	i.startEventSyncMonitor()
	i.startDataSyncProcess()
	i.startDataSyncListener()

	// Wait for initialization to complete
	return i.waitForInitialization()
}

// setupShutdownInfrastructure initializes context and channels for coordinated shutdown
func (i *InProcess) setupShutdownInfrastructure() {
	i.ctx, i.cancelFunc = context.WithCancel(context.Background())
	i.shutdownChannels = &shutdownChannels{
		listenerShutdown: make(chan struct{}),
		syncData:         make(chan isync.DataSync, syncChannelBuffer),
		initSuccess:      make(chan struct{}),
		initError:        make(chan error, 1),
	}
}

// startEventSyncMonitor starts monitoring events from EventSync providers
func (i *InProcess) startEventSyncMonitor() {
	eventSync, ok := i.syncProvider.(EventSync)
	if !ok {
		return // No event monitoring needed
	}

	i.eventSync = eventSync
	go i.runEventSyncMonitor()
}

// runEventSyncMonitor handles events from the sync provider
func (i *InProcess) runEventSyncMonitor() {
	i.logger.Debug("starting event sync monitor")
	defer i.logger.Debug("event sync monitor stopped")

	for {
		select {
		case <-i.ctx.Done():
			return
		case <-i.shutdownChannels.listenerShutdown:
			return
		case msg := <-i.eventSync.Events():
			i.handleSyncEvent(msg)
		}
	}
}

// handleSyncEvent processes individual sync events
func (i *InProcess) handleSyncEvent(event SyncEvent) {
	switch event.event {
	case of.ProviderError:
		i.handleProviderError()
		// Reset the sync.Once so it can fire again on recovery
		i.sendReadyOnNextData = sync.Once{}
	case of.ProviderReady:
		i.handleProviderReady()
	}
}

// handleProviderError handles provider error events by starting stale timer
func (i *InProcess) handleProviderError() {
	i.events <- of.Event{
		ProviderName:         providerName,
		EventType:            of.ProviderStale,
		ProviderEventDetails: of.ProviderEventDetails{Message: "connection error"},
	}

	// Start stale timer - when it expires, send error event
	i.staleTimer.start(time.Duration(i.configuration.RetryGracePeriod)*time.Second, func() {
		i.events <- of.Event{
			ProviderName:         providerName,
			EventType:            of.ProviderError,
			ProviderEventDetails: of.ProviderEventDetails{Message: "provider error"},
		}
	})
}

// handleProviderReady handles provider ready events by stopping stale timer
func (i *InProcess) handleProviderReady() {
	i.staleTimer.stop()
}

// startDataSyncProcess starts the main data synchronization goroutine
func (i *InProcess) startDataSyncProcess() {
	i.wg.Add(1)
	go i.runDataSyncProcess()
}

// runDataSyncProcess runs the main sync process and handles errors appropriately
func (i *InProcess) runDataSyncProcess() {
	defer i.wg.Done()
	i.logger.Debug("starting data sync process")
	defer i.logger.Debug("data sync process stopped")

	err := i.syncProvider.Sync(i.ctx, i.shutdownChannels.syncData)
	if err != nil && i.ctx.Err() == nil {
		// Only report non-cancellation errors
		select {
		case i.shutdownChannels.initError <- err:
		default:
			// Don't block if channel is full or no reader
		}
	}
}

// startDataSyncListener starts the data sync listener goroutine
func (i *InProcess) startDataSyncListener() {
	i.wg.Add(1)
	go i.runDataSyncListener()
}

// runDataSyncListener processes incoming sync data and handles shutdown
func (i *InProcess) runDataSyncListener() {
	defer i.wg.Done()
	i.logger.Debug("starting data sync listener")
	defer i.logger.Debug("data sync listener stopped")

	for {
		select {
		case data := <-i.shutdownChannels.syncData:
			i.processSyncData(data)

		case <-i.ctx.Done():
			i.logger.Info("data sync listener stopping due to context cancellation")
			i.shutdownSyncProvider()
			return

		case <-i.shutdownChannels.listenerShutdown:
			i.logger.Info("data sync listener stopping due to shutdown signal")
			i.shutdownSyncProvider()
			return
		}
	}
}

// processSyncData handles individual sync data updates
func (i *InProcess) processSyncData(data isync.DataSync) {
	changes, _, err := i.evaluator.SetState(data)
	if err != nil {
		i.events <- of.Event{
			ProviderName:         providerName,
			EventType:            of.ProviderError,
			ProviderEventDetails: of.ProviderEventDetails{Message: "Error from flag sync " + err.Error()},
		}
		return
	}

	i.logger.Info("staletimer stop")
	// Stop stale timer - we've successfully received and processed data
	i.staleTimer.stop()

	// Send ready event using sync.Once - handles initial ready and recovery automatically
	i.sendReadyOnNextData.Do(func() {
		i.events <- of.Event{ProviderName: providerName, EventType: of.ProviderReady}
	})

	// Handle initialization completion (only happens once ever)
	i.initOnce.Do(func() {
		close(i.shutdownChannels.initSuccess)
	})

	// Send config change event for data updates
	if len(changes) > 0 {
		i.events <- of.Event{
			ProviderName: providerName,
			EventType:    of.ProviderConfigChange,
			ProviderEventDetails: of.ProviderEventDetails{
				Message:     "New flag sync",
				FlagChanges: maps.Keys(changes),
			},
		}
	}
}

// shutdownSyncProvider gracefully shuts down the sync provider
func (i *InProcess) shutdownSyncProvider() {
	if shutdowner, ok := i.syncProvider.(Shutdowner); ok {
		if err := shutdowner.Shutdown(); err != nil {
			i.logger.Error("error shutting down sync provider", zap.Error(err))
		}
	}
}

// waitForInitialization waits for the service to initialize or fail
func (i *InProcess) waitForInitialization() error {
	select {
	case <-i.shutdownChannels.initSuccess:
		i.logger.Info("InProcess service initialized successfully")
		return nil
	case err := <-i.shutdownChannels.initError:
		return fmt.Errorf("initialization failed: %w", err)
	}
}

// Shutdown gracefully shuts down the service
func (i *InProcess) Shutdown() {
	i.shutdownOnce.Do(func() {
		i.logger.Info("starting InProcess service shutdown")

		// Stop stale timer
		i.staleTimer.stop()

		// Cancel context to signal all goroutines
		if i.cancelFunc != nil {
			i.cancelFunc()
		}

		// Close shutdown channels
		if i.shutdownChannels != nil {
			close(i.shutdownChannels.listenerShutdown)
		}

		i.logger.Info("waiting for background processes to complete")
		i.wg.Wait()
		i.logger.Info("InProcess service shutdown completed successfully")
	})
}

// EventChannel returns the event channel for external consumers
func (i *InProcess) EventChannel() <-chan of.Event {
	return i.events
}

// appendMetadata adds service metadata to evaluation metadata
func (i *InProcess) appendMetadata(evalMetadata model.Metadata) {
	for k, v := range i.serviceMetadata {
		evalMetadata[k] = v
	}
}

// ResolveBoolean resolves a boolean flag value
func (i *InProcess) ResolveBoolean(ctx context.Context, key string, defaultValue bool, evalCtx map[string]interface{}) of.BoolResolutionDetail {
	value, variant, reason, metadata, err := i.evaluator.ResolveBooleanValue(ctx, "", key, evalCtx)
	i.appendMetadata(metadata)

	if err != nil {
		return of.BoolResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				ResolutionError: mapError(key, err),
				Reason:          of.Reason(reason),
				Variant:         variant,
				FlagMetadata:    metadata,
			},
		}
	}

	return of.BoolResolutionDetail{
		Value: value,
		ProviderResolutionDetail: of.ProviderResolutionDetail{
			Reason:       of.Reason(reason),
			Variant:      variant,
			FlagMetadata: metadata,
		},
	}
}

// ResolveString resolves a string flag value
func (i *InProcess) ResolveString(ctx context.Context, key string, defaultValue string, evalCtx map[string]interface{}) of.StringResolutionDetail {
	value, variant, reason, metadata, err := i.evaluator.ResolveStringValue(ctx, "", key, evalCtx)
	i.appendMetadata(metadata)

	if err != nil {
		return of.StringResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				ResolutionError: mapError(key, err),
				Reason:          of.Reason(reason),
				Variant:         variant,
				FlagMetadata:    metadata,
			},
		}
	}

	return of.StringResolutionDetail{
		Value: value,
		ProviderResolutionDetail: of.ProviderResolutionDetail{
			Reason:       of.Reason(reason),
			Variant:      variant,
			FlagMetadata: metadata,
		},
	}
}

// ResolveFloat resolves a float flag value
func (i *InProcess) ResolveFloat(ctx context.Context, key string, defaultValue float64, evalCtx map[string]interface{}) of.FloatResolutionDetail {
	value, variant, reason, metadata, err := i.evaluator.ResolveFloatValue(ctx, "", key, evalCtx)
	i.appendMetadata(metadata)

	if err != nil {
		return of.FloatResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				ResolutionError: mapError(key, err),
				Reason:          of.Reason(reason),
				Variant:         variant,
				FlagMetadata:    metadata,
			},
		}
	}

	return of.FloatResolutionDetail{
		Value: value,
		ProviderResolutionDetail: of.ProviderResolutionDetail{
			Reason:       of.Reason(reason),
			Variant:      variant,
			FlagMetadata: metadata,
		},
	}
}

// ResolveInt resolves an integer flag value
func (i *InProcess) ResolveInt(ctx context.Context, key string, defaultValue int64, evalCtx map[string]interface{}) of.IntResolutionDetail {
	value, variant, reason, metadata, err := i.evaluator.ResolveIntValue(ctx, "", key, evalCtx)
	i.appendMetadata(metadata)

	if err != nil {
		return of.IntResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				ResolutionError: mapError(key, err),
				Reason:          of.Reason(reason),
				Variant:         variant,
				FlagMetadata:    metadata,
			},
		}
	}

	return of.IntResolutionDetail{
		Value: value,
		ProviderResolutionDetail: of.ProviderResolutionDetail{
			Reason:       of.Reason(reason),
			Variant:      variant,
			FlagMetadata: metadata,
		},
	}
}

// ResolveObject resolves an object flag value
func (i *InProcess) ResolveObject(ctx context.Context, key string, defaultValue interface{}, evalCtx map[string]interface{}) of.InterfaceResolutionDetail {
	value, variant, reason, metadata, err := i.evaluator.ResolveObjectValue(ctx, "", key, evalCtx)
	i.appendMetadata(metadata)

	if err != nil {
		return of.InterfaceResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: of.ProviderResolutionDetail{
				ResolutionError: mapError(key, err),
				Reason:          of.Reason(reason),
				Variant:         variant,
				FlagMetadata:    metadata,
			},
		}
	}

	return of.InterfaceResolutionDetail{
		Value: value,
		ProviderResolutionDetail: of.ProviderResolutionDetail{
			Reason:       of.Reason(reason),
			Variant:      variant,
			FlagMetadata: metadata,
		},
	}
}

// createSyncProvider creates the appropriate sync provider based on configuration
func createSyncProvider(cfg Configuration, log *logger.Logger) (isync.ISync, string) {
	if cfg.CustomSyncProvider != nil {
		log.Info("using custom sync provider at " + cfg.CustomSyncProviderUri)
		return cfg.CustomSyncProvider, cfg.CustomSyncProviderUri
	}

	if cfg.OfflineFlagSource != "" {
		log.Info("using file sync provider with source: " + cfg.OfflineFlagSource)
		return &file.Sync{
			URI:    cfg.OfflineFlagSource,
			Logger: log,
			Mux:    &sync.RWMutex{},
		}, cfg.OfflineFlagSource
	}

	// Default to gRPC sync provider
	uri := buildGrpcUri(cfg)
	log.Info("using gRPC sync provider with URI: " + uri)

	return &Sync{
		CredentialBuilder:       &credentials.CredentialBuilder{},
		GrpcDialOptionsOverride: cfg.GrpcDialOptionsOverride,
		Logger:                  log,
		Secure:                  cfg.TLSEnabled,
		CertPath:                cfg.CertificatePath,
		ProviderID:              cfg.ProviderID,
		Selector:                cfg.Selector,
		URI:                     uri,
		RetryGracePeriod:        cfg.RetryGracePeriod,
		RetryBackOffMs: 		 cfg.RetryBackOffMs,
		RetryBackOffMaxMs: 		 cfg.RetryBackOffMaxMs,
		FatalStatusCodes: 		 cfg.FatalStatusCodes,
	}, uri
}

// buildGrpcUri constructs the gRPC URI from configuration
func buildGrpcUri(cfg Configuration) string {
	if cfg.TargetUri != "" && isValidTargetScheme(cfg.TargetUri) {
		return cfg.TargetUri
	}
	return fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
}

// mapError maps evaluation errors to OpenFeature errors
func mapError(flagKey string, err error) of.ResolutionError {
	switch err.Error() {
	case model.FlagNotFoundErrorCode:
		return of.NewFlagNotFoundResolutionError(fmt.Sprintf("flag: %s not found", flagKey))
	case model.FlagDisabledErrorCode:
		return of.NewFlagNotFoundResolutionError(fmt.Sprintf("flag: %s is disabled", flagKey))
	case model.TypeMismatchErrorCode:
		return of.NewTypeMismatchResolutionError(fmt.Sprintf("flag: %s evaluated type not valid", flagKey))
	case model.ParseErrorCode:
		return of.NewParseErrorResolutionError(fmt.Sprintf("flag: %s parsing error", flagKey))
	default:
		return of.NewGeneralResolutionError(fmt.Sprintf("flag: %s unable to evaluate", flagKey))
	}
}

// isValidTargetScheme validates the gRPC target URI scheme
func isValidTargetScheme(targetUri string) bool {
	regx := regexp.MustCompile("^" + grpc.SupportedScheme)
	return regx.Match([]byte(targetUri))
}
