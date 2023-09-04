# Flagd Provider

[Flagd](https://github.com/open-feature/flagd) is a simple command line tool for fetching and presenting feature flags to services. It is designed to conform to OpenFeature schema for flag definitions. This repository and package provides the client side code for interacting with it via the [OpenFeature SDK](https://github.com/open-feature/go-sdk).

## Setup
Using remote buf packages requires a one-time registry configuration:
```shell
export GOPRIVATE=buf.build/gen/go
```
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
flagd.WithTLS(certPath string)          // default of false, if certPath is not given, system certs are used
flagd.WithoutCache()                    // disables caching of flag evaluations
flagd.WithLRUCache(1000)                // enables LRU caching (see configuring caching section)
flagd.WithBasicInMemoryCache()          // enables basic in memory cache (see configuring caching section)
flagd.WithLogger(logger)                // sets a custom logger (see logging section)
flagd.WithOtelInterceptor(bool)         // enable or disable OpenTelemetry interceptor for flagd communication
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
        flagd.WithPort(8013),
    ))
}
```

### Using flagd.FromEnv()  
By default the flagd provider will read non-empty environment variables to set its own configuration with the lowest priority. Use the `flagd.FromEnv()` option as an argument for the `flagd.NewProvider()` method to give environment variables a higher priority.

| Option name           | Environment variable name      | Type    | Options            | Default   |
|-----------------------|--------------------------------|---------|--------------------|-----------|
| host                  | FLAGD_HOST                     | string  |                    | localhost |
| port                  | FLAGD_PORT                     | number  |                    | 8013      |
| tls                   | FLAGD_TLS                      | boolean | true, false        | false     |
| socketPath            | FLAGD_SOCKET_PATH              | string  |                    |           |
| certPath              | FLAGD_SERVER_CERT_PATH         | string  |                    |           |
| cache                 | FLAGD_CACHE                    | string  | lru, mem, disabled | lru       |
| maxCacheSize          | FLAGD_MAX_CACHE_SIZE           | int     |                    | 1000      |
| maxEventStreamRetries | FLAGD_MAX_EVENT_STREAM_RETRIES | int     |                    | 5         |

In the event that another configuration option is passed to the `flagd.NewProvider()` method, such as `flagd.WithPort(8013)` then priority is decided by the order in which the options are passed to the constructor from lowest to highest priority.

e.g. below the values set by `FromEnv()` overwrite the value set by `WithHost("localhost")`.
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

## Supported Events

The flagd provider emits `PROVIDER_READY`, `PROVIDER_ERROR` and `PROVIDER_CONFIGURATION_CHANGED` events.

| SDK event                        | Originating action in flagd                                                     |
|----------------------------------|---------------------------------------------------------------------------------|
| `PROVIDER_READY`                 | The streaming connection with flagd has been established.                       |
| `PROVIDER_ERROR`                 | The streaming connection with flagd has been broken.                            |
| `PROVIDER_CONFIGURATION_CHANGED` | A flag configuration (default value, targeting rule, etc) in flagd has changed. |

For general information on events, see the [official documentation](https://openfeature.dev/docs/reference/concepts/events).

## Flag Metadata

The flagd provider currently support following flag evaluation metadata,

| Field   | Type   | Value                                             |
|---------|--------|---------------------------------------------------|
| `scope` | string | "selector" set for the associated source in flagd |

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
