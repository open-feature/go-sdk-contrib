package process

import (
	"context"
	"fmt"

	"regexp"
	parallel "sync"

	"github.com/open-feature/flagd/core/pkg/evaluator"
	"github.com/open-feature/flagd/core/pkg/logger"
	"github.com/open-feature/flagd/core/pkg/model"
	"github.com/open-feature/flagd/core/pkg/store"
	"github.com/open-feature/flagd/core/pkg/sync"
	"github.com/open-feature/flagd/core/pkg/sync/file"
	"github.com/open-feature/flagd/core/pkg/sync/grpc"
	"github.com/open-feature/flagd/core/pkg/sync/grpc/credentials"
	of "github.com/open-feature/go-sdk/openfeature"
	"golang.org/x/exp/maps"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

// InProcess service implements flagd flag evaluation in-process.
// Flag configurations are obtained from supported sources.
type InProcess struct {
	evaluator        evaluator.IEvaluator
	events           chan of.Event
	listenerShutdown chan interface{}
	logger           *logger.Logger
	serviceMetadata  model.Metadata
	sync             sync.ISync
	syncEnd          context.CancelFunc
}

type Configuration struct {
	Host                  any
	Port                  any
	TargetUri             string
	ProviderID            string
	Selector              string
	TLSEnabled            bool
	OfflineFlagSource     string
	CustomSyncProvider    sync.ISync
	CustomSyncProviderUri string
}

func NewInProcessService(cfg Configuration) *InProcess {
	log := logger.NewLogger(zap.NewRaw(), false)

	iSync, uri := makeSyncProvider(cfg, log)

	// service specific metadata
	var svcMetadata model.Metadata
	if cfg.Selector != "" {
		svcMetadata = make(model.Metadata, 1)
		svcMetadata["scope"] = cfg.Selector
	}
	if cfg.ProviderID != "" {
		svcMetadata["providerID"] = cfg.ProviderID
	}

	flagStore := store.NewFlags()
	flagStore.FlagSources = append(flagStore.FlagSources, uri)
	return &InProcess{
		evaluator:        evaluator.NewJSON(log, flagStore),
		events:           make(chan of.Event, 5),
		logger:           log,
		listenerShutdown: make(chan interface{}),
		serviceMetadata:  svcMetadata,
		sync:             iSync,
	}
}

func (i *InProcess) Init() error {
	var ctx context.Context
	ctx, i.syncEnd = context.WithCancel(context.Background())

	err := i.sync.Init(ctx)
	if err != nil {
		return err
	}

	initOnce := parallel.Once{}
	syncInitSuccess := make(chan interface{})
	syncInitErr := make(chan error)

	syncChan := make(chan sync.DataSync, 1)

	// start data sync
	go func() {
		err := i.sync.Sync(ctx, syncChan)
		if err != nil {
			syncInitErr <- err
		}
	}()

	// start data sync listener and listen to listener shutdown hook
	go func() {
		for {
			select {
			case data := <-syncChan:
				// re-syncs are ignored as we only support single flag sync source
				changes, _, err := i.evaluator.SetState(data)
				if err != nil {
					i.events <- of.Event{
						ProviderName: "flagd", EventType: of.ProviderError,
						ProviderEventDetails: of.ProviderEventDetails{Message: "Error from flag sync " + err.Error()}}
				}
				initOnce.Do(func() {
					i.events <- of.Event{ProviderName: "flagd", EventType: of.ProviderReady}
					syncInitSuccess <- nil
				})
				i.events <- of.Event{
					ProviderName: "flagd", EventType: of.ProviderConfigChange,
					ProviderEventDetails: of.ProviderEventDetails{Message: "New flag sync", FlagChanges: maps.Keys(changes)}}
			case <-i.listenerShutdown:
				i.logger.Info("Shutting down data sync listener")
				return
			}
		}
	}()

	// wait for initialization or error
	select {
	case <-syncInitSuccess:
		return nil
	case err := <-syncInitErr:
		return err
	}
}

func (i *InProcess) Shutdown() {
	i.syncEnd()
	close(i.listenerShutdown)
}

func (i *InProcess) ResolveBoolean(ctx context.Context, key string, defaultValue bool,
	evalCtx map[string]interface{}) of.BoolResolutionDetail {
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

func (i *InProcess) ResolveString(ctx context.Context, key string, defaultValue string,
	evalCtx map[string]interface{}) of.StringResolutionDetail {
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

func (i *InProcess) ResolveFloat(ctx context.Context, key string, defaultValue float64,
	evalCtx map[string]interface{}) of.FloatResolutionDetail {
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

func (i *InProcess) ResolveInt(ctx context.Context, key string, defaultValue int64,
	evalCtx map[string]interface{}) of.IntResolutionDetail {
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

func (i *InProcess) ResolveObject(ctx context.Context, key string, defaultValue interface{},
	evalCtx map[string]interface{}) of.InterfaceResolutionDetail {
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

func (i *InProcess) EventChannel() <-chan of.Event {
	return i.events
}

func (i *InProcess) appendMetadata(evalMetadata model.Metadata) {
	// For a nil slice, the number of iterations is 0
	for k, v := range i.serviceMetadata {
		evalMetadata[k] = v
	}
}

// makeSyncProvider is a helper to create sync.ISync and return the underlying uri used by it to the caller
func makeSyncProvider(cfg Configuration, log *logger.Logger) (sync.ISync, string) {
	if cfg.CustomSyncProvider != nil {
		log.Info("operating in in-process mode with a custom sync provider at " + cfg.CustomSyncProviderUri)
		return cfg.CustomSyncProvider, cfg.CustomSyncProviderUri
	}

	if cfg.OfflineFlagSource != "" {
		// file sync provider
		log.Info("operating in in-process mode with offline flags sourced from " + cfg.OfflineFlagSource)
		return &file.Sync{
			URI:    cfg.OfflineFlagSource,
			Logger: log,
			Mux:    &parallel.RWMutex{},
		}, cfg.OfflineFlagSource
	}

	// grpc sync provider (default uri based on `dns`)
	uri := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)

	if cfg.TargetUri != "" && isValidTargetScheme(cfg.TargetUri) {
		uri = cfg.TargetUri
	}

	log.Info("operating in in-process mode with flags sourced from " + uri)

	return &grpc.Sync{
		CredentialBuilder: &credentials.CredentialBuilder{},
		Logger:            log,
		Secure:            cfg.TLSEnabled,
		ProviderID:        cfg.ProviderID,
		Selector:          cfg.Selector,
		URI:               uri,
	}, uri
}

// mapError is a helper to map evaluation errors to OF errors
func mapError(flagKey string, err error) of.ResolutionError {
	switch err.Error() {
	case model.FlagNotFoundErrorCode:
		return of.NewFlagNotFoundResolutionError(fmt.Sprintf("flag: " + flagKey + " not found"))
	case model.FlagDisabledErrorCode:
		return of.NewFlagNotFoundResolutionError(fmt.Sprintf("flag: " + flagKey + " is disabled"))
	case model.TypeMismatchErrorCode:
		return of.NewTypeMismatchResolutionError(fmt.Sprintf("flag: " + flagKey + " evaluated type not valid"))
	case model.ParseErrorCode:
		return of.NewParseErrorResolutionError(fmt.Sprintf("flag: " + flagKey + " parsing error"))
	default:
		return of.NewGeneralResolutionError(fmt.Sprintf("flag: " + flagKey + " unable to evaluate"))
	}
}

func isValidTargetScheme(targetUri string) bool {
	regx := regexp.MustCompile("^" + grpc.SupportedScheme)
	return regx.Match([]byte(targetUri))
}
