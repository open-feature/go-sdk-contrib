# Environment Variable JSON Flag Provider

This repository contains a very simple environment variable based feature flag provider.
This provider uses a JSON evaluation for matching a flag `Variant` to a provided `EvaluationContext`. Each flag `Variant` contains a slice of `Criteria`, if all `Criteria` match then the flags value is returned. Each `Variant` is evaluated starting at index 0, therefore the first matching `Variant` is returned. Each variant also has a `TargetingKey`, when set it must match the `TargetingKey` provided in the `EvaluationContext` for the `Variant` to be returned.

## Flag Configuration Structure

Flag configurations are stored as JSON strings, with one configuration per flag key. An example configuration is described below.

```json
{
	"defaultVariant": "not-yellow",
	"variants": [
		{
			"name": "yellow-with-key",
			"targetingKey": "user",
			"criteria": [
				{
					"key": "color",
					"value": "yellow"
				}
			],
			"value": true
		},
		{
			"name": "yellow",
			"targetingKey": "",
			"criteria": [
				{
					"key": "color",
					"value": "yellow"
				}
			],
			"value": true
		},
		{
			"name": "not-yellow",
			"targetingKey": "",
			"criteria": [],
			"value": false
		}
	]
}
```

## Example Usage

Below is a simple example of using this `Provider`, in this example the above flag configuration is saved with the key `AM_I_YELLOW.`

```sh
export AM_I_YELLOW='{"defaultVariant":"not-yellow","variants":[{"name":"yellow-with-key","targetingKey":"user","criteria":[{"key":"color","value":"yellow"}],"value":true},{"name":"yellow","targetingKey":"","criteria":[{"key":"color","value":"yellow"}],"value":true},{"name":"not-yellow","targetingKey":"","criteria": [],"value":false}]}'
```

```go
package main

import (
	"context"
	"fmt"

	fromEnv "github.com/open-feature/go-sdk-contrib/providers/from-env/pkg"
	"github.com/open-feature/go-sdk/openfeature"
)

func main() {
	// register the provider against the go-sdk
	openfeature.SetProvider(&fromEnv.FromEnvProvider{})
	// create a client from via the go-sdk
	client := openfeature.NewClient("am-i-yellow-client")

	// we are now able to evaluate our stored flags
	resB, err := client.BooleanValueDetails(
		context.Background(),
		"AM_I_YELLOW",
		false,
		openfeature.NewEvaluationContext(
			"",
			map[string]interface{}{
				"color": "yellow",
			},
		),
	)
	fmt.Println(resB, err)

	resB, err = client.BooleanValueDetails(
		context.Background(),
		"AM_I_YELLOW",
		false,
		openfeature.NewEvaluationContext(
			"user",
			map[string]interface{}{
				"color": "yellow",
			},
		),
	)
	fmt.Println(resB, err)

	resS, err := client.StringValueDetails(
		context.Background(),
		"AM_I_YELLOW",
		"i am a default value",
		openfeature.NewEvaluationContext(
			"",
			map[string]interface{}{
				"color": "not yellow",
			},
		),
	)
	fmt.Println(resS, err)
}
```

Console output:

```console
{AM_I_YELLOW 0 {true  TARGETING_MATCH yellow}} <nil>
{AM_I_YELLOW 0 {true  TARGETING_MATCH yellow-with-key}} <nil>
{i am a default value {AM_I_YELLOW string { ERROR TYPE_MISMATCH }}} error code: TYPE_MISMATCH
```

### Name Mapping

To transform the flag name into an environment variable name at runtime, you can use the option `WithFlagToEnvMapper`.

For example:

```go
mapper := func(flagKey string) string {
	return fmt.Sprintf("MY_%s", strings.ToUpper(strings.ReplaceAll(flagKey, "-", "_")))
}

p := fromEnv.NewProvider(fromEnv.WithFlagToEnvMapper(mapper))

// This will look up MY_SOME_FLAG env variable
res := p.BooleanEvaluation(context.Background(), "some-flag", false, evalCtx)
```

## Common Error Response Types

| Error Value    | Error Reason                                                                                                                |
| -------------- | --------------------------------------------------------------------------------------------------------------------------- |
| PARSE_ERROR    | A required `DefaultVariant` does not exist, or, the stored flag configuration cannot be parsed into the `StoredFlag` struct |
| TYPE_MISMATCH  | The responses value type does not match that of the request.                                                                |
| FLAG_NOT_FOUND | The requested flag key does not have an associated environment variable.                                                    |
