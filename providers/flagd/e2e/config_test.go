package e2e

import (
	context2 "context"
	"testing"

	"github.com/cucumber/godog"
	"github.com/open-feature/go-sdk-contrib/tests/flagd/testframework"
	"github.com/open-feature/go-sdk/openfeature"
)

// TestConfigurationGherkin runs the config.feature gherkin scenarios as unit tests
// This does NOT use testcontainers and runs as fast unit tests
func TestConfigurationGherkin(t *testing.T) {
	// Create test suite for config scenarios
	suite := godog.TestSuite{
		Name: "flagd-config",
		ScenarioInitializer: func(context *godog.ScenarioContext) {
			state := testframework.TestState{
				EnvVars:       make(map[string]string),
				EvalContext:   make(map[string]interface{}),
				Events:        []testframework.EventRecord{},
				EventHandlers: make(map[string]func(openfeature.EventDetails)),
			}
			testframework.InitializeConfigScenario(context, &state)
			context.After(func(ctx context2.Context, sc *godog.Scenario, err error) (context2.Context, error) {
				// Note: OpenFeature providers don't have a Shutdown method
				state.CleanupEnvironmentVariables()
				return ctx, nil
			})
		},
		Options: &godog.Options{
			Format:      "pretty",
			Paths:       []string{"../flagd-testbed/gherkin/config.feature"},
			TestingT:    t,
			Concurrency: 1,
		},
	}

	// Run the tests
	if status := suite.Run(); status != 0 {
		t.Fatalf("Configuration gherkin tests failed with status: %d", status)
	}
}

// TestRPCConfiguration tests RPC-specific configuration scenarios
func TestRPCConfiguration(t *testing.T) {
	suite := godog.TestSuite{
		Name: "flagd-config-rpc",
		ScenarioInitializer: func(context *godog.ScenarioContext) {
			state := testframework.TestState{
				EnvVars:       make(map[string]string),
				EvalContext:   make(map[string]interface{}),
				Events:        []testframework.EventRecord{},
				EventHandlers: make(map[string]func(openfeature.EventDetails)),
			}
			testframework.InitializeConfigScenario(context, &state)
			context.After(func(ctx context2.Context, sc *godog.Scenario, err error) (context2.Context, error) {
				// Note: OpenFeature providers don't have a Shutdown method
				state.CleanupEnvironmentVariables()
				return ctx, nil
			})
		},
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../flagd-testbed/gherkin/config.feature"},
			Tags:     "@rpc",
			TestingT: t,
		},
	}

	if status := suite.Run(); status != 0 {
		t.Fatalf("RPC configuration tests failed with status: %d", status)
	}
}

// TestInProcessConfiguration tests in-process-specific configuration scenarios
func TestInProcessConfiguration(t *testing.T) {
	suite := godog.TestSuite{
		Name: "flagd-config-inprocess",
		ScenarioInitializer: func(context *godog.ScenarioContext) {
			state := testframework.TestState{
				EnvVars:       make(map[string]string),
				EvalContext:   make(map[string]interface{}),
				Events:        []testframework.EventRecord{},
				EventHandlers: make(map[string]func(openfeature.EventDetails)),
			}
			testframework.InitializeConfigScenario(context, &state)
			context.After(func(ctx context2.Context, sc *godog.Scenario, err error) (context2.Context, error) {
				// Note: OpenFeature providers don't have a Shutdown method
				state.CleanupEnvironmentVariables()
				return ctx, nil
			})
		},
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../flagd-testbed/gherkin/config.feature"},
			Tags:     "@in-process",
			TestingT: t,
		},
	}

	if status := suite.Run(); status != 0 {
		t.Fatalf("In-process configuration tests failed with status: %d", status)
	}
}

// TestFileConfiguration tests file-specific configuration scenarios
func TestFileConfiguration(t *testing.T) {
	suite := godog.TestSuite{
		Name: "flagd-config-file",
		ScenarioInitializer: func(context *godog.ScenarioContext) {
			state := testframework.TestState{
				EnvVars:       make(map[string]string),
				EvalContext:   make(map[string]interface{}),
				Events:        []testframework.EventRecord{},
				EventHandlers: make(map[string]func(openfeature.EventDetails)),
			}
			testframework.InitializeConfigScenario(context, &state)
			context.After(func(ctx context2.Context, sc *godog.Scenario, err error) (context2.Context, error) {
				// Note: OpenFeature providers don't have a Shutdown method
				state.CleanupEnvironmentVariables()
				return ctx, nil
			})
		},
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../flagd-testbed/gherkin/config.feature"},
			Tags:     "@file",
			TestingT: t,
		},
	}

	if status := suite.Run(); status != 0 {
		t.Fatalf("File configuration tests failed with status: %d", status)
	}
}
