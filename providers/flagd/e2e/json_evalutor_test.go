//go:build e2e

package e2e

import (
	"flag"
	flagd "github.com/open-feature/go-sdk-contrib/providers/flagd/pkg"
	"testing"

	"github.com/cucumber/godog"
	"github.com/open-feature/go-sdk-contrib/tests/flagd/pkg/integration"
	"github.com/open-feature/go-sdk/openfeature"
)

func TestJsonEvaluatorInRPC(t *testing.T) {
	if testing.Short() {
		// skip e2e if testing -short
		t.Skip()
	}

	flag.Parse()

	name := "flagd-json-evaluator.feature"

	testSuite := godog.TestSuite{
		Name: name,
		TestSuiteInitializer: integration.InitializeFlagdJsonTestSuite(func() openfeature.FeatureProvider {
			provider, err := flagd.NewProvider(flagd.WithPort(8013))

			if err != nil {
				t.Fatal("Creating provider failed:", err)
			}

			return provider
		}),
		ScenarioInitializer: integration.InitializeFlagdJsonScenario,
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../flagd-testbed/gherkin/flagd-json-evaluator.feature"},
			TestingT: t, // Testing instance that will run subtests.
			Strict:   true,
		},
	}

	if testSuite.Run() != 0 {
		t.Fatal("non-zero status returned, failed to run evaluation tests")
	}
}

func TestJsonEvaluatorInProcess(t *testing.T) {
	if testing.Short() {
		// skip e2e if testing -short
		t.Skip()
	}

	flag.Parse()

	name := "flagd-json-evaluator.feature"

	testSuite := godog.TestSuite{
		Name: name,
		TestSuiteInitializer: integration.InitializeFlagdJsonTestSuite(func() openfeature.FeatureProvider {
			provider, err := flagd.NewProvider(flagd.WithInProcessResolver(), flagd.WithPort(9090))

			if err != nil {
				t.Fatal("Creating provider failed:", err)
			}

			return provider
		}),
		ScenarioInitializer: integration.InitializeFlagdJsonScenario,
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../flagd-testbed/gherkin/flagd-json-evaluator.feature"},
			TestingT: t, // Testing instance that will run subtests.
			Strict:   true,
		},
	}

	if testSuite.Run() != 0 {
		t.Fatal("non-zero status returned, failed to run evaluation tests")
	}
}
