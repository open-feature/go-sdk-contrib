package otel_test

import (
	"context"
	"errors"
	"testing"

	otelHook "github.com/open-feature/go-sdk-contrib/hooks/open-telemetry/pkg"
	"github.com/open-feature/go-sdk/pkg/openfeature"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
)

func TestHookMethods(t *testing.T) {

	t.Run("After hook should add feature_flag event as well as key, provider_name and variant attributes", func(t *testing.T) {
		flagKey := "flag-key"
		providerName := "provider-name"
		variant := "variant"
		exp := tracetest.NewInMemoryExporter()
		tp := trace.NewTracerProvider(
			trace.WithSyncer(exp),
		)
		otel.SetTracerProvider(tp)
		ctx, span := otel.Tracer("test-tracer").Start(context.Background(), "Run")
		hook := otelHook.NewHook()
		err := hook.After(
			ctx,
			openfeature.NewHookContext(
				flagKey,
				openfeature.String,
				"default",
				openfeature.ClientMetadata{},
				openfeature.Metadata{
					Name: providerName,
				},
				openfeature.NewEvaluationContext(
					"test-targeting-key",
					map[string]interface{}{
						"this": "that",
					},
				),
			),
			openfeature.InterfaceEvaluationDetails{
				EvaluationDetails: openfeature.EvaluationDetails{
					ResolutionDetail: openfeature.ResolutionDetail{
						Variant: variant,
					},
				},
			},
			openfeature.NewHookHints(
				map[string]interface{}{},
			),
		)
		if err != nil {
			t.Error(err)
		}
		span.End()
		spans := exp.GetSpans()
		if len(spans) != 1 {
			t.Errorf("expected 1 span, got %d", len(spans))
		}
		if len(spans[0].Events) != 1 {
			t.Errorf("expected 1 event, got %d", len(spans[0].Events))
		}
		if spans[0].Events[0].Name != otelHook.EventName {
			t.Errorf("unexpected event name: %s", spans[0].Events[0].Name)
		}
		for _, attr := range spans[0].Events[0].Attributes {
			switch attr.Key {
			case otelHook.EventPropertyFlagKey:
				if attr.Value.AsString() != flagKey {
					t.Errorf("unexpected feature_flag.key attribute value: %s", attr.Value.AsString())
				}
			case otelHook.EventPropertyProviderName:
				if attr.Value.AsString() != providerName {
					t.Errorf("unexpected feature_flag.provider_name attribute value: %s", attr.Value.AsString())
				}
			case otelHook.EventPropertyVariant:
				if attr.Value.AsString() != variant {
					t.Errorf("unexpected feature_flag.variant attribute value: %s", attr.Value.AsString())
				}
			default:
				t.Errorf("unexpected attribute key: %s", attr.Key)
			}
		}
	})

	t.Run("Error hook should record exception on span", func(t *testing.T) {
		exp := tracetest.NewInMemoryExporter()
		tp := trace.NewTracerProvider(
			trace.WithSyncer(exp),
		)
		otel.SetTracerProvider(tp)
		ctx, span := otel.Tracer("test-tracer").Start(context.Background(), "Run")
		hook := otelHook.NewHook()
		err := errors.New("a terrible error")
		hook.Error(ctx, openfeature.HookContext{}, err, openfeature.HookHints{})
		span.End()

		spans := exp.GetSpans()
		if len(spans) != 1 {
			t.Errorf("expected 1 span, got %d", len(spans))
		}
		if len(spans[0].Events) != 1 {
			t.Errorf("expected 1 event, got %d", len(spans[0].Events))
		}
		if spans[0].Events[0].Name != semconv.ExceptionEventName {
			t.Errorf("unexpected event name: %s", spans[0].Events[0].Name)
		}
	})

	t.Run("a nil context should not cause a panic", func(t *testing.T) {
		flagKey := "flag-key"
		providerName := "provider-name"
		variant := "variant"
		ctx := context.Background()
		hook := otelHook.NewHook()

		err := hook.After(
			ctx,
			openfeature.NewHookContext(
				flagKey,
				openfeature.String,
				"default",
				openfeature.ClientMetadata{},
				openfeature.Metadata{
					Name: providerName,
				},
				openfeature.NewEvaluationContext(
					"test-targeting-key",
					map[string]interface{}{
						"this": "that",
					},
				),
			),
			openfeature.InterfaceEvaluationDetails{
				EvaluationDetails: openfeature.EvaluationDetails{
					ResolutionDetail: openfeature.ResolutionDetail{
						Variant: variant,
					},
				},
			},
			openfeature.NewHookHints(
				map[string]interface{}{},
			),
		)
		if err != nil {
			t.Error(err)
		}

		hook.Error(ctx, openfeature.HookContext{}, err, openfeature.HookHints{})
	})

}
