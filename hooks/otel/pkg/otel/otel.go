package otel

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	of "github.com/open-feature/golang-sdk/pkg/openfeature"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const (
	FlagKey          = "feature_flag.flag_key"
	ProviderName     = "feature_flag.provider_name"
	EvaluatedVariant = "feature_flag.evaluated_variant"
	EvaluatedValue   = "feature_flag.evaluated_value"
)

type Hook struct {
	spans map[string]*mutexWrapper
	ctx   context.Context
	wg    *sync.WaitGroup
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
	ctx, span := otel.Tracer("Flag Evaluation").Start(h.ctx, key)
	ctx, cancel := context.WithCancel(ctx)
	span.SetAttributes(
		attribute.String(FlagKey, hookContext.FlagKey()),
		attribute.String(ProviderName, hookContext.ProviderMetadata().Name),
	)
	h.spans[key].ss = &storedSpan{
		cancel: cancel,
		span:   span,
	}
	go func() {
		<-ctx.Done()
		span.End()
		h.spans[key].ss = nil
		h.spans[key].mu.Unlock()
		h.wg.Done()
	}()
	evCtx := hookContext.EvaluationContext()
	return &evCtx, nil
}

// After sets the EvaluatedVariant and EvaluatedValue on the evaluation specific span
func (h *Hook) After(hookContext of.HookContext, flagEvaluationDetails of.EvaluationDetails, hookHints of.HookHints) error {
	key := fmt.Sprintf("%s.%s", hookContext.ClientMetadata().Name(), hookContext.FlagKey())
	mw, ok := h.spans[key]
	if !ok {
		return errors.New("no span stored for provided hook context")
	}
	mw.ss.span.SetAttributes(
		attribute.String(EvaluatedVariant, flagEvaluationDetails.Variant),
	)
	switch flagEvaluationDetails.FlagType {
	case of.Boolean:
		mw.ss.span.SetAttributes(
			attribute.Bool(EvaluatedValue, flagEvaluationDetails.Value.(bool)),
		)
	case of.String:
		mw.ss.span.SetAttributes(
			attribute.String(EvaluatedValue, flagEvaluationDetails.Value.(string)),
		)
	case of.Float:
		mw.ss.span.SetAttributes(
			attribute.Float64(EvaluatedValue, flagEvaluationDetails.Value.(float64)),
		)
	case of.Int:
		mw.ss.span.SetAttributes(
			attribute.Int64(EvaluatedValue, flagEvaluationDetails.Value.(int64)),
		)
	case of.Object:
		val, err := json.Marshal(flagEvaluationDetails.Value)
		if err != nil {
			return err
		}
		mw.ss.span.SetAttributes(
			attribute.String(EvaluatedValue, string(val)),
		)
	}
	mw.ss.cancel()
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
func (h Hook) Finally(hookContext of.HookContext, hookHints of.HookHints) {}

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
