# Validator Hook

The `validator hook` taps into the `After` lifecycle event to validate the result of flag evaluations.

The hook defines a `validator` interface with function
```go
IsValid(flagEvaluationDetails of.EvaluationDetails) error
```
to allow application authors to supply their own validators.

There are, however, [ready to be used validators](#validators) that conform to the interface.

## Setup

Import the [OpenFeature SDK](https://github.com/open-feature/golang-sdk) and the validator.

Create an instance of the validator hook struct (using the hex validator as an example):
```go
package main

import (
"github.com/open-feature/golang-sdk-contrib/hooks/validator/pkg/hex"
"github.com/open-feature/golang-sdk-contrib/hooks/validator/pkg/validator"
"github.com/open-feature/golang-sdk/pkg/openfeature"
)

func main() {
	v := validator.Hook{Validator: hex.Validator{}}
}
```

Use the validator hook (on invocation as an example):
```go
client := openfeature.NewClient("foo")
evalOptions := openfeature.NewEvaluationOptions([]openfeature.Hook{v}, openfeature.HookHints{})
value, err := client.
    StringValueDetails("blue", "#0000FF", openfeature.EvaluationContext{}, evalOptions)
if err != nil {
    fmt.Println("err: ", err)
}
```

## Example

Following [setup](#setup), use the `NoopProvider`, this simply returns the given default value on flag evaluation.
Give a false hex color as the default value call and check that the flag evaluation returns an error.

```go
package main

import (
	"fmt"
	"github.com/open-feature/golang-sdk-contrib/hooks/validator/pkg/hex"
	"github.com/open-feature/golang-sdk-contrib/hooks/validator/pkg/validator"
	"github.com/open-feature/golang-sdk/pkg/openfeature"
)

func main() {
	openfeature.SetProvider(openfeature.NoopProvider{})
	v := validator.Hook{Validator: hex.Validator{}}
	client := openfeature.NewClient("foo")
	evalOptions := openfeature.NewEvaluationOptions([]openfeature.Hook{v}, openfeature.HookHints{})

	result, err := client.
		StringValueDetails("blue", "invalidhex", openfeature.EvaluationContext{}, evalOptions)
	if err != nil {
		fmt.Println("err:", err)
	}
	fmt.Println("result:", result)
}
```

```shell
go run main.go
err: execute after hook: invalid format
result {blue 1 {invalidhex   }}
```
Note that despite getting an error we still get a result.

## Validators

- [Hex](./pkg/hex/hex.go) validates the result is a valid hex color (e.g. #FFFFFF)

## License

Apache 2.0 - See [LICENSE](./../../LICENSE) for more information.
