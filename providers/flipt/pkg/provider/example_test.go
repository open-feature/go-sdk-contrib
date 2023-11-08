package flipt_test

import (
	"context"

	flipt "github.com/open-feature/go-sdk-contrib/providers/flipt/pkg/provider"
	"github.com/open-feature/go-sdk/pkg/openfeature"
)

func Example() {
	openfeature.SetProvider(flipt.NewProvider(
		flipt.WithAddress("localhost:9000"),
	))

	client := openfeature.NewClient("my-app")
	value, err := client.BooleanValue(
		context.Background(), "v2_enabled", false, openfeature.NewEvaluationContext("tim@apple.com", map[string]interface{}{
			"favorite_color": "blue",
		}),
	)

	if err != nil {
		panic(err)
	}

	if value {
		// do something
	} else {
		// do something else
	}
}
