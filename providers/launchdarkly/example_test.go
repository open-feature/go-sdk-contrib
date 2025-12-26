package launchdarkly_test

import (
	"context"
	"time"

	"github.com/launchdarkly/go-sdk-common/v3/ldlog"
	"github.com/launchdarkly/go-server-sdk/v7/ldcomponents"
	"go.openfeature.dev/openfeature/v2"

	ld "github.com/launchdarkly/go-server-sdk/v7"
	ofld "go.openfeature.dev/contrib/providers/launchdarkly/v2/pkg"
)

var emptyEvalCtx = openfeature.EvaluationContext{}

func Example() {
	var config ld.Config
	config.Logging = ldcomponents.Logging().MinLevel(ldlog.Debug)
	ldClient, err := ld.MakeCustomClient("my api key", config, 5*time.Second)
	if err != nil {
		panic(err)
	}

	// Flushes all pending analytics events.
	defer func() {
		_ = ldClient.Close()
	}()

	// Set Launchdarkly as OpenFeature provider
	err = openfeature.SetProvider(context.TODO(), ofld.NewProvider(ldClient))
	if err != nil {
		// handle error for provider initialization
	}

	// Set a multi-context evaluation context as example
	evalCtx := openfeature.NewEvaluationContext("redpanda-12342", map[string]any{
		"kind": "multi",
		"organization": map[string]any{
			"key":           "blah1234",
			"name":          "Redpanda",
			"customer_tier": "GOLD",
		},
		"redpanda-id": map[string]any{
			"key":            "redpanda-12342",
			"cloud-provider": "aws",
		},
	})

	// Get an openfeature client and set the evaluation context to it as example.
	// For more information about OpenFeature evaluation contexts please refer to
	// https://openfeature.dev/docs/reference/concepts/evaluation-context/
	client := openfeature.NewClient(openfeature.WithDomain("hello-world"))
	client.SetEvaluationContext(evalCtx)

	if err := doSomething(context.TODO(), client); err != nil {
		panic(err)
	}
}

func doSomething(ctx context.Context, ofclient *openfeature.Client) error {
	mtlsEnabled := ofclient.Boolean(ctx, "mtls_enabled", false, emptyEvalCtx)

	if mtlsEnabled {
		println("configuring mTLS...")
	}

	return nil
}
