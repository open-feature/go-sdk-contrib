package flipt

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	of "github.com/open-feature/go-sdk/openfeature"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.flipt.io/flipt/rpc/flipt/evaluation"
)

func TestMetadata(t *testing.T) {
	p := NewProvider()
	assert.Equal(t, "flipt-provider", p.Metadata().Name)
}

func TestBooleanEvaluation(t *testing.T) {
	tests := []struct {
		name                  string
		flagKey               string
		defaultValue          bool
		mockRespEvaluation    *evaluation.BooleanEvaluationResponse
		mockRespEvaluationErr error
		expected              of.BoolResolutionDetail
	}{
		{
			name:         "false",
			flagKey:      "boolean-false",
			defaultValue: true,
			mockRespEvaluation: &evaluation.BooleanEvaluationResponse{
				Enabled: false,
				Reason:  evaluation.EvaluationReason_MATCH_EVALUATION_REASON,
			},
			expected: of.BoolResolutionDetail{Value: false, ProviderResolutionDetail: of.ProviderResolutionDetail{Reason: of.TargetingMatchReason}},
		},
		{
			name:                  "resolution error",
			flagKey:               "boolean-res-error",
			defaultValue:          false,
			mockRespEvaluationErr: of.NewInvalidContextResolutionError("boom"),
			expected: of.BoolResolutionDetail{
				Value: false,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					Reason:          of.DefaultReason,
					ResolutionError: of.NewInvalidContextResolutionError("boom"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := newMockService(t)
			mockSvc.On("Boolean", mock.Anything, "flipt", tt.flagKey, mock.Anything).Return(tt.mockRespEvaluation, tt.mockRespEvaluationErr).Maybe()

			p := NewProvider(WithService(mockSvc), ForNamespace("flipt"))

			actual := p.BooleanEvaluation(context.Background(), tt.flagKey, tt.defaultValue, map[string]any{})

			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestStringEvaluation(t *testing.T) {
	tests := []struct {
		name                  string
		flagKey               string
		defaultValue          string
		mockRespEvaluation    *evaluation.VariantEvaluationResponse
		mockRespEvaluationErr error
		expected              of.StringResolutionDetail
	}{
		{
			name:         "flag enabled",
			flagKey:      "string-true",
			defaultValue: "false",
			mockRespEvaluation: &evaluation.VariantEvaluationResponse{
				Match:      true,
				VariantKey: "true",
			},
			expected: of.StringResolutionDetail{Value: "true", ProviderResolutionDetail: of.ProviderResolutionDetail{Reason: of.TargetingMatchReason}},
		},
		{
			name:         "flag disabled",
			flagKey:      "string-true",
			defaultValue: "false",
			mockRespEvaluation: &evaluation.VariantEvaluationResponse{
				Match:  false,
				Reason: evaluation.EvaluationReason_FLAG_DISABLED_EVALUATION_REASON,
			},
			expected: of.StringResolutionDetail{Value: "false", ProviderResolutionDetail: of.ProviderResolutionDetail{Reason: of.DisabledReason}},
		},
		{
			name:                  "resolution error",
			flagKey:               "string-res-error",
			defaultValue:          "true",
			mockRespEvaluationErr: of.NewInvalidContextResolutionError("boom"),
			expected: of.StringResolutionDetail{
				Value: "true",
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					Reason:          of.DefaultReason,
					ResolutionError: of.NewInvalidContextResolutionError("boom"),
				},
			},
		},
		{
			name:                  "error",
			flagKey:               "string-error",
			defaultValue:          "true",
			mockRespEvaluationErr: errors.New("boom"),
			expected: of.StringResolutionDetail{
				Value: "true",
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					Reason:          of.DefaultReason,
					ResolutionError: of.NewGeneralResolutionError("boom"),
				},
			},
		},
		{
			name:    "no match",
			flagKey: "string-no-match",

			defaultValue: "default",
			mockRespEvaluation: &evaluation.VariantEvaluationResponse{
				Match: false,
			},
			expected: of.StringResolutionDetail{Value: "default", ProviderResolutionDetail: of.ProviderResolutionDetail{Reason: of.DefaultReason}},
		},
		{
			name:    "match",
			flagKey: "string-match",

			defaultValue: "default",
			mockRespEvaluation: &evaluation.VariantEvaluationResponse{
				Match:      true,
				VariantKey: "abc",
			},
			expected: of.StringResolutionDetail{
				Value: "abc",
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					Reason: of.TargetingMatchReason,
				},
			},
		},
		{
			name:         "match",
			flagKey:      "string-match",
			defaultValue: "default",
			mockRespEvaluation: &evaluation.VariantEvaluationResponse{
				Match:      true,
				VariantKey: "abc",
			},
			expected: of.StringResolutionDetail{
				Value: "abc",
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					Reason: of.TargetingMatchReason,
				},
			},
		},
		{
			name:         "flipt-default",
			flagKey:      "string-match",
			defaultValue: "default",
			mockRespEvaluation: &evaluation.VariantEvaluationResponse{
				Match:      false,
				Reason:     evaluation.EvaluationReason_DEFAULT_EVALUATION_REASON,
				VariantKey: "abc",
			},
			expected: of.StringResolutionDetail{
				Value: "abc",
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					Reason: of.DefaultReason,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := newMockService(t)
			mockSvc.On("Evaluate", mock.Anything, "default", tt.flagKey, mock.Anything).Return(tt.mockRespEvaluation, tt.mockRespEvaluationErr).Maybe()

			p := NewProvider(WithService(mockSvc))

			actual := p.StringEvaluation(context.Background(), tt.flagKey, tt.defaultValue, map[string]any{})

			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestFloatEvaluation(t *testing.T) {
	tests := []struct {
		name                  string
		flagKey               string
		defaultValue          float64
		mockRespEvaluation    *evaluation.VariantEvaluationResponse
		mockRespEvaluationErr error
		expected              of.FloatResolutionDetail
	}{
		{
			name:    "flag enabled",
			flagKey: "float-one",

			defaultValue: 1.0,
			mockRespEvaluation: &evaluation.VariantEvaluationResponse{
				Match:      true,
				VariantKey: "1.0",
			},
			expected: of.FloatResolutionDetail{Value: 1.0, ProviderResolutionDetail: of.ProviderResolutionDetail{Reason: of.TargetingMatchReason}},
		},
		{
			name:    "flag disabled",
			flagKey: "float-zero",

			defaultValue: 0.0,
			mockRespEvaluation: &evaluation.VariantEvaluationResponse{
				Match:  false,
				Reason: evaluation.EvaluationReason_FLAG_DISABLED_EVALUATION_REASON,
			},
			expected: of.FloatResolutionDetail{Value: 0.0, ProviderResolutionDetail: of.ProviderResolutionDetail{Reason: of.DisabledReason}},
		},
		{
			name:                  "resolution error",
			flagKey:               "float-res-error",
			defaultValue:          0.0,
			mockRespEvaluationErr: of.NewInvalidContextResolutionError("boom"),
			expected: of.FloatResolutionDetail{
				Value: 0.0,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					Reason:          of.DefaultReason,
					ResolutionError: of.NewInvalidContextResolutionError("boom"),
				},
			},
		},
		{
			name:    "parse error",
			flagKey: "float-parse-error",

			defaultValue: 1.0,
			mockRespEvaluation: &evaluation.VariantEvaluationResponse{
				Match:      true,
				VariantKey: "not-a-float",
			},
			expected: of.FloatResolutionDetail{
				Value: 1.0,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					Reason:          of.ErrorReason,
					ResolutionError: of.NewTypeMismatchResolutionError("value is not a float"),
				},
			},
		},
		{
			name:                  "error",
			flagKey:               "float-error",
			defaultValue:          1.0,
			mockRespEvaluationErr: errors.New("boom"),
			expected: of.FloatResolutionDetail{
				Value: 1.0,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					Reason:          of.DefaultReason,
					ResolutionError: of.NewGeneralResolutionError("boom"),
				},
			},
		},
		{
			name:    "no match",
			flagKey: "float-no-match",

			defaultValue: 1.0,
			mockRespEvaluation: &evaluation.VariantEvaluationResponse{
				Match: false,
			},
			expected: of.FloatResolutionDetail{Value: 1.0, ProviderResolutionDetail: of.ProviderResolutionDetail{Reason: of.DefaultReason}},
		},
		{
			name:    "match",
			flagKey: "float-match",

			defaultValue: 1.0,
			mockRespEvaluation: &evaluation.VariantEvaluationResponse{
				Match:      true,
				VariantKey: "2.0",
			},
			expected: of.FloatResolutionDetail{
				Value: 2.0,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					Reason: of.TargetingMatchReason,
				},
			},
		},
		{
			name:    "flipt-default",
			flagKey: "float-default",

			defaultValue: 1.0,
			mockRespEvaluation: &evaluation.VariantEvaluationResponse{
				Match:      false,
				Reason:     evaluation.EvaluationReason_DEFAULT_EVALUATION_REASON,
				VariantKey: "2.0",
			},
			expected: of.FloatResolutionDetail{
				Value: 2.0,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					Reason: of.DefaultReason,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := newMockService(t)
			mockSvc.On("Evaluate", mock.Anything, "flipt", tt.flagKey, mock.Anything).Return(tt.mockRespEvaluation, tt.mockRespEvaluationErr).Maybe()

			p := NewProvider(WithService(mockSvc), ForNamespace("flipt"))

			actual := p.FloatEvaluation(context.Background(), tt.flagKey, tt.defaultValue, map[string]any{})
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestIntEvaluation(t *testing.T) {
	tests := []struct {
		name                  string
		flagKey               string
		defaultValue          int64
		mockRespEvaluation    *evaluation.VariantEvaluationResponse
		mockRespEvaluationErr error
		expected              of.IntResolutionDetail
	}{
		{
			name:    "flag enabled",
			flagKey: "int-one",

			defaultValue: 1,
			mockRespEvaluation: &evaluation.VariantEvaluationResponse{
				Match:      true,
				VariantKey: "1",
			},
			expected: of.IntResolutionDetail{Value: 1, ProviderResolutionDetail: of.ProviderResolutionDetail{Reason: of.TargetingMatchReason}},
		},
		{
			name:    "flag disabled",
			flagKey: "int-zero",

			defaultValue: 0,
			mockRespEvaluation: &evaluation.VariantEvaluationResponse{
				Match:  false,
				Reason: evaluation.EvaluationReason_FLAG_DISABLED_EVALUATION_REASON,
			},
			expected: of.IntResolutionDetail{Value: 0, ProviderResolutionDetail: of.ProviderResolutionDetail{Reason: of.DisabledReason}},
		},
		{
			name:                  "resolution error",
			flagKey:               "int-res-error",
			defaultValue:          0,
			mockRespEvaluationErr: of.NewInvalidContextResolutionError("boom"),
			expected: of.IntResolutionDetail{
				Value: 0,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					Reason:          of.DefaultReason,
					ResolutionError: of.NewInvalidContextResolutionError("boom"),
				},
			},
		},
		{
			name:    "parse error",
			flagKey: "int-parse-error",

			defaultValue: 1,
			mockRespEvaluation: &evaluation.VariantEvaluationResponse{
				Match:      true,
				VariantKey: "not-an-int",
			},
			expected: of.IntResolutionDetail{
				Value: 1,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					Reason:          of.ErrorReason,
					ResolutionError: of.NewTypeMismatchResolutionError("value is not an integer"),
				},
			},
		},
		{
			name:                  "error",
			flagKey:               "int-error",
			defaultValue:          1,
			mockRespEvaluationErr: errors.New("boom"),
			expected: of.IntResolutionDetail{
				Value: 1,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					Reason:          of.DefaultReason,
					ResolutionError: of.NewGeneralResolutionError("boom"),
				},
			},
		},
		{
			name:    "no match",
			flagKey: "int-no-match",

			defaultValue: 1,
			mockRespEvaluation: &evaluation.VariantEvaluationResponse{
				Match: false,
			},
			expected: of.IntResolutionDetail{Value: 1, ProviderResolutionDetail: of.ProviderResolutionDetail{Reason: of.DefaultReason}},
		},
		{
			name:    "match",
			flagKey: "int-match",

			defaultValue: 1,
			mockRespEvaluation: &evaluation.VariantEvaluationResponse{
				Match:      true,
				VariantKey: "2",
			},
			expected: of.IntResolutionDetail{
				Value: 2,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					Reason: of.TargetingMatchReason,
				},
			},
		},
		{
			name:         "match",
			flagKey:      "int-match",
			defaultValue: 1,
			mockRespEvaluation: &evaluation.VariantEvaluationResponse{
				Match:      true,
				VariantKey: "2",
			},
			expected: of.IntResolutionDetail{
				Value: 2,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					Reason: of.TargetingMatchReason,
				},
			},
		},
		{
			name:         "flipt-default",
			flagKey:      "int-match",
			defaultValue: 1,
			mockRespEvaluation: &evaluation.VariantEvaluationResponse{
				Match:      false,
				Reason:     evaluation.EvaluationReason_DEFAULT_EVALUATION_REASON,
				VariantKey: "2",
			},
			expected: of.IntResolutionDetail{
				Value: 2,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					Reason: of.DefaultReason,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := newMockService(t)
			mockSvc.On("Evaluate", mock.Anything, "default", tt.flagKey, mock.Anything).Return(tt.mockRespEvaluation, tt.mockRespEvaluationErr).Maybe()

			p := NewProvider(WithService(mockSvc))

			actual := p.IntEvaluation(context.Background(), tt.flagKey, tt.defaultValue, map[string]any{})

			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestObjectEvaluation(t *testing.T) {
	attachment := map[string]any{
		"foo": "bar",
	}

	b, _ := json.Marshal(attachment)
	attachmentJSON := string(b)

	tests := []struct {
		name                  string
		flagKey               string
		defaultValue          map[string]any
		mockRespEvaluation    *evaluation.VariantEvaluationResponse
		mockRespEvaluationErr error
		expected              of.InterfaceResolutionDetail
	}{
		{
			name:    "flag enabled",
			flagKey: "obj-enabled",

			defaultValue: map[string]any{
				"baz": "qux",
			},
			mockRespEvaluation: &evaluation.VariantEvaluationResponse{
				Match:             true,
				VariantAttachment: attachmentJSON,
			},
			expected: of.InterfaceResolutionDetail{
				Value:                    attachment,
				ProviderResolutionDetail: of.ProviderResolutionDetail{Reason: of.TargetingMatchReason},
			},
		},
		{
			name:    "flag disabled",
			flagKey: "obj-disabled",

			defaultValue: map[string]any{
				"baz": "qux",
			},
			mockRespEvaluation: &evaluation.VariantEvaluationResponse{
				Match:  false,
				Reason: evaluation.EvaluationReason_FLAG_DISABLED_EVALUATION_REASON,
			},
			expected: of.InterfaceResolutionDetail{
				Value: map[string]any{
					"baz": "qux",
				},
				ProviderResolutionDetail: of.ProviderResolutionDetail{Reason: of.DisabledReason},
			},
		},
		{
			name:    "resolution error",
			flagKey: "obj-res-error",

			defaultValue: map[string]any{
				"baz": "qux",
			},
			mockRespEvaluationErr: of.NewInvalidContextResolutionError("boom"),
			expected: of.InterfaceResolutionDetail{
				Value: map[string]any{
					"baz": "qux",
				}, ProviderResolutionDetail: of.ProviderResolutionDetail{
					Reason:          of.DefaultReason,
					ResolutionError: of.NewInvalidContextResolutionError("boom"),
				},
			},
		},
		{
			name:    "unmarshal error",
			flagKey: "obj-unmarshal-error",

			defaultValue: map[string]any{
				"baz": "qux",
			},
			mockRespEvaluation: &evaluation.VariantEvaluationResponse{
				Match:             true,
				VariantAttachment: "x",
			},
			expected: of.InterfaceResolutionDetail{
				Value: map[string]any{
					"baz": "qux",
				},
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					Reason:          of.ErrorReason,
					ResolutionError: of.NewTypeMismatchResolutionError("value is not an object: \"x\""),
				},
			},
		},
		{
			name:    "error",
			flagKey: "obj-error",

			defaultValue: map[string]any{
				"baz": "qux",
			},
			mockRespEvaluationErr: errors.New("boom"),
			expected: of.InterfaceResolutionDetail{
				Value: map[string]any{
					"baz": "qux",
				},
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					Reason:          of.DefaultReason,
					ResolutionError: of.NewGeneralResolutionError("boom"),
				},
			},
		},
		{
			name:    "no match",
			flagKey: "obj-no-match",

			defaultValue: map[string]any{
				"baz": "qux",
			},
			mockRespEvaluation: &evaluation.VariantEvaluationResponse{
				Match: false,
			},
			expected: of.InterfaceResolutionDetail{
				Value: map[string]any{
					"baz": "qux",
				},
				ProviderResolutionDetail: of.ProviderResolutionDetail{Reason: of.DefaultReason},
			},
		},
		{
			name:    "match",
			flagKey: "obj-match",

			defaultValue: map[string]any{
				"baz": "qux",
			},
			mockRespEvaluation: &evaluation.VariantEvaluationResponse{
				Match:             true,
				VariantKey:        "2",
				VariantAttachment: "{\"foo\": \"bar\"}",
			},
			expected: of.InterfaceResolutionDetail{
				Value: map[string]any{
					"foo": "bar",
				},
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					Reason: of.TargetingMatchReason,
				},
			},
		},
		{
			name:    "match no attachment",
			flagKey: "obj-match-no-attach",

			defaultValue: map[string]any{
				"baz": "qux",
			},
			mockRespEvaluation: &evaluation.VariantEvaluationResponse{
				Match:      true,
				VariantKey: "2",
			},
			expected: of.InterfaceResolutionDetail{
				Value: map[string]any{
					"baz": "qux",
				},
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					Reason: of.DefaultReason,
				},
			},
		},
		{
			name:    "flipt-default",
			flagKey: "obj-match",

			defaultValue: map[string]any{
				"baz": "qux",
			},
			mockRespEvaluation: &evaluation.VariantEvaluationResponse{
				Match:             false,
				Reason:            evaluation.EvaluationReason_DEFAULT_EVALUATION_REASON,
				VariantKey:        "2",
				VariantAttachment: "{\"foo\": \"bar\"}",
			},
			expected: of.InterfaceResolutionDetail{
				Value: map[string]any{
					"foo": "bar",
				},
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					Reason: of.DefaultReason,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := newMockService(t)
			mockSvc.On("Evaluate", mock.Anything, "flipt", tt.flagKey, mock.Anything).Return(tt.mockRespEvaluation, tt.mockRespEvaluationErr).Maybe()

			p := NewProvider(WithService(mockSvc), ForNamespace("flipt"))

			actual := p.ObjectEvaluation(context.Background(), tt.flagKey, tt.defaultValue, map[string]any{})

			assert.Equal(t, tt.expected.Value, actual.Value)
		})
	}
}
