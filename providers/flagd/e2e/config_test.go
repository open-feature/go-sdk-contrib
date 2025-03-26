//go:build e2e

package e2e

import (
	"flag"
	"github.com/open-feature/go-sdk-contrib/tests/flagd/pkg/integration"
	"os"
	"testing"

	"github.com/cucumber/godog"
)

// usedEnvVars list of env vars that have been set
var usedEnvVars []string

func TestConfig(t *testing.T) {
	if testing.Short() {
		// skip e2e if testing -short
		t.Skip()
	}

	flag.Parse()

	name := "config.feature"

	testSuite := godog.TestSuite{
		Name: name,
		TestSuiteInitializer: func(testSuiteContext *godog.TestSuiteContext) {
			integration.PrepareConfigTestSuite(
				func(envVar, envVarValue string) {
					t.Setenv(envVar, envVarValue)
					usedEnvVars = append(usedEnvVars, envVar)
				},
			)
		},
		ScenarioInitializer: func(ctx *godog.ScenarioContext) {
			for _, envVar := range usedEnvVars {
				err := os.Unsetenv(envVar)

				if err != nil {
					t.Fatal("unsetting environment variable: non-zero status returned")
				}
			}
			usedEnvVars = nil
			integration.InitializeConfigScenario(ctx)
		},
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../flagd-testbed/gherkin/config.feature"},
			TestingT: t, // Testing instance that will run subtests.
			Strict:   true,
		},
	}

	if testSuite.Run() != 0 {
		t.Fatal("non-zero status returned, failed to run evaluation tests")
	}
}
