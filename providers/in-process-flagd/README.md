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
        flagd.WithHost("localhost"),
        flagd.FromEnv(),
    ))
```

## Caching

The provider attempts to establish a connection to flagd's event stream (up to 5 times by default). If the connection is successful and caching is enabled each flag returned with reason `STATIC` is cached until an event is received concerning the cached flag (at which point it is removed from cache).

On invocation of a flag evaluation (if caching is available) an attempt is made to retrieve the entry from cache, if found the flag is returned with reason `CACHED`.

By default, the provider is configured to use LRU caching with up to 1000 entries.

### Configuration

#### [Least recently used (LRU) caching](https://github.com/hashicorp/golang-lru)

Configure the provider with this caching implementation to set a maximum number, n, of entries. Once the limit is reached each new entry replaces the least recently used entry.

```go
flagd.WithLRUCache(n)
```

#### Basic in memory caching

Configure the provider with this caching implementation if memory limit is no concern.

```go
flagd.WithBasicInMemoryCache()
```

#### Disable caching

```go
flagd.WithoutCache()
```

## Logging

If not configured, logging falls back to the standard Go log package at error level only.

In order to avoid coupling to any particular logging implementation, the provider uses the structured logging [logr](https://github.com/go-logr/logr)
API. This allows integration to any package that implements the layer between their logger and this API.
Thankfully, there is already [integration implementations](https://github.com/go-logr/logr#implementations-non-exhaustive)
for many of the popular logger packages.

```go
var l logr.Logger
l = integratedlogr.New() // replace with your chosen integrator

provider := flagd.NewProvider(flagd.WithLogger(l)) // set the provider's logger
```

[logr](https://github.com/go-logr/logr) uses incremental verbosity levels (akin to named levels but in integer form).
The provider logs `warning` at level `0`, `info` at level `1` and `debug` at level `2`. Errors are always logged.

## License

Apache 2.0 - See [LICENSE](./../../LICENSE) for more information.
