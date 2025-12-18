package otel

import (
	"context"
	"maps"
	"slices"
	"testing"

	"go.openfeature.dev/openfeature/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
)

func TestTracesHook_Finally(t *testing.T) {
	tests := []struct {
		name           string
		flagKey        string
		providerName   string
		variant        string
		targetingKey   string
		reason         openfeature.Reason
		hasSpan        bool
		ctx            context.Context
		expectedAttrs  map[attribute.Key]string
		shouldNotPanic bool
	}{
		{
			name:         "should_add_feature_flag_event_and_attributes",
			flagKey:      "flag-key",
			providerName: "provider-name",
			variant:      "variant",
			targetingKey: "test-targeting-key",
			reason:       openfeature.TargetingMatchReason,
			hasSpan:      true,
			ctx:          t.Context(),
			expectedAttrs: map[attribute.Key]string{
				semconv.FeatureFlagKeyKey:           "flag-key",
				semconv.FeatureFlagProviderNameKey:  "provider-name",
				semconv.FeatureFlagResultVariantKey: "variant",
				semconv.FeatureFlagResultReasonKey:  "targeting_match",
				semconv.FeatureFlagContextIDKey:     "test-targeting-key",
			},
		},
		{
			name:           "nil_context_should_not_panic",
			flagKey:        "flag-key",
			providerName:   "provider-name",
			variant:        "variant",
			targetingKey:   "test-targeting-key",
			hasSpan:        false,
			ctx:            nil,
			shouldNotPanic: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exp := tracetest.NewInMemoryExporter()
			tp := trace.NewTracerProvider(trace.WithSyncer(exp))
			otel.SetTracerProvider(tp)

			ctx, span := otel.Tracer("test-tracer").Start(tt.ctx, "Run")

			hook := NewTracesHook()

			hook.Finally(
				ctx,
				openfeature.NewHookContext(
					tt.flagKey,
					openfeature.String,
					"default",
					openfeature.ClientMetadata{},
					openfeature.Metadata{Name: tt.providerName},
					openfeature.NewEvaluationContext(tt.targetingKey, map[string]any{"this": "that"}),
				),
				openfeature.EvaluationDetails[openfeature.FlagTypes]{
					ResolutionDetail: openfeature.ResolutionDetail{
						Variant: tt.variant,
						Reason:  tt.reason,
					},
				},
				openfeature.NewHookHints(nil),
			)

			if tt.hasSpan {
				span.End()
			}

			if tt.shouldNotPanic {
				return
			}

			attrs := extractAttributes(t, exp)
			m := maps.Collect(func(yield func(attribute.Key, string) bool) {
				for _, kv := range attrs {
					if !yield(kv.Key, kv.Value.Emit()) {
						return
					}
				}
			})

			for expKey, expValue := range tt.expectedAttrs {
				val, ok := m[expKey]
				if !ok {
					t.Errorf("missing %s attribute", expKey)
				}
				if val != expValue {
					t.Errorf("unexpected %s attribute value: want %s, got: %s", expKey, expValue, val)
				}
			}
		})
	}
}

func TestTracesHook_MetadataExtractionOption(t *testing.T) {
	tests := []struct {
		name  string
		value any
	}{
		{
			name:  "metadata_extraction_with_custom_callback",
			value: "ok",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exp := tracetest.NewInMemoryExporter()
			tp := trace.NewTracerProvider(trace.WithSyncer(exp))
			otel.SetTracerProvider(tp)

			ctx, span := otel.Tracer("test-tracer").Start(t.Context(), "Run")
			hook := NewTracesHook(WithTracesAttributeSetter(extractionCallback))

			hook.Finally(
				ctx,
				openfeature.HookContext{},
				openfeature.EvaluationDetails[openfeature.FlagTypes]{
					Value:    tt.value,
					FlagKey:  "stringFlag",
					FlagType: openfeature.String,
					ResolutionDetail: openfeature.ResolutionDetail{
						FlagMetadata: evalMetadata,
					},
				},
				openfeature.HookHints{},
			)
			span.End()

			attrs := extractAttributes(t, exp)
			for _, attribute := range attrs {
				switch string(attribute.Key) {
				case scopeKey:
					if attribute.Value.AsString() != scopeValue {
						t.Errorf("want %s, got type %s", scopeValue, attribute.Value.Type().String())
					}
				case stageKey:
					if attribute.Value.AsInt64() != int64(stageValue) {
						t.Errorf("want %d, got type %s", stageValue, attribute.Value.Type().String())
					}
				case scoreKey:
					if attribute.Value.AsFloat64() != scoreValue {
						t.Errorf("want %f, got type %s", scoreValue, attribute.Value.Type().String())
					}
				case cachedKey:
					if attribute.Value.AsBool() != cacheValue {
						t.Errorf("want %t, got type %s", cacheValue, attribute.Value.Type().String())
					}
				}
			}
		})
	}
}

