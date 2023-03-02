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

type hook struct {
	openfeature.UnimplementedHook
}

// NewHook return a reference to a new instance of the OpenTelemetry Hook
func NewHook() *hook {
	return &hook{}
}

// After sets the feature_flag event and associated attributes on the span stored in the context
func (h *hook) After(ctx context.Context, hookContext openfeature.HookContext, flagEvaluationDetails openfeature.InterfaceEvaluationDetails, hookHints openfeature.HookHints) error {
	span := trace.SpanFromContext(ctx)
	span.AddEvent(EventName, trace.WithAttributes(
		attribute.String(EventPropertyFlagKey, hookContext.FlagKey()),
		attribute.String(EventPropertyProviderName, hookContext.ProviderMetadata().Name),
		attribute.String(EventPropertyVariant, flagEvaluationDetails.Variant),
	))
	return nil
}

// Error records the given error against the span and sets the span to an error status
func (h *hook) Error(ctx context.Context, hookContext openfeature.HookContext, err error, hookHints openfeature.HookHints) {
	span := trace.SpanFromContext(ctx)
	span.RecordError(err)
}
