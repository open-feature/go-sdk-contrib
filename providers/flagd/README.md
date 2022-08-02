# Flagd Provider

![Experimental](https://img.shields.io/badge/experimental-breaking%20changes%20allowed-yellow)
![Alpha](https://img.shields.io/badge/alpha-release-red)

[Flagd](https://github.com/open-feature/flagd) is a simple command line tool for fetching and presenting feature flags to services. It is designed to conform to OpenFeature schema for flag definitions. This repository and package provides the client side code for interacting with it via the [OpenFeature SDK](https://github.com/open-feature/golang-sdk).

## Setup
To use flagd with the [OpenFeature SDK](https://github.com/open-feature/golang-sdk) set the provider to the `openfeature` global singleton as shown below (using default values which align with those of `flagd`)
```
openfeature.SetProvider(flagd.NewProvider())
```  
You may also provide additional options to configure the provider client
```
flagd.WithService("http" or "grpc") // defaults to http
flagd.WithHost(string)              // defaults to localhost
flagd.WithPort(int32)               // defaults to 8080
flagd.WithProtocol(string)          // defaults to http (https is not currently supported by flagd)
```
for example:
```
package main

import (
	"github.com/open-feature/golang-sdk-contrib/providers/flagd/pkg"
   	 "github.com/open-feature/golang-sdk/pkg/openfeature"
)

func main() {
    openfeature.SetProvider(flagd.NewProvider(
        flagd.WithService("grpc"),
        flagd.WithHost("localhost"),
        flagd.WithPort(8000),
    ))
}

```

## License

Apache 2.0 - See [LICENSE](./../../license) for more information.