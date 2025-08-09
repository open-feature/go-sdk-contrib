package e2e

import (
	"os"
	"testing"

	"github.com/cucumber/godog"
	"github.com/open-feature/go-sdk-contrib/tests/flagd/pkg/integration"
)

// TestConfigurationGherkin runs the config.feature gherkin scenarios as unit tests
// This does NOT use testcontainers and runs as fast unit tests
func TestConfigurationGherkin(t *testing.T) {
	// Prepare the integration package
	integration.PrepareConfigTestSuite(setEnvironmentVariable)
	
	// Create test suite for config scenarios
	suite := godog.TestSuite{
		Name:                "flagd-config",
		ScenarioInitializer: integration.InitializeConfigScenario,
		Options: &godog.Options{
			Format: "pretty",
			Paths:  []string{"../flagd-testbed/gherkin/config.feature"},
			TestingT: t,
		},
	}
	
	// Run the tests
	if status := suite.Run(); status != 0 {
		t.Fatalf("Configuration gherkin tests failed with status: %d", status)
	}
}

// TestRPCConfiguration tests RPC-specific configuration scenarios
func TestRPCConfiguration(t *testing.T) {
	integration.PrepareConfigTestSuite(setEnvironmentVariable)
	
	suite := godog.TestSuite{
		Name:                "flagd-config-rpc",
		ScenarioInitializer: integration.InitializeConfigScenario,
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
	integration.PrepareConfigTestSuite(setEnvironmentVariable)
	
	suite := godog.TestSuite{
		Name:                "flagd-config-inprocess",
		ScenarioInitializer: integration.InitializeConfigScenario,
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
	integration.PrepareConfigTestSuite(setEnvironmentVariable)
	
	suite := godog.TestSuite{
		Name:                "flagd-config-file",
		ScenarioInitializer: integration.InitializeConfigScenario,
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

// TestBasicConfiguration tests basic scenarios that work across all resolver types
func TestBasicConfiguration(t *testing.T) {
	integration.PrepareConfigTestSuite(setEnvironmentVariable)
	
	suite := godog.TestSuite{
		Name:                "flagd-config-basic",
		ScenarioInitializer: integration.InitializeConfigScenario,
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../flagd-testbed/gherkin/config.feature"},
			Tags:     "~@rpc && ~@in-process && ~@file", // Basic scenarios without resolver-specific tags
			TestingT: t,
		},
	}
	
	if status := suite.Run(); status != 0 {
		t.Fatalf("Basic configuration tests failed with status: %d", status)
	}
}

// setEnvironmentVariable is used by the integration package to set environment variables
func setEnvironmentVariable(key, value string) {
	os.Setenv(key, value)
}