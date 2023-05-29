package otel

import (
	"context"
	"github.com/open-feature/go-sdk/pkg/openfeature"
	"go.opentelemetry.io/otel/attribute"
	api "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/sdk/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.18.0"
)

const (
	meterName = "OpenFeature/Metrics"

	evaluationActive   = "feature_flag.evaluation_active_count"
	evaluationRequests = "feature_flag.evaluation_requests_total"
	evaluationSuccess  = "feature_flag.evaluation_success_total"
	evaluationErrors   = "feature_flag.evaluation_error_total"
)

type MetricsHook struct {
	activeCounter  api.Int64UpDownCounter
	requestCounter api.Int64Counter
	successCounter api.Int64Counter
	errorCounter   api.Int64Counter

	flagEvalMetadataDimensions []DimensionDescription
}

type MetricOptions func(*MetricsHook)

// NewMetricsHook builds a metric hook backed by provided metric.Reader. Reader must be provided by developer and
// its configurations govern metric exports
func NewMetricsHook(reader metric.Reader, opts ...MetricOptions) (*MetricsHook, error) {
	provider := metric.NewMeterProvider(metric.WithReader(reader))
	meter := provider.Meter(meterName)

	activeCounter, err := meter.Int64UpDownCounter(evaluationActive, api.WithDescription("active flag evaluations counter"))
	if err != nil {
		return nil, err
	}

	evalCounter, err := meter.Int64Counter(evaluationRequests, api.WithDescription("feature flag evaluation request counter"))
	if err != nil {
		return nil, err
	}

	successCounter, err := meter.Int64Counter(evaluationSuccess, api.WithDescription("feature flag evaluation success counter"))
	if err != nil {
		return nil, err
	}

	errorCounter, err := meter.Int64Counter(evaluationErrors, api.WithDescription("feature flag evaluation error counter"))
	if err != nil {
		return nil, err
	}

	m := &MetricsHook{
		activeCounter:  activeCounter,
		requestCounter: evalCounter,
		successCounter: successCounter,
		errorCounter:   errorCounter,
	}

	for _, opt := range opts {
		opt(m)
	}

	return m, nil
}

func (h *MetricsHook) Before(ctx context.Context, hCtx openfeature.HookContext,
	hint openfeature.HookHints) (*openfeature.EvaluationContext, error) {

	h.activeCounter.Add(ctx, +1, api.WithAttributes(semconv.FeatureFlagVariant(hCtx.FlagType().String())))
	h.requestCounter.Add(ctx, 1, api.WithAttributes(SemConvFeatureFlagAttributes(hCtx)...))

	return nil, nil
}

func (h *MetricsHook) After(ctx context.Context, hCtx openfeature.HookContext,
	details openfeature.InterfaceEvaluationDetails, hint openfeature.HookHints) error {

	reasonAttrib := attribute.String("reason", string(details.Reason))
	attribs := append(SemConvFeatureFlagAttributes(hCtx), reasonAttrib)

	fromMetadata, err := descriptionsToAttributes(details.FlagMetadata, h.flagEvalMetadataDimensions)
	if err != nil {
		return err
	}

	attribs = append(attribs, fromMetadata...)

	h.successCounter.Add(ctx, 1, api.WithAttributes(attribs...))

	return nil
}

func (h *MetricsHook) Error(ctx context.Context, hCtx openfeature.HookContext, err error, hint openfeature.HookHints) {
	errorReason := attribute.String(semconv.ExceptionEventName, err.Error())
	h.errorCounter.Add(ctx, 1, api.WithAttributes(append(SemConvFeatureFlagAttributes(hCtx), errorReason)...))
}

func (h *MetricsHook) Finally(ctx context.Context, hCtx openfeature.HookContext, hint openfeature.HookHints) {
	h.activeCounter.Add(ctx, -1, api.WithAttributes(semconv.FeatureFlagVariant(hCtx.FlagType().String())))
}

// Extra options for metrics hook

type Type int

// Type helper
const (
	Bool = iota
	String
	Int
	Float
)

// DimensionDescription is key and Type description of the dimension
type DimensionDescription struct {
	Key string
	Type
}

// WithFlagMetadataDimensions allows configuring extra dimensions for feature_flag.evaluation_success_total metric.
// If provided, dimensions will be extracted from openfeature.FlagMetadata & added to the metric with the same key
func WithFlagMetadataDimensions(descriptions ...DimensionDescription) MetricOptions {
	return func(metricsHook *MetricsHook) {
		metricsHook.flagEvalMetadataDimensions = descriptions
	}
}

// Metric helpers

// SemConvFeatureFlagAttributes a helper to derive feature flag semantic convention attributes from
// openfeature.HookContext
// Read more - https://opentelemetry.io/docs/reference/specification/trace/semantic_conventions/feature-flags/
func SemConvFeatureFlagAttributes(hookContext openfeature.HookContext) []attribute.KeyValue {
	return []attribute.KeyValue{
		semconv.FeatureFlagKey(hookContext.FlagKey()),
		semconv.FeatureFlagVariant(hookContext.FlagType().String()),
		semconv.FeatureFlagProviderName(hookContext.ProviderMetadata().Name),
	}
}

// descriptionsToAttributes is a helper to extract dimensions from openfeature.FlagMetadata
func descriptionsToAttributes(metadata openfeature.FlagMetadata, descriptions []DimensionDescription) (
	[]attribute.KeyValue, error) {

	attribs := []attribute.KeyValue{}
	for _, dimension := range descriptions {
		switch dimension.Type {
		case Bool:
			b, err := metadata.GetBool(dimension.Key)
			if err != nil {
				return nil, err
			}
			attribs = append(attribs, attribute.Bool(dimension.Key, b))
		case String:
			s, err := metadata.GetString(dimension.Key)
			if err != nil {
				return nil, err
			}
			attribs = append(attribs, attribute.String(dimension.Key, s))
		case Int:
			i, err := metadata.GetInt(dimension.Key)
			if err != nil {
				return nil, err
			}
			attribs = append(attribs, attribute.Int64(dimension.Key, i))
		case Float:
			f, err := metadata.GetFloat(dimension.Key)
			if err != nil {
				return nil, err
			}
			attribs = append(attribs, attribute.Float64(dimension.Key, f))
		}
	}

	return attribs, nil
}
