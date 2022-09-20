# OpenTelemetry Hook

For this hook to function correctly a global `TracerProvider` must be set, an example of how to do this can be found below.

The `open telemetry hook` taps into the full hook lifecycle to write `traces` to the global `TracerProvider`.
To ensure thread safety a lock is generated for a given `span` key, constructed from the `OpenFeature client` name and `flagKey`, any 
spans attempting to reuse a currently active key will be blocked until the lock becomes available.

By default, the hook uses an internal `context.Background()`, however a context can be provided to the hook using the 
`hook.WithContext(context.Context)` method.
To wait for all threads to finish processing, the `hook.Wait()` method will be used. This is managed by an internal `sync.WaitGroup{}`.
## Example
The following example demonstrates the use of the `OpenTelemetry hook` with the `OpenFeature golang-sdk`. The traces are sent to a `zipkin` server running at `:9411` which will receive the following trace:
```json
{
    "traceId":"edc1a5f076c0afb7ea8bd2a56dfb3dd3",
    "id":"37ea9ce3b8638962",
    "name":"test-client.my-bool-flag",
    "timestamp":1662116610661173,
    "duration":100,
    "localEndpoint":{
        "serviceName":"hook-example"
    },
    "tags":{
        "feature_flag.evaluated_variant":"default-variant",
        "feature_flag.flag_key":"my-bool-flag",
        "feature_flag.provider_name":"NoopProvider",
        "otel.library.name":"github.com/open-feature/go-sdk-contrib/hooks/opentelemetry",
        "service.name":"hook-example"
    }
}
            
```

```go
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"

	otelHook "github.com/open-feature/go-sdk-contrib/hooks/open-telemetry/pkg"
	"github.com/open-feature/go-sdk/pkg/openfeature"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/zipkin"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
)

var logger = log.New(os.Stderr, "hook-example", log.Ldate|log.Ltime|log.Llongfile)

// initTracer creates a new trace provider instance and registers it as global trace provider.
func initTracer(url string) (func(context.Context) error, error) {
	exporter, err := zipkin.New(
		url,
		zipkin.WithLogger(logger),
	)
	if err != nil {
		return nil, err
	}

	batcher := sdktrace.NewBatchSpanProcessor(exporter)

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(batcher),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String("hook-example"),
		)),
	)
	otel.SetTracerProvider(tp)

	return tp.Shutdown, nil
}

func main() {
	url := flag.String("zipkin", "http://localhost:9411/api/v2/spans", "zipkin url")
	flag.Parse()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	shutdown, err := initTracer(*url)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := shutdown(ctx); err != nil {
			log.Fatal("failed to shutdown TracerProvider: %w", err)
		}
	}()

	// set the opentelemetry hook
	openfeature.AddHooks(otelHook.NewHook())
	// create a new client
	client := openfeature.NewClient("test-client")
	// evaluate a flag value
    client.ObjectValueDetails(
		"my-bool-flag",
		map[string]interface{}{
			"foo": "bar",
		},
		openfeature.EvaluationContext{},
		openfeature.EvaluationOptions{},
	)
}

```

## License

Apache 2.0 - See [LICENSE](./../../LICENSE) for more information.
