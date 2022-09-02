package otel

import (
	"context"
	"log"
	"os"
	"testing"
	"time"

	"github.com/open-feature/golang-sdk/pkg/openfeature"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/zipkin"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

func initTracer() (func(context.Context) error, error) {
	var logger = log.New(os.Stderr, "zipkin-example", log.Ldate|log.Ltime|log.Llongfile)
	exporter, err := zipkin.New(
		"http://localhost:9411/api/v2/spans",
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

func TestOtelHookMethods(t *testing.T) {

	t.Run("Before should start a new span", func(t *testing.T) {
		cleanup, err := initTracer()
		if err != nil {
			t.Fatal(err)
		}
		defer cleanup(context.Background())
		otelHook := Hook{}
		otelHook.Before(openfeature.HookContext{}, openfeature.HookHints{})
		if len(otelHook.spans) != 1 {
			t.Fatal("before hook did not create a new span")
		}
	})

	t.Run("Finally hook should trigger the span to close with no error", func(t *testing.T) {
		cleanup, err := initTracer()
		if err != nil {
			t.Fatal(err)
		}
		defer cleanup(context.Background())
		otelHook := Hook{}
		otelHook.Before(openfeature.HookContext{}, openfeature.HookHints{})
		if len(otelHook.spans) != 1 {
			t.Fatal("before hook did not create a new span")
		}
		otelHook.Finally(openfeature.HookContext{}, openfeature.HookHints{})
		otelHook.Wait()

		if err != nil {
			t.Fatal(err)
		}
		for _, x := range otelHook.spans {
			if x.ss != nil {
				t.Fatal("after hook did not trigger the closing of the span")
			}
		}
	})

	t.Run("context cancellation should close an open span", func(t *testing.T) {
		cleanup, err := initTracer()
		if err != nil {
			t.Fatal(err)
		}
		defer cleanup(context.Background())
		ctx, cancel := context.WithCancel(context.Background())
		otelHook := Hook{}
		otelHook.WithContext(ctx)
		otelHook.Before(openfeature.HookContext{}, openfeature.HookHints{})
		if len(otelHook.spans) != 1 {
			t.Fatal("before hook did not create a new span")
		}
		cancel()
		otelHook.Wait()
		for _, x := range otelHook.spans {
			if x.ss != nil {
				t.Fatal("stored span has not been cleaned up")
			}
		}
	})

	// if updates have been made causing the tests suite to hang, they will be within this test
	// however, in most cases the reasons for the tests to hang will be caught by the above tests
	t.Run("duplicate keys should be blocked from running concurrently", func(t *testing.T) {
		cleanup, err := initTracer()
		if err != nil {
			t.Fatal(err)
		}
		defer cleanup(context.Background())
		otelHook := Hook{}
		blocked := true

		// Trigger the initial before hook, the empty context will always provide the same key
		otelHook.Before(openfeature.HookContext{}, openfeature.HookHints{})
		if len(otelHook.spans) != 1 {
			t.Fatal("before hook did not create a new span")
		}
		// this before hook should be blocked until the after hook for the locked resource has been run
		go func() {
			otelHook.Before(openfeature.HookContext{}, openfeature.HookHints{})
			blocked = false
		}()
		time.Sleep(500 * time.Millisecond) // account for slow execution to ensure that the goroutine is blocked
		if !blocked {
			t.Fatal("duplicate keys are not being blocked")
		}

		// unlock the resource and ensure that the previously blocked goroutine can now complete
		otelHook.Finally(openfeature.HookContext{}, openfeature.HookHints{})

		// account for slow execution time to ensure that goroutine is no longer blocked
		// (cannot use the .Wait method in this example) as the after method has not yet been called for the blocked goroutine
		time.Sleep(500 * time.Millisecond)
		if blocked {
			t.Fatal("blocked goroutine has not been unblocked by the release of the lock")
		}

		// complete the final hooks lifecycle and ensure that it is being cleaned up
		otelHook.Finally(openfeature.HookContext{}, openfeature.HookHints{})
		otelHook.Wait()
		for _, x := range otelHook.spans {
			if x.ss != nil {
				t.Fatal("stored span has not been cleaned up")
			}
		}
	})
}
