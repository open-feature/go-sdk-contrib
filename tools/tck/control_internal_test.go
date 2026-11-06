package tck

import (
	"context"
	"testing"
)

// untypedControl is a control that does not state its backend type, which is
// what an adopter writing their own control gets if they say nothing.
type untypedControl struct{}

func (untypedControl) PrepareScenario(context.Context) error { return nil }
func (untypedControl) ChangeFlag(context.Context) error      { return nil }
func (untypedControl) Description() string                   { return "untyped" }

// typedControl states one, as both controls this package ships do.
type typedControl struct{ backendType BackendType }

func (typedControl) PrepareScenario(context.Context) error { return nil }
func (typedControl) ChangeFlag(context.Context) error      { return nil }
func (typedControl) Description() string                   { return "typed" }
func (c typedControl) BackendType() BackendType            { return c.backendType }

func TestBackendTypeDefaultsToCustom(t *testing.T) {
	if got := backendTypeOf(untypedControl{}); got != BackendTypeCustom {
		t.Fatalf("a control that states nothing should report %q, got %q", BackendTypeCustom, got)
	}
}

func TestBackendTypeIsTakenFromTheControlThatStatesIt(t *testing.T) {
	for _, want := range []BackendType{BackendTypeHTTP, BackendTypeInProcess, BackendTypeCustom} {
		if got := backendTypeOf(typedControl{backendType: want}); got != want {
			t.Errorf("want %q, got %q", want, got)
		}
	}
}

func TestShippedControlsStateTheirBackendType(t *testing.T) {
	// A regression guard: these two must keep implementing BackendTyper, or
	// every report they produce silently becomes "custom".
	if _, ok := any(&HTTPControl{}).(BackendTyper); !ok {
		t.Error("HTTPControl no longer implements BackendTyper")
	}
	if _, ok := any(&InProcessControl{}).(BackendTyper); !ok {
		t.Error("InProcessControl no longer implements BackendTyper")
	}
}
