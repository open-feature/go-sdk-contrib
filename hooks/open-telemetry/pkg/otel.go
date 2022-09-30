package otel

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	of "github.com/open-feature/go-sdk/pkg/openfeature"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const (
	// AttributeFlagKey trace attribute name for the evaluated flag key
	AttributeFlagKey = "feature_flag.flag_key"
	// AttributeProviderName trace attribute name for the provider used for evaluation
	AttributeProviderName = "feature_flag.provider_name"
	// AttributeEvaluatedVariant trace attribute name for the variant returned from the flag evaluation
	AttributeEvaluatedVariant = "feature_flag.evaluated_variant"
	// AttributeEvaluatedValue trace attribute name for the value returned from the flag evaluation
	AttributeEvaluatedValue = "feature_flag.evaluated_value"
	// AttributeTraceName the name of the trace exported by the hook
	AttributeTraceName = "github.com/open-feature/go-sdk-contrib/hooks/open-telemetry/pkg"
)

type Hook struct {
	spans        map[string]*mutexWrapper
	ctx          context.Context
	wg           *sync.WaitGroup
	tracerClient tracerClientInterface
}

// mutex wrapper is used to prevent colliding keys from overwriting each other / closing a partially completed span
type mutexWrapper struct {
	mu *sync.Mutex
	ss *storedSpan // ss has been set as a pointer so it can be set to nil during cleanup
}
type storedSpan struct {
	cancel func()
	span   trace.Span
}

// NewHook return a reference to a new instance of the OpenTelemetry Hook
func NewHook() *Hook {
	return &Hook{
		tracerClient: &tracerClient{},
	}
}

// Wait blocks until all spans have been closed
func (h *Hook) Wait() {
	h.wg.Wait()
}

// WithContext sets the parent context used when new spans are formed. When the parent context is closed so are all spans.
// if no parent context is provided context.Background() is used internally
func (h *Hook) WithContext(ctx context.Context) {
	h.ctx = ctx
}

// Before creates the flag evaluations open-telemetry span and sets the FlagKey and ProviderName attributes
func (h *Hook) Before(hookContext of.HookContext, hookHints of.HookHints) (*of.EvaluationContext, error) {
	key := fmt.Sprintf("%s.%s", hookContext.ClientMetadata().Name(), hookContext.FlagKey())
	h.setup()
	if _, ok := h.spans[key]; !ok {
		h.spans[key] = &mutexWrapper{
			mu: &sync.Mutex{},
		}
	}
	h.spans[key].mu.Lock()
	h.wg.Add(1)
	ctx, span := h.tracerClient.tracer().Start(h.ctx, key)
	ctx, cancel := context.WithCancel(ctx)
	span.SetAttributes(
		attribute.String(AttributeFlagKey, hookContext.FlagKey()),
		attribute.String(AttributeProviderName, hookContext.ProviderMetadata().Name),
	)
	h.spans[key].ss = &storedSpan{
		cancel: cancel,
		span:   span,
	}
	// this goroutine cleans up the span, if the associated context is closed then the stored
	// span data is removed and the resource is unlocked. This context close can come from either the cancel() method
	// or from the closing of the parent context outside the scope of the hook
	go func() {
		<-ctx.Done()
		h.spans[key].ss = nil
		h.spans[key].mu.Unlock()
		h.wg.Done()
	}()
	return nil, nil
}

// After sets the EvaluatedVariant and EvaluatedValue on the evaluation specific span
func (h *Hook) After(hookContext of.HookContext, flagEvaluationDetails of.InterfaceEvaluationDetails, hookHints of.HookHints) error {
	key := fmt.Sprintf("%s.%s", hookContext.ClientMetadata().Name(), hookContext.FlagKey())
	mw, ok := h.spans[key]
	if !ok {
		return errors.New("no span stored for provided hook context")
	}
	if flagEvaluationDetails.ResolutionDetail.Variant != "" {
		mw.ss.span.SetAttributes(
			attribute.String(AttributeEvaluatedVariant, flagEvaluationDetails.ResolutionDetail.Variant),
		)
		return nil
	}
	var value string
	switch flagEvaluationDetails.FlagType {
	case of.Boolean:
		value = fmt.Sprintf("%t", flagEvaluationDetails.Value.(bool))
	case of.String:
		value = flagEvaluationDetails.Value.(string)
	case of.Float:
		value = fmt.Sprintf("%64f", flagEvaluationDetails.Value.(float64))
	case of.Int:
		value = fmt.Sprintf("%d", flagEvaluationDetails.Value.(int64))
	case of.Object:
		val, err := json.Marshal(flagEvaluationDetails.Value)
		if err != nil {
			return err
		}
		value = string(val)
	default:
		return fmt.Errorf("unknown data type received: %d", flagEvaluationDetails.FlagType)
	}
	mw.ss.span.SetAttributes(
		attribute.String(AttributeEvaluatedValue, value),
	)
	return nil
}

// Error records the given error against the span and sets the span to an error status
func (h *Hook) Error(hookContext of.HookContext, err error, hookHints of.HookHints) {
	key := fmt.Sprintf("%s.%s", hookContext.ClientMetadata().Name(), hookContext.FlagKey())
	mw, ok := h.spans[key]
	if ok {
		mw.ss.span.RecordError(err)
		mw.ss.span.SetStatus(codes.Error, err.Error())
	}
}

// Finally this method is unused for this hook, spans are closed via context
func (h Hook) Finally(hookContext of.HookContext, hookHints of.HookHints) {
	key := fmt.Sprintf("%s.%s", hookContext.ClientMetadata().Name(), hookContext.FlagKey())
	mw, ok := h.spans[key]
	if ok {
		mw.ss.span.End()
		mw.ss.cancel()
	}
}

func (h *Hook) setup() {
	if h.wg == nil {
		wg := sync.WaitGroup{}
		h.wg = &wg
	}
	if h.spans == nil {
		h.spans = map[string]*mutexWrapper{}
	}
	if h.ctx == nil {
		h.ctx = context.Background()
	}
}
