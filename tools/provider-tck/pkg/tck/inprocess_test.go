package tck_test

import (
	"context"
	"errors"
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
