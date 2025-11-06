# OpenFeature Remote Evaluation Protocol Provider

This is the Go implementation of the OFREP provider.
The provider works by evaluating flags against OFREP single flag evaluation endpoint.

## Installation

Use OFREP provider with the latest OpenFeature Go SDK

```sh
go get github.com/open-feature/go-sdk-contrib/providers/ofrep
```

## Usage

Initialize the provider with the URL of the OFREP implementing service,

```go
ofrepProvider := ofrep.NewProvider("http://localhost:8016")
```

Then, register the provider with the OpenFeature Go SDK and use derived clients for flag evaluations,

```go
openfeature.SetProvider(ofrepProvider)
```

## Configuration

You can configure the provider using following configuration options,

| Configuration option | Details                                                                                                                 |
| -------------------- | ----------------------------------------------------------------------------------------------------------------------- |
| WithApiKeyAuth       | Set the token to be used with "X-API-Key" header                                                                        |
| WithBearerToken      | Set the token to be used with "Bearer" HTTP Authorization schema                                                        |
| WithClient           | Provide a custom, pre-configured http.Client for OFREP service communication                                            |
| WithHeaderProvider   | Register a custom header provider for OFREP calls. You may utilize this for custom authentication/authorization headers |
| WithHeader           | Set a custom header to be used for authorization                                                                        |
| WithBaseURI          | Set the base URI of the OFREP service                                                                                   |
| WithTimeout          | Set the timeout for the http client used for communication with the OFREP service (ignored if custom client is used)    |
| WithFromEnv          | Configure the provider using environment variables (experimental)                                                       |

For example, consider below example which sets bearer token and provides a customized http client,

```go
provider := ofrep.NewProvider(
    "http://localhost:8016",
    ofrep.WithBearerToken("TOKEN"),
    ofrep.WithClient(&http.Client{
        Timeout: 1 * time.Second,
    }))
```

### Environment Variable Configuration (Experimental)

You can use the `WithFromEnv()` option to configure the provider using environment variables:

```go
provider := ofrep.NewProvider(
    "http://localhost:8016",
    ofrep.WithFromEnv())
```

Supported environment variables:

| Environment Variable | Description                                                           | Example                                   |
| -------------------- | --------------------------------------------------------------------- | ----------------------------------------- |
| OFREP_ENDPOINT       | Base URI for the OFREP service (overrides the baseUri parameter)      | `http://localhost:8016`                   |
| OFREP_TIMEOUT        | Timeout duration for HTTP requests (ignored if custom client is used) | `30s`, `1m` or raw `5000` in milliseconds |
| OFREP_HEADERS        | Comma-separated custom headers                                        | `Key1=Value1,Key2=Value2`                 |
