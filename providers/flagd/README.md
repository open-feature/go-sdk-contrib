# Flagd Provider

![Experimental](https://img.shields.io/badge/experimental-breaking%20changes%20allowed-yellow)
![Alpha](https://img.shields.io/badge/alpha-release-red)

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
flagd.WithoutCache()                    // disables caching of flag evaluations
flagd.WithLRUCache(1000)                // enables LRU caching (see configuring caching section)
flagd.WithBasicInMemoryCache()          // enables basic in memory cache (see configuring caching section)
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
By default the flagd provider will not read environment variables to set its own configuration, however, if the `flagd.FromEnv()` option is set as an argument for the `flagd.NewProvider()` method, then the following environment variables will be checked: `FLAGD_HOST`, `FLAGD_PORT`, `FLAGD_SERVER_CERT_PATH` & `FLAGD_CACHING_DISABLED`.

In the event that another configuration option is passed to the `flagd.NewProvider()` method, such as `flagd.WithPort(8013)` then priority is decided by the order in which the options are passed to the constructor from lowest to highest priority.

e.g. below the values set by `FromEnv()` overwrite the value set by `WithHost("localhost")`.
```go
openfeature.SetProvider(flagd.NewProvider(
        flagd.WithHost("localhost"),
        flagd.FromEnv(),
    ))
```

## Caching

The provider establishes a connection to flagd's event stream. If the connection is successful and caching is enabled each flag returned with reason `STATIC` is cached until an event is received concerning the cached flag (at which point it is removed from cache).

On invocation of a flag evaluation (if caching is available) an attempt is made to retrieve the entry from cache, if found the flag is returned with reason `CACHED`.

By default, the provider is configured to use LRU caching with up to 1000 entries per type of flag.

### Configuration

#### [Least recently used (LRU) caching](https://github.com/hashicorp/golang-lru)

Configure the provider with this caching implementation to set a maximum number, n, of entries of each type (boolean/string/int/float/object) of flag. Once the limit is reached each new entry replaces the least recently used entry.

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

## License

Apache 2.0 - See [LICENSE](./../../LICENSE) for more information.
