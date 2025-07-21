package rocketflag_test

import (
	"context"
	"fmt"

	rocketflag "github.com/open-feature/go-sdk-contrib/providers/rocketflag"
	"github.com/open-feature/go-sdk/openfeature"
	client "github.com/rocketflag/go-sdk"
)

func Sample() {
	provider := rocketflag.NewProvider(client.NewClient())
	err := openfeature.SetProviderAndWait(provider)
	if err != nil {
		panic(err)
	}
	ofClient := openfeature.NewClient("")

	// If you want to provide a cohort, you can do that here. OpenFeature uses "targetingKey" instead of cohorts.
	// In this example, the cohort value is "user@example.com"
	evaluationCtx := openfeature.NewEvaluationContext("user@example.com", nil)

	// If you don't want to provide a cohort for a flag, create a targetless context.
	// blankCtx := openfeature.NewTargetlessEvaluationContext(nil)

	flagResult := ofClient.Boolean(
		context.TODO(),
		"flag_id",
		false, // default value
		evaluationCtx,
	)
	fmt.Println("OpenFeature RocketFlag Result: ", flagResult)
}
