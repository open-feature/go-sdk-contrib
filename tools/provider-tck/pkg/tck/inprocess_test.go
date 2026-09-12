package tck_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/open-feature/go-sdk-contrib/tools/provider-tck/pkg/tck"
	"github.com/open-feature/go-sdk/openfeature"
	"github.com/open-feature/go-sdk/openfeature/memprovider"
)

// disabledFlagKeys is every flag the canonical set disables, which is the only
// group in it whose configured variant is not what an evaluation resolves.
//
// Named once because two tests need the same list from opposite sides: the
// table in TestCanonicalFlagSetMatchesTheFile asserts that each of these
// decoded as DISABLED, and TestOnlyTheDisabledFlagsAreDisabled asserts that
// nothing else did. Appendix F states that these four are the only ones, and a
// fifth appearing in a later revision of the assets breaks the untagged
// scenarios that expect every flag to serve its own value — which is a thing to
// find out from a failing test here rather than from a provider suite.
var disabledFlagKeys = []string{
	"disabled-boolean-flag",
	"disabled-string-flag",
	"disabled-integer-flag",
	"disabled-float-flag",
}

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
//
// The state column is load-bearing for the same kind of reason. Every row here
// names the variant the flag is configured to serve, and for the four disabled
// rows that configuration is exactly what must NOT reach an evaluation: the
// variant is in the file, the decoder has to preserve it, and the provider has
// to ignore it in favour of the caller's default. A decoder that dropped State
// would leave those four resolving their configured value, which is the one
// thing the @disabled-flags scenarios are built to catch, and every assertion
// in this test would still pass. So the state each flag was written with is
// asserted here rather than assumed, and disabledFlagKeys below is the same
// statement in the other direction: nothing outside that set is disabled.
func TestCanonicalFlagSetMatchesTheFile(t *testing.T) {
	flags := tck.CanonicalFlagSet()

	for _, want := range []struct {
		key     string
		variant string
		value   any
		state   memprovider.State
	}{
		{"boolean-flag", "on", true, memprovider.Enabled},
		{"string-flag", "greeting", "hi", memprovider.Enabled},
		{"integer-flag", "ten", int64(10), memprovider.Enabled},
		{"float-flag", "half", 0.5, memprovider.Enabled},
		{"large-integer-flag", "max-int32", int64(2147483647), memprovider.Enabled},
		{"huge-integer-flag", "max-safe", int64(9007199254740991), memprovider.Enabled},
		{"integral-float-flag", "ten", 10.0, memprovider.Enabled},
		{"boolean-zero-flag", "zero", false, memprovider.Enabled},
		{"integer-zero-flag", "zero", int64(0), memprovider.Enabled},
		{"string-zero-flag", "zero", "", memprovider.Enabled},
		{"wrong-flag", "one", "uno", memprovider.Enabled},
		{"targeting-key-flag", "miss", "miss", memprovider.Enabled},
		{tck.ChangingFlagKey, "foo", "foo", memprovider.Enabled},
		{"disabled-boolean-flag", "on", true, memprovider.Disabled},
		{"disabled-string-flag", "greeting", "hi", memprovider.Disabled},
		{"disabled-integer-flag", "ten", int64(10), memprovider.Disabled},
		{"disabled-float-flag", "half", 0.5, memprovider.Disabled},
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
		if flag.State != want.state {
			t.Errorf("%s decoded with state %q, want %q: the state is what decides whether the "+
				"variant below is served or stood in for by the caller's default",
				want.key, flag.State, want.state)
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

// TestOnlyTheDisabledFlagsAreDisabled pins the other half of the state
// property: the four disabled-* flags are disabled, and nothing else is.
//
// Every scenario outside @disabled-flags assumes the flag it asks for serves
// its own configured value. Disabling any other flag would break those
// quietly — the resolved value becomes whatever default the row passes in,
// which for several rows is a plausible value — so the set is asserted whole
// rather than flag by flag.
func TestOnlyTheDisabledFlagsAreDisabled(t *testing.T) {
	expected := make(map[string]bool, len(disabledFlagKeys))
	for _, key := range disabledFlagKeys {
		expected[key] = true
	}

	for key, flag := range tck.CanonicalFlagSet() {
		disabled := flag.State == memprovider.Disabled
		switch {
		case disabled && !expected[key]:
			t.Errorf("%s is disabled but is not one of the four flags Appendix F says are: every "+
				"other scenario assumes the flag it asks for serves its own value, so this breaks "+
				"them with a plausible-looking result rather than an error", key)
		case !disabled && expected[key]:
			t.Errorf("%s is not disabled, but the @disabled-flags scenarios ask it for the "+
				"caller's default; enabled, it serves its own value and the row fails", key)
		}
	}
}

// TestCanonicalFlagSetDisabledFlagsCarryAnError is the evidence behind every
// in-memory suite leaving tck.DisabledFlags undeclared, and it is the
// uncomfortable kind: the gap is in the Go SDK rather than in this suite or in
// the flag set.
//
// memprovider.InMemoryProvider does return the caller's default for a disabled
// flag, which is the value half of the capability and the half that matters
// most. But it attaches a GENERAL resolution error to it while setting reason
// DISABLED (Resolve in openfeature/memprovider/in_memory_provider.go), and the
// two do not go together: DISABLED is one of the reason strings 2.2.5 lists for
// a resolution that worked, and an error code alongside it tells the
// application something went wrong when nothing did. So "the error-code should
// be \"\"" fails, all four rows of the outline with it, and the capability is
// withheld.
//
// Pinned here because the withholding is otherwise invisible — it is an
// absence from three Config literals — and because this is a bug rather than a
// property of in-memory evaluation: an in-memory provider is the architecture
// that CAN satisfy this capability, since the caller's default never has to
// leave the process. When the SDK stops attaching the error this test fails,
// and the fix is to declare the capability in the self-test suites rather than
// to relax the assertion.
func TestCanonicalFlagSetDisabledFlagsCarryAnError(t *testing.T) {
	ctx := context.Background()
	provider := tck.NewInProcessControl().NewProvider()

	result := provider.BooleanEvaluation(ctx, "disabled-boolean-flag", false, nil)

	if result.Value {
		t.Errorf("disabled-boolean-flag resolved to true, so the state was ignored and its "+
			"configured variant was served; the caller passed false")
	}
	code := result.ResolutionDetail().ErrorCode
	if code == "" {
		t.Fatal("disabled-boolean-flag now resolves with no error code, which is the whole of " +
			"tck.DisabledFlags: declare the capability in the in-memory self-test suites and " +
			"delete this test")
	}
	if code != openfeature.GeneralCode {
		t.Errorf("disabled-boolean-flag resolved with error code %q; the capability is withheld "+
			"on the strength of it being %q, so a different code means the reason for "+
			"withholding has changed", code, openfeature.GeneralCode)
	}
	if result.Reason != openfeature.DisabledReason {
		t.Errorf("disabled-boolean-flag resolved with reason %q, want %q", result.Reason,
			openfeature.DisabledReason)
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
