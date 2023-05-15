package otel

import (
	"context"
	"fmt"
	"github.com/open-feature/go-sdk/pkg/openfeature"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const (
	EventName                 = "feature_flag"
	EventPropertyFlagKey      = "feature_flag.key"
	EventPropertyProviderName = "feature_flag.provider_name"
	EventPropertyVariant      = "feature_flag.variant"
)

type hook struct {
	setErrorStatus bool
	openfeature.UnimplementedHook
}

// NewHook return a reference to a new instance of the OpenTelemetry Hook
func NewHook(opts ...Options) *hook {
	h := &hook{
		setErrorStatus: true,
	}

	for _, opt := range opts {
		opt(h)
	}

	return h
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

	if h.setErrorStatus {
		span.SetStatus(codes.Error,
			fmt.Sprintf("error evaluating flag '%s' of type '%s'", hookContext.FlagKey(), hookContext.FlagType().String()))
	}

	span.RecordError(err)
}

// Options of the hook

type Options func(*hook)

// WithErrorStatusDisabled prevents setting span status to codes.Error in case of an error. Default behavior is enabled
func WithErrorStatusDisabled() Options {
	return func(h *hook) {
		h.setErrorStatus = false
	}
}

var _ openfeature.Hook = &hook{}
