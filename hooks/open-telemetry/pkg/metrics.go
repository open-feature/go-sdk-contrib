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

	evaluationActive  = "evaluationActive"
	evaluationCounter = "evaluationRequests"
	successCounter    = "evaluationSuccess"
	errorCounter      = "evaluationError"
)

type MetricsHook struct {
	activeCounter  api.Int64UpDownCounter
	evalCounter    api.Int64Counter
	successCounter api.Int64Counter
	errorCounter   api.Int64Counter
}

// NewMetricsHook builds a metric hook backed by provided metric.Reader. Reader must be provided by library user and
func NewMetricsHook(reader metric.Reader) (*MetricsHook, error) {
	provider := metric.NewMeterProvider(metric.WithReader(reader))
	meter := provider.Meter(meterName)

	activeCounter, err := meter.Int64UpDownCounter(evaluationActive, api.WithDescription("active evaluations counter"))
	if err != nil {
		return nil, err
	}

	evalCounter, err := meter.Int64Counter(evaluationCounter, api.WithDescription("feature flag evaluation counter"))
	if err != nil {
		return nil, err
	}

	successCounter, err := meter.Int64Counter(successCounter, api.WithDescription("feature flag success counter"))
	if err != nil {
		return nil, err
	}

	errorCounter, err := meter.Int64Counter(errorCounter, api.WithDescription("feature flag error counter"))
	if err != nil {
		return nil, err
	}

	return &MetricsHook{
		activeCounter:  activeCounter,
		evalCounter:    evalCounter,
		successCounter: successCounter,
		errorCounter:   errorCounter,
	}, nil
}

func (h *MetricsHook) Before(ctx context.Context, hCtx openfeature.HookContext, hint openfeature.HookHints) (*openfeature.EvaluationContext, error) {
	h.activeCounter.Add(ctx, +1, api.WithAttributes(semconv.FeatureFlagVariant(hCtx.FlagType().String())))

	h.evalCounter.Add(ctx, 1,
		api.WithAttributes(
			SemConvFeatureFlagAttributes(hCtx.FlagKey(), hCtx.FlagType().String(), hCtx.ProviderMetadata().Name)...))

	return nil, nil
}

func (h *MetricsHook) After(ctx context.Context, hCtx openfeature.HookContext,
	details openfeature.InterfaceEvaluationDetails, hint openfeature.HookHints) error {

	reasonAttrib := attribute.String("reason", string(details.Reason))
	attribs := append(
		SemConvFeatureFlagAttributes(hCtx.FlagKey(), hCtx.FlagType().String(), hCtx.ProviderMetadata().Name), reasonAttrib)

	h.successCounter.Add(ctx, 1, api.WithAttributes(attribs...))

	return nil
}

func (h *MetricsHook) Error(ctx context.Context, hCtx openfeature.HookContext, err error, hint openfeature.HookHints) {
	errorReason := attribute.String(semconv.ExceptionEventName, err.Error())

	attribs := append(
		SemConvFeatureFlagAttributes(hCtx.FlagKey(), hCtx.FlagType().String(), hCtx.ProviderMetadata().Name), errorReason)

	h.errorCounter.Add(ctx, 1, api.WithAttributes(attribs...))
}

func (h *MetricsHook) Finally(ctx context.Context, hCtx openfeature.HookContext, hint openfeature.HookHints) {
	h.activeCounter.Add(ctx, -1, api.WithAttributes(semconv.FeatureFlagVariant(hCtx.FlagType().String())))
}

// SemConvFeatureFlagAttributes is helper to derive semantic convention adhering feature flag attributes
// refer - https://opentelemetry.io/docs/reference/specification/trace/semantic_conventions/feature-flags/
func SemConvFeatureFlagAttributes(ffKey string, ffVariant string, provider string) []attribute.KeyValue {
	return []attribute.KeyValue{
		semconv.FeatureFlagKey(ffKey),
		semconv.FeatureFlagVariant(ffVariant),
		semconv.FeatureFlagProviderName(provider),
	}
}
