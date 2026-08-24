package tck

import (
	"context"
	"errors"
	"fmt"
)

// ErrUnsupportedControl is returned by BackendControl operations a particular
// backend cannot perform.
//
// It is always a test-configuration bug rather than a provider defect. The
// scenarios that need connection control are gated behind Stale and
// UnavailableInit, so reaching an unsupported operation means a capability was
// declared that the backend cannot back up. The TCK fails loudly on it rather
// than skipping, because a silent no-op would report the scenario as passed.
var ErrUnsupportedControl = errors.New("backend control operation not supported")

// BackendControl is the single seam between the TCK's scenarios and whatever
// manipulates the backend under test.
//
// Step definitions never talk to a backend directly. They talk to this
// interface, which is why the same Gherkin runs unchanged against a
// containerised backend driven over HTTP and against a provider manipulated
// in-process. Nothing below this line knows about ports, containers or
// transports.
//
// # Which implementation is right for your provider
//
// If your provider talks to a backend — a server, a service, anything out of
// process — drive it over the HTTP control API described in
// spec/specification/assets/provider-tck/openapi/control-api.yaml. That API is
// the normative contract for those providers, and it is what makes a
// conformance claim portable: another language's TCK drives the same endpoints
// against the same stack and must get the same answers.
//
// Do not write an in-process BackendControl that reaches into an external
// backend through a side channel — a test-only admin client, a shared database
// handle, a hook inside the provider. It will pass, and it will prove nothing,
// because the path it exercised is not the path the contract describes.
//
// In-process control exists for providers that have no backend to contract
// with: in-memory, environment-variable and file-based providers, where "the
// backend" is a data structure in the same process. See InProcessControl.
//
// # Operations a backend may not support
//
// PrepareScenario and ChangeFlag are mandatory: a backend that can neither
// reset itself nor change a flag cannot run the suite at all.
//
// Connection control is not. A provider with nothing to disconnect from simply
// does not implement ConnectionControl, and leaves the Stale and
// UnavailableInit capabilities undeclared, so those scenarios are skipped
// before any step can reach an unsupported operation.
type BackendControl interface {
	// PrepareScenario brings the backend to the state every scenario starts
	// from: reachable, with flag state at the baseline of the canonical flag
	// set.
	//
	// Called once before each scenario. This is the TCK's only isolation
	// mechanism — scenarios share one backend for the whole suite, and
	// containers are never restarted between them.
	PrepareScenario(ctx context.Context) error

	// ChangeFlag mutates flag configuration so that a conforming provider
	// observes a configuration change and afterwards resolves a different value
	// for changing-flag.
	//
	// Which value it changes to is deliberately unspecified; the suite asserts
	// only that the resolved value differs from what it was before.
	ChangeFlag(ctx context.Context) error

	// Description returns a short description of what is being controlled, for
	// startup logging and for the failure messages of unsupported operations.
	Description() string
}

// ConnectionControl is implemented by a BackendControl whose backend can be cut
// off from the provider and restored.
//
// It is a separate interface rather than two more methods on BackendControl
// so that a backend-less provider cannot accidentally supply a no-op
// implementation: not implementing it at all is the honest answer, and the TCK
// turns the resulting gap into an explicit, reported skip.
type ConnectionControl interface {
	// Disconnect makes the backend unreachable for the rest of the scenario,
	// without stopping any container.
	Disconnect(ctx context.Context) error

	// Reconnect makes the backend reachable again after Disconnect, preserving
	// flag state so the provider observes an availability change rather than a
	// configuration change.
	//
	// Preserving flag state is a requirement, not an implementation detail. An
	// outage must be observable as a change in availability and never as a
	// change in flag values, or the stale scenario cannot distinguish the two.
	Reconnect(ctx context.Context) error
}

// unsupportedControl builds the error the connection operations return when the
// backend has no connection to control. The message names the fix, because the
// mistake it reports is always the same one.
func unsupportedControl(control BackendControl, operation string) error {
	return fmt.Errorf(
		"%w: %s does not support %q. This is a test-configuration bug rather than a provider "+
			"defect: a scenario needing connection control ran, so the suite declared "+
			"tck.Stale or tck.UnavailableInit for a backend that cannot simulate an outage. "+
			"Remove those capabilities from Config.Capabilities, or supply a BackendControl "+
			"that implements tck.ConnectionControl",
		ErrUnsupportedControl, control.Description(), operation)
}

// connectionControl returns the ConnectionControl view of a BackendControl, or
// an error naming the operation that is missing.
func connectionControl(control BackendControl, operation string) (ConnectionControl, error) {
	cc, ok := control.(ConnectionControl)
	if !ok {
		return nil, unsupportedControl(control, operation)
	}
	return cc, nil
}
