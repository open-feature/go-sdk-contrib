package tck_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/open-feature/go-sdk-contrib/tools/provider-tck/pkg/tck"
	"github.com/open-feature/go-sdk/openfeature"
)

// The Gherkin cannot assert these things about itself, so they are pinned here.
// Each one is a way the in-process control path could look correct while
// quietly making the conformance suites meaningless.

// TestChangeFlagActuallyChangesTheResolvedValue guards the assumption every
// configuration-change scenario rests on.
//
// If ChangeFlag emitted an event without altering what the provider resolves,
// the scenario would still pass its event assertion, and the suite would be
// certifying a signal with nothing behind it.
func TestChangeFlagActuallyChangesTheResolvedValue(t *testing.T) {
	ctx := context.Background()
	control := tck.NewInProcessControl()
	provider := control.NewProvider()

	before := provider.StringEvaluation(ctx, tck.ChangingFlagKey, "unset", nil)
	if before.Error() != nil {
		t.Fatalf("resolving %s before the change failed: %v", tck.ChangingFlagKey, before.Error())
	}

	if err := control.ChangeFlag(ctx); err != nil {
		t.Fatalf("ChangeFlag: %v", err)
	}

	after := provider.StringEvaluation(ctx, tck.ChangingFlagKey, "unset", nil)
	if after.Error() != nil {
		t.Fatalf("resolving %s after the change failed: %v", tck.ChangingFlagKey, after.Error())
	}

	if before.Value == after.Value {
		t.Fatalf("ChangeFlag left %s resolving to %q; the change was not applied",
			tck.ChangingFlagKey, after.Value)
	}
}

// TestChangeFlagEmitsAConfigurationChangeEvent checks that the event the
// scenarios await is the provider's own, and that it names the flag.
func TestChangeFlagEmitsAConfigurationChangeEvent(t *testing.T) {
	ctx := context.Background()
	control := tck.NewInProcessControl()
	provider := control.NewProvider()

	if err := control.ChangeFlag(ctx); err != nil {
		t.Fatalf("ChangeFlag: %v", err)
	}

	select {
	case event := <-provider.EventChannel():
		if event.EventType != openfeature.ProviderConfigChange {
			t.Fatalf("got event type %q, want %q", event.EventType, openfeature.ProviderConfigChange)
		}
		if len(event.FlagChanges) != 1 || event.FlagChanges[0] != tck.ChangingFlagKey {
			t.Fatalf("event named %v, want [%s]", event.FlagChanges, tck.ChangingFlagKey)
		}
	case <-time.After(time.Second):
		t.Fatal("no configuration-change event was emitted")
	}
}

// TestChangeDoesNotLeakIntoTheNextScenario pins scenario isolation.
//
// A leak here would make the suite order-dependent: a scenario that ran after
// the configuration-change one would start with changing-flag already flipped,
// and the failure would look like a provider defect.
func TestChangeDoesNotLeakIntoTheNextScenario(t *testing.T) {
	ctx := context.Background()
	control := tck.NewInProcessControl()

	first := control.NewProvider()
	baseline := first.StringEvaluation(ctx, tck.ChangingFlagKey, "unset", nil).Value

	if err := control.ChangeFlag(ctx); err != nil {
		t.Fatalf("ChangeFlag: %v", err)
	}
	if changed := first.StringEvaluation(ctx, tck.ChangingFlagKey, "unset", nil).Value; changed == baseline {
		t.Fatalf("precondition failed: ChangeFlag did not change the resolved value")
	}

	if err := control.PrepareScenario(ctx); err != nil {
		t.Fatalf("PrepareScenario: %v", err)
	}

	second := control.NewProvider()
	if got := second.StringEvaluation(ctx, tck.ChangingFlagKey, "unset", nil).Value; got != baseline {
		t.Fatalf("the next scenario starts with %s resolving to %q, want the baseline %q",
			tck.ChangingFlagKey, got, baseline)
	}
}

// TestChangeFlagWithoutAProviderFailsClearly checks the diagnostic for a
// scenario that manipulates flags before creating a provider. In-process the
// flag store and the provider are the same object, so there is nothing to
// change; saying so beats a nil dereference.
func TestChangeFlagWithoutAProviderFailsClearly(t *testing.T) {
	control := tck.NewInProcessControl()

	err := control.ChangeFlag(context.Background())
	if err == nil {
		t.Fatal("ChangeFlag succeeded with no provider for the scenario")
	}
}

// TestInProcessControlDoesNotPretendToHaveAConnection is the load-bearing one.
//
// A no-op Disconnect would report the @stale scenarios as passed against a
// provider that cannot go stale, which is precisely the silent-green failure a
// conformance suite must never have. tck.InProcessControl therefore does not
// implement tck.ConnectionControl at all, and the TCK turns that into a skip
// with a reason rather than a pass.
func TestInProcessControlDoesNotPretendToHaveAConnection(t *testing.T) {
	control := tck.NewInProcessControl()

	if _, implements := any(control).(tck.ConnectionControl); implements {
		t.Fatal("InProcessControl implements ConnectionControl: an in-memory provider has no " +
			"connection to lose, and a no-op implementation would make the @stale scenarios " +
			"pass without testing anything")
	}
}

// TestCanonicalFlagSetOmitsMissingFlag pins the property the FLAG_NOT_FOUND
// scenario depends on. Seeding missing-flag would turn that scenario green for
// the wrong reason, and nothing else in the suite would notice.
func TestCanonicalFlagSetOmitsMissingFlag(t *testing.T) {
	if _, present := tck.CanonicalFlagSet()["missing-flag"]; present {
		t.Fatal("the canonical flag set contains missing-flag; its absence is what the " +
			"FLAG_NOT_FOUND scenario tests")
	}
}

// TestCanonicalFlagSetMatchesTheFile pins what the decoding of
// canonical-flags.json must preserve, because a loader that "cleans up" values
// destroys exactly what the scenarios test.
//
// The numeric rows are the load-bearing ones. encoding/json decodes every
// number as float64, so a loader that turned integral float64s back into
// int64 would silently make integral-float-flag an integer flag and let the
// lossless-coercion scenario pass without coercing; and 9007199254740991 has
// to arrive as an int64 rather than through a float. The falsy rows pin that
// false, 0 and "" are values rather than absences, which is what their
// scenarios are about.
func TestCanonicalFlagSetMatchesTheFile(t *testing.T) {
	flags := tck.CanonicalFlagSet()

	for _, want := range []struct {
		key     string
		variant string
		value   any
	}{
		{"boolean-flag", "on", true},
		{"string-flag", "greeting", "hi"},
		{"integer-flag", "ten", int64(10)},
		{"float-flag", "half", 0.5},
		{"large-integer-flag", "max-int32", int64(2147483647)},
		{"huge-integer-flag", "max-safe", int64(9007199254740991)},
		{"integral-float-flag", "ten", 10.0},
		{"boolean-zero-flag", "zero", false},
		{"integer-zero-flag", "zero", int64(0)},
		{"string-zero-flag", "zero", ""},
		{"wrong-flag", "one", "uno"},
		{"targeting-key-flag", "miss", "miss"},
		{tck.ChangingFlagKey, "foo", "foo"},
	} {
		flag, present := flags[want.key]
		if !present {
			t.Errorf("%s is missing from the canonical flag set", want.key)
			continue
		}
		if flag.Key != want.key {
			t.Errorf("%s carries Key %q", want.key, flag.Key)
		}
		if flag.DefaultVariant != want.variant {
			t.Errorf("%s resolves to variant %q, want %q", want.key, flag.DefaultVariant, want.variant)
		}
		got, present := flag.Variants[want.variant]
		if !present {
			t.Errorf("%s has no variant %q", want.key, want.variant)
			continue
		}
		// reflect.DeepEqual distinguishes int64(10) from float64(10), which is
		// the distinction under test.
		if !reflect.DeepEqual(got, want.value) {
			t.Errorf("%s variant %q is %v (%T), want %v (%T)",
				want.key, want.variant, got, got, want.value, want.value)
		}
	}

	// ChangeFlag flips changing-flag between two variants it names itself, so
	// the file has to define exactly those.
	changing := flags[tck.ChangingFlagKey]
	for _, variant := range []string{"foo", "bar"} {
		if _, present := changing.Variants[variant]; !present {
			t.Errorf("%s has no variant %q, which ChangeFlag switches to", tck.ChangingFlagKey, variant)
		}
	}

	// The member inside object-flag has to be converted like a top-level
	// number, or the type-aware comparison of the object scenario sees a
	// json.Number.
	template, ok := flags["object-flag"].Variants["template"].(map[string]any)
	if !ok {
		t.Fatalf("object-flag's template variant is a %T, want map[string]any", flags["object-flag"].Variants["template"])
	}
	if got := template["imagesPerPage"]; !reflect.DeepEqual(got, int64(100)) {
		t.Errorf("object-flag's imagesPerPage is %v (%T), want int64(100)", got, got)
	}
}

// TestCanonicalFlagSetEvaluatesNoTargeting is the evidence behind every
// in-process suite leaving tck.Targeting undeclared.
//
// canonical-flags.json gives targeting-key-flag a JsonLogic rule, and
// tck.CanonicalFlagSet deliberately does not read it: translating that rule
// into a memprovider ContextEvaluator would make these self-tests a test of a
// rule engine written here. The consequence is that a matching targeting key
// resolves the miss variant, so the @targeting scenarios cannot pass against
// this flag set — which is a fact about the flag set, not a provider defect,
// and the reason the capability is withheld rather than recorded as a
// deviation.
//
// Pinned here because it is otherwise invisible: the capability is simply
// absent from four Config literals, and a future decoder that started honouring
// the targeting member would make those omissions wrong with nothing failing to
// say so.
func TestCanonicalFlagSetEvaluatesNoTargeting(t *testing.T) {
	const (
		key     = "targeting-key-flag"
		hitting = "5c3d8535-f81a-4478-a6d3-afaa4d51199e"
	)

	flag, present := tck.CanonicalFlagSet()[key]
	if !present {
		t.Fatalf("%s is missing from the canonical flag set", key)
	}
	if flag.ContextEvaluator != nil {
		t.Fatalf("%s carries a ContextEvaluator: the in-memory suites declare no Targeting "+
			"capability on the strength of it having none, so declaring it is now the honest "+
			"report and those Config literals have to say so", key)
	}

	ctx := context.Background()
	provider := tck.NewInProcessControl().NewProvider()
	result := provider.StringEvaluation(ctx, key, "fallback", map[string]any{
		openfeature.TargetingKey: hitting,
	})
	if result.Error() != nil {
		t.Fatalf("resolving %s with a matching targeting key failed: %v", key, result.Error())
	}
	if result.Value != "miss" {
		t.Fatalf("%s resolved to %q for a matching targeting key; the rule is being evaluated "+
			"after all, so the in-memory suites should declare tck.Targeting", key, result.Value)
	}
}

// TestControllableProviderRejectsUpdatesAfterShutdown checks that a mutation
// arriving after the SDK has shut the provider down is reported rather than
// panicking on a closed channel.
func TestControllableProviderRejectsUpdatesAfterShutdown(t *testing.T) {
	control := tck.NewInProcessControl()
	provider := control.NewProvider()

	provider.Shutdown()
	provider.Shutdown() // must be idempotent; the SDK may call it more than once

	err := control.ChangeFlag(context.Background())
	if !errors.Is(err, tck.ErrProviderShutdown) {
		t.Fatalf("ChangeFlag after shutdown returned %v, want %v", err, tck.ErrProviderShutdown)
	}
}
