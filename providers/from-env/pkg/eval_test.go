package from_env

import (
	"encoding/json"
	"testing"

	"github.com/open-feature/go-sdk/openfeature"
)

func TestValuesEqual(t *testing.T) {
	tests := []struct {
		name  string
		left  any
		right any
		want  bool
	}{
		// nil
		{name: "nil_nil", left: nil, right: nil, want: true},
		{name: "nil_nonnil", left: nil, right: "x", want: false},
		{name: "nonnil_nil", left: "x", right: nil, want: false},

		// bool
		{name: "bool_equal_true", left: true, right: true, want: true},
		{name: "bool_equal_false", left: false, right: false, want: true},
		{name: "bool_unequal", left: true, right: false, want: false},
		{name: "bool_vs_string", left: true, right: "true", want: false},
		{name: "bool_vs_int", left: true, right: 1, want: false},

		// float64
		{name: "float64_equal", left: float64(3.14), right: float64(3.14), want: true},
		{name: "float64_unequal", left: float64(1.0), right: float64(2.0), want: false},
		{name: "float64_vs_int", left: float64(1), right: 1, want: false},
		{name: "float64_vs_string", left: float64(1), right: "1", want: false},
		{name: "float64_nan", left: float64(0), right: float64(0), want: true},

		// string (existing comparable path)
		{name: "string_equal", left: "abc", right: "abc", want: true},
		{name: "string_unequal", left: "abc", right: "xyz", want: false},

		// deep equal – slices
		{name: "slice_equal", left: []any{"a", "b"}, right: []any{"a", "b"}, want: true},
		{name: "slice_unequal", left: []any{"a", "b"}, right: []any{"a", "c"}, want: false},
		{name: "slice_diff_len", left: []any{"a"}, right: []any{"a", "b"}, want: false},

		// deep equal – maps
		{name: "map_equal", left: map[string]any{"k": "v"}, right: map[string]any{"k": "v"}, want: true},
		{name: "map_unequal", left: map[string]any{"k": "v"}, right: map[string]any{"k": "x"}, want: false},
		{name: "map_diff_keys", left: map[string]any{"a": 1}, right: map[string]any{"b": 1}, want: false},

		// complex types via reflect.DeepEqual
		{
			name:  "nested_struct_equal",
			left:  struct{ A int }{1},
			right: struct{ A int }{1},
			want:  true,
		},
		{
			name:  "nested_struct_unequal",
			left:  struct{ A int }{1},
			right: struct{ A int }{2},
			want:  false,
		},
		{
			name:  "noncomparable_struct_equal",
			left:  struct{ S []string }{[]string{"a"}},
			right: struct{ S []string }{[]string{"a"}},
			want:  true,
		},
		{
			name:  "noncomparable_struct_unequal",
			left:  struct{ S []string }{[]string{"a"}},
			right: struct{ S []string }{[]string{"b"}},
			want:  false,
		},
		{
			name:  "noncomparable_mismatched_types",
			left:  struct{ S []string }{[]string{"a"}},
			right: struct{ N []int }{[]int{1}},
			want:  false,
		},
		{
			name:  "slice_of_maps_equal",
			left:  []any{map[string]any{"x": 1}},
			right: []any{map[string]any{"x": 1}},
			want:  true,
		},
		{
			name:  "slice_of_maps_unequal",
			left:  []any{map[string]any{"x": 1}},
			right: []any{map[string]any{"x": 2}},
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := valuesEqual(tt.left, tt.right); got != tt.want {
				t.Errorf("valuesEqual(%v, %v) = %v, want %v", tt.left, tt.right, got, tt.want)
			}
		})
	}
}

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
		{
			name: "equal_nested_json",
			criteriaValue: []any{
				map[string]any{"roles": []any{"admin", "beta"}},
			},
			evalValue: []any{
				map[string]any{"roles": []any{"admin", "beta"}},
			},
			wantVariant: "match",
			wantReason:  openfeature.TargetingMatchReason,
			wantValue:   true,
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
