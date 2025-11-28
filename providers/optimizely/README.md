# Unofficial Optimizely OpenFeature Provider for Go

OpenFeature Go provider implementation for [Optimizely](https://optimizely.com) that uses the official [Optimizely Go SDK](https://github.com/optimizely/go-sdk).

## Installation

```shell
# Optimizely SDK
go get github.com/optimizely/go-sdk/v2

# OpenFeature SDK
go get github.com/open-feature/go-sdk/openfeature
go get github.com/open-feature/go-sdk-contrib/providers/optimizely
```

## Usage

```go
import (
    "github.com/open-feature/go-sdk/openfeature"
    "github.com/optimizely/go-sdk/v2/pkg/client"
    optimizely "github.com/open-feature/go-sdk-contrib/providers/optimizely"
)

func main() {
    optimizelyClient, err := (&client.OptimizelyFactory{
        SDKKey: "your-sdk-key",
    }).Client()
    if err != nil {
        panic(err)
    }

    provider := optimizely.NewProvider(optimizelyClient)
    openfeature.SetProviderAndWait(provider)
    defer openfeature.Shutdown()

    ofClient := openfeature.NewClient("my-app")

    evalCtx := openfeature.NewEvaluationContext("user-123", map[string]any{
        "email": "user@example.com",
    })
    value, err := ofClient.BooleanValue(ctx, "my_flag", false, evalCtx)
}
```

See [example/example.go](./example/example.go) for a complete example.

## Evaluation Context

The `targetingKey` is required and maps to the Optimizely user ID. Additional attributes are passed to Optimizely for audience targeting.

## Variable Key Selection

Optimizely flags can have multiple variables. By default, the provider looks for a variable named `"value"`. Specify a different variable using the `variableKey` attribute:

```go
evalCtx := openfeature.NewEvaluationContext("user-123", map[string]any{
    "variableKey": "button_color",
})
```

## References
* [Optimizely Go SDK documentation](https://docs.developers.optimizely.com/feature-experimentation/docs/go-sdk)
