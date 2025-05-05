//go:build e2e

package e2e

import (
	"context"
	"flag"
	"github.com/cucumber/godog"
	"github.com/open-feature/go-sdk-contrib/providers/flagd/e2e/containers"
	flagd "github.com/open-feature/go-sdk-contrib/providers/flagd/pkg"
	"github.com/open-feature/go-sdk-contrib/tests/flagd/pkg/integration"
	"github.com/open-feature/go-sdk/openfeature"
	"os"
	"testing"
)

// usedEnvVars list of env vars that have been set
var usedEnvVars []string

type gherkinTestRunConfig struct {
	t *testing.T
	// prepareTestSuite    func(func() openfeature.FeatureProvider)
	scenarioInitializer func(scenarioContext *godog.ScenarioContext)
	name                string
	gherkinFiles        []string
	port                containers.ExposedPort
	providerOptions     []flagd.ProviderOption
	Tags                string
}

func runGherkinTestWithFeatureProvider(config gherkinTestRunConfig) {
	if testing.Short() {
		// skip e2e if testing -short
		config.t.Skip()
	}

	container, err := containers.NewFlagd(context.TODO())
	if err != nil {
		config.t.Fatal(err)
	}
	flag.Parse()

	var opts []flagd.ProviderOption
	opts = append(opts, config.providerOptions...)
	opts = append(opts, flagd.WithPort(uint16(container.GetPort(config.port))))

	integration.RegisterSetEnvVarFunc(func(envVar, envVarValue string) {
		config.t.Setenv(envVar, envVarValue)
		usedEnvVars = append(usedEnvVars, envVar)
	})

	integration.RegisterProviderSupplier(func() openfeature.FeatureProvider {
		provider, err := flagd.NewProvider(opts...)

		if err != nil {
			config.t.Fatal("Creating provider failed:", err)
		}

		return provider
	})

	testSuite := godog.TestSuite{
		Name: config.name,
		TestSuiteInitializer: func(testSuiteContext *godog.TestSuiteContext) {
			testSuiteContext.AfterSuite(func() {
				err = container.Terminate(context.Background())

				if err != nil {
					config.t.Fatal(err)
				}
			})
		},
		ScenarioInitializer: func(ctx *godog.ScenarioContext) {
			for _, envVar := range usedEnvVars {
				err := os.Unsetenv(envVar)

				if err != nil {
					config.t.Fatal("unsetting environment variable: non-zero status returned")
				}
			}
			usedEnvVars = nil
			config.scenarioInitializer(ctx)
		},
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    config.gherkinFiles,
			TestingT: config.t, // Testing instance that will run subtests.
			Strict:   true,
			Tags:     config.Tags,
		},
	}

	if testSuite.Run() != 0 {
		config.t.Fatal("non-zero status returned, failed to run evaluation tests")
	}
}
