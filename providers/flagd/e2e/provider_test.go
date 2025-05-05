//go:build e2e

package e2e

import (
	"github.com/open-feature/go-sdk-contrib/providers/flagd/e2e/containers"
	flagd "github.com/open-feature/go-sdk-contrib/providers/flagd/pkg"
	"github.com/open-feature/go-sdk-contrib/tests/flagd/pkg/integration"
	"testing"
)

var gherkinFiles = []string{
	"../flagd-testbed/gherkin/config.feature",
	"../flagd-testbed/gherkin/flagd-json-evaluator.feature",
}

// tags to be ignored (currently not supported)
var tags = "~@sync && ~@offline && ~@events"

func TestRPC(t *testing.T) {
	runGherkinTestWithFeatureProvider(
		gherkinTestRunConfig{
			t:                   t,
			scenarioInitializer: integration.InitializeGenericScenario,
			name:                "flagd-rpc",
			gherkinFiles:        gherkinFiles,
			port:                containers.Remote,
			Tags:                tags,
		},
	)
}

func TestInProcess(t *testing.T) {

	runGherkinTestWithFeatureProvider(
		gherkinTestRunConfig{
			t: t,
			// prepareTestSuite:    integration.PrepareGenericTestSuite,
			scenarioInitializer: integration.InitializeGenericScenario,
			name:                "flagd-in-process",
			gherkinFiles:        gherkinFiles,
			port:                containers.InProcess,
			providerOptions:     []flagd.ProviderOption{flagd.WithInProcessResolver()},
			Tags:                tags,
		},
	)
}
