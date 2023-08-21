package otel

import (
	"context"
	"fmt"
	"github.com/open-feature/go-sdk/pkg/openfeature"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.18.0"
	"go.opentelemetry.io/otel/trace"
)

const (
	EventName                 = "feature_flag"
	EventPropertyFlagKey      = "feature_flag.key"
	EventPropertyProviderName = "feature_flag.provider_name"
	EventPropertyVariant      = "feature_flag.variant"
)

// traceHook is the hook implementation for OTel traces
type traceHook struct {
	setErrorStatus          bool
	attributeMapperCallback func(openfeature.FlagMetadata) []attribute.KeyValue

	openfeature.UnimplementedHook
}

var _ openfeature.Hook = &traceHook{}

// NewTracesHook return a reference to a new instance of the OpenTelemetry Hook
func NewTracesHook(opts ...Options) *traceHook {
	h := &traceHook{}

	for _, opt := range opts {
		opt(h)
	}

	return h
}

// After sets the feature_flag event and associated attributes on the span stored in the context
func (h *traceHook) After(ctx context.Context, hookContext openfeature.HookContext, flagEvaluationDetails openfeature.InterfaceEvaluationDetails, hookHints openfeature.HookHints) error {
	attribs := []attribute.KeyValue{
		semconv.FeatureFlagKey(hookContext.FlagKey()),
		semconv.FeatureFlagProviderName(hookContext.ProviderMetadata().Name),
	}
	if flagEvaluationDetails.Variant != "" {
		attribs = append(attribs, semconv.FeatureFlagVariant(flagEvaluationDetails.Variant))
	}

	if h.attributeMapperCallback != nil {
		attribs = append(attribs, h.attributeMapperCallback(flagEvaluationDetails.FlagMetadata)...)
	}

	span := trace.SpanFromContext(ctx)
	span.AddEvent(EventName, trace.WithAttributes(attribs...))
	return nil
}

// Error records the given error against the span and sets the span to an error status
func (h *traceHook) Error(ctx context.Context, hookContext openfeature.HookContext, err error, hookHints openfeature.HookHints) {
	span := trace.SpanFromContext(ctx)

	if h.setErrorStatus {
		span.SetStatus(codes.Error,
			fmt.Sprintf("error evaluating flag '%s' of type '%s'", hookContext.FlagKey(), hookContext.FlagType().String()))
	}

	span.RecordError(err, trace.WithAttributes(
		semconv.FeatureFlagKey(hookContext.FlagKey()),
		semconv.FeatureFlagProviderName(hookContext.ProviderMetadata().Name),
	))
}

// Options of the hook

type Options func(*traceHook)

// WithErrorStatusEnabled enable setting span status to codes.Error in case of an error. Default behavior is disabled
func WithErrorStatusEnabled() Options {
	return func(h *traceHook) {
		h.setErrorStatus = true
	}
}

// WithTracesAttributeSetter allows to set a extractionCallback which accept openfeature.FlagMetadata and returns
// []attribute.KeyValue derived from those metadata.
// If present, returned attributes will be added to successful evaluation traces
func WithTracesAttributeSetter(callback func(openfeature.FlagMetadata) []attribute.KeyValue) Options {
	return func(tracesHook *traceHook) {
		tracesHook.attributeMapperCallback = callback
	}
}
