package main

import (
	"context"
	"fmt"
	gofeatureflag "github.com/open-feature/go-sdk-contrib/providers/go-feature-flag/pkg"
)

func main() {

	provider, err := gofeatureflag.NewProvider(gofeatureflag.ProviderOptions{Endpoint: "http://localhost:1031"})
	if err != nil {
		panic(err)
	}

	evalCtx := map[string]interface{}{"targetingKey": "xxx"}
	res := provider.StringEvaluation(context.Background(), "hexColor2", "true", evalCtx)
	fmt.Println(res)
}
