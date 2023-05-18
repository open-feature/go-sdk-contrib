package otel

import (
	"context"
	"github.com/open-feature/go-sdk/pkg/openfeature"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"testing"
)

func TestMetricsHook_Before(t *testing.T) {
	manualReader := metric.NewManualReader()
	metricsHook, err := NewMetricsHook(manualReader)
	if err != nil {
		t.Error(err)
		return
	}

	ctx := context.Background()

	hookContext := openfeature.NewHookContext("flagA",
		openfeature.Boolean,
		false, openfeature.NewClientMetadata(""),
		openfeature.Metadata{
			Name: "flagd",
		},
		openfeature.EvaluationContext{},
	)

	hookHints := openfeature.NewHookHints(map[string]interface{}{})

	_, err = metricsHook.Before(ctx, hookContext, hookHints)
	if err != nil {
		t.Error(err)
		return
	}

	var data metricdata.ResourceMetrics

	err = manualReader.Collect(ctx, &data)
	if err != nil {
		t.Error(err)
		return
	}
}
