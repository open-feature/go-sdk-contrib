# Vercel OpenFeature Provider

OpenFeature provider for Vercel Flags.

## Installation

```sh
go get github.com/open-feature/go-sdk-contrib/providers/vercel
```

## Usage

```go
package main

import (
	"context"
	"log"

	vercel "github.com/open-feature/go-sdk-contrib/providers/vercel/pkg"
	"github.com/open-feature/go-sdk/openfeature"
)

func main() {
	provider, err := vercel.NewProvider()
	if err != nil {
		log.Fatal(err)
	}

	if err := openfeature.SetProviderAndWait(provider); err != nil {
		log.Fatal(err)
	}
	defer openfeature.Shutdown()

	client := openfeature.NewClient("app")
	evalCtx := openfeature.NewEvaluationContext("", map[string]any{
		"user": map[string]any{"id": "user-123"},
	})

	enabled, err := client.BooleanValue(context.Background(), "new-checkout", false, evalCtx)
	if err != nil {
		log.Fatal(err)
	}

	log.Println(enabled)
}
```

By default the provider reads the `FLAGS` environment variable, matching
Vercel's TypeScript provider. You can also pass the SDK key or connection
string explicitly:

```go
provider, err := vercel.NewProvider(
	vercel.WithConnectionString("flags:edgeConfigId=...&edgeConfigToken=...&sdkKey=vf_server_..."),
)
```

The provider fetches the Vercel Flags datafile from
`https://flags.vercel.com/v1/datafile`, evaluates flags locally, and refreshes
the datafile in the background. Use `WithPollingDisabled` to disable refreshes,
or `WithDatafile` to seed the provider with an existing datafile.
