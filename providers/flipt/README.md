# Flipt OpenFeature Provider (Go)

This repository and package provides a [Flipt](https://github.com/flipt-io/flipt) [OpenFeature Provider](https://docs.openfeature.dev/docs/specification/sections/providers) for interacting with the Flipt service backend using the [OpenFeature Go SDK](https://github.com/open-feature/go-sdk).

From the [OpenFeature Specification](https://docs.openfeature.dev/docs/specification/sections/providers):

> Providers are the "translator" between the flag evaluation calls made in application code, and the flag management system that stores flags and in some cases evaluates flags.

## Requirements

- Go 1.20+
- A running instance of [Flipt](https://www.flipt.io/docs/installation)

## Breaking Changes

### v0.2.0

Version [v0.2.0](https://github.com/flipt-io/flipt-openfeature-provider-go/releases/tag/v0.2.0) of this client correlates Boolean flag evaluations to [Boolean flag types](https://www.flipt.io/docs/concepts#boolean-flags) on the Flipt server. Upgrading to this version will require you to convert your flags that were using Boolean evaluation to the Boolean flag type on the Flipt server.

:warning: Boolean flag evaluations were introduced in Flipt server (>= [v.1.24.0](https://github.com/flipt-io/flipt/releases/tag/v1.24.0)).

### v0.1.5

Version [v0.1.5](https://github.com/flipt-io/flipt-openfeature-provider-go/releases/tag/v0.1.5) of this client introduced a change to use a newer version of the Flipt API which requires use of the `namespace` parameter. This is to support the new namespace functionality added to [Flipt v1.20.0](https://www.flipt.io/docs/reference/overview#v1-20-0).

This client uses the `default` namespace by default. If you are using a different namespace, you will need to set the `namespace` parameter when creating the provider:

```go
provider := flipt.NewProvider(flipt.ForNamespace("my-namespace"))
```

:warning: If you are running an older version of Flipt server (< [v1.20.0](https://github.com/flipt-io/flipt/releases/tag/v1.20.0)), you should use a pre 0.1.5 version of this client.

## Usage

### Installation

```bash
go get github.com/open-feature/go-sdk-contrib/providers/flipt
```

### Example

```go
package main

import (
    "context"

    "github.com/open-feature/go-sdk-contrib/providers/flipt"
    "github.com/open-feature/go-sdk/pkg/openfeature"
)


func main() {
    // http://localhost:8080 is the default Flipt address
    openfeature.SetProvider(flipt.NewProvider())

    client := openfeature.NewClient("my-app")
    value, err := client.BooleanValue(context.Background(), "v2_enabled", false, openfeature.EvaluationContext{
        TargetingKey: "tim@apple.com",
        Attributes: map[string]interface{}{
            "favorite_color": "blue",
        },
    })

    if err != nil {
        panic(err)
    }

    if value {
        // do something
    } else {
        // do something else
    }
}
```

## Configuration

The Flipt provider allows you to communicate with Flipt over either HTTP(S) or GRPC, depending on the address provided.

### HTTP(S)

```go
provider := flipt.NewProvider(flipt.WithAddress("https://localhost:443"))
```

#### Unix Socket

```go
provider := flipt.NewProvider(flipt.WithAddress("unix:///path/to/socket"))
```

### GRPC

#### HTTP/2

```go
type Token string

func (t Token) ClientToken() (string, error) {
    return t, nil
}

provider := flipt.NewProvider(
    flipt.WithAddress("grpc://localhost:9000"),
    flipt.WithCertificatePath("/path/to/cert.pem"), // optional
    flipt.WithClientProvider(Token("a-client-token")), // optional
)
```

#### Unix Socket

```go
provider := flipt.NewProvider(
    flipt.WithAddress("unix:///path/to/socket"),
)
```
