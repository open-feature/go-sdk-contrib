package flagsmith_test

import (
	"context"
	"fmt"

	flagsmithClient "github.com/Flagsmith/flagsmith-go-client/v4"
	flagsmith "github.com/open-feature/go-sdk-contrib/providers/flagsmith/pkg"
	"github.com/open-feature/go-sdk/openfeature"
)


func Example() {
	provider := flagsmith.NewProvider(
		// See https://docs.flagsmith.com/clients/server-side?language=go
		flagsmithClient.NewClient(
			// Local evaluation is the recommended mode for server-side applications.
			// Client-side applications should use remote evaluation.
			// See https://docs.flagsmith.com/clients/
			"ser.your-server-side-sdk-key",
			flagsmithClient.WithLocalEvaluation(context.TODO()),

			// Only needed if not using Flagsmith SaaS, i.e. https://app.flagsmith.com
			// flagsmithClient.WithBaseURL("https://flagsmith-api.example.com/api/v1/"),
		),
		// Makes of.Boolean return the enabled/disabled state of a feature, and not its value.
		flagsmith.WithUsingBooleanConfigValue(),
	)
	err := openfeature.SetProviderAndWait(provider)
	if err != nil {
		panic(err)
	}
	of := openfeature.NewClient("")

	evaluationCtx := openfeature.NewEvaluationContext(
		// The context targeting key is used as the Flagsmith identity's identifier.
		// It is required to perform any kind of flag targeting.
		"my-user-id",
		// The context attributes correspond to Flagsmith traits.
		// Segments are sets of identities, defined by rules that match on these traits.
		// They can be used to override flags only for certain identities.
		// See https://docs.flagsmith.com/basic-features/segments
		map[string]any{
			"favourite_drink": "tea",
		},
	)

	hasMyFeature := of.Boolean(
		context.TODO(),
		"my_feature",
		false,
		evaluationCtx,
	)
	fmt.Println(hasMyFeature)
	// Output: false
}
