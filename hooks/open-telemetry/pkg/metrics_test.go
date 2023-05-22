package otel

import (
	"context"
	"errors"
	"github.com/open-feature/go-sdk/pkg/openfeature"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"reflect"
	"testing"
)

func TestMetricsHook_BeforeStage(t *testing.T) {
	// Validate metrics of before stage

	// given
	manualReader := metric.NewManualReader()

	ctx := context.Background()

	hookContext := hookContext()
	hookHints := openfeature.NewHookHints(map[string]interface{}{})

	metricsHook, err := NewMetricsHook(manualReader)
	if err != nil {
		t.Error(err)
		return
	}

	// when
	_, err = metricsHook.Before(ctx, hookContext, hookHints)
	if err != nil {
		t.Error(err)
		return
	}

	// then
	var data metricdata.ResourceMetrics

	err = manualReader.Collect(ctx, &data)
	if err != nil {
		t.Error(err)
		return
	}

	scopeMetrics := data.ScopeMetrics
	if len(scopeMetrics) < 1 {
		t.Error("expected scope metrics to be non empty with before hook")
	}

	metrics := scopeMetrics[0].Metrics
	if len(metrics) != 2 {
		t.Errorf("expected 2 metrics, %s & %s to be present with before hook", evaluationCounter, evaluationActive)
	}
}

func TestMetricsHook_AfterStage(t *testing.T) {
	// Validate metrics of after stage

	// given
	manualReader := metric.NewManualReader()

	ctx := context.Background()

	hookContext := hookContext()
	evalDetails := openfeature.InterfaceEvaluationDetails{
		Value: true,
		EvaluationDetails: openfeature.EvaluationDetails{
			FlagKey:          "flagA",
			FlagType:         openfeature.Boolean,
			ResolutionDetail: openfeature.ResolutionDetail{},
		},
	}
	hookHints := openfeature.NewHookHints(map[string]interface{}{})

	metricsHook, err := NewMetricsHook(manualReader)
	if err != nil {
		t.Error(err)
		return
	}

	// when
	err = metricsHook.After(ctx, hookContext, evalDetails, hookHints)
	if err != nil {
		t.Error(err)
		return
	}

	// then
	var data metricdata.ResourceMetrics

	err = manualReader.Collect(ctx, &data)
	if err != nil {
		t.Error(err)
		return
	}

	scopeMetrics := data.ScopeMetrics
	if len(scopeMetrics) < 1 {
		t.Error("expected scope metrics to be non empty with after hook")
	}

	metrics := scopeMetrics[0].Metrics
	if len(metrics) < 1 {
		t.Errorf("expected metric, %s to be present with after hook", successCounter)
	}

	if metrics[0].Name != successCounter {
		t.Errorf("expected %s to be present with after hook", successCounter)
	}
}

func TestMetricsHook_ErrorStage(t *testing.T) {
	// Validate metrics of error stage

	// given
	manualReader := metric.NewManualReader()

	ctx := context.Background()

	hookContext := hookContext()
	evalError := errors.New("some eval error")
	hookHints := openfeature.NewHookHints(map[string]interface{}{})

	metricsHook, err := NewMetricsHook(manualReader)
	if err != nil {
		t.Error(err)
		return
	}

	// when
	metricsHook.Error(ctx, hookContext, evalError, hookHints)

	// then
	var data metricdata.ResourceMetrics

	err = manualReader.Collect(ctx, &data)
	if err != nil {
		t.Error(err)
		return
	}

	scopeMetrics := data.ScopeMetrics
	if len(scopeMetrics) < 1 {
		t.Error("expected scope metrics to be non empty with error hook")
	}

	metrics := scopeMetrics[0].Metrics
	if len(metrics) < 1 {
		t.Errorf("expected metric, %s to be present with error hook", errorCounter)
	}

	errorCounterMetric := metrics[0]

	if errorCounterMetric.Name != errorCounter {
		t.Errorf("expected %s to be present with error hook", errorCounter)
	}

	m := errorCounterMetric.Data.(metricdata.Sum[int64])

	// verify for zero count with before + finally execution
	if m.DataPoints[0].Value != 1 {
		t.Errorf("expected value 1 for error counter")
	}
}

func TestMetricsHook_FinallyStage(t *testing.T) {
	// Validate metrics of finally stage

	// given
	manualReader := metric.NewManualReader()

	ctx := context.Background()

	hookContext := hookContext()
	hookHints := openfeature.NewHookHints(map[string]interface{}{})

	metricsHook, err := NewMetricsHook(manualReader)
	if err != nil {
		t.Error(err)
		return
	}

	// when
	metricsHook.Finally(ctx, hookContext, hookHints)

	// then
	var data metricdata.ResourceMetrics

	err = manualReader.Collect(ctx, &data)
	if err != nil {
		t.Error(err)
		return
	}

	scopeMetrics := data.ScopeMetrics
	if len(scopeMetrics) < 1 {
		t.Error("expected scope metrics to be non empty with finally hook")
	}

	metrics := scopeMetrics[0].Metrics
	if len(metrics) != 1 {
		t.Errorf("expected metric, %s to be present with finally hook", evaluationActive)
	}

	if metrics[0].Name != evaluationActive {
		t.Errorf("expected %s to be present with finally hook", evaluationActive)
	}
}

func TestMetricsHook_ActiveCounterShouldBeZero(t *testing.T) {
	// Validate active evaluation count to be zero with before & after stage completion

	// given
	manualReader := metric.NewManualReader()

	ctx := context.Background()

	hookContext := hookContext()
	hookHints := openfeature.NewHookHints(map[string]interface{}{})

	metricsHook, err := NewMetricsHook(manualReader)
	if err != nil {
		t.Error(err)
		return
	}

	// when - executed before & after hooks
	_, err = metricsHook.Before(ctx, hookContext, hookHints)
	if err != nil {
		t.Error(err)
		return
	}

	metricsHook.Finally(ctx, hookContext, hookHints)

	// then
	var data metricdata.ResourceMetrics

	err = manualReader.Collect(ctx, &data)
	if err != nil {
		t.Error(err)
		return
	}

	scopeMetrics := data.ScopeMetrics
	if len(scopeMetrics) < 1 {
		t.Error("expected scope metrics to be non")
	}

	metrics := scopeMetrics[0].Metrics
	if len(metrics) < 1 {
		t.Error("expected metrics to be present")
	}

	// extract evaluation active
	var activeEvalMetric metricdata.Metrics

	for _, m := range metrics {
		if m.Name == evaluationActive {
			activeEvalMetric = m
			break
		}
	}

	if reflect.ValueOf(activeEvalMetric).IsZero() {
		t.Errorf("expected %s to be present", evaluationActive)
	}

	m := activeEvalMetric.Data.(metricdata.Sum[int64])

	// verify for zero count with before + finally execution
	if m.DataPoints[0].Value != 0 {
		t.Errorf("expected 0 value with before & finally stage executions")
	}
}

func hookContext() openfeature.HookContext {
	return openfeature.NewHookContext("flagA",
		openfeature.Boolean,
		false,
		openfeature.NewClientMetadata(""),
		openfeature.Metadata{
			Name: "provider",
		},
		openfeature.EvaluationContext{},
	)
}
