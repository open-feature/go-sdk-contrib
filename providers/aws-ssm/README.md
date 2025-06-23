# AWS SSM OpenFeature Provider for Go

OpenFeature Go provider implementation that integrates with AWS Systems Manager Parameter Store (SSM) for feature flag management.

## Installation

```shell
go get github.com/open-feature/go-sdk/openfeature
go get github.com/open-feature/go-sdk-contrib/providers/aws-ssm
```

## Usage

Here's a basic example of using the AWS SSM provider:

```go
import (
	"context"
	"fmt"

	"github.com/open-feature/go-sdk/openfeature"
	"github.com/open-feature/go-sdk-contrib/providers/aws-ssm"
)

func main() {
	// Initialize the provider
	provider := aws.NewProvider()
	err := openfeature.SetProvider(provider)

	// Create OpenFeature client
	client := openfeature.NewClient("")

	// Evaluate a feature flag
	val, err := client.BooleanValue(context.Background(), "/path/to/feature/flag", false, nil)
	fmt.Printf("val: %+v - error: %v\n", val, err)
}
```

## Configuration

The provider uses AWS SSM Parameter Store to store and retrieve feature flag values. Feature flag paths in OpenFeature correspond to SSM parameter paths. For example:

- OpenFeature flag path: `/features/new-feature`
- SSM parameter path: `/features/new-feature`

The provider supports both secure string parameters (encrypted with AWS KMS) and standard string parameters in SSM Parameter Store.

## AWS Credentials

The provider uses the default AWS credential chain, which means it will look for credentials in this order:
1. Environment variables
2. Shared credentials file (~/.aws/credentials)
3. AWS config file (~/.aws/config)
4. IAM role credentials (when running on AWS)

## Security

The provider supports secure parameters in SSM Parameter Store, which are encrypted at rest using AWS KMS. When using secure parameters, make sure your AWS credentials have the necessary permissions to decrypt the parameters.
