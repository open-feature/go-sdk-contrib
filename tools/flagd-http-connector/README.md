# flagd-http-connector (Go)

This is a custom sync provider for `flagd`, written in Go. It fetches feature flags from an HTTP endpoint and supplies them to the OpenFeature Go SDK.

## Usage

```go
provider := flagdhttp.NewHTTPProvider("http://localhost:8080/flags")
flags, err := provider.Sync(context.Background())
```