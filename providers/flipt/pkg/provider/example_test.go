package flipt_test

import (
	"context"

	flipt "go.openfeature.dev/contrib/providers/flipt/v2/pkg/provider"
	"go.openfeature.dev/openfeature/v2"
)

func Example() {
	err := openfeature.SetProviderAndWait(context.TODO(), flipt.NewProvider(
		flipt.WithAddress("localhost:9000"),
	))
	if err != nil {
		panic(err)
	}

	client := openfeature.NewClient(openfeature.WithDomain("my-app"))
	value := client.Boolean(
		context.TODO(), "v2_enabled", false, openfeature.NewEvaluationContext("tim@apple.com", map[string]any{
			"favorite_color": "blue",
		}),
	)

	if value {
		// do something
	} else {
		// do something else
	}
}
