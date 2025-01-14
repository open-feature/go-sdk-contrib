# GO Feature Flag GO Provider

GO Feature Flag provider allows you to connect to your GO Feature Flag instance.  

[GO Feature Flag](https://gofeatureflag.org) believes in simplicity and offers a simple and lightweight solution to use feature flags.  
Our focus is to avoid any complex infrastructure work to use GO Feature Flag.

This is a complete feature flagging solution with the possibility to target only a group of users, use any types of flags, store your configuration in various location and advanced rollout functionality. You can also collect usage data of your flags and be notified of configuration changes.


# GO SDK usage

## Install dependencies

The first things we will do are to install the **Open Feature SDK** and the **GO Feature Flag provider**.

```shell
go get github.com/open-feature/go-sdk-contrib/providers/go-feature-flag
```

## Initialize your Open Feature provider

### Connecting to the relay proxy

This provider has to connect with the **relay proxy**, to do that you should set the field `Endpoint` in the options.  
By default it will use a default `HTTPClient` with a **timeout** configured at **10000** milliseconds. You can change
this configuration by providing your own configuration of the `HTTPClient`.

#### Example
```go
options := gofeatureflag.ProviderOptions{
  Endpoint: "http://localhost:1031",
  HTTPClient: &http.Client{
    Timeout:   1 * time.Second,
  },
}
provider, _ := gofeatureflag.NewProviderWithContext(ctx, options)
```

## Initialize your Open Feature client

To evaluate the flag you need to have an Open Feature configured in you app.
This code block shows you how you can create a client that you can use in your application.

```go
import (
  // ...
  gofeatureflag "github.com/open-feature/go-sdk-contrib/providers/go-feature-flag/pkg"
  of "github.com/open-feature/go-sdk/openfeature"
)

// ...

options := gofeatureflag.ProviderOptions{
    Endpoint: "http://localhost:1031",
}
provider, err := gofeatureflag.NewProviderWithContext(ctx, options)
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
		"firstname": "john",
		"lastname":  "doe",
		"email":     "john.doe@gofeatureflag.org",
		"admin":     true,
		"anonymous": false,
	})
adminFlag, _ := client.BooleanValue(context.TODO(), "flag-only-for-admin", false, evaluationCtx)
if adminFlag {
   // flag "flag-only-for-admin" is true for the user
} else {
  // flag "flag-only-for-admin" is false for the user
}
```
