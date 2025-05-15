OpenFeature Multi-Provider
------------

The Multi-Provider allows you to use multiple underlying providers as sources of flag data for the OpenFeature server SDK. 
When a flag is being evaluated, the Multi-Provider will consult each underlying provider it is managing in order to 
determine the final result. Different evaluation strategies can be defined to control which providers get evaluated and 
which result is used.

The Multi-Provider is a powerful tool for performing migrations between flag providers, or combining multiple providers 
into a single feature flagging interface. For example:

- **Migration**: When migrating between two providers, you can run both in parallel under a unified flagging interface. 
As flags are added to the new provider, the Multi-Provider will automatically find and return them, falling back to the old provider 
if the new provider does not have
- **Multiple Data Sources**: The Multi-Provider allows you to seamlessly combine many sources of flagging data, such as 
environment variables, local files, database values and SaaS hosted feature management systems.

# Installation

```sh
go get github.com/open-feature/go-sdk-contrib/providers/multi-provider
go get github.com/open-feature/go-sdk
```

# Usage

```go
import (
	"github.com/open-feature/go-sdk/openfeature"
	mp "github.com/open-feature/go-sdk-contrib/providers/multi-provider"
)

providers := make(mp.ProviderMap)
providers["providerA"] = providerA
providers["providerB"] = providerB
provider, err := mp.NewMultiProvider(providers, mp.StrategyFirstMatch, WithLogger(myLogger))
openfeature.SetProvider(provider)
```

# Options

- `WithTimeout` - the duration is used for the total timeout across parallel operations. If none is set it will default 
to 5 seconds. This is not supported for `FirstMatch` yet, which executes sequentially
- `WithFallbackProvider` - Used for setting a fallback provider for the `Comparison` strategy
- `WithLogger` - Provides slog support using the specified logger
- `WithLoggerDefault` - Default setting. Uses the slog default logger
- `WithoutLogging` - Disables internal logging of the multiprovider
- `WithCustomStrategy` - Allows for passing in an instance of a custom `Strategy` implementation. Must be used in 
conjunction with the `StrategyCustom` `EvaluationStrategy` parameter.
- `WithGlobalHooks` - Sets any hooks that should be executed globally across all internal providers. For hooks targeting
specific providers they should either be attached directly to the provider or use `WithProviderHooks`
- `WithProviderHooks` - Sets any hooks that should be executed only for a specific named provider

# Strategies

There are multiple strategies that can be used to determine the result returned to the caller. A strategy must be set at
initialization time.

There are 3 strategies available currently:

- _First Match_
- _First Success_
- _Comparison_

## First Match Strategy

The first match strategy works by **sequentially**  calling each provider in the order that they are provided to the mutli-provider.
The first provider that returns a result. It will try calling the next provider whenever it encounters a `FLAG_NOT_FOUND`
error. However, if a provider returns an error other than `FLAG_NOT_FOUND` the provider will stop and return the default
value along with setting the error details if a detailed request is issued. (allow changing this behavior?)

## First Success Strategy

The First Success strategy works by calling each provider in **parallel**. The first provider that returns a response
with no errors is returned and all other calls are cancelled. If no provider provides a successful result the default
value will be returned to the caller.

## Comparison

The Comparison strategy works by calling each provider in **parallel**. All results are collected from each provider and
then the resolved results are compared to each other. If they all agree then that value is returned. If not and a fallback
provider is specified then the fallback will be executed. If no fallback is configured then the default value will be 
returned. If a provider returns `FLAG_NOT_FOUND` that is not included in the comparison. If all providers
return not found then the default value is returned. Finally, if any provider returns an error other than `FLAG_NOT_FOUND`
the evaluation immediately stops and that error result is returned. This strategy does NOT support `ObjectEvaluation`

## Custom

Users can opt to write their own strategy by implementing the interface if they have needs that the three built-in
strategies cannot meet. When setting the `StrategyCustom` strategy make sure to pass in an instance of your `Strategy`
implementation using the `WithCustomStrategy` option.

# Not Yet Implemented

- Full slog support