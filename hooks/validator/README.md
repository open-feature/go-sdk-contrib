# Validator Hook

The `validator hook` taps into the `After` lifecycle event to validate the result of flag evaluations. If the result is
invalid, the default value and an error are returned.

The hook defines a `validator` interface with function
```go
IsValid(flagEvaluationDetails of.EvaluationDetails) error
```
to allow application authors to supply their own validators.

There are, however, [ready to be used validators](#validators) that conform to the interface.

## Setup

Import the [OpenFeature SDK](https://github.com/open-feature/go-sdk) and the validator.

Create an instance of the validator hook struct (using the hex validator as an example):
```go
package main

import (
	"fmt"
	"github.com/open-feature/go-sdk-contrib/hooks/validator/pkg/regex"
	"github.com/open-feature/go-sdk-contrib/hooks/validator/pkg/validator"
	"github.com/open-feature/go-sdk/openfeature"
	"log"
)

func main() {
	hexValidator, err := regex.Hex()
	if err != nil {
		log.Fatal(err)
	}
	v := validator.Hook{Validator: hexValidator}
}
```

Use the validator hook (on invocation as an example):
```go
client := openfeature.NewClient("foo")
value, err := client.
    StringValueDetails("blue", "#0000FF", openfeature.EvaluationContext{}, openfeature.WithHooks(v))
if err != nil {
    fmt.Println("err:", err)
}
```

## Example

Following [setup](#setup), use the `NoopProvider`, this simply returns the given default value on flag evaluation.
Give a false hex color as the default value call and check that the flag evaluation returns an error.

```go
package main

import (
	"fmt"
	"github.com/open-feature/go-sdk-contrib/hooks/validator/pkg/regex"
	"github.com/open-feature/go-sdk-contrib/hooks/validator/pkg/validator"
	"github.com/open-feature/go-sdk/openfeature"
	"log"
)

func main() {
	openfeature.SetProvider(openfeature.NoopProvider{})
	hexValidator, err := regex.Hex()
	if err != nil {
		log.Fatal(err)
	}
	v := validator.Hook{Validator: hexValidator}
	client := openfeature.NewClient("foo")

	result, err := client.
		StringValueDetails("blue", "invalidhex", openfeature.EvaluationContext{}, openfeature.WithHooks(v))
	if err != nil {
		fmt.Println("err:", err)
	}

	fmt.Println("result:", result)
}
```

```shell
go run main.go
err: execute after hook: regex doesn't match on flag value
result: {blue 1 {invalidhex   }}
```
Note that despite getting an error we still get a result.

## Validators

- [Regex](./pkg/regex/regex.go) validates the result matches the given regex
- [Hex](./pkg/regex/hex.go) validates the result is a valid hex color (e.g. #FFFFFF)

## License

Apache 2.0 - See [LICENSE](./../../LICENSE) for more information.
