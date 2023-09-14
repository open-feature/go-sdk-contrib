//go:build e2e
// +build e2e

package e2e

import (
	"flag"
	"testing"

	"github.com/cucumber/godog"
	flagd "github.com/open-feature/go-sdk-contrib/providers/flagd/pkg"
	"github.com/open-feature/go-sdk-contrib/tests/flagd/pkg/integration"
)

func TestEvaluation(t *testing.T) {
	if testing.Short() {
		// skip e2e if testing -short
		t.Skip()
	}

	flag.Parse()

	var providerOptions []flagd.ProviderOption
	name := "evaluation.feature"

	testSuite := godog.TestSuite{
		Name:                name,
		ScenarioInitializer: integration.InitializeEvaluationScenario(providerOptions...),
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../../../spec/specification/assets/gherkin/evaluation.feature"},
			TestingT: t, // Testing instance that will run subtests.
			Strict:   true,
		},
	}

	if testSuite.Run() != 0 {
		t.Fatal("non-zero status returned, failed to run evaluation tests")
	}
}
