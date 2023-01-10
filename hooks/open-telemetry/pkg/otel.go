package otel

import (
	"context"

	"github.com/open-feature/go-sdk/pkg/openfeature"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const (
	EventName                 = "feature_flag"
	EventPropertyFlagKey      = "feature_flag.key"
	EventPropertyProviderName = "feature_flag.provider_name"
	EventPropertyVariant      = "feature_flag.variant"
)

type Hook struct {
	ctx context.Context
	openfeature.UnimplementedHook
}

// NewHook return a reference to a new instance of the OpenTelemetry Hook
func NewHook(ctx context.Context) *Hook {
	return &Hook{
		ctx: ctx,
	}
}

// After sets the feature_flag event and associated attributes on the span stored in the context
func (h *Hook) After(hookContext openfeature.HookContext, flagEvaluationDetails openfeature.InterfaceEvaluationDetails, hookHints openfeature.HookHints) error {
	if h.ctx == nil {
		// if no context is set trace.SpanFromContext will return a noop span
		return nil
	}
	span := trace.SpanFromContext(h.ctx)
	span.SetAttributes(
		attribute.String(EventPropertyFlagKey, hookContext.FlagKey()),
		attribute.String(EventPropertyProviderName, hookContext.ProviderMetadata().Name),
		attribute.String(EventPropertyVariant, flagEvaluationDetails.Variant),
	)
	span.AddEvent(EventName)
	return nil
}

// Error records the given error against the span and sets the span to an error status
func (h *Hook) Error(hookContext openfeature.HookContext, err error, hookHints openfeature.HookHints) {
	if h.ctx == nil {
		// if no context is set trace.SpanFromContext will return a noop span
		return
	}
	span := trace.SpanFromContext(h.ctx)
	span.RecordError(err)
}
