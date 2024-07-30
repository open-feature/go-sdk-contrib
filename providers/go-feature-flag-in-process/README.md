# GO Feature Flag In Process Provider

> [!WARNING]  
> This provider is in process and has as dependency to GO Feature Flag completely; it means that it will include a lot of dependencies in your project.
>
> This provider is recommended if you want an OpenFeature facade in front of the GO Feature Flag go module.
> If you aim to use the relay proxy, please check the [GO Feature Flag provider](../go-feature-flag/README.md).

## Install dependencies

The first things we will do are to install the **Open Feature SDK** and the **GO Feature Flag In Process provider**.

```shell
go get github.com/open-feature/go-sdk-contrib/providers/go-feature-flag-in-process
```

## Initialize your Open Feature provider

You can check the [GO Feature Flag documentation website](https://docs.gofeatureflag.org) to look how to configure the
GO module.

#### Example
```go
options := gofeatureflaginprocess.ProviderOptions{
  GOFeatureFlagConfig: &ffclient.Config{
      PollingInterval: 10 * time.Second,
      Context:         context.Background(),
      Retriever: &fileretriever.Retriever{
        Path: "../testutils/module/flags.yaml",
      },
    },
}
provider, _ := gofeatureflaginprocess.NewProviderWithContext(ctx, options)
```

## Initialize your Open Feature client

To evaluate a flag, you need to have an OpenFeature configured in your app.
This code block shows you how you can create a client that you can use in your application.

```go
import (
  // ...
  gofeatureflaginprocess "github.com/open-feature/go-sdk-contrib/providers/go-feature-flag/pkg"
  of "github.com/open-feature/go-sdk/openfeature"
)

// ...

options := gofeatureflaginprocess.ProviderOptions{
    GOFeatureFlagConfig: &ffclient.Config{
        PollingInterval: 10 * time.Second,
        Context:         context.Background(),
        Retriever: &fileretriever.Retriever{
            Path: "../testutils/module/flags.yaml",
        },
    },
}
provider, err := gofeatureflaginprocess.NewProviderWithContext(ctx, options)
of.SetProvider(provider)
client := of.NewClient("my-app")
```

## Evaluate your flag

This code block explain how you can create an `EvaluationContext` and use it to evaluate your flag.


> In this example we are evaluating a `boolean` flag, but other types are available.
>
> **Refer to the [Open Feature documentation](https://openfeature.dev/docs/reference/concepts/evaluation-api#basic-evaluation) to know more about it.**

```go
evaluationCtx := of.NewEvaluationContext(
    "1d1b9238-2591-4a47-94cf-d2bc080892f1",
    map[string]interface{}{
      "firstname", "john",
      "lastname", "doe",
      "email", "john.doe@gofeatureflag.org",
      "admin", true,
      "anonymous", false,
    })
adminFlag, _ := client.BoolValue(context.TODO(), "flag-only-for-admin", false, evaluationCtx)
if adminFlag {
   // flag "flag-only-for-admin" is true for the user
} else {
  // flag "flag-only-for-admin" is false for the user
}
```