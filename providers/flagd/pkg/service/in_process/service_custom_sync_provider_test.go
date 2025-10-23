package process

import (
	"testing"
)

func TestInProcessWithCustomSyncProvider(t *testing.T) {
	customSyncProvider := NewDoNothingCustomSyncProvider()
	service := NewInProcessService(Configuration{CustomSyncProvider: customSyncProvider, CustomSyncProviderUri: "not tested here"})

	// If custom sync provider is supplied the in-process service should use it.
	if service.syncProvider != customSyncProvider {
		t.Fatalf("Expected service.sync to be the mockCustomSyncProvider, but got %s", service.syncProvider)
	}
}
