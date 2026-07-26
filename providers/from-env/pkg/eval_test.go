package from_env

import "testing"

func TestComparableCriteriaUseDirectComparison(t *testing.T) {
	left, right := 1, 1
	flag := StoredFlag{
		DefaultVariant: "default",
		Variants: []Variant{
			{
				Name: "match",
				Criteria: []Criteria{
					{Key: "attribute", Value: &right},
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

	variant, _, _, err := flag.evaluate(map[string]any{"attribute": &left})
	if err != nil {
		t.Fatalf("unexpected evaluation error: %v", err)
	}
	if variant != "default" {
		t.Fatalf("expected default variant, got %q", variant)
	}
}
