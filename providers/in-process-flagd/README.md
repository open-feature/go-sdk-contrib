# In-process Flagd Provider

This provider implements parts of the functionality of [Flagd](https://github.com/open-feature/flagd), and follows its
specification for [in-process providers](https://github.com/open-feature/flagd/blob/main/docs/other_resources/in-process-providers/specification.md).

## Setup
To use flagd with the [OpenFeature SDK](https://github.com/open-feature/go-sdk) set the provider to the `openfeature` global singleton as shown below (using default values which align with those of `flagd`)
```go
openfeature.SetProvider(inprocessflagd.NewProvider())
```  
You may also provide additional options to configure the provider client
```go
inprocessflagd.WithHost(string)                  // defaults to localhost
inprocessflagd.WithPort(uint16)                  // defaults to 8013
inprocessflagd.FromEnv()                         // sets the provider configuration from environment variables
inprocessflagd.WithSocketPath(string)            // no default, when set a unix socket connection is used (only available for GRPC)
inprocessflagd.WithTLS(certPath string)          // default of false, if certPath is not given, system certs are used
inprocessflagd.WithoutCache()                    // disables caching of flag evaluations
inprocessflagd.WithLRUCache(1000)                // enables LRU caching (see configuring caching section)
inprocessflagd.WithBasicInMemoryCache()          // enables basic in memory cache (see configuring caching section)
inprocessflagd.WithLogger(logger)                // sets a custom logger (see logging section)
inprocessflagd.WithOtelInterceptor(bool)         // enable or disable OpenTelemetry interceptor for flagd communication
```
for example:
```go
package main

import (
	"github.com/open-feature/go-sdk-contrib/providers/flagd/pkg"
   	"github.com/open-feature/go-sdk/pkg/openfeature"
)

func main() {
    openfeature.SetProvider(inprocessflagd.NewProvider(
		inprocessflagd.WithSourcURI("localhost:8015"),
		inprocessflagd.WithSourceProviderType(inprocessflagd.SourceTypeGrpc),
    ))
}
```

### Using inprocessflagd.FromEnv()  
By default the flagd provider will read non-empty environment variables to set its own configuration with the lowest priority. Use the `flagd.FromEnv()` option as an argument for the `flagd.NewProvider()` method to give environment variables a higher priority.

| Option name                 | Environment variable name             | Type    | Options      | Default                                |
| --------------------------- | ------------------------------------- | ------- | ------------ | -------------------------------------- |
| host                        | FLAGD_PROXY_HOST                      | string  |              | localhost                              |
| port                        | FLAGD_PROXY_PORT                      | number  |              | 8013                                   |
| tls                         | FLAGD_PROXY_TLS                       | boolean |              | false                                  |
| socketPath                  | FLAGD_PROXY_SOCKET_PATH               | string  |              |                                        |
| certPath                    | FLAGD_PROXY_SERVER_CERT_PATH          | string  |              |                                        |
| sourceURI                   | FLAGD_SOURCE_URI                      | string  |              |                                        |
| sourceProviderType          | FLAGD_SOURCE_PROVIDER_TYPE            | string  |              | grpc                                   |
| sourceSelector              | FLAGD_SOURCE_SELECTOR                 | string  |              |                                        |
| maxSyncRetries              | FLAGD_MAX_SYNC_RETRIES                | int     |              | 0 (0 means unlimited)                  |
| maxSyncRetryInterval        | FLAGD_MAX_SYNC_RETRY_INTERVAL         | int     |              | 60s                                    |

In the event that another configuration option is passed to the `flagd.NewProvider()` method, such as `WithSourceURI("localhost:8015")` then priority is decided by the order in which the options are passed to the constructor from lowest to highest priority.

e.g. below the values set by `FromEnv()` overwrite the value set by `WithSourceURI("localhost:8015")`.
```go
openfeature.SetProvider(flagd.NewProvider(
        flagd.WithSourceURI("localhost:8015"),
        flagd.FromEnv(),
    ))
```

## License

Apache 2.0 - See [LICENSE](./../../LICENSE) for more information.
