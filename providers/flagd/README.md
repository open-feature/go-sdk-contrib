# Flagd Provider

![Experimental](https://img.shields.io/badge/experimental-breaking%20changes%20allowed-yellow)
![Alpha](https://img.shields.io/badge/alpha-release-red)

[Flagd](https://github.com/open-feature/flagd) is a simple command line tool for fetching and presenting feature flags to services. It is designed to conform to OpenFeature schema for flag definitions. This repository and package provides the client side code for interacting with it via the [OpenFeature SDK](https://github.com/open-feature/go-sdk).

## Setup
To use flagd with the [OpenFeature SDK](https://github.com/open-feature/go-sdk) set the provider to the `openfeature` global singleton as shown below (using default values which align with those of `flagd`)
```go
openfeature.SetProvider(flagd.NewProvider())
```  
You may also provide additional options to configure the provider client
```go
flagd.WithHost(string)                  // defaults to localhost
flagd.WithPort(uint16)                  // defaults to 8013
flagd.FromEnv()                         // sets the provider configuration from environment variables
flagd.WithSocketPath(string)            // no default, when set a unix socket connection is used (only available for GRPC)
```
for example:
```go
package main

import (
	"github.com/open-feature/go-sdk-contrib/providers/flagd/pkg"
   	"github.com/open-feature/go-sdk/pkg/openfeature"
)

func main() {
    openfeature.SetProvider(flagd.NewProvider(
        flagd.WithHost("localhost"),
        flagd.WithPort(8000),
    ))
}
```

### Using flagd.FromEnv()  
By default the flagd provider will not read environment variables to set its own configuration, however, if the `flagd.FromEnv()` option is set as an argument for the `flagd.NewProvider()` method, then the following environment variables will be checked: `FLAGD_HOST`, `FLAGD_PORT`, `FLAGD_SERVER_CERT_PATH`.

In the event that another configuration option is passed to the `flagd.NewProvider()` method, such as `flagd.WithPort(8013)` then this value will be prioritized over any existing environment variable configuration. This means that the priority order is as follows:
1. Explicitly set configuration via `WithXXX` options
1. Environment variable configuration values (if the `flagd.FromEnv()` option is set)
1. Default values (host `localhost`, port `8013`)

## License

Apache 2.0 - See [LICENSE](./../../LICENSE) for more information.