func TestTracesHook_ResultValueTypes(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		expected string
	}{
		{"bool_true", true, "true"},
		{"bool_false", false, "false"},
		{"string", "test-value", "test-value"},
		{"int64", int64(42), "42"},
		{"float64", float64(3.14), "3.14"},
		{"int", int(99), "99"},
		{"float32", float32(2.5), "2.5"},
		{"bool_slice", []bool{true, false, true}, "[true false true]"},
		{"string_slice", []string{"a", "b", "c"}, `["a","b","c"]`},
		{"int64_slice", []int64{1, 2, 3}, "[1,2,3]"},
		{"float64_slice", []float64{1.1, 2.2}, "[1.1,2.2]"},
		{"int_slice", []int{10, 20}, "[10,20]"},
		{"float32_slice", []float32{1.5, 2.5}, "[1.5,2.5]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exp := tracetest.NewInMemoryExporter()
			tp := trace.NewTracerProvider(trace.WithSyncer(exp))
			otel.SetTracerProvider(tp)

			ctx, span := otel.Tracer("test-tracer").Start(t.Context(), "Run")
			hook := NewTracesHook()

			hook.Finally(
				ctx,
				openfeature.NewHookContext(
					"test-flag",
					openfeature.Object,
					nil,
					openfeature.ClientMetadata{},
					openfeature.Metadata{Name: "test-provider"},
					openfeature.NewEvaluationContext("", nil),
				),
				openfeature.EvaluationDetails[openfeature.FlagTypes]{
					Value: tt.value,
					ResolutionDetail: openfeature.ResolutionDetail{
						Reason: openfeature.StaticReason,
					},
				},
				openfeature.NewHookHints(nil),
			)
			span.End()

			attrs := extractAttributes(t, exp)
			found := slices.ContainsFunc(attrs, func(attr attribute.KeyValue) bool {
				return attr.Key == semconv.FeatureFlagResultValueKey && attr.Value.Emit() == tt.expected
			})
			if !found {
				t.Error("expected feature_flag.result.value attribute to be set")
			}
		})
	}
}

