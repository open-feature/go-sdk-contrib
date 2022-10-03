package otel

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/open-feature/go-sdk/pkg/openfeature"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type tracerClientMock struct {
	t *tracerMock
}
type tracerMock struct {
	trace.Tracer
	span *spanMock
}
type spanMock struct {
	trace.Span
	attributes []attribute.KeyValue
	closed     bool
	err        error
	code       codes.Code
	errString  string
}

func (t tracerClientMock) tracer() trace.Tracer {
	return t.t
}
func (t tracerMock) Start(ctx context.Context, _ string, _ ...trace.SpanStartOption) (context.Context, trace.Span) {
	return ctx, t.span
}
func (s *spanMock) SetAttributes(kv ...attribute.KeyValue) {
	s.attributes = append(s.attributes, kv...)
}
func (s *spanMock) RecordError(err error, _ ...trace.EventOption) {
	s.err = err
}
func (s *spanMock) SetStatus(code codes.Code, err string) {
	s.code = code
	s.errString = err
}
func (s *spanMock) End(...trace.SpanEndOption) {
	s.closed = true
}

func TestOtelHookMethods(t *testing.T) {

	t.Run("Before should start a new span", func(t *testing.T) {
		otelHook := Hook{
			tracerClient: tracerClientMock{
				t: &tracerMock{
					span: &spanMock{},
				},
			},
		}
		otelHook.tracerClient.tracer().Start(context.Background(), "test")
		_, _ = otelHook.Before(openfeature.HookContext{}, openfeature.HookHints{})
		if len(otelHook.spans) != 1 {
			t.Fatal("before hook did not create a new span")
		}
	})

	t.Run("Finally hook should trigger the span to close with no error", func(t *testing.T) {
		otelHook := Hook{
			tracerClient: &tracerClientMock{
				t: &tracerMock{
					span: &spanMock{},
				},
			},
		}
		_, _ = otelHook.Before(openfeature.HookContext{}, openfeature.HookHints{})
		if len(otelHook.spans) != 1 {
			t.Fatal("before hook did not create a new span")
		}
		otelHook.Finally(openfeature.HookContext{}, openfeature.HookHints{})
		otelHook.Wait()
		for _, x := range otelHook.spans {
			if x.ss != nil {
				t.Fatal("after hook did not trigger the closing of the span")
			}
		}
	})

	t.Run("context cancellation should close an open span", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		otelHook := Hook{
			tracerClient: &tracerClientMock{
				t: &tracerMock{
					span: &spanMock{},
				},
			},
		}
		otelHook.WithContext(ctx)
		_, _ = otelHook.Before(openfeature.HookContext{}, openfeature.HookHints{})
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
		otelHook := Hook{
			tracerClient: &tracerClientMock{
				t: &tracerMock{
					span: &spanMock{},
				},
			},
		}
		blocked := true

		// Trigger the initial before hook, the empty context will always provide the same key
		_, _ = otelHook.Before(openfeature.HookContext{}, openfeature.HookHints{})
		if len(otelHook.spans) != 1 {
			t.Fatal("before hook did not create a new span")
		}
		// this before hook should be blocked until the after hook for the locked resource has been run
		go func() {
			_, _ = otelHook.Before(openfeature.HookContext{}, openfeature.HookHints{})
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

	t.Run("ensure all attributes are correctly set on span and it is closed", func(t *testing.T) {
		span := spanMock{}
		hook := Hook{
			tracerClient: &tracerClientMock{
				t: &tracerMock{
					span: &span,
				},
			},
		}
		openfeature.AddHooks(&hook)
		client := openfeature.NewClient("test-client")
		_, _ = client.ObjectValue(
			context.Background(),
			"my-bool-flag",
			map[string]interface{}{"foo": "bar"},
			openfeature.EvaluationContext{},
		)
		hook.Wait()
		if !span.closed {
			t.Fatalf("span has not been closed")
		}
		valuePresent := false
		variantPreset := false
		for _, att := range span.attributes {
			switch att.Key {
			case AttributeFlagKey:
				if att.Value.AsString() != "my-bool-flag" {
					t.Fatalf("unexpected flagKey value received: %s", att.Value.AsString())
				}
			case AttributeProviderName:
				if att.Value.AsString() != "NoopProvider" {
					t.Fatalf("unexpected ProviderName value received expected %s, got %s", "NoopProvider", att.Value.AsString())
				}
			case AttributeEvaluatedVariant:
				variantPreset = true
				if att.Value.AsString() != "default-variant" {
					t.Fatalf("unexpected EvaluatedVariant value received expected %s, got %s", "default-variant", att.Value.AsString())
				}
			case AttributeEvaluatedValue:
				valuePresent = true
			default:
				t.Fatalf("unexpected span key received %s", att.Key)
			}
		}
		if valuePresent && variantPreset {
			t.Fatal("both AttributeEvaluatedVariant and AttributeEvaluatedValue are present in the span")
		}
	})

	t.Run("ensure Error method is behaving correctly", func(t *testing.T) {
		mySpan := spanMock{}
		myError := errors.New("my error")
		hook := Hook{
			spans: map[string]*mutexWrapper{
				".": {
					ss: &storedSpan{
						span: &mySpan,
					},
				},
			},
		}

		hook.Error(openfeature.HookContext{}, myError, openfeature.HookHints{})
		if !errors.Is(myError, mySpan.err) {
			t.Fatalf("unexpected error received, want %v, got %v", myError, mySpan.err)
		}
		if mySpan.code != codes.Error {
			t.Fatalf("unexpected code received, want %v, got %v", codes.Error, mySpan.code)
		}
		if mySpan.errString != myError.Error() {
			t.Fatalf("unexpected errorString received, want %v, got %v", mySpan.errString, myError.Error())
		}
	})

	t.Run("all values are being cast to string", func(t *testing.T) {
		tests := map[string]struct {
			value    interface{}
			flagType openfeature.Type
		}{
			"bool": {
				value:    true,
				flagType: openfeature.Boolean,
			},
			"string": {
				value:    "hello",
				flagType: openfeature.String,
			},
			"int": {
				value:    int64(1),
				flagType: openfeature.Int,
			},
			"float": {
				value:    float64(1.001),
				flagType: openfeature.Float,
			},
			"object": {
				value:    map[string]interface{}{"foo": "bar"},
				flagType: openfeature.Object,
			},
		}
		for name, test := range tests {
			t.Run(name, func(t *testing.T) {
				mySpan := spanMock{}
				hook := Hook{
					tracerClient: &tracerClientMock{
						t: &tracerMock{
							span: &mySpan,
						},
					},
				}
				_, _ = hook.Before(openfeature.HookContext{}, openfeature.HookHints{})
				_ = hook.After(openfeature.HookContext{}, openfeature.InterfaceEvaluationDetails{
					Value: test.value,
					EvaluationDetails: openfeature.EvaluationDetails{
						FlagType: test.flagType,
					},
				}, openfeature.HookHints{})
				hook.Finally(openfeature.HookContext{}, openfeature.HookHints{})
				hook.Wait()
				found := false
				for _, att := range mySpan.attributes {
					if att.Key == AttributeEvaluatedValue {
						found = true
						if att.Value.Type() != attribute.STRING {
							t.Fatalf("unexpected value type received, expected string, got %v", att.Value.Type())
						}
					}
				}
				if !found {
					t.Fatalf("%s not found in span", name)
				}
			})
		}

	})
}
