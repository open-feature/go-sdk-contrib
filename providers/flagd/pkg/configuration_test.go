package flagd

import (
	"testing"
)

func TestConfigureProviderConfigurationInProcessWithOfflineFile(t *testing.T) {
	// given
	providerConfiguration := &ProviderConfiguration{
		Resolver:              inProcess,
		OfflineFlagSourcePath: "somePath",
	}

	// when
	configureProviderConfiguration(providerConfiguration)

	// then
	if providerConfiguration.Resolver != file {
		t.Errorf("incorrect Resolver, expected %v, got %v", file, providerConfiguration.Resolver)
	}
}

func TestConfigureProviderConfigurationRpcWithoutPort(t *testing.T) {
	// given
	providerConfiguration := &ProviderConfiguration{
		Resolver: rpc,
	}

	// when
	configureProviderConfiguration(providerConfiguration)

	// then
	if providerConfiguration.Port != defaultRpcPort {
		t.Errorf("incorrect Port, expected %v, got %v", defaultRpcPort, providerConfiguration.Port)
	}
}

func TestConfigureProviderConfigurationInProcessWithoutPort(t *testing.T) {
	// given
	providerConfiguration := &ProviderConfiguration{
		Resolver: inProcess,
	}

	// when
	configureProviderConfiguration(providerConfiguration)

	// then
	if providerConfiguration.Port != defaultInProcessPort {
		t.Errorf("incorrect Port, expected %v, got %v", defaultInProcessPort, providerConfiguration.Port)
	}
}

func TestValidateProviderConfigurationFileMissingData(t *testing.T) {
	// given
	providerConfiguration := &ProviderConfiguration{
		Resolver: file,
	}

	// when
	err := validateProviderConfiguration(providerConfiguration)

	// then
	if err == nil {
		t.Errorf("Error expected but check succeeded")
	}
}
