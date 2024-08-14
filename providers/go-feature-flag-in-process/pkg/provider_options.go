package gofeatureflaginprocess

import (
	ffclient "github.com/thomaspoignant/go-feature-flag"
)

// ProviderOptions is the struct containing the provider options you can
// use while initializing GO Feature Flag.
// To have a valid configuration you need to have an Endpoint or GOFeatureFlagConfig set.
type ProviderOptions struct {
	// GOFeatureFlagConfig is the configuration struct for the GO Feature Flag module.
	// If not nil we will launch the provider using the GO Feature Flag module.
	GOFeatureFlagConfig *ffclient.Config
}
