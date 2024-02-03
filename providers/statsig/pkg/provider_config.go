package statsig

import (
	statsig "github.com/statsig-io/go-sdk"
)

// ProviderConfig is the struct containing the provider options.
type ProviderConfig struct {
	Options statsig.Options
	SdkKey  string
}
