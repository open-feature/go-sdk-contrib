package from_env

import (
	"encoding/json"
	"testing"

	"github.com/open-feature/go-sdk/openfeature"
)

func TestNonComparableCriteria(t *testing.T) {
	tests := []struct {
		name          string
		criteriaValue any
		evalValue     any
		wantVariant   string
		wantReason    openfeature.Reason
		wantValue     bool
	}{
		{
			name:          "equal_slice",
			criteriaValue: []any{"admin", "beta"},
			evalValue:     []any{"admin", "beta"},
			wantVariant:   "match",
			wantReason:    openfeature.TargetingMatchReason,
			wantValue:     true,
		},
		{
			name:          "unequal_slice",
			criteriaValue: []any{"admin", "beta"},
			evalValue:     []any{"admin", "stable"},
			wantVariant:   "default",
			wantReason:    openfeature.DefaultReason,
			wantValue:     false,
		},
		{
			name:          "equal_map",
			criteriaValue: map[string]any{"tier": "gold"},
			evalValue:     map[string]any{"tier": "gold"},
			wantVariant:   "match",
			wantReason:    openfeature.TargetingMatchReason,
			wantValue:     true,
		},
		{
			name:          "unequal_map",
			criteriaValue: map[string]any{"tier": "gold"},
			evalValue:     map[string]any{"tier": "silver"},
			wantVariant:   "default",
			wantReason:    openfeature.DefaultReason,
			wantValue:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flagJSON, err := json.Marshal(StoredFlag{
				DefaultVariant: "default",
				Variants: []Variant{
					{
						Name: "match",
						Criteria: []Criteria{
							{Key: "attribute", Value: tt.criteriaValue},
						},
						Value: true,
					},
					{
						Name: "default",
						Criteria: []Criteria{
							{Key: "missing", Value: true},
						},
						Value: false,
					},
				},
			})
			if err != nil {
				t.Fatalf("failed to encode flag: %v", err)
			}

			const flagKey = "NON_COMPARABLE_CRITERIA"
			t.Setenv(flagKey, string(flagJSON))

			result := (&FromEnvProvider{}).BooleanEvaluation(
				t.Context(),
				flagKey,
				false,
				map[string]any{"attribute": tt.evalValue},
			)
			if err := result.Error(); err != nil {
				t.Fatalf("unexpected evaluation error: %v", err)
			}
			if result.Variant != tt.wantVariant {
				t.Fatalf("expected variant %q, got %q", tt.wantVariant, result.Variant)
			}
			if result.Reason != tt.wantReason {
				t.Fatalf("expected reason %q, got %q", tt.wantReason, result.Reason)
			}
			if result.Value != tt.wantValue {
				t.Fatalf("expected value %t, got %t", tt.wantValue, result.Value)
			}
		})
	}
}

func TestCriteriaComparison(t *testing.T) {
	left, right := 1, 1
	tests := []struct {
		name          string
		criteriaValue any
		evalValue     any
		wantVariant   string
	}{
		{
			name:          "equal_comparable",
			criteriaValue: "admin",
			evalValue:     "admin",
			wantVariant:   "match",
		},
		{
			name:          "unequal_comparable",
			criteriaValue: "admin",
			evalValue:     "viewer",
			wantVariant:   "default",
		},
		{
			name:          "distinct_pointers",
			criteriaValue: &right,
			evalValue:     &left,
			wantVariant:   "default",
		},
		{
			name:          "mismatched_dynamic_types",
			criteriaValue: int64(1),
			evalValue:     int(1),
			wantVariant:   "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := StoredFlag{
				DefaultVariant: "default",
				Variants: []Variant{
					{
						Name: "match",
						Criteria: []Criteria{
							{Key: "attribute", Value: tt.criteriaValue},
						},
					},
					{
						Name: "default",
						Criteria: []Criteria{
							{Key: "missing", Value: true},
						},
					},
				},
			}

			variant, _, _, err := flag.evaluate(map[string]any{"attribute": tt.evalValue})
			if err != nil {
				t.Fatalf("unexpected evaluation error: %v", err)
			}
			if variant != tt.wantVariant {
				t.Fatalf("expected variant %q, got %q", tt.wantVariant, variant)
			}
		})
	}
}
