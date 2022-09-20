# Environment Variable JSON Flag Provider

This repository contains a very simple environment variable based feature flag provider.  
This provider uses a JSON evaluation for matching a flag `Variant` to a provided `EvaluationContext`. Each flag `Variant` contains a slice of `Conditions`, if all `Conditions` match then the flags value is returned. Each `Variant` is evaluated starting at index 0, therefore the first matching `Variant` is returned. Each variant also has a `TargetingKey`, when set it must match the `TargetingKey` provided in the `EvaluationContext` for the `Variant` to be returned.  


## Flag Configuration Structure. 

Flag configurations are stored as JSON strings, with one configuration per flag key. An example configuration is described below.  
```json
{
    "defaultVariant":"not-yellow",
    "variants": [
        {
	    "name": "yellow-with-key",
            "targetingKey":"user",
            "criteria": [
                {
                    "color":"yellow"
                }
            ],
            "value":true
        },
	{
	    "name": "yellow",
            "targetingKey":"",
            "criteria": [
                {
                    "color":"yellow"
                }
            ],
            "value":true
        },
        {
	    "name": "not-yellow",
            "targetingKey":"",
            "criteria": [],
            "value":false
        }
    ]
}
```

## Example Usage  
Below is a simple example of using this `Provider`, in this example the above flag configuration is saved with the key `AM_I_YELLOW.` 

```go
package main

import (
	"fmt"

	fromEnv "github.com/open-feature/go-sdk-contrib/providers/from-env/pkg"
	"github.com/open-feature/go-sdk/pkg/openfeature"
)

func main() {
	// register the provider against the go-sdk
	openfeature.SetProvider(&fromEnv.Provider{})
	// create a client from via the go-sdk
	client := openfeature.NewClient("am-i-yellow-client")

	// we are now able to evaluate our stored flags
	res, err := client.BooleanValueDetails(
		"AM_I_YELLOW",
		false,
		openfeature.EvaluationContext{
			Attributes: map[string]interface{}{
				"color": "yellow",
			},
		},
	)
	fmt.Println(res, err)

	res, err := client.BooleanValueDetails(
	"AM_I_YELLOW",
	false,
	openfeature.EvaluationContext{
		Attributes: map[string]interface{}{
			"color": "yellow",
		},
		TargetingKey: "user",
	},
	)
	fmt.Println(res, err)

	res, err = client.StringValueDetails(
		"AM_I_YELLOW",
		"i am a default value",
		openfeature.EvaluationContext{
			Attributes: map[string]interface{}{
				"color": "not yellow",
			},
		},
	)
	fmt.Println(res, err)
}
```
Console output:
```
{AM_I_YELLOW 0 {true  TARGETING_MATCH yellow}} <nil>
{AM_I_YELLOW 0 {true  TARGETING_MATCH yellow-with-key}} <nil>
{AM_I_YELLOW 1 {i am a default value   }} evaluate the flag: TYPE_MISMATCH
```

## Common Error Response Types

Error Value  | Error Reason
------------- | -------------
PARSE_ERROR  | A required `DefaultVariant` does not exist, or, the stored flag configuration cannot be parsed into the `StoredFlag` struct
TYPE_MISMATCH  | The responses value type does not match that of the request.
FLAG_NOT_FOUND  | The requested flag key does not have an associated environment variable.

