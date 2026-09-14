package flipt_test

import (
	"context"
	"fmt"

	flipt "github.com/open-feature/go-sdk-contrib/providers/flipt/pkg/provider"
	"github.com/open-feature/go-sdk/openfeature"
)

func Example() {
	err := openfeature.SetProviderAndWait(flipt.NewProvider(
		flipt.WithAddress("localhost:9000"),
	))
	if err != nil {
		panic(err)
	}

	client := openfeature.NewDefaultClient()
	value := client.Boolean(
		context.TODO(), "v2_enabled", false, openfeature.NewEvaluationContext("tim@apple.com", map[string]any{
			"favorite_color": "blue",
		}),
	)

	if value {
		// do something
		fmt.Println("flag is on")
	} else {
		// do something else
		fmt.Println("flag is off")
	}
	// Output: flag is off
}
