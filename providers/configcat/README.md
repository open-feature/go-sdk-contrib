# ConfigCat OpenFeature provider for Go

OpenFeature Go provider implementation for [ConfigCat](https://configcat.com) that uses the official [ConfigCat Go SDK](https://github.com/configcat/go-sdk).

## Installation

```shell
# ConfigCat SDK
go get github.com/configcat/go-sdk/v9

# OpenFeature SDK
go get go.openfeature.dev/v2
go get go.openfeature.dev/contrib/providers/configcat/v2
```

## Usage

Here's a basic example:

```go
import (
 "context"
 "fmt"

 sdk "github.com/configcat/go-sdk/v9"
 configcat "go.openfeature.dev/contrib/providers/configcat/v2/pkg"
 "go.openfeature.dev/openfeature/v2"
)

func main() {
 provider := configcat.NewProvider(sdk.NewClient("..."))
 openfeature.SetProvider(context.TODO(), provider)

 client := openfeature.NewClient("app")

 val := client.Boolean(context.TODO(), "flag_name", false, openfeature.NewEvaluationContext("123", map[string]any{
  configcat.EmailKey: "test@example.com",
 }))
 fmt.Printf("val: %+v\n", val)
}
```
