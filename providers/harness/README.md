# Unofficial Harness OpenFeature GO Provider

 [Harness](https://developer.harness.io/docs/feature-flags) OpenFeature Provider can provide usage for Harness via OpenFeature GO SDK.

# Installation

To use the Harness provider, you'll need to install [Harness Go client](github.com/harness/ff-golang-server-sdk) and Harness provider. You can install the packages using the following command

```shell
go get github.com/harness/ff-golang-server-sdk
go get github.com/open-feature/go-sdk-contrib/providers/harness
```

## Concepts
* Provider Object evaluation gets Harness JSON evaluation.
* Other provider types evaluation gets Harness matching type evaluation.

## Usage
Harness OpenFeature Provider is using Harness GO SDK.

### Evaluation Context
Evaluation Context is mapped to Harness [target](https://developer.harness.io/docs/feature-flags/ff-sdks/server-sdks/feature-flag-sdks-go-application/#add-a-target).
OpenFeature targetingKey is mapped to _Identifier_, _Name_ is mapped to _Name_ and other fields are mapped to Attributes 
fields.

### Usage Example

```go
import (
  harness "github.com/harness/ff-golang-server-sdk/client"
  harnessProvider "github.com/open-feature/go-sdk-contrib/providers/harness/pkg"
)

providerConfig := harnessProvider.ProviderConfig{
    Options: []harness.ConfigOption{
        harness.WithWaitForInitialized(true),
        harness.WithURL(URL),
        harness.WithStreamEnabled(false),
        harness.WithHTTPClient(http.DefaultClient),
        harness.WithStoreEnabled(false),
    },
    SdkKey: ValidSDKKey,
}

provider, err := harnessProvider.NewProvider(providerConfig)
if err != nil {
    t.Fail()
}
err = provider.Init(of.EvaluationContext{})
if err != nil {
    t.Fail()
}

ctx := context.Background()

of.SetProvider(provider)
ofClient := of.NewClient("my-app")

evalCtx := of.NewEvaluationContext(
    "john",
    map[string]interface{}{
        "Firstname": "John",
        "Lastname":  "Doe",
        "Email":     "john@doe.com",
    },
)
enabled, err := ofClient.BooleanValue(context.Background(), "TestTrueOn", false, evalCtx)
if enabled == false {
    t.Fatalf("Expected feature to be enabled")
}

```
See [provider_test.go](./pkg/provider_test.go) for more information.

