package tck

import (
	"context"
	"errors"
	"fmt"
)

// ErrUnsupportedControl is returned by BackendControl operations a particular
// backend cannot perform.
//
// It is always a test-configuration bug rather than a provider defect: the
// scenarios needing connection control are gated behind Stale and
// UnavailableInit, so reaching an unsupported operation means a capability was
// declared the backend cannot back up. Appendix F requires this to fail loudly
// rather than skip, a silent no-op reporting the scenario as passed.
var ErrUnsupportedControl = errors.New("backend control operation not supported")

// ControlAPI names the path a run took to manipulate the backend under test.
//
// It is a defined type with a closed set of values rather than a string,
// because the value ends up in the conformance report against a schema enum:
// "HTTP" or "in_process" would compile, pass every test, and produce a report
// that fails validation somewhere the author cannot see it.
type ControlAPI string

const (
	// ControlAPIHTTP means the backend was driven over the normative HTTP
	// control API. Every provider with a real backend answers this.
	ControlAPIHTTP ControlAPI = "http"

	// ControlAPIInProcess means flag state was manipulated in this process,
	// which is the narrow allowance made for a provider that has no backend at
	// all. A report claiming it for a provider that does have one should be
	// treated with suspicion, which is precisely why it is recorded rather
	// than assumed.
	ControlAPIInProcess ControlAPI = "in-process"
)

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
// A provider that talks to a backend — a server, a service, anything out of
// process — drives it over the HTTP control API: WithComposeFile builds an
// HTTPControl for you. A provider that has no backend to contract with may
// control one in-process; see InProcessControl.
//
// The choice is not a matter of taste, and [Appendix F] states why in the terms
// that matter: in-process control is a narrow allowance for backend-less
// providers, and a control that reaches into an external backend through a side
// channel passes while proving nothing.
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
//
// [Appendix F]: https://github.com/open-feature/spec/blob/main/specification/appendix-f-provider-conformance.md#providers-with-no-backend
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

	// ControlAPI states which path this control drives the backend over, for
	// the conformance report.
	//
	// Required, with no default and nothing inferred from the concrete type,
	// which is Appendix F's rule and its argument for it. Both controls this
	// package ships answer it already, so an adopter using WithComposeFile or
	// InProcessControl writes nothing. The only author who has to state it is
	// the one writing a control of their own — which is exactly the case where
	// it cannot be guessed.
	ControlAPI() ControlAPI
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
			"Remove those capabilities from tck.WithCapabilities, or supply a BackendControl "+
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
