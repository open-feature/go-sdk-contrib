# Unofficial Prefab OpenFeature Provider for GO

[Prefab](https://www.prefab.cloud/) OpenFeature Provider can provide usage for Prefab via OpenFeature Java SDK.

## Installation

To use the provider, you'll need to install [Prefab Go client](https://github.com/prefab-cloud/prefab-cloud-go) and Prefab provider. You can install the packages using the following command

```shell
go get github.com/prefab-cloud/prefab-cloud-go
go get github.com/open-feature/go-sdk-contrib/providers/prefab
```

## Usage
Prefab OpenFeature Provider is using Prefab GO SDK.

### Usage Example

```go
import (
  provider "github.com/open-feature/go-sdk-contrib/providers/prefab/pkg"
  of "github.com/open-feature/go-sdk/openfeature"
  prefab "github.com/prefab-cloud/prefab-cloud-go/pkg"
)

of.SetProvider(provider)
ofClient := of.NewClient("my-app")

evalCtx := of.NewEvaluationContext(
  "",
  map[string]interface{}{
    "user.key":         "key1",
    "team.domain":      "prefab.cloud",
    "team.description": "team1",
  },
)
enabled, _ := ofClient.BooleanValue(context.Background(), "always_on_gate", false, evalCtx)

value, _ := ofClient.StringValue(context.Background(), "string", "fallback", evalCtx)

slice, err := ofClient.ObjectValueDetails(context.Background(), "sample_list", []string{"a2", "b2"}, evalCtx)

of.Shutdown()

```
See [provider_test.go](./pkg/provider_test.go) for more information.

## Notes
Some Prefab custom operations are supported from the Prefab client via PrefabClient.

## Prefab Provider Tests Strategies

Unit test based on Prefab yaml config file. 
Can be enhanced pending [JSON dump data source](https://github.com/prefab-cloud/prefab-cloud-go/blob/0e3d5a4ba7171bbc4484cc99ccaad4c0c32d7e81/README.md?plain=1#L58)
JSON evaluation not tested properly until then.
See [provider_test.go](./pkg/provider_test.go) for more information.
