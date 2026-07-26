package from_env_test

import (
	"encoding/json"
	"testing"

	fromEnv "github.com/open-feature/go-sdk-contrib/providers/from-env/pkg"
	"github.com/open-feature/go-sdk/openfeature"
)

func TestNonComparableCriteria(t *testing.T) {
	tests := map[string]any{
		"slice": []any{"admin", "beta"},
		"map":   map[string]any{"tier": "gold"},
	}

	for name, criteriaValue := range tests {
		t.Run(name, func(t *testing.T) {
			flagJSON, err := json.Marshal(fromEnv.StoredFlag{
				DefaultVariant: "default",
				Variants: []fromEnv.Variant{
					{
						Name: "match",
						Criteria: []fromEnv.Criteria{
							{Key: "attribute", Value: criteriaValue},
						},
						Value: true,
					},
					{Name: "default", Value: false},
				},
			})
			if err != nil {
				t.Fatalf("failed to encode flag: %v", err)
			}

			const flagKey = "NON_COMPARABLE_CRITERIA"
			t.Setenv(flagKey, string(flagJSON))

			result := (&fromEnv.FromEnvProvider{}).BooleanEvaluation(
				t.Context(),
				flagKey,
				false,
				map[string]any{"attribute": criteriaValue},
			)
			if err := result.Error(); err != nil {
				t.Fatalf("unexpected evaluation error: %v", err)
			}
			if result.Variant != "match" {
				t.Fatalf("expected matching variant, got %q", result.Variant)
			}
			if result.Reason != openfeature.TargetingMatchReason {
				t.Fatalf("expected targeting match reason, got %q", result.Reason)
			}
			if !result.Value {
				t.Fatal("expected true value")
			}
		})
	}
}
