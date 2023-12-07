# Flipt OpenFeature Provider (Go)

This repository and package provides a [Flipt](https://github.com/flipt-io/flipt) [OpenFeature Provider](https://docs.openfeature.dev/docs/specification/sections/providers) for interacting with the Flipt service backend using the [OpenFeature Go SDK](https://github.com/open-feature/go-sdk).

From the [OpenFeature Specification](https://docs.openfeature.dev/docs/specification/sections/providers):

> Providers are the "translator" between the flag evaluation calls made in application code, and the flag management system that stores flags and in some cases evaluates flags.

## Requirements

- Go 1.20+
- A running instance of [Flipt](https://www.flipt.io/docs/installation)

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
    flipt.WithAddress("localhost:9000"),
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
