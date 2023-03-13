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
    // Initialize the flagsmith client
	client := flagsmithClient.NewClient(os.Getenv("FLAGSMITH_ENVIRONMENT_KEY"))

    // Initialize the flagsmith provider
	provider := flagsmith.NewProvider(client, flagsmith.WithUsingBooleanConfigValue())

	of.SetProvider(provider)

    // Create open feature client
	ofClient := of.NewClient("my-app")

    // Start interacting with the client
	Value, err := ofClient.BooleanValue(context.Background(), "bool_feature", defaultboolValue,  evalCtx)
    ....

    // With traits
    traitKey := "some_key"
    traitValue := "some_value"

	evalCtx := of.NewEvaluationContext(
            "openfeature_user",
            map[string]interface{}{
                 traitKey:traitValue
            },
        )
	valueForIdentity, err := ofClient.BooleanValue(context.Background(), "bool_feature", defaultboolValue,  evalCtx)
    ...

```
You can find the flagsmith client document [here](https://docs.flagsmith.com/clients/server-side)

### Options
- `WithUsingBooleanConfigValue`: Determines whether to resolve a feature value as a boolean or use the isFeatureEnabled as the flag itself.
i.e: if the flag is enabled, the value will be true, otherwise it will be false