func TestTracesHook_ReasonTypes(t *testing.T) {
	tests := []struct {
		name   string
		reason openfeature.Reason
	}{
		{"cached", openfeature.CachedReason},
		{"default", openfeature.DefaultReason},
		{"disabled", openfeature.DisabledReason},
		{"error", openfeature.ErrorReason},
		{"split", openfeature.SplitReason},
		{"static", openfeature.StaticReason},
		{"targeting_match", openfeature.TargetingMatchReason},
		{"unknown", openfeature.Reason("custom")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exp := tracetest.NewInMemoryExporter()
			tp := trace.NewTracerProvider(trace.WithSyncer(exp))
			otel.SetTracerProvider(tp)

			ctx, span := otel.Tracer("test-tracer").Start(t.Context(), "Run")
			hook := NewTracesHook()

			evaluationDetails := openfeature.EvaluationDetails[openfeature.FlagTypes]{
				Value: "test",
				ResolutionDetail: openfeature.ResolutionDetail{
					Reason: tt.reason,
				},
			}

			if tt.reason == openfeature.ErrorReason {
				evaluationDetails.ErrorCode = openfeature.FlagNotFoundCode
				evaluationDetails.ErrorMessage = "some error message"
			}

			hook.Finally(
				ctx,
				openfeature.NewHookContext(
					"test-flag",
					openfeature.String,
					"default",
					openfeature.ClientMetadata{},
					openfeature.Metadata{Name: "test-provider"},
					openfeature.NewEvaluationContext("", nil),
				),
				evaluationDetails,
				openfeature.NewHookHints(nil),
			)
			span.End()

			attrs := extractAttributes(t, exp)
			found := slices.ContainsFunc(attrs, func(attr attribute.KeyValue) bool {
				return attr.Key == semconv.FeatureFlagResultReasonKey && tt.name == attr.Value.AsString()
			})
			if !found {
				t.Error("expected feature_flag.result.reason attribute to be set")
			}
			if tt.reason == openfeature.ErrorReason {
				found = slices.ContainsFunc(attrs, func(attr attribute.KeyValue) bool {
					return attr.Key == semconv.ErrorTypeKey && attr.Value.AsString() == "flag_not_found"
				})
				if !found {
					t.Error("expected error_type attribute to be set")
				}
				found = slices.ContainsFunc(attrs, func(attr attribute.KeyValue) bool {
					return attr.Key == semconv.ErrorMessageKey && attr.Value.AsString() == "some error message"
				})
				if !found {
					t.Error("expected error_message attribute to be set")
				}
			}
		})
	}
}

func TestTracesHook_FlagMetadata(t *testing.T) {
	tests := []struct {
		name            string
		metadataKey     string
		metadataValue   string
		targetingKey    string
		expectedAttrKey attribute.Key
		errorMessage    string
	}{
		{
			name:            "context_id_from_metadata",
			metadataKey:     "contextId",
			metadataValue:   "custom-context-123",
			targetingKey:    "targeting-key",
			expectedAttrKey: semconv.FeatureFlagContextIDKey,
			errorMessage:    "expected feature_flag.context.id attribute",
		},
		{
			name:            "flag_set_id_from_metadata",
			metadataKey:     "flagSetId",
			metadataValue:   "set-123",
			targetingKey:    "",
			expectedAttrKey: semconv.FeatureFlagSetIDKey,
			errorMessage:    "expected feature_flag.set.id attribute",
		},
		{
			name:            "version_from_metadata",
			metadataKey:     "version",
			metadataValue:   "v1.2.3",
			targetingKey:    "",
			expectedAttrKey: semconv.FeatureFlagVersionKey,
			errorMessage:    "expected feature_flag.version attribute",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exp := tracetest.NewInMemoryExporter()
			tp := trace.NewTracerProvider(trace.WithSyncer(exp))
			otel.SetTracerProvider(tp)

			ctx, span := otel.Tracer("test-tracer").Start(t.Context(), "Run")
			hook := NewTracesHook()

			hook.Finally(
				ctx,
				openfeature.NewHookContext(
					"test-flag",
					openfeature.String,
					"default",
					openfeature.ClientMetadata{},
					openfeature.Metadata{Name: "test-provider"},
					openfeature.NewEvaluationContext(tt.targetingKey, nil),
				),
				openfeature.EvaluationDetails[openfeature.FlagTypes]{
					Value: "test",
					ResolutionDetail: openfeature.ResolutionDetail{
						FlagMetadata: map[string]any{
							tt.metadataKey: tt.metadataValue,
						},
					},
				},
				openfeature.NewHookHints(nil),
			)
			span.End()

			attrs := extractAttributes(t, exp)
			found := slices.ContainsFunc(attrs, func(attr attribute.KeyValue) bool {
				return attr.Key == tt.expectedAttrKey && attr.Value.Emit() == tt.metadataValue
			})
			if !found {
				t.Error(tt.errorMessage)
			}
		})
	}
}

func extractAttributes(t *testing.T, exp *tracetest.InMemoryExporter) []attribute.KeyValue {
	t.Helper()
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
	return spans[0].Events[0].Attributes
}
