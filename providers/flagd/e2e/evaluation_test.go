//go:build e2e

package e2e

import (
	"github.com/open-feature/go-sdk-contrib/providers/flagd/e2e/containers"
	flagd "github.com/open-feature/go-sdk-contrib/providers/flagd/pkg"
	"github.com/open-feature/go-sdk-contrib/tests/flagd/pkg/integration"
	"testing"
)

func TestETestEvaluationFlagdInRPC(t *testing.T) {
	testJsonEvaluatorFlagd(t, containers.Remote)
}

func TestJsonEvaluatorFlagdInProcess(t *testing.T) {
	testJsonEvaluatorFlagd(t, containers.InProcess, flagd.WithInProcessResolver())
}

func testJsonEvaluatorFlagd(
	t *testing.T,
	exposedPort containers.ExposedPort,
	providerOptions ...flagd.ProviderOption,
) {
	runGherkinTestWithFeatureProvider(
		gherkinTestRunConfig{
			t:                   t,
			scenarioInitializer: integration.InitializeEvaluationScenario,
			name:                "evaluation.feature",
			gherkinFiles:        []string{"../spec/specification/assets/gherkin/evaluation.feature"},
			port:                exposedPort,
			providerOptions:     providerOptions,
		},
	)
}
