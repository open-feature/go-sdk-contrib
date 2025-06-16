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

    flipt "github.com/open-feature/go-sdk-contrib/providers/flipt/pkg/provider"
    "github.com/open-feature/go-sdk/openfeature"
)


func main() {
    // http://localhost:8080 is the default Flipt address
    err := openfeature.SetProviderAndWait(flipt.NewProvider())
    if err != nil {
      panic(err)
    }

    client := openfeature.NewClient("my-app")
    value, err := client.BooleanValue(context.Background(), "v2_enabled", false, openfeature.NewEvaluationContext(
        "tim@apple.com",
        map[string]any{
            "favorite_color": "blue",
        },
    ))

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

The Flipt provider allows you to change the [namespace](https://docs.flipt.io/concepts#namespaces) that the evaluation is performed against. If not provided, it defaults to the `Default` namespace:

### Target Namespace

```go
provider := flipt.NewProvider(flipt.ForNamespace("your-namespace"))
```

### Protocol

The Flipt provider allows you to communicate with Flipt over either HTTP(S) or GRPC, depending on the address provided.

#### HTTP(S)

```go
provider := flipt.NewProvider(flipt.WithAddress("https://localhost:443"))
```

##### Unix Socket

```go
provider := flipt.NewProvider(flipt.WithAddress("unix:///path/to/socket"))
```

#### GRPC

##### HTTP/2

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

##### Unix Socket

```go
provider := flipt.NewProvider(
    flipt.WithAddress("unix:///path/to/socket"),
)
```
