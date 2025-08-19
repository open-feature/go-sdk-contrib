package e2e

import (
	context2 "context"
	"testing"

	"github.com/cucumber/godog"
	"github.com/open-feature/go-sdk-contrib/tests/flagd/testframework"
)

// configTestCase defines a test case for configuration tests
type configTestCase struct {
	name        string
	tags        string
	concurrency int
}

// TestConfiguration runs all configuration test scenarios using table-driven tests
func TestConfiguration(t *testing.T) {
	testCases := []configTestCase{
		{
			name: "All",
			tags: "",
		},
		{
			name: "RPC",
			tags: "@rpc",
		},
		{
			name: "InProcess",
			tags: "@in-process",
		},
		{
			name: "File",
			tags: "@file",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			suite := godog.TestSuite{
				Name: "flagd-config-" + tc.name,
				ScenarioInitializer: func(context *godog.ScenarioContext) {
					state := testframework.TestState{
						EnvVars:      make(map[string]string),
						EvalContext:  make(map[string]interface{}),
						EventChannel: make(chan testframework.EventRecord, 100),
					}
					testframework.InitializeConfigScenario(context, &state)
					context.After(func(ctx context2.Context, sc *godog.Scenario, err error) (context2.Context, error) {
						state.CleanupEnvironmentVariables()
						return ctx, nil
					})
				},
				Options: &godog.Options{
					Format:   "pretty",
					Paths:    []string{"../flagd-testbed/gherkin/config.feature"},
					Tags:     tc.tags,
					TestingT: t,
				},
			}

			if status := suite.Run(); status != 0 {
				t.Fatalf("%s configuration tests failed with status: %d", tc.name, status)
			}
		})
	}
}
