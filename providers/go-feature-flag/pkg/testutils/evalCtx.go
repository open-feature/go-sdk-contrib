package testutils

import (
	"github.com/open-feature/go-sdk/openfeature"
)

// DefaultEvaluationContext is a default evaluation context for testing.
var DefaultEvaluationContext = func() openfeature.EvaluationContext {
	return openfeature.NewEvaluationContext(
		"d45e303a-38c2-11ed-a261-0242ac120002",
		map[string]interface{}{
			"email":        "john.doe@gofeatureflag.org",
			"firstname":    "john",
			"lastname":     "doe",
			"anonymous":    false,
			"professional": true,
			"rate":         3.14,
			"age":          30,
			"admin":        true,
			"company_info": map[string]interface{}{
				"name": "my_company",
				"size": 120,
			},
			"labels": []string{
				"pro", "beta",
			},
		},
	)
}()
