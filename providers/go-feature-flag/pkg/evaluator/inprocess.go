package evaluator

import (
	"context"
	"errors"
	"maps"
	"sync"
	"time"

	"github.com/open-feature/go-sdk-contrib/providers/go-feature-flag/pkg/api"
	"github.com/open-feature/go-sdk/openfeature"
	"github.com/thomaspoignant/go-feature-flag/modules/core/evaluation"
	"github.com/thomaspoignant/go-feature-flag/modules/core/ffcontext"
	"github.com/thomaspoignant/go-feature-flag/modules/core/flag"
	"github.com/thomaspoignant/go-feature-flag/modules/core/model"
)

var _ Evaluator = &InProcess{}

const inProcessProviderName = "go-feature-flag"
const pollingIntervalDefault = 2 * time.Minute

type configurationRefreshStatus int

const (
	configurationRefreshNotModified configurationRefreshStatus = iota
	configurationRefreshChanged
)

type InProcess struct {
	flagConfig                  map[string]flag.InternalFlag
	evaluationContextEnrichment map[string]any
	flagChangePollingInterval   time.Duration
	goffAPI                     *api.GoFeatureFlagAPI
	etag                        string
	mu                          sync.RWMutex
	stopPolling                 chan struct{}
	pollingDone                 chan struct{}
	shutdownOnce                sync.Once
	eventStream                 chan openfeature.Event
	stale                       bool
}

func NewInprocessEvaluator(flagChangePollingInterval time.Duration, goffAPI *api.GoFeatureFlagAPI, eventStream chan openfeature.Event) *InProcess {
	pollingDone := make(chan struct{})
	close(pollingDone) // pre-closed so Shutdown() doesn't block if Init() was never called or failed
	return &InProcess{
		flagChangePollingInterval: flagChangePollingInterval,
		goffAPI:                   goffAPI,
		stopPolling:               make(chan struct{}),
		pollingDone:               pollingDone,
		eventStream:               eventStream,
	}
}

// evaluate is a generic helper that performs flag evaluation for any supported type.
func evaluate[T model.JSONType](
	i *InProcess,
	flagName string,
	defaultValue T,
	flatCtx openfeature.FlattenedContext,
	typeStr string,
) (T, openfeature.ProviderResolutionDetail) {
	f, errFExists := i.checkFlagExists(flagName)
	if errFExists != nil {
		return defaultValue, *errFExists
	}
	enrichment := i.getEvaluationContextEnrichment()
	varResult, err := evaluation.Evaluate(
		f, flagName, toFFContext(flatCtx),
		flag.Context{DefaultSdkValue: defaultValue, EvaluationContextEnrichment: enrichment},
		typeStr, defaultValue,
	)
	if err != nil || varResult.Failed {
		msg := varResult.ErrorDetails
		if err != nil {
			msg = err.Error()
		}
		return defaultValue, openfeature.ProviderResolutionDetail{
			ResolutionError: toResolutionError(varResult.ErrorCode, msg),
			Reason:          openfeature.ErrorReason,
		}
	}
	return varResult.Value, openfeature.ProviderResolutionDetail{
		Reason:       openfeature.Reason(varResult.Reason),
		Variant:      varResult.VariationType,
		FlagMetadata: varResult.Metadata,
	}
}

// BooleanEvaluation implements [Evaluator].
func (i *InProcess) BooleanEvaluation(_ context.Context, flagName string, defaultValue bool, flatCtx openfeature.FlattenedContext) openfeature.BoolResolutionDetail {
	v, prd := evaluate(i, flagName, defaultValue, flatCtx, "bool")
	return openfeature.BoolResolutionDetail{Value: v, ProviderResolutionDetail: prd}
}

// FloatEvaluation implements [Evaluator].
func (i *InProcess) FloatEvaluation(_ context.Context, flagName string, defaultValue float64, flatCtx openfeature.FlattenedContext) openfeature.FloatResolutionDetail {
	v, prd := evaluate(i, flagName, defaultValue, flatCtx, "float")
	return openfeature.FloatResolutionDetail{Value: v, ProviderResolutionDetail: prd}
}

// loadConfiguration fetches flag config from the relay proxy and updates state.
func (i *InProcess) loadConfiguration(ctx context.Context) (configurationRefreshStatus, error) {
	i.mu.RLock()
	etag := i.etag
	i.mu.RUnlock()
	resp, err := i.goffAPI.GetConfiguration(ctx, nil, etag)
	if errors.Is(err, api.ErrNotModified) {
		return configurationRefreshNotModified, nil
	}
	if err != nil {
		return configurationRefreshNotModified, err
	}
	i.mu.Lock()
	i.flagConfig = resp.Flags
	i.evaluationContextEnrichment = resp.EvaluationContextEnrichment
	i.etag = resp.Etag
	i.mu.Unlock()
	return configurationRefreshChanged, nil
}

func (i *InProcess) emitEvent(eventType openfeature.EventType, message string) {
	select {
	case i.eventStream <- openfeature.Event{
		ProviderName: inProcessProviderName,
		EventType:    eventType,
		ProviderEventDetails: openfeature.ProviderEventDetails{
			Message: message,
		},
	}:
	default:
		// event stream full; drop event to avoid blocking the polling goroutine
	}
}

func (i *InProcess) markStale() bool {
	i.mu.Lock()
	defer i.mu.Unlock()
	if i.stale {
		return false
	}
	i.stale = true
	return true
}

