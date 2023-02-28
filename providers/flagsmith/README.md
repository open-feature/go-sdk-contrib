# Flagsmith OpenFeature GO Provider

[Flagsmith](https://flagsmith.com/) provides an all-in-one platform for developing, implementing, and managing your feature flags.

# Installation

To use the Flagsmith provider, you'll need to install [flagsmith Go client](https://github.com/Flagsmith/flagsmith-go-client) and flagsmith provider. You can install the packages using the following command

```shell
go get github.com/Flagsmith/flagsmith-go-client/v2
go get github.com/open-feature/go-sdk-contrib/providers/flagsmith
```

## Usage
Here's an example of how you can use the Flagsmith provider:

```go
import (
    flagsmithClient "github.com/Flagsmith/flagsmith-go-client/v2"
    of "github.com/open-feature/go-sdk/pkg/openfeature"
    flagsmith "github.com/open-feature/go-sdk-contrib/providers/flagsmith/pkg"
)
    ...
    // Intialise the flagsmith client
	client := flagsmithClient.NewClient(os.Getenv("FLAGSMITH_ENVIRONMENT_KEY"))

    // Inlitalise the flagsmith provider
	provider := flagsmith.NewProvider(client, flagsmith.WithUsingBooleanConfigValue())

	of.SetProvider(provider)

    // Create open feature client
	ofClient := of.NewClient("my-app")

    // Start interacting with the client
	Value, err := ofClient.BooleanValue(context.Background(), "bool_feature", defaultboolValue,  evalCtx)

    ...

```
In the example above, we first import the necessary packages including the Flagsmith client, the OpenFeature SDK, and the Flagsmith provider.
We then initialize the [Flagsmith client](https://docs.flagsmith.com/clients/server-side) with the `FLAGSMITH_ENVIRONMENT_KEY` environment variable.
We initialize the Flagsmith provider with the client and an optional configuration option, and set the provider(`of.SetProvider()`)

### Options
- WithUsingBooleanConfigValue: Determines whether to resolve a feature value as a boolean or use the isFeatureEnabled as the flag itself.
i.e: if the flag is enabled, the value will be true, otherwise it will be false
