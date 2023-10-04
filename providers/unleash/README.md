# Unofficial Unleash OpenFeature GO Provider

 [Unleash](https://getunleash.io) OpenFeature Provider can provide usage for Unleash via OpenFeature GO SDK.

# Installation

To use the Unleash provider, you'll need to install [unleash Go client](github.com/Unleash/unleash-client-go/v3) and unleash provider. You can install the packages using the following command

```shell
go get github.com/Unleash/unleash-client-go/v3
go get github.com/open-feature/go-sdk-contrib/providers/unleash
```

## Concepts
* Boolean evaluation gets feature enabled status.
* String evaluation gets feature variant value.

## Usage
Unleash OpenFeature Provider is using Unleash GO SDK.

## Usage Example

```go
import (
  "github.com/Unleash/unleash-client-go/v3"
  unleashProvider "github.com/open-feature/go-sdk-contrib/providers/unleash/pkg"
)

providerConfig := unleashProvider.ProviderConfig{
  Options: []unleash.ConfigOption{
    unleash.WithListener(&unleash.DebugListener{}),
    unleash.WithAppName("my-application"),
    unleash.WithRefreshInterval(5 * time.Second),
    unleash.WithMetricsInterval(5 * time.Second),
    unleash.WithStorage(&unleash.BootstrapStorage{Reader: demoReader}),
    unleash.WithUrl("https://localhost:4242"),
  },
}

provider, err := unleashProvider.NewProvider(providerConfig)
err = provider.Init(of.EvaluationContext{})

ctx := context.Background()

of.SetProvider(provider)
ofClient := of.NewClient("my-app")

evalCtx := of.NewEvaluationContext(
  "",
  map[string]interface{}{
    "UserId": "111",
  },
)
enabled, err := ofClient.BooleanValue(context.Background(), "users-flag", false, evalCtx)

evalCtx := of.NewEvaluationContext(
  "",
  map[string]interface{}{},
)
value, err := ofClient.StringValue(context.Background(), "variant-flag", "", evalCtx)

```
See [provider_test.go](./pkg/provider_test.go) for more information.


### Additional Usage Details

* When default value is used and returned, default variant is not used and variant name is not set.
* json/csv payloads are evaluated via object evaluation as what returned from Unleash - string, wrapped with Value.
* Additional evaluation data can be received via flag metadata, such as:
  * *enabled* - boolean