func (i *InProcess) clearStale() bool {
	i.mu.Lock()
	defer i.mu.Unlock()
	wasStale := i.stale
	i.stale = false
	return wasStale
}

func (i *InProcess) handleRefreshResult(status configurationRefreshStatus, err error) {
	if err != nil {
		if i.markStale() {
			i.emitEvent(openfeature.ProviderStale, "Configuration refresh failed: "+err.Error())
		}
		return
	}

	wasStale := i.clearStale()
	switch status {
	case configurationRefreshChanged:
		i.emitEvent(openfeature.ProviderConfigChange, "Configuration has changed")
	case configurationRefreshNotModified:
		if wasStale {
			i.emitEvent(openfeature.ProviderReady, "Configuration refresh recovered")
		}
	}
}

// Init implements [Evaluator].
func (i *InProcess) Init(ctx context.Context) error {
	status, err := i.loadConfiguration(ctx)
	if err != nil {
		return err
	}
	i.handleRefreshResult(status, nil)
	interval := i.flagChangePollingInterval
	if interval <= 0 {
		interval = pollingIntervalDefault
	}
	i.stopPolling = make(chan struct{})
	i.pollingDone = make(chan struct{}) // replace pre-closed channel with a fresh one for this goroutine
	i.shutdownOnce = sync.Once{}
	go func() {
		defer close(i.pollingDone)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-i.stopPolling:
				return
			case <-ticker.C:
				status, err := i.loadConfiguration(context.Background())
				i.handleRefreshResult(status, err)
			}
		}
	}()
	return nil
}

// IntEvaluation implements [Evaluator].
func (i *InProcess) IntEvaluation(_ context.Context, flagName string, defaultValue int64, flatCtx openfeature.FlattenedContext) openfeature.IntResolutionDetail {
	v, prd := evaluate(i, flagName, int(defaultValue), flatCtx, "int")
	return openfeature.IntResolutionDetail{Value: int64(v), ProviderResolutionDetail: prd}
}

// ObjectEvaluation implements [Evaluator].
func (i *InProcess) ObjectEvaluation(_ context.Context, flagName string, defaultValue any, flatCtx openfeature.FlattenedContext) openfeature.InterfaceResolutionDetail {
	v, prd := evaluate[any](i, flagName, defaultValue, flatCtx, "object")
	return openfeature.InterfaceResolutionDetail{Value: v, ProviderResolutionDetail: prd}
}

// Shutdown implements [Evaluator].
func (i *InProcess) Shutdown(ctx context.Context) error {
	i.shutdownOnce.Do(func() { close(i.stopPolling) })
	<-i.pollingDone
	return nil
}

// StringEvaluation implements [Evaluator].
func (i *InProcess) StringEvaluation(_ context.Context, flagName string, defaultValue string, flatCtx openfeature.FlattenedContext) openfeature.StringResolutionDetail {
	v, prd := evaluate(i, flagName, defaultValue, flatCtx, "string")
	return openfeature.StringResolutionDetail{Value: v, ProviderResolutionDetail: prd}
}

// getEvaluationContextEnrichment returns a copy of the current evaluation context enrichment (thread-safe).
func (i *InProcess) getEvaluationContextEnrichment() map[string]any {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return maps.Clone(i.evaluationContextEnrichment)
}

// checkFlagExists checks if the flag exists in the flag config.
// If the flag does not exist, it returns a flag not found resolution error.
func (i *InProcess) checkFlagExists(flagName string) (*flag.InternalFlag, *openfeature.ProviderResolutionDetail) {
	i.mu.RLock()
	flagCfg, ok := i.flagConfig[flagName]
	i.mu.RUnlock()
	if !ok {
		return nil, &openfeature.ProviderResolutionDetail{
			ResolutionError: openfeature.NewFlagNotFoundResolutionError(flagName),
		}
	}
	return &flagCfg, nil
}

// toFFContext converts an openfeature FlattenedContext to an ffcontext.Context.
func toFFContext(flatCtx openfeature.FlattenedContext) ffcontext.Context {
	var key string
	if targetingKey, ok := flatCtx["targetingKey"]; ok {
		if keyStr, isString := targetingKey.(string); isString {
			key = keyStr
		}
	}
	b := ffcontext.NewEvaluationContextBuilder(key)
	for k, v := range flatCtx {
		if k == "targetingKey" {
			continue
		}
		b.AddCustom(k, v)
	}
	return b.Build()
}

// toResolutionError maps a go-feature-flag error code to the appropriate openfeature ResolutionError.
func toResolutionError(errorCode flag.ErrorCode, msg string) openfeature.ResolutionError {
	switch errorCode {
	case flag.ErrorCodeProviderNotReady:
		return openfeature.NewProviderNotReadyResolutionError(msg)
	case flag.ErrorCodeFlagNotFound:
		return openfeature.NewFlagNotFoundResolutionError(msg)
	case flag.ErrorCodeParseError:
		return openfeature.NewParseErrorResolutionError(msg)
	case flag.ErrorCodeTypeMismatch:
		return openfeature.NewTypeMismatchResolutionError(msg)
	case flag.ErrorCodeTargetingKeyMissing:
		return openfeature.NewTargetingKeyMissingResolutionError(msg)
	case flag.ErrorCodeInvalidContext:
		return openfeature.NewInvalidContextResolutionError(msg)
	default:
		return openfeature.NewGeneralResolutionError(msg)
	}
}
