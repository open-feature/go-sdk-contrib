# ConfigCat OpenFeature provider for Go

OpenFeature Go provider implementation for [ConfigCat](https://configcat.com) that uses the official [ConfigCat Go SDK](https://github.com/configcat/go-sdk).

## Installation

```shell
# ConfigCat SDK
go get github.com/configcat/go-sdk/v9

# OpenFeature SDK
go get github.com/open-feature/go-sdk/openfeature
go get github.com/open-feature/go-sdk-contrib/providers/configcat
```

## Usage

Here's a basic example:

```go
import (
	"context"
	"fmt"

	sdk "github.com/configcat/go-sdk/v9"
	configcat "github.com/open-feature/go-sdk-contrib/providers/configcat/pkg"
	"github.com/open-feature/go-sdk/openfeature"
)

func main() {
	provider := configcat.NewProvider(sdk.NewClient("..."))
	openfeature.SetProvider(provider)

	client := openfeature.NewClient("app")

	val, err := client.BooleanValue(context.Background(), "flag_name", false, openfeature.NewEvaluationContext("123", map[string]any{
		configcat.EmailKey: "test@example.com",
	}))
	fmt.Printf("val: %+v - error: %v\n", val, err)
}
```
