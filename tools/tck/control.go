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
// declared the backend cannot back up. It fails loudly rather than skipping,
// because a silent no-op would report the scenario as passed.
var ErrUnsupportedControl = errors.New("backend control operation not supported")

// BackendType names what kind of backend a run drove.
//
// It is a defined type with a closed set of values rather than a string,
// because the value ends up in the conformance report against a schema enum:
// "HTTP" or "in_process" would compile, pass every test, and produce a report
// that fails validation somewhere the author cannot see it.
type BackendType string

const (
	// BackendTypeHTTP means the backend was driven over the normative HTTP
	// control API. Every provider with a real backend answers this.
	BackendTypeHTTP BackendType = "http"

	// BackendTypeInProcess means flag state was manipulated in this process,
	// which is the narrow allowance made for a provider that has no backend at
	// all. A report claiming it for a provider that does have one should be
	// treated with suspicion, which is why it is recorded rather than assumed.
	BackendTypeInProcess BackendType = "in-process"

	// BackendTypeCustom means the run drove a control this package does not
	// ship, so the claim is not directly comparable with the two above. It is
	// the honest answer for a control of one's own: better a report that says
	// the suite cannot vouch for how the backend was driven than one claiming
	// the normative path it did not take.
	BackendTypeCustom BackendType = "custom"
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
// In-process control is a narrow allowance for backend-less providers, not a
// choice; [Appendix F] explains why.
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
}

// BackendTyper is implemented by a BackendControl that states what kind of
// backend it drives, for the conformance report.
//
// It is a separate interface rather than a method on BackendControl so that a
// control which cannot honestly claim either shipped path does not have to
// claim one: a control that does not implement this is reported as
// BackendTypeCustom. Both controls this package ships implement it, so the only
// author who decides is the one writing a control of their own -- and the
// default is the claim that overstates nothing.
type BackendTyper interface {
	// BackendType states what kind of backend this control drives.
	BackendType() BackendType
}

// backendTypeOf reports what kind of backend a control drives, defaulting to
// BackendTypeCustom for a control that does not say.
func backendTypeOf(control BackendControl) BackendType {
	if typed, ok := any(control).(BackendTyper); ok {
		return typed.BackendType()
	}
	return BackendTypeCustom
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

// ConfigurationControl is implemented by a BackendControl whose backend serves
// more than one named flag configuration.
//
// A separate interface for the reason ConnectionControl is one: a control over
// a single in-process flag set has no second configuration to offer, and a
// no-op would keep serving the baseline while the scenario believed it had
// asked for something else.
//
// The names belong to the backend rather than to the TCK — the control API
// takes config as a free string defaulting to DefaultBackendConfiguration — so
// a scenario naming one is naming something only that backend's control
// defines. That makes this an extension-scenario operation. The canonical
// scenarios never ask: the baseline every control already serves is the
// configuration they are written against.
type ConfigurationControl interface {
	// ServeConfiguration restarts the backend on one of its own named flag
	// configurations, which also seeds that configuration's flag state.
	//
	// It has to take effect before the provider connects, which is why the one
	// step that calls it registers the provider too. Reconnect afterwards
	// brings this configuration back rather than the suite's, because losing
	// the connection is not how a scenario changes what is served; the next
	// PrepareScenario takes the backend back to the suite's own.
	ServeConfiguration(ctx context.Context, configuration string) error
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

// configurationControl returns the ConfigurationControl view of a
// BackendControl, or an error naming what is missing.
//
// Its own message rather than unsupportedControl's: no capability gates this
// operation, so the fix is never "remove a capability". A scenario reaches it
// by naming a configuration, which only an extension feature does.
func configurationControl(control BackendControl) (ConfigurationControl, error) {
	cc, ok := control.(ConfigurationControl)
	if !ok {
		return nil, fmt.Errorf(
			"%w: %s does not support %q. A scenario asked the backend to serve a named flag "+
				"configuration, which only an extension feature does — the canonical scenarios "+
				"are written against the baseline every control already serves. Either drop the "+
				"configuration from that scenario, or supply a BackendControl that implements "+
				"tck.ConfigurationControl",
			ErrUnsupportedControl, control.Description(), "ServeConfiguration")
	}
	return cc, nil
}
