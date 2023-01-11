# OpenTelemetry Hook

For this hook to function correctly a global `TracerProvider` must be set, an example of how to do this can be found below.

The `open telemetry hook` taps into the after and error methods of the hook lifecycle to write `events` and `attributes` to an existing `span`.
A `context.Context` containing a `span` must be passed to the hook `NewHook` method, otherwise the hook will no-op.

## Example
The following example demonstrates the use of the `OpenTelemetry hook` with the `OpenFeature go-sdk`. The traces are sent to a `zipkin` server running at `:9411` which will receive the following trace:
```json
{
{
	"traceId":"ac4464e6387c552b4b55ab3d19bf64f9",
	"id":"f677ca41dbfd6bfe",
	"name":"run",
	"timestamp":1673431556236064,
	"duration":45,
	"localEndpoint":{
		"serviceName":"hook-example"
		},
		"annotations":[
			{
				"timestamp":1673431556236107,
				"value":"feature_flag: {\"feature_flag.key\":\"my-bool-flag\",\"feature_flag.provider_name\":\"NoopProvider\",\"feature_flag.variant\":\"default-variant\"}"
			}
		],
		"tags":{
			"otel.library.name":"test-tracer",
			"service.name":"hook-example"
		}
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
			log.Fatalf("failed to shutdown TracerProvider: %w", err)
		}
	}()

	// set up the span
	ctx, s := otel.Tracer("test-tracer").Start(ctx, "run")
	// set the opentelemetry hook
	openfeature.AddHooks(otelHook.NewHook(ctx))
	// create a new client
	client := openfeature.NewClient("test-client")
	// evaluate a flag value
	client.ObjectValueDetails(
		ctx,
		"my-bool-flag",
		map[string]interface{}{
			"foo": "bar",
		},
		openfeature.EvaluationContext{},
	)
	s.End()
}

```

## License

Apache 2.0 - See [LICENSE](./../../LICENSE) for more information.
