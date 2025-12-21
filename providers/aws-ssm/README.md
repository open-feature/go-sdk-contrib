# AWS SSM OpenFeature Provider for Go

OpenFeature Go provider implementation that integrates with AWS Systems Manager Parameter Store (SSM) for feature flag management.

## Installation

```shell
go get go.openfeature.dev/openfeature/v2
go get go.openfeature.dev/contrib/providers/aws-ssm/v2
```

## Usage

Here's a basic example of using the AWS SSM provider:

```go
import (
 "context"
 "fmt"
 "log"

 "github.com/aws/aws-sdk-go-v2/config"
 "go.openfeature.dev/openfeature/v2"
 ssm "go.openfeature.dev/contrib/providers/aws-ssm/v2"
)

func main() {

 // Retrieve AWS Config
 cfg, err := config.LoadDefaultConfig(context.TODO())

 if err != nil {
  log.Fatalf("failed to load AWS config: %v", err)
 }

 // Initialize the provider
 provider := ssm.NewProvider(cfg)
 err := openfeature.SetProvider(context.TODO(), provider)

 // Create OpenFeature client
 client := openfeature.NewClient("")

 // Evaluate a feature flag
 val := client.Boolean(context.TODO(), "/path/to/feature/flag", false, nil)
 fmt.Printf("val: %+v\n", val)
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
