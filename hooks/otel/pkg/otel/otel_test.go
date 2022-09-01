package otel

import (
	"testing"

	"github.com/open-feature/golang-sdk/pkg/openfeature"
)

func TestOtelHookMethods(t *testing.T) {
	otelHook := Hook{}
	t.Run("Before should start a new span", func(t *testing.T) {
		otelHook.Before(openfeature.HookContext{}, openfeature.HookHints{})
		if len(otelHook.spans) != 1 {
			t.Fatal("before hook did not create a new span")
		}
	})

	t.Run("After hook should trigger the span to close with no error", func(t *testing.T) {
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
}
