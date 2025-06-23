package otel

import (
	"context"

	"github.com/open-feature/go-sdk/openfeature"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.32.0"
)

const (
	meterName = "go.openfeature.dev"

	evaluationActive   = "feature_flag.evaluation_active_count"
	evaluationRequests = "feature_flag.evaluation_requests_total"
	evaluationSuccess  = "feature_flag.evaluation_success_total"
	evaluationErrors   = "feature_flag.evaluation_error_total"
)

type MetricsHook struct {
	activeCounter  metric.Int64UpDownCounter
	requestCounter metric.Int64Counter
	successCounter metric.Int64Counter
	errorCounter   metric.Int64Counter

	flagEvalMetadataDimensions []DimensionDescription
	attributeMapperCallback    func(openfeature.FlagMetadata) []attribute.KeyValue
}

var _ openfeature.Hook = &MetricsHook{}

// NewMetricsHook builds a metric hook backed by a globally set metric.MeterProvider.
// Use otel.SetMeterProvider to set the global provider or use NewMetricsHookForProvider.
func NewMetricsHook(opts ...MetricOptions) (*MetricsHook, error) {
	return NewMetricsHookForProvider(otel.GetMeterProvider(), opts...)
}

// NewMetricsHookForProvider builds a metric hook backed by metric.MeterProvider.
func NewMetricsHookForProvider(provider metric.MeterProvider, opts ...MetricOptions) (*MetricsHook, error) {
	meter := provider.Meter(meterName)

	activeCounter, err := meter.Int64UpDownCounter(evaluationActive, metric.WithDescription("active flag evaluations counter"))
	if err != nil {
		return nil, err
	}

	evalCounter, err := meter.Int64Counter(evaluationRequests, metric.WithDescription("feature flag evaluation request counter"))
	if err != nil {
		return nil, err
	}

	successCounter, err := meter.Int64Counter(evaluationSuccess, metric.WithDescription("feature flag evaluation success counter"))
	if err != nil {
		return nil, err
	}

	errorCounter, err := meter.Int64Counter(evaluationErrors, metric.WithDescription("feature flag evaluation error counter"))
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
	hint openfeature.HookHints,
) (*openfeature.EvaluationContext, error) {
	h.activeCounter.Add(ctx, +1, metric.WithAttributes(semconv.FeatureFlagKey(hCtx.FlagKey())))

	h.requestCounter.Add(ctx, 1,
		metric.WithAttributes(
			semconv.FeatureFlagKey(hCtx.FlagKey()),
			semconv.FeatureFlagProviderName(hCtx.ProviderMetadata().Name)))

	return nil, nil
}

func (h *MetricsHook) After(ctx context.Context, hCtx openfeature.HookContext,
	details openfeature.InterfaceEvaluationDetails, hint openfeature.HookHints,
) error {
	attribs := []attribute.KeyValue{
		semconv.FeatureFlagKey(hCtx.FlagKey()),
		semconv.FeatureFlagProviderName(hCtx.ProviderMetadata().Name),
	}

	if details.Variant != "" {
		attribs = append(attribs, semconv.FeatureFlagResultVariant(details.Variant))
	}

	if details.Reason != "" {
		attribs = append(attribs, attribute.String("reason", string(details.Reason)))
	}
	fromMetadata := descriptionsToAttributes(details.FlagMetadata, h.flagEvalMetadataDimensions)
	attribs = append(attribs, fromMetadata...)

	if h.attributeMapperCallback != nil {
		attribs = append(attribs, h.attributeMapperCallback(details.FlagMetadata)...)
	}

	h.successCounter.Add(ctx, 1, metric.WithAttributes(attribs...))

	return nil
}

func (h *MetricsHook) Error(ctx context.Context, hCtx openfeature.HookContext, err error, hint openfeature.HookHints) {
	h.errorCounter.Add(ctx, 1,
		metric.WithAttributes(
			semconv.FeatureFlagKey(hCtx.FlagKey()),
			semconv.FeatureFlagProviderName(hCtx.ProviderMetadata().Name),
			attribute.String(semconv.ExceptionEventName, err.Error())))
}

func (h *MetricsHook) Finally(ctx context.Context, hCtx openfeature.HookContext, flagEvaluationDetails openfeature.InterfaceEvaluationDetails, hint openfeature.HookHints) {
	h.activeCounter.Add(ctx, -1, metric.WithAttributes(semconv.FeatureFlagKey(hCtx.FlagKey())))
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

// Options of the hook

type MetricOptions func(*MetricsHook)

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

// WithMetricsAttributeSetter allows to set a extractionCallback which accept openfeature.FlagMetadata and returns
// []attribute.KeyValue derived from those metadata.
func WithMetricsAttributeSetter(callback func(openfeature.FlagMetadata) []attribute.KeyValue) MetricOptions {
	return func(metricsHook *MetricsHook) {
		metricsHook.attributeMapperCallback = callback
	}
}

// descriptionsToAttributes is a helper to extract dimensions from openfeature.FlagMetadata. Missing metadata
// dimensions are ignore.
func descriptionsToAttributes(metadata openfeature.FlagMetadata, descriptions []DimensionDescription) []attribute.KeyValue {
	attribs := []attribute.KeyValue{}
	for _, dimension := range descriptions {
		switch dimension.Type {
		case Bool:
			b, err := metadata.GetBool(dimension.Key)
			if err == nil {
				attribs = append(attribs, attribute.Bool(dimension.Key, b))
			}
		case String:
			s, err := metadata.GetString(dimension.Key)
			if err == nil {
				attribs = append(attribs, attribute.String(dimension.Key, s))
			}
		case Int:
			i, err := metadata.GetInt(dimension.Key)
			if err == nil {
				attribs = append(attribs, attribute.Int64(dimension.Key, i))
			}
		case Float:
			f, err := metadata.GetFloat(dimension.Key)
			if err == nil {
				attribs = append(attribs, attribute.Float64(dimension.Key, f))
			}
		}
	}

	return attribs
}
