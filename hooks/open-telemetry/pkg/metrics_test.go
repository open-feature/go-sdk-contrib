package otel

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/open-feature/go-sdk/pkg/openfeature"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestMetricsHook_BeforeStage(t *testing.T) {
	// Validate metrics of before stage

	// given
	manualReader := metric.NewManualReader()

	ctx := context.Background()

	hookHints := openfeature.NewHookHints(map[string]interface{}{})

	metricsHook, err := NewMetricsHookForProvider(metric.NewMeterProvider(metric.WithReader(manualReader)))
	if err != nil {
		t.Error(err)
		return
	}

	// when
	_, err = metricsHook.Before(ctx, hookContext(), hookHints)
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
		t.Errorf("expected 2 metrics, %s & %s to be present with before hook", evaluationRequests, evaluationActive)
	}
}

func TestMetricsHook_AfterStage(t *testing.T) {
	// Validate metrics of after stage

	// given
	manualReader := metric.NewManualReader()

	ctx := context.Background()

	evalDetails := openfeature.InterfaceEvaluationDetails{
		Value: true,
		EvaluationDetails: openfeature.EvaluationDetails{
			FlagKey:          "flagA",
			FlagType:         openfeature.Boolean,
			ResolutionDetail: openfeature.ResolutionDetail{},
		},
	}
	hookHints := openfeature.NewHookHints(map[string]interface{}{})

	metricsHook, err := NewMetricsHookForProvider(metric.NewMeterProvider(metric.WithReader(manualReader)))
	if err != nil {
		t.Error(err)
		return
	}

	// when
	err = metricsHook.After(ctx, hookContext(), evalDetails, hookHints)
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
		t.Errorf("expected metric, %s to be present with after hook", evaluationSuccess)
	}

	if metrics[0].Name != evaluationSuccess {
		t.Errorf("expected %s to be present with after hook", evaluationSuccess)
	}
}

func TestMetricsHook_ErrorStage(t *testing.T) {
	// Validate metrics of error stage

	// given
	manualReader := metric.NewManualReader()

	ctx := context.Background()

	evalError := errors.New("some eval error")
	hookHints := openfeature.NewHookHints(map[string]interface{}{})

	metricsHook, err := NewMetricsHookForProvider(metric.NewMeterProvider(metric.WithReader(manualReader)))
	if err != nil {
		t.Error(err)
		return
	}

	// when
	metricsHook.Error(ctx, hookContext(), evalError, hookHints)

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
		t.Errorf("expected metric, %s to be present with error hook", evaluationErrors)
	}

	errorCounterMetric := metrics[0]

	if errorCounterMetric.Name != evaluationErrors {
		t.Errorf("expected %s to be present with error hook", evaluationErrors)
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

	metricsHook, err := NewMetricsHookForProvider(metric.NewMeterProvider(metric.WithReader(manualReader)))
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

	metricsHook, err := NewMetricsHookForProvider(metric.NewMeterProvider(metric.WithReader(manualReader)))
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

func TestMetricHook_OptionMetadataDimensions(t *testing.T) {
	// Test validates WithFlagMetadataDimensions option

	// given
	manualReader := metric.NewManualReader()
	ctx := context.Background()

	scopeDescription := DimensionDescription{
		Key:  "scope",
		Type: String,
	}
	scopeValue := "7c34165e-fbef-11ed-be56-0242ac120002"

	stageDescription := DimensionDescription{
		Key:  "stage",
		Type: Int,
	}
	stageValue := 1

	scoreDescription := DimensionDescription{
		Key:  "score",
		Type: Float,
	}
	scoreValue := 4.5

	cachedDescription := DimensionDescription{
		Key:  "cached",
		Type: Bool,
	}
	cachedValue := false

	evalMetadata := map[string]interface{}{
		scopeDescription.Key:  scopeValue,
		stageDescription.Key:  stageValue,
		scoreDescription.Key:  scoreValue,
		cachedDescription.Key: cachedValue,
	}

	evalDetails := openfeature.InterfaceEvaluationDetails{
		Value: true,
		EvaluationDetails: openfeature.EvaluationDetails{
			FlagKey:  "flagA",
			FlagType: openfeature.Boolean,
			ResolutionDetail: openfeature.ResolutionDetail{
				FlagMetadata: evalMetadata,
			},
		},
	}
	hookHints := openfeature.NewHookHints(map[string]interface{}{})

	metricsHook, err := NewMetricsHookForProvider(metric.NewMeterProvider(metric.WithReader(manualReader)),
		WithFlagMetadataDimensions(scopeDescription, stageDescription, scoreDescription, cachedDescription))
	if err != nil {
		t.Error(err)
		return
	}

	// when
	err = metricsHook.After(ctx, hookContext(), evalDetails, hookHints)
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
		t.Errorf("expected metric, %s to be present with after hook", metrics)
	}

	successMetric := metrics[0]

	if successMetric.Name != evaluationSuccess {
		t.Errorf("expected %s to be present with after hook", evaluationSuccess)
	}

	instrument := successMetric.Data.(metricdata.Sum[int64])

	if len(instrument.DataPoints) < 1 {
		t.Error("expected data points, but found none")
	}

	attributes := instrument.DataPoints[0].Attributes

	value, ok := attributes.Value(attribute.Key(scopeDescription.Key))
	if !ok || value.AsString() != scopeValue {
		t.Errorf("attribute %s is incorrectly configured", scopeDescription.Key)
	}

	value, ok = attributes.Value(attribute.Key(stageDescription.Key))
	if !ok || value.AsInt64() != int64(stageValue) {
		t.Errorf("attribute %s is incorrectly configured", stageDescription.Key)
	}

	value, ok = attributes.Value(attribute.Key(scoreDescription.Key))
	if !ok || value.AsFloat64() != scoreValue {
		t.Errorf("attribute %s is incorrectly configured", scoreDescription.Key)
	}

	value, ok = attributes.Value(attribute.Key(cachedDescription.Key))
	if !ok || value.AsBool() != cachedValue {
		t.Errorf("attribute %s is incorrectly configured", cachedDescription.Key)
	}
}

