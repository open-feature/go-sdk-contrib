//go:build e2e

package e2e

import (
	"context"
	"flag"
	"github.com/cucumber/godog"
	"github.com/open-feature/go-sdk-contrib/providers/flagd/e2e/containers"
	flagd "github.com/open-feature/go-sdk-contrib/providers/flagd/pkg"
	"github.com/open-feature/go-sdk/openfeature"
	"testing"
)

type gherkinTestRunConfig struct {
	t                   *testing.T
	prepareTestSuite    func(func() openfeature.FeatureProvider)
	scenarioInitializer func(scenarioContext *godog.ScenarioContext)
	name                string
	gherkinFile         string
	port                containers.ExposedPort
	providerOptions     []flagd.ProviderOption
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

	testSuite := godog.TestSuite{
		Name: config.name,
		TestSuiteInitializer: func(testSuiteContext *godog.TestSuiteContext) {
			config.prepareTestSuite(func() openfeature.FeatureProvider {
				provider, err := flagd.NewProvider(opts...)

				if err != nil {
					config.t.Fatal("Creating provider failed:", err)
				}

				return provider
			})

			testSuiteContext.AfterSuite(func() {
				err = container.Terminate(context.Background())

				if err != nil {
					config.t.Fatal(err)
				}
			})
		},
		ScenarioInitializer: config.scenarioInitializer,
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{config.gherkinFile},
			TestingT: config.t, // Testing instance that will run subtests.
			Strict:   true,
		},
	}

	if testSuite.Run() != 0 {
		config.t.Fatal("non-zero status returned, failed to run evaluation tests")
	}
}
