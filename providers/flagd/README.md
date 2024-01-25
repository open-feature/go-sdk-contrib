# Flagd Provider

This provider is designed to use flagd's [evaluation protocol](https://github.com/open-feature/schemas/blob/main/protobuf/schema/v1/schema.proto), or locally evaluate flags defined in a flagd [flag definition](https://github.com/open-feature/schemas/blob/main/json/flagd-definitions.json).

## Installation

Use the latest flagd release with [OpenFeature Go SDK](https://github.com/open-feature/go-sdk)

```sh
go get github.com/open-feature/go-sdk-contrib/providers/flagd
go get github.com/open-feature/go-sdk
```

The flagd provider can operate in two modes: [RPC](#remote-resolver-rpc) (evaluation takes place in flagd, via gRPC calls) or [in-process](#in-process-resolver) (evaluation takes place in-process, with the provider getting a ruleset from a compliant sync-source).

### Remote resolver (RPC)

This is the default mode of operation of the provider.
In this mode, provider communicates with [flagd](https://github.com/open-feature/flagd) via the gRPC protocol.
Flag evaluations take place remotely at the connected flagd instance.

To use in this mode, set the provider to the `openfeature` global singleton as shown below (using default values which align with those of `flagd`)

```go
openfeature.SetProvider(flagd.NewProvider())
```  

### In-process resolver

This mode performs flag evaluations locally (in-process).
Flag configurations for evaluation are obtained via gRPC protocol using [sync protobuf schema](https://buf.build/open-feature/flagd/file/main:sync/v1/sync_service.proto) service definition.

Consider following example to create a `FlagdProvider` with in-process evaluations,

```go
provider := flagd.NewProvider(
        flagd.WithInProcessResolver(),
        flagd.WithHost("localhost"),
        flagd.WithPort(8013))
openfeature.SetProvider(provider)
```

In the above example, in-process handlers attempt to connect to a sync service on address `localhost:8013` to obtain [flag definitions](https://github.com/open-feature/schemas/blob/main/json/flagd-definitions.json).

#### Offline mode

In-process resolvers can also work in an offline mode.
To enable this mode, you should provide a [valid flag configuration](https://flagd.dev/reference/flag-definitions/) file with the option `WithOfflineFilePath`.

```go
provider := flagd.NewProvider(
        flagd.WithInProcessResolver(),
        flagd.WithOfflineFilePath(OFFLINE_FLAG_PATH))
openfeature.SetProvider(provider)
```

The provider will attempt to detect file changes, but this is a best-effort attempt as file system events differ between operating systems.
This mode is useful for local development, tests and offline applications.

> [!IMPORTANT]
> Note that you can only use a single flag source (either gRPC or offline file) for the in-process resolver. 
> If both sources are configured, offline mode will be selected.

## Configuration options

Configuration can be provided as constructor options or as environment variables, where constructor options having the highest precedence.

| Option name                                              | Environment variable name      | Type & supported value      | Default   | Compatible resolver |
|----------------------------------------------------------|--------------------------------|-----------------------------|-----------|---------------------|
| WithHost                                                 | FLAGD_HOST                     | string                      | localhost | rpc & in-process    |
| WithPort                                                 | FLAGD_PORT                     | number                      | 8013      | rpc & in-process    |
| WithTLS                                                  | FLAGD_TLS                      | boolean                     | false     | rpc & in-process    |
| WithSocketPath                                           | FLAGD_SOCKET_PATH              | string                      | ""        | rpc & in-process    |
| WithCertificatePath                                      | FLAGD_SERVER_CERT_PATH         | string                      | ""        | rpc & in-process    |
| WithLRUCache<br/>WithBasicInMemoryCache<br/>WithoutCache | FLAGD_CACHE                    | string (lru, mem, disabled) | lru       | rpc                 |
| WithEventStreamConnectionMaxAttempts                     | FLAGD_MAX_EVENT_STREAM_RETRIES | int                         | 5         | rpc                 |
| WithOfflineFilePath                                      | FLAGD_OFFLINE_FLAG_SOURCE_PATH | string                      | ""        | in-process          |

### Overriding behavior

By default, the flagd provider will read non-empty environment variables to set its own configuration with the lowest priority.
Use the `flagd.FromEnv()` option to give environment variables a higher priority.

In the event that another configuration option is passed to the `flagd.NewProvider()` method, such as `flagd.WithPort(8013)` then priority is decided by the order in which the options are passed to the constructor from lowest to highest priority.

e.g. below the values set by `FromEnv()` overwrite the value set by `WithHost("localhost")`.
```go
openfeature.SetProvider(flagd.NewProvider(
        flagd.WithHost("localhost"),
        flagd.FromEnv(),
    ))
```

### Caching

The provider attempts to establish a connection to flagd's event stream (up to 5 times by default).
If the connection is successful and caching is enabled each flag returned with reason `STATIC` is cached until an event is received concerning the cached flag (at which point it is removed from cache).

On invocation of a flag evaluation (if caching is available) an attempt is made to retrieve the entry from cache, if found the flag is returned with reason `CACHED`.
By default, the provider is configured to use LRU caching with up to 1000 entries.
This can be changed through constructor option or environment variable `FLAGD_MAX_CACHE_SIZE`

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
