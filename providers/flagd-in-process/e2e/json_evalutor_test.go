package e2e

import (
	"context"
	"flag"
	"testing"

	"github.com/cucumber/godog"
	in_process "github.com/open-feature/go-sdk-contrib/providers/flagd-in-process/pkg"
	"github.com/open-feature/go-sdk-contrib/tests/flagd/pkg/integration"
	"github.com/open-feature/go-sdk/openfeature"
)

func TestJsonEvaluatorFlagdInProcess(t *testing.T) {
	if testing.Short() {
		// skip e2e if testing -short
		t.Skip()
	}

	flag.Parse()

	name := "flagd-json-evaluator.feature"

	testSuite := godog.TestSuite{
		Name: name,
		TestSuiteInitializer: integration.InitializeFlagdJsonTestSuite(func() openfeature.FeatureProvider {
			return in_process.NewProvider(context.Background(), in_process.WithSourceURI("localhost:9090"), in_process.WithSourceType(in_process.SourceTypeGrpc))
		}),
		ScenarioInitializer: integration.InitializeFlagdJsonScenario,
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../../../flagd-testbed/gherkin/flagd-json-evaluator.feature"},
			TestingT: t, // Testing instance that will run subtests.
			Strict:   true,
		},
	}

	if testSuite.Run() != 0 {
		t.Fatal("non-zero status returned, failed to run evaluation tests")
	}
}
