# Unofficial Statsig OpenFeature GO Provider

[Statsig](https://statsig.com/) OpenFeature Provider can provide usage for Statsig via OpenFeature GO SDK.

# Installation

To use the provider, you'll need to install [Statsig Go client](github.com/statsig-io/go-sdk) and Statsig provider. You can install the packages using the following command

```shell
go get github.com/statsig-io/go-sdk
go get github.com/open-feature/go-sdk-contrib/providers/statsig
```

## Concepts
* Boolean evaluation gets [gate](https://docs.statsig.com/server/javaSdk#checking-a-gate) status.
* String/Integer/Double evaluations evaluation gets [Dynamic config](https://docs.statsig.com/server/javaSdk#reading-a-dynamic-config) or [Layer](https://docs.statsig.com/server/javaSdk#getting-an-layerexperiment) evaluation.
  As the key represents an inner attribute, feature config is required as a parameter with data needed for evaluation.
  For an example of dynamic config of product alias, need to differentiate between dynamic config or layer, and the dynamic config name.
* Object evaluation gets a structure representing the dynamic config or layer.
* [Private Attributes](https://docs.statsig.com/server/javaSdk#private-attributes) are supported as 'privateAttributes' context key.


## Usage
Statsig OpenFeature Provider is using Statsig GO SDK.

## Usage Example

```go
import (
  statsigProvider "github.com/open-feature/go-sdk-contrib/providers/statsig/pkg"
	of "github.com/open-feature/go-sdk/openfeature"
	statsig "github.com/statsig-io/go-sdk"
)

of.SetProvider(provider)
ofClient := of.NewClient("my-app")

evalCtx := of.NewEvaluationContext(
  "",
  map[string]interface{}{
    "UserID": "123",
  },
)
enabled, _ := ofClient.BooleanValue(context.Background(), "always_on_gate", false, evalCtx)

featureConfig := statsigProvider.FeatureConfig{
  FeatureConfigType: statsigProvider.FeatureConfigType("CONFIG"),
  Name:              "test_config",
}

evalCtx := of.NewEvaluationContext(
  "",
  map[string]interface{}{
    "UserID":         "123",
    "Email":          "testuser1@statsig.com",
    "feature_config": featureConfig,
  },
)
expected := "statsig"
value, _ := ofClient.StringValue(context.Background(), "string", "fallback", evalCtx)

of.Shutdown()

```
See [provider_test.go](./pkg/provider_test.go) for more information.


## Notes
Some Statsig custom operations are supported from the Statsig client via statsig.

## Statsig Provider Tests Strategies

Unit test based on Statsig [BootstrapValues](https://docs.statsig.com/server/golangSDK#statsig-options) config file. 
As it is limited, evaluation context based tests are limited.
See [provider_test.go](./pkg/provider_test.go) for more information.

