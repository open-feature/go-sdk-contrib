package flagd

import (
	"os"
	"testing"

	"github.com/go-logr/logr"
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

func TestUpdatePortFromEnvVarInProcessWithSyncPort(t *testing.T) {
	// given
	os.Setenv("FLAGD_SYNC_PORT", "9999")
	defer os.Unsetenv("FLAGD_SYNC_PORT")

	providerConfiguration := &ProviderConfiguration{
		Resolver: inProcess,
		log:      logr.Discard(),
	}

	// when
	providerConfiguration.updatePortFromEnvVar()

	// then
	if providerConfiguration.Port != 9999 {
		t.Errorf("incorrect Port, expected %v, got %v", 9999, providerConfiguration.Port)
	}
}

func TestUpdatePortFromEnvVarInProcessWithLegacyPort(t *testing.T) {
	// given - for backwards compatibility, FLAGD_PORT should work for in-process
	os.Setenv("FLAGD_PORT", "8888")
	defer os.Unsetenv("FLAGD_PORT")

	providerConfiguration := &ProviderConfiguration{
		Resolver: inProcess,
		log:      logr.Discard(),
	}

	// when
	providerConfiguration.updatePortFromEnvVar()

	// then
	if providerConfiguration.Port != 8888 {
		t.Errorf("incorrect Port, expected %v, got %v", 8888, providerConfiguration.Port)
	}
}

func TestUpdatePortFromEnvVarInProcessSyncPortPriority(t *testing.T) {
	// given - FLAGD_SYNC_PORT takes priority over FLAGD_PORT
	os.Setenv("FLAGD_SYNC_PORT", "9999")
	os.Setenv("FLAGD_PORT", "8888")
	defer os.Unsetenv("FLAGD_SYNC_PORT")
	defer os.Unsetenv("FLAGD_PORT")

	providerConfiguration := &ProviderConfiguration{
		Resolver: inProcess,
		log:      logr.Discard(),
	}

	// when
	providerConfiguration.updatePortFromEnvVar()

	// then
	if providerConfiguration.Port != 9999 {
		t.Errorf("incorrect Port, expected %v, got %v", 9999, providerConfiguration.Port)
	}
}

func TestUpdatePortFromEnvVarRpcWithPort(t *testing.T) {
	// given - RPC resolver uses FLAGD_PORT
	os.Setenv("FLAGD_PORT", "8888")
	defer os.Unsetenv("FLAGD_PORT")

	providerConfiguration := &ProviderConfiguration{
		Resolver: rpc,
		log:      logr.Discard(),
	}

	// when
	providerConfiguration.updatePortFromEnvVar()

	// then
	if providerConfiguration.Port != 8888 {
		t.Errorf("incorrect Port, expected %v, got %v", 8888, providerConfiguration.Port)
	}
}

func TestUpdatePortFromEnvVarRpcIgnoresSyncPort(t *testing.T) {
	// given - RPC resolver should NOT use FLAGD_SYNC_PORT
	os.Setenv("FLAGD_SYNC_PORT", "9999")
	defer os.Unsetenv("FLAGD_SYNC_PORT")

	providerConfiguration := &ProviderConfiguration{
		Resolver: rpc,
		log:      logr.Discard(),
	}

	// when
	providerConfiguration.updatePortFromEnvVar()

	// then - port should remain 0 (unset) since FLAGD_PORT is not set
	if providerConfiguration.Port != 0 {
		t.Errorf("incorrect Port, expected %v, got %v", 0, providerConfiguration.Port)
	}
}
