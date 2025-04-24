//go:build e2e

package e2e

import (
	"github.com/open-feature/go-sdk-contrib/providers/flagd/e2e/containers"
	flagd "github.com/open-feature/go-sdk-contrib/providers/flagd/pkg"
	"github.com/open-feature/go-sdk-contrib/tests/flagd/pkg/integration"
	"testing"
)

func TestJsonEvaluatorInRPC(t *testing.T) {
	testJsonEvaluator(t, containers.Remote)
}

func TestJsonEvaluatorInProcess(t *testing.T) {
	testJsonEvaluator(t, containers.InProcess, flagd.WithInProcessResolver())
}

func testJsonEvaluator(t *testing.T, exposedPort containers.ExposedPort, providerOptions ...flagd.ProviderOption) {
	runGherkinTestWithFeatureProvider(
		gherkinTestRunConfig{
			t:                   t,
			prepareTestSuite:    integration.PrepareFlagdJsonTestSuite,
			scenarioInitializer: integration.InitializeFlagdJsonScenario,
			name:                "flagd-json-evaluator.feature",
			gherkinFile:         "../flagd-testbed/gherkin/flagd-json-evaluator.feature",
			port:                exposedPort,
			providerOptions:     providerOptions,
		},
	)
}
