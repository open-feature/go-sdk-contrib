package process

import (
	"context"
	"fmt"
	"github.com/open-feature/flagd/core/pkg/eval"
	"github.com/open-feature/flagd/core/pkg/logger"
	"github.com/open-feature/flagd/core/pkg/model"
	"github.com/open-feature/flagd/core/pkg/runtime"
	"github.com/open-feature/flagd/core/pkg/store"
	"github.com/open-feature/flagd/core/pkg/sync"
	of "github.com/open-feature/go-sdk/pkg/openfeature"
	"golang.org/x/exp/maps"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	parallel "sync"
)

// InProcess service implements flagd flag evaluation in-process.
// Flag configurations are obtained from supported sources.
type InProcess struct {
	evaluator        eval.IEvaluator
	events           chan of.Event
	listenerShutdown chan interface{}
	logger           *logger.Logger
	sync             sync.ISync
	syncEnd          context.CancelFunc
}

type Configuration struct {
	Host       any
	Port       any
	Selector   string
	TLSEnabled bool
}

func NewInProcessService(cfg Configuration) *InProcess {
	log := logger.NewLogger(zap.NewRaw(), false)

	// currently supports grpc syncs for in-process flag fetch
	var uri string
	if cfg.TLSEnabled {
		uri = fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	} else {
		uri = fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	}

	grpcSync := runtime.NewGRPC(runtime.SourceConfig{
		URI:      uri,
		TLS:      cfg.TLSEnabled,
		Selector: cfg.Selector,
	}, log)

	flagStore := store.NewFlags()
	flagStore.FlagSources = append(flagStore.FlagSources, uri)

	jsonEvaluator := eval.NewJSONEvaluator(log,
		flagStore,
		eval.WithEvaluator(
			"fractional",
			eval.NewFractionalEvaluator(log).FractionalEvaluation,
		),
		eval.WithEvaluator(
			"starts_with",
			eval.NewStringComparisonEvaluator(log).StartsWithEvaluation,
		),
		eval.WithEvaluator(
			"ends_with",
			eval.NewStringComparisonEvaluator(log).EndsWithEvaluation,
		),
		eval.WithEvaluator(
			"sem_ver",
			eval.NewSemVerComparisonEvaluator(log).SemVerEvaluation,
		))

	return &InProcess{
		evaluator:        jsonEvaluator,
		events:           make(chan of.Event, 5),
		logger:           log,
		listenerShutdown: make(chan interface{}),
		sync:             grpcSync,
	}
}

func (i *InProcess) Init() error {
	var ctx context.Context
	ctx, i.syncEnd = context.WithCancel(context.Background())

	err := i.sync.Init(ctx)
	if err != nil {
		return err
	}

	syncInitSuccess := make(chan interface{})
	readyOnce := parallel.OnceFunc(func() {
		i.events <- of.Event{ProviderName: "flagd", EventType: of.ProviderReady}
		syncInitSuccess <- nil
	})
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
				readyOnce()
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
	i.listenerShutdown <- nil
}

func (i *InProcess) ResolveBoolean(ctx context.Context, key string, defaultValue bool,
	evalCtx map[string]interface{}) of.BoolResolutionDetail {
	value, variant, reason, metadata, err := i.evaluator.ResolveBooleanValue(ctx, "", key, evalCtx)
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

// mapError is a helper to map evaluation errors to OF errors
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
