package otel

import (
	"context"
	"testing"
	"time"

	"github.com/open-feature/golang-sdk/pkg/openfeature"
)

func TestOtelHookMethods(t *testing.T) {

	t.Run("Before should start a new span", func(t *testing.T) {
		otelHook := Hook{}
		otelHook.Before(openfeature.HookContext{}, openfeature.HookHints{})
		if len(otelHook.spans) != 1 {
			t.Fatal("before hook did not create a new span")
		}
	})

	t.Run("After hook should trigger the span to close with no error", func(t *testing.T) {
		otelHook := Hook{}
		otelHook.Before(openfeature.HookContext{}, openfeature.HookHints{})
		if len(otelHook.spans) != 1 {
			t.Fatal("before hook did not create a new span")
		}
		err := otelHook.After(openfeature.HookContext{}, openfeature.EvaluationDetails{
			FlagKey:  "testKey",
			FlagType: openfeature.Boolean,
			ResolutionDetail: openfeature.ResolutionDetail{
				Value: true,
			},
		}, openfeature.HookHints{})
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
		err := otelHook.After(openfeature.HookContext{}, openfeature.EvaluationDetails{
			FlagKey:  "testKey",
			FlagType: openfeature.Boolean,
			ResolutionDetail: openfeature.ResolutionDetail{
				Value: true,
			},
		}, openfeature.HookHints{})
		if err != nil {
			t.Fatal(err)
		}
		// account for slow execution time to ensure that goroutine is no longer blocked
		// (cannot use the .Wait method in this example) as the after method has not yet been called for the blocked goroutine
		time.Sleep(500 * time.Millisecond)
		if blocked {
			t.Fatal("blocked goroutine has not been unblocked by the release of the lock")
		}
		// complete the final hooks lifecycle and ensure that it is being cleaned up
		err = otelHook.After(openfeature.HookContext{}, openfeature.EvaluationDetails{
			FlagKey:  "testKey",
			FlagType: openfeature.Boolean,
			ResolutionDetail: openfeature.ResolutionDetail{
				Value: true,
			},
		}, openfeature.HookHints{})
		if err != nil {
			t.Fatal(err)
		}
		otelHook.Wait()
		for _, x := range otelHook.spans {
			if x.ss != nil {
				t.Fatal("stored span has not been cleaned up")
			}
		}
	})
}
