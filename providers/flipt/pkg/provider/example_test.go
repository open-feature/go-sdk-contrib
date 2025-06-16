package flipt_test

import (
	"context"

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

	client := openfeature.NewClient("my-app")
	value, err := client.BooleanValue(
		context.TODO(), "v2_enabled", false, openfeature.NewEvaluationContext("tim@apple.com", map[string]any{
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
