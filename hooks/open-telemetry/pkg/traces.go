package otel

import (
	"context"
	"fmt"
	"strings"

	"github.com/open-feature/go-sdk/openfeature"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.34.0"
	"go.opentelemetry.io/otel/trace"
)

// EventName is the name of the span event.
const EventName = "feature_flag.evaluation"

const (
	flagMetaContextIDKey string = "contextId"
	flagMetaFlagSetIDKey string = "flagSetId"
	flagMetaVersionKey   string = "version"
)

// traceHook is the hook implementation for OTel traces.
type traceHook struct {
	attributeMapperCallback func(openfeature.FlagMetadata) []attribute.KeyValue

	openfeature.UnimplementedHook
}

var _ openfeature.Hook = &traceHook{}

// NewTracesHook return a reference to a new instance of the OpenTelemetry Hook.
func NewTracesHook(opts ...Options) *traceHook {
	h := &traceHook{}

	for _, opt := range opts {
		opt(h)
	}

	return h
}

// Finally adds the feature_flag event and associated attributes on the span stored in the context.
func (h *traceHook) Finally(ctx context.Context, hookContext openfeature.HookContext, flagEvaluationDetails openfeature.InterfaceEvaluationDetails, hookHints openfeature.HookHints) {
	attrs := eventAttributes(hookContext, flagEvaluationDetails)
	if h.attributeMapperCallback != nil {
		attrs = append(attrs, h.attributeMapperCallback(flagEvaluationDetails.FlagMetadata)...)
	}
	trace.SpanFromContext(ctx).AddEvent(EventName, trace.WithAttributes(attrs...))
}

// eventAttributes returns a slice of OpenTelemetry attributes that can be used to create an event for a feature flag evaluation.
func eventAttributes(hookContext openfeature.HookContext, details openfeature.InterfaceEvaluationDetails) []attribute.KeyValue {
	attrs := []attribute.KeyValue{
		semconv.FeatureFlagKey(hookContext.FlagKey()),
		semconv.FeatureFlagProviderName(hookContext.ProviderMetadata().Name),
	}

	switch v := details.Value.(type) {
	case bool:
		attrs = append(attrs, semconv.FeatureFlagResultValueKey.Bool(v))
	case string:
		attrs = append(attrs, semconv.FeatureFlagResultValueKey.String(v))
	case int64:
		attrs = append(attrs, semconv.FeatureFlagResultValueKey.Int64(v))
	case float64:
		attrs = append(attrs, semconv.FeatureFlagResultValueKey.Float64(v))

	// try to cover common types for object value supported by otel
	case []bool:
		attrs = append(attrs, semconv.FeatureFlagResultValueKey.BoolSlice(v))
	case []string:
		attrs = append(attrs, semconv.FeatureFlagResultValueKey.StringSlice(v))
	case []int64:
		attrs = append(attrs, semconv.FeatureFlagResultValueKey.Int64Slice(v))
	case []float64:
		attrs = append(attrs, semconv.FeatureFlagResultValueKey.Float64Slice(v))
	case int:
		attrs = append(attrs, semconv.FeatureFlagResultValueKey.Int(v))
	case []int:
		attrs = append(attrs, semconv.FeatureFlagResultValueKey.IntSlice(v))
	case float32:
		attrs = append(attrs, semconv.FeatureFlagResultValueKey.Float64(float64(v)))
	case []float32:
		vals := make([]float64, len(v))
		for i, val := range v {
			vals[i] = float64(val)
		}
		attrs = append(attrs, semconv.FeatureFlagResultValueKey.Float64Slice(vals))
	default:
		if val, ok := v.(fmt.Stringer); ok {
			attrs = append(attrs, semconv.FeatureFlagResultValueKey.String(val.String()))
		}
	}

	if details.Variant != "" {
		attrs = append(attrs, semconv.FeatureFlagResultVariant(details.Variant))
	}

	switch details.Reason {
	case openfeature.CachedReason:
		attrs = append(attrs, semconv.FeatureFlagResultReasonCached)
	case openfeature.DefaultReason:
		attrs = append(attrs, semconv.FeatureFlagResultReasonDefault)
	case openfeature.DisabledReason:
		attrs = append(attrs, semconv.FeatureFlagResultReasonDisabled)
	case openfeature.ErrorReason:
		attrs = append(attrs, semconv.FeatureFlagResultReasonError)
		errorType := openfeature.GeneralCode
		if details.ErrorCode != "" {
			errorType = details.ErrorCode
		}
		attrs = append(attrs, semconv.ErrorTypeKey.String(
			strings.ToLower(string(errorType)),
		))

		if details.ErrorMessage != "" {
			attrs = append(attrs, semconv.ErrorMessage(details.ErrorMessage))
		}
	case openfeature.SplitReason:
		attrs = append(attrs, semconv.FeatureFlagResultReasonSplit)
	case openfeature.StaticReason:
		attrs = append(attrs, semconv.FeatureFlagResultReasonStatic)
	case openfeature.TargetingMatchReason:
		attrs = append(attrs, semconv.FeatureFlagResultReasonTargetingMatch)
	default:
		attrs = append(attrs, semconv.FeatureFlagResultReasonUnknown)
	}

	contextID := hookContext.EvaluationContext().TargetingKey()
	if flagMetaContextID, ok := details.FlagMetadata[flagMetaContextIDKey].(string); ok {
		contextID = flagMetaContextID
	}
	attrs = append(attrs, semconv.FeatureFlagContextID(contextID))

	if setID, ok := details.FlagMetadata[flagMetaFlagSetIDKey].(string); ok {
		attrs = append(attrs, semconv.FeatureFlagSetID(setID))
	}

	if version, ok := details.FlagMetadata[flagMetaVersionKey].(string); ok {
		attrs = append(attrs, semconv.FeatureFlagVersion(version))
	}

	return attrs
}

// Options of the hook

type Options func(*traceHook)

// WithErrorStatusEnabled enable setting span status to codes.Error in case of an error. Default behavior is disabled.
//
// Deprecated: this option has no effect. It will be removed in a future release.
func WithErrorStatusEnabled() Options {
	return func(h *traceHook) {
	}
}

// WithTracesAttributeSetter allows to set a extractionCallback which accept [openfeature.FlagMetadata] and returns
// []attribute.KeyValue derived from those metadata.
// If present, returned attributes will be added to successful evaluation traces.
func WithTracesAttributeSetter(callback func(openfeature.FlagMetadata) []attribute.KeyValue) Options {
	return func(tracesHook *traceHook) {
		tracesHook.attributeMapperCallback = callback
	}
}
