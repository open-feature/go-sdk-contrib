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
    "github.com/open-feature/go-sdk/openfeature/hooks"
)

namedProviderA := UniqueNameProvider{ Provider: myProvider, UniqueName: "Provider A" }
namedProviderB := UniqueNameProvider{ Provider: myOtherProvider, UniqueName: "Provider B" }

provider, err := NewMultiProvider([]UniqueNameProvider{namedProviderA, namedProviderB}, StrategyFirstMatch, hooks.NewLoggingHook(false))
openfeature.SetProvider(provider)
```
 

# Strategies

There are multiple strategies that can be used to determine the result returned to the caller. A strategy must be set at
initialization time.

There are 2 strategies available currently:

- _First Match_
- _First Success_

## First Match Strategy

The first match strategy works by **sequentially**  calling each provider in the order that they are provided to the mutli-provider.
The first provider that returns a result. It will try calling the next provider whenever it encounters a `FLAG_NOT_FOUND`
error. However, if a provider returns an error other than `FLAG_NOT_FOUND` the provider will stop and return the default
value along with setting the error details if a detailed request is issued. (allow changing this behavior?)

## First Success Strategy

The First Success strategy works by calling each provider in **parallel**. The first provider that returns a response
with no errors is returned and all other calls are cancelled. If no provider provides a successful result the default
value will be returned to the caller.