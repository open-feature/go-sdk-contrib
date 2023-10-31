package harness

import (
	harness "github.com/harness/ff-golang-server-sdk/client"
)

type ProviderConfig struct {
	Options []harness.ConfigOption
	SdkKey  string
}
