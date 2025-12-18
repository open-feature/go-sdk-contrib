# Flagsmith OpenFeature Provider for Go

[Flagsmith](https://www.flagsmith.com/) is an open source feature flagging and remote config service.

## Installation

Install the OpenFeature Go SDK, [Flagsmith Go client](https://github.com/Flagsmith/flagsmith-go-client) and OpenFeature provider:

```shell
go get github.com/Flagsmith/flagsmith-go-client/v4
go get go.openfeature.dev/v2
go get go.openfeature.dev/contrib/providers/flagsmith/v2
```

## Usage

Refer to the [Go client documentation](https://docs.flagsmith.com/clients/server-side?language=go) for details
on creating and configuring the Flagsmith client.

## Example

See [`example_test.go`](./example_test.go) for a runnable example.

## Provider-specific options

### `WithUsingBooleanConfigValue`

Determines whether to resolve a feature value as a boolean or use the isFeatureEnabled as the flag itself.
i.e: if the flag is enabled, the value will be true, otherwise it will be false