func TestMetricHook_OptionWithAttributeMapper(t *testing.T) {
	// Test validates WithAttributeMapper option

	// given
	manualReader := metric.NewManualReader()
	ctx := context.Background()

	scopeKey := "scope"
	scopeValue := "7c34165e-fbef-11ed-be56-0242ac120002"

	stageKey := "stage"
	stageValue := 1

	scoreKey := "score"
	scoreValue := 4.5

	cachedKey := "cached"
	cachedValue := false

	attributeMapper := func(md openfeature.FlagMetadata) []attribute.KeyValue {
		scopeValue, _ := md.GetString(scopeKey)
		stageValue, _ := md.GetInt(stageKey)
		scoreValue, _ := md.GetFloat(scoreKey)
		cachedValue, _ := md.GetBool(cachedKey)

		return []attribute.KeyValue{
			attribute.String(scopeKey, scopeValue),
			attribute.Int64(stageKey, stageValue),
			attribute.Float64(scoreKey, scoreValue),
			attribute.Bool(cachedKey, cachedValue),
		}
	}

	metricsHook, err := NewMetricsHookForProvider(
		metric.NewMeterProvider(metric.WithReader(manualReader)),
		WithAttributeMapper(attributeMapper),
	)
	if err != nil {
		t.Error(err)
		return
	}

	evalDetails := openfeature.InterfaceEvaluationDetails{
		Value: true,
		EvaluationDetails: openfeature.EvaluationDetails{
			FlagKey:  "flagA",
			FlagType: openfeature.Boolean,
			ResolutionDetail: openfeature.ResolutionDetail{
				FlagMetadata: map[string]interface{}{
					scopeKey:  scopeValue,
					stageKey:  stageValue,
					scoreKey:  scoreValue,
					cachedKey: cachedValue,
				},
			},
		},
	}
	hookHints := openfeature.NewHookHints(map[string]interface{}{})

	// when
	err = metricsHook.After(ctx, hookContext(), evalDetails, hookHints)
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
		t.Errorf("expected metric, %s to be present with after hook", metrics)
	}

	successMetric := metrics[0]

	if successMetric.Name != evaluationSuccess {
		t.Errorf("expected %s to be present with after hook", evaluationSuccess)
	}

	instrument := successMetric.Data.(metricdata.Sum[int64])

	if len(instrument.DataPoints) < 1 {
		t.Error("expected data points, but found none")
	}

	attributes := instrument.DataPoints[0].Attributes

	value, ok := attributes.Value(attribute.Key(scopeKey))
	if !ok || value.AsString() != scopeValue {
		t.Errorf("attribute %s is incorrectly configured", scopeKey)
	}

	value, ok = attributes.Value(attribute.Key(stageKey))
	if !ok || value.AsInt64() != int64(stageValue) {
		t.Errorf("attribute %s is incorrectly configured", stageKey)
	}

	value, ok = attributes.Value(attribute.Key(scoreKey))
	if !ok || value.AsFloat64() != scoreValue {
		t.Errorf("attribute %s is incorrectly configured", scoreKey)
	}

	value, ok = attributes.Value(attribute.Key(cachedKey))
	if !ok || value.AsBool() != cachedValue {
		t.Errorf("attribute %s is incorrectly configured", cachedKey)
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
