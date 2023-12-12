package otel

import (
	"context"
	"errors"
	"go.opentelemetry.io/otel/codes"
	"testing"

	"github.com/open-feature/go-sdk/openfeature"
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
		hook := NewTracesHook()
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
		if spans[0].Events[0].Name != EventName {
			t.Errorf("unexpected event name: %s", spans[0].Events[0].Name)
		}
		for _, attr := range spans[0].Events[0].Attributes {
			switch attr.Key {
			case EventPropertyFlagKey:
				if attr.Value.AsString() != flagKey {
					t.Errorf("unexpected feature_flag.key attribute value: %s", attr.Value.AsString())
				}
			case EventPropertyProviderName:
				if attr.Value.AsString() != providerName {
					t.Errorf("unexpected feature_flag.provider_name attribute value: %s", attr.Value.AsString())
				}
			case EventPropertyVariant:
				if attr.Value.AsString() != variant {
					t.Errorf("unexpected feature_flag.variant attribute value: %s", attr.Value.AsString())
				}
			default:
				t.Errorf("unexpected attribute key: %s", attr.Key)
			}
		}
	})

	t.Run("Error hook should record exception on span & avoid setting span status", func(t *testing.T) {
		exp := tracetest.NewInMemoryExporter()
		tp := trace.NewTracerProvider(
			trace.WithSyncer(exp),
		)
		otel.SetTracerProvider(tp)
		ctx, span := otel.Tracer("test-tracer").Start(context.Background(), "Run")
		hook := NewTracesHook()
		err := errors.New("a terrible error")
		hook.Error(ctx, openfeature.HookContext{}, err, openfeature.HookHints{})
		span.End()

		spans := exp.GetSpans()
		if len(spans) != 1 {
			t.Errorf("expected 1 span, got %d", len(spans))
		}

		errSpan := spans[0]

		// check for codes.Unset - default span status
		if errSpan.Status.Code != codes.Unset {
			t.Errorf("expected status %s, got %s", codes.Unset.String(), errSpan.Status.Code.String())
		}
		if len(errSpan.Events) != 1 {
			t.Errorf("expected 1 event, got %d", len(errSpan.Events))
		}
		if errSpan.Events[0].Name != semconv.ExceptionEventName {
			t.Errorf("unexpected event name: %s", errSpan.Events[0].Name)
		}
	})

	t.Run("Error hook should set span status if build option is provided", func(t *testing.T) {
		exp := tracetest.NewInMemoryExporter()
		tp := trace.NewTracerProvider(
			trace.WithSyncer(exp),
		)
		otel.SetTracerProvider(tp)
		ctx, span := otel.Tracer("test-tracer").Start(context.Background(), "Run")

		// build traceHook with option WithErrorStatusEnabled
		hook := NewTracesHook(WithErrorStatusEnabled())

		err := errors.New("a terrible error")
		hook.Error(ctx, openfeature.HookContext{}, err, openfeature.HookHints{})
		span.End()

		spans := exp.GetSpans()
		if len(spans) != 1 {
			t.Errorf("expected 1 span, got %d", len(spans))
		}

		errSpan := spans[0]

		if errSpan.Status.Code != codes.Error {
			t.Errorf("expected status %s, got %s", codes.Error.String(), errSpan.Status.Code.String())
		}
	})

	t.Run("a nil context should not cause a panic", func(t *testing.T) {
		flagKey := "flag-key"
		providerName := "provider-name"
		variant := "variant"
		ctx := context.Background()
		hook := NewTracesHook()

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

func TestTracesHook_MedataExtractionOption(t *testing.T) {
	// given
	exp := tracetest.NewInMemoryExporter()
	tp := trace.NewTracerProvider(
		trace.WithSyncer(exp),
	)
	otel.SetTracerProvider(tp)

	evaluationDetails := openfeature.InterfaceEvaluationDetails{
		Value: "ok",
		EvaluationDetails: openfeature.EvaluationDetails{
			FlagKey:  "stringFlag",
			FlagType: openfeature.String,
			ResolutionDetail: openfeature.ResolutionDetail{
				FlagMetadata: evalMetadata,
			},
		},
	}

	hook := NewTracesHook(WithTracesAttributeSetter(extractionCallback))

	// when
	ctx, span := otel.Tracer("test-tracer").Start(context.Background(), "Run")
	err := hook.After(ctx, openfeature.HookContext{}, evaluationDetails, openfeature.HookHints{})
	if err != nil {
		t.Fatal(err)
	}
	span.End()

	// then
	spans := exp.GetSpans()
	if len(spans) != 1 {
		t.Errorf("expected 1 span, got %d", len(spans))
	}

	if len(spans[0].Events) != 1 {
		t.Errorf("expected 1 event, got %d", len(spans[0].Events))
	}

	if spans[0].Events[0].Name != EventName {
		t.Errorf("unexpected event name: %s", spans[0].Events[0].Name)
	}

	attributes := spans[0].Events[0].Attributes

	for _, attribute := range attributes {
		switch string(attribute.Key) {
		case scopeKey:
			if attribute.Value.AsString() != scopeValue {
				t.Errorf("expected %s, got type %s", scopeValue, attribute.Value.Type().String())
			}
		case stageKey:
			if attribute.Value.AsInt64() != int64(stageValue) {
				t.Errorf("expected %d, got type %s", stageValue, attribute.Value.Type().String())
			}
		case scoreKey:
			if attribute.Value.AsFloat64() != scoreValue {
				t.Errorf("expected %f, got type %s", scoreValue, attribute.Value.Type().String())
			}
		case cachedKey:
			if attribute.Value.AsBool() != cacheValue {
				t.Errorf("expected %t, got type %s", cacheValue, attribute.Value.Type().String())
			}
		}
	}
}
