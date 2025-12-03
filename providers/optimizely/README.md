# Unofficial Optimizely OpenFeature Provider for Go

OpenFeature Go provider implementation for [Optimizely](https://optimizely.com) that uses the official [Optimizely Go SDK](https://github.com/optimizely/go-sdk).

## Installation

```shell
go get github.com/optimizely/go-sdk/v2
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

## Flag Variable Mapping

The evaluation method you use depends on the number of variables configured in your Optimizely flag:

| Variables | Evaluation Method | Returns |
|-----------|-------------------|---------|
| 0 | `BooleanEvaluation` | Flag enabled state (`true`/`false`) |
| 1 | Type-specific method | The single variable's value |
| N (>1) | `ObjectEvaluation` | Map of all variable names to values |

### Flags with No Variables

Use `BooleanEvaluation` to get the flag's enabled state:

```go
// Returns true if flag is enabled for user, false otherwise
enabled, err := ofClient.BooleanValue(ctx, "feature_flag", false, evalCtx)
```

### Flags with One Variable

Use the evaluation method matching the variable's type:

```go
// String variable
message, err := ofClient.StringValue(ctx, "welcome_message_flag", "Hello", evalCtx)

// Integer variable
limit, err := ofClient.IntValue(ctx, "rate_limit_flag", 100, evalCtx)

// Double variable
price, err := ofClient.FloatValue(ctx, "price_flag", 9.99, evalCtx)

// Boolean variable
enabled, err := ofClient.BooleanValue(ctx, "dark_mode_flag", false, evalCtx)

// Any type (returns the value as interface{})
value, err := ofClient.ObjectValue(ctx, "config_flag", nil, evalCtx)
```

### Flags with Multiple Variables

Use `ObjectEvaluation` to get all variables as a map:

```go
// Returns map[string]any with all variable values
config, err := ofClient.ObjectValue(ctx, "ui_config_flag", nil, evalCtx)
if err == nil {
    configMap := config.(map[string]any)
    buttonColor := configMap["button_color"].(string)
    fontSize := configMap["font_size"].(int)
}
```

### Error Cases

Using the wrong evaluation method for your flag configuration returns an error:

- `BooleanEvaluation` on a flag with multiple variables returns an error
- `StringEvaluation`, `IntEvaluation`, `FloatEvaluation` on a flag with 0 or multiple variables returns an error
- Type mismatches (e.g., `IntEvaluation` on a string variable) return an error

## References

- [Optimizely Go SDK documentation](https://docs.developers.optimizely.com/feature-experimentation/docs/go-sdk)
- [OpenFeature Go SDK](https://github.com/open-feature/go-sdk)
