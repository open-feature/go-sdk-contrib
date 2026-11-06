package tck

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/open-feature/go-sdk/openfeature"
	"github.com/open-feature/go-sdk/openfeature/memprovider"
)

// recordingConfigurationControl records the order in which it is driven, which
// is what the combined provider-and-configuration step exists to fix.
type recordingConfigurationControl struct {
	mu    sync.Mutex
	calls []string
}

var (
	_ BackendControl       = (*recordingConfigurationControl)(nil)
	_ ConfigurationControl = (*recordingConfigurationControl)(nil)
)

func (c *recordingConfigurationControl) record(call string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.calls = append(c.calls, call)
}

func (c *recordingConfigurationControl) recorded() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]string(nil), c.calls...)
}

func (c *recordingConfigurationControl) PrepareScenario(context.Context) error {
	c.record("PrepareScenario")
	return nil
}

func (c *recordingConfigurationControl) ChangeFlag(context.Context) error {
	c.record("ChangeFlag")
	return nil
}

func (c *recordingConfigurationControl) Description() string {
	return "a control that records what it was asked to do"
}

func (c *recordingConfigurationControl) ServeConfiguration(_ context.Context, configuration string) error {
	c.record("ServeConfiguration(" + configuration + ")")
	return nil
}

func TestTheConfigurationIsServedBeforeTheProviderIsCreated(t *testing.T) {
	control := &recordingConfigurationControl{}
	ctx, _ := stepState(t,
		WithName("configuration-order"),
		WithControl(control),
		WithProvider(func(context.Context) (openfeature.FeatureProvider, error) {
			control.record("NewProvider")
			return memprovider.NewInMemoryProvider(nil), nil
		}),
	)
	t.Cleanup(func() {
		if err := openfeature.SetNamedProvider("tck/configuration-order", openfeature.NoopProvider{}); err != nil {
			t.Logf("could not release the provider: %v", err)
		}
	})

	if _, err := aProviderWithConfiguration(ctx, "stable", "metadata"); err != nil {
		t.Fatalf("a stable provider with \"metadata\" configuration: %v", err)
	}

	// This order is the entire reason the two are one step. Served afterwards,
	// the provider would have connected to the baseline flag set and the
	// scenario would assert against flags it never saw.
	want := []string{"ServeConfiguration(metadata)", "NewProvider"}
	if got := control.recorded(); strings.Join(got, " | ") != strings.Join(want, " | ") {
		t.Errorf("control call order\n got: %s\nwant: %s", strings.Join(got, " | "), strings.Join(want, " | "))
	}
}

func TestAControlOverOneFlagSetRefusesTheConfigurationStep(t *testing.T) {
	ctx, _ := stepState(t, WithName("one-flag-set"), WithControl(untypedControl{}))

	_, err := aProviderWithConfiguration(ctx, "stable", "metadata")
	if err == nil {
		t.Fatal("a control with one flag set accepted a request for a named configuration")
	}
	if !errors.Is(err, ErrUnsupportedControl) {
		t.Errorf("the error is not an unsupported-control error: %v", err)
	}
	// The fix is never "remove a capability" here: no capability gates this
	// operation, and a scenario reaches it by naming a configuration.
	if !strings.Contains(err.Error(), "tck.ConfigurationControl") {
		t.Errorf("the error does not name what to implement: %v", err)
	}
	if strings.Contains(err.Error(), "WithCapabilities") {
		t.Errorf("the error offers a capability fix that does not apply: %v", err)
	}
}
