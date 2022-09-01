package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"

	otelHook "github.com/open-feature/golang-sdk-contrib/hooks/otel/pkg/otel"
	"github.com/open-feature/golang-sdk/pkg/openfeature"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/zipkin"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
)

var logger = log.New(os.Stderr, "zipkin-example", log.Ldate|log.Ltime|log.Llongfile)

// initTracer creates a new trace provider instance and registers it as global trace provider.
func initTracer(url string) (func(context.Context) error, error) {
	// Create Zipkin Exporter and install it as a global tracer.
	//
	// For demoing purposes, always sample. In a production application, you should
	// configure the sampler to a trace.ParentBased(trace.TraceIDRatioBased) set at the desired
	// ratio.
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
			semconv.ServiceNameKey.String("zipkin-test"),
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
	hook := otelHook.Hook{}
	hook.Before(openfeature.HookContext{}, openfeature.HookHints{})
	fmt.Println("----")
	// hook.After(openfeature.HookContext{}, openfeature.EvaluationDetails{
	// 	FlagType: openfeature.Boolean,
	// 	ResolutionDetail: openfeature.ResolutionDetail{
	// 		Value: false,
	// 	},
	// }, openfeature.HookHints{})
	// hook.Finally(openfeature.HookContext{}, openfeature.HookHints{})
	openfeature.AddHooks(&otelHook.Hook{})
	client := openfeature.NewClient("test-client")
	fmt.Println(client.BooleanValueDetails("my-bool-flag", true, openfeature.EvaluationContext{}, openfeature.EvaluationOptions{}))
}
