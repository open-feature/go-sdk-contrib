package tck

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// DefaultBackendConfiguration is the backend configuration name every backend
// under test must support, and the one that serves the canonical flag set.
//
// "Backend" is in the name on purpose: the report's "configuration" field is
// the provider's own mode, which WithName feeds. This is the named flag
// configuration the backend is asked to serve.
const DefaultBackendConfiguration = "default"

// defaultControlTimeout bounds a single control-API request. Control calls are
// local HTTP to a container on the same host; anything slower than this is a
// wedged backend rather than a slow one.
const defaultControlTimeout = 30 * time.Second

// HTTPControl drives a backend under test over the HTTP control API defined in
// assets/openapi/control-api.yaml.
//
// This is the normative control path for any provider with a real backend, and
// it is what makes a conformance claim portable: another language's TCK drives
// the same endpoints against the same stack and must get the same answers.
//
// # What it never does
//
// It never stops, kills or recreates a container: unavailability is simulated
// inside the running stack, through POST /stop. That is Appendix F's
// control-API invariant. Starting and stopping the stack itself belongs to the
// suite, once per run.
//
// # Scenario isolation
//
// PrepareScenario prefers POST /reset, which restores the flag baseline with no
// availability blip and therefore cannot inject a spurious lifecycle event into
// the next scenario. That operation is optional, and a backend that does not
// implement it answers 404 or 501; the TCK then falls back to
// POST /start?config=..., which also resets flag state at the cost of a process
// restart. The fallback is probed once and remembered.
//
// After a disconnect the backend may be down, and /reset is specified to reset
// flag state rather than to start a stopped backend. HTTPControl tracks that
// and uses /start for the scenario following any disconnect. A scenario that
// served a configuration of its own is tracked the same way and for the same
// reason: /reset restores flag state, not which configuration is being served.
type HTTPControl struct {
	baseURL              string
	backendConfiguration string
	client               *http.Client

	mu sync.Mutex
	// resetSupported is nil until the first /reset call tells us.
	resetSupported *bool
	// backendMaybeDown records that a disconnect happened, so the next
	// PrepareScenario starts the backend rather than merely resetting flags.
	backendMaybeDown bool
	// scenarioConfiguration is a configuration one scenario asked for, which
	// the next scenario has to be taken off again. Empty means the suite's own
	// is in effect.
	scenarioConfiguration string
}

var (
	_ BackendControl       = (*HTTPControl)(nil)
	_ ConnectionControl    = (*HTTPControl)(nil)
	_ ConfigurationControl = (*HTTPControl)(nil)
)

// HTTPControlOption configures an HTTPControl.
type HTTPControlOption func(*HTTPControl)

// WithControlBackendConfiguration names the flag configuration the backend is
// asked to serve. Without it the control asks for DefaultBackendConfiguration,
// which is the only name every backend must support and the one serving the
// canonical flag set.
//
// Not to be confused with the provider's own configuration — its mode, which
// the report calls "configuration" and WithName supplies.
func WithControlBackendConfiguration(name string) HTTPControlOption {
	return func(c *HTTPControl) { c.backendConfiguration = name }
}

// WithControlHTTPClient sets the HTTP client the control uses. Without it the
// control uses one with a 30 second timeout.
func WithControlHTTPClient(client *http.Client) HTTPControlOption {
	return func(c *HTTPControl) { c.client = client }
}

// NewHTTPControl returns a control that drives the backend whose control API is
// rooted at baseURL, for example http://localhost:32768.
//
// baseURL must be built from the dynamically mapped host port of the control
// service, discovered after the stack is up. A stack under test must not pin
// host ports.
func NewHTTPControl(baseURL string, opts ...HTTPControlOption) (*HTTPControl, error) {
	if baseURL == "" {
		return nil, errors.New("the HTTP control needs a base URL: it is the root of the " +
			"control API, built from the dynamically mapped host port of the control service")
	}
	if _, err := url.Parse(baseURL); err != nil {
		return nil, fmt.Errorf("control API base URL %q is not a valid URL: %w", baseURL, err)
	}

	control := &HTTPControl{baseURL: strings.TrimSuffix(baseURL, "/")}
	for _, opt := range opts {
		opt(control)
	}
	if control.backendConfiguration == "" {
		control.backendConfiguration = DefaultBackendConfiguration
	}
	if control.client == nil {
		control.client = &http.Client{Timeout: defaultControlTimeout}
	}
	return control, nil
}

// AwaitReady blocks until the control API is willing to accept commands, or
// until timeout expires.
//
// It probes GET /healthz, which the control API specification marks optional:
// a 404 means the control port is answering HTTP but does not serve that path,
// which is the reference implementation's behaviour and is ready enough — the
// TCP wait the stack already passed is the documented fallback. Anything else
// is retried until the deadline.
//
// This is deliberately the only wait in the suite that is a wait rather than an
// assertion. There is no settle after a control call: control-api.yaml promises
// that /start, /change and /reset have taken effect when they return, so a
// scenario that flaps immediately after one is a defect in the backend's
// control API.
//
// The promise is the backend's: a fresh evaluation must resolve the new state
// once the call returns. How long the provider takes to notice is a property of
// its transport, which is what WithEventTimeout is for. Sleeping here would
// make that detection latency unmeasurable.
func (c *HTTPControl) AwaitReady(ctx context.Context, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	var last error

	for attempt := 0; ; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return fmt.Errorf("gave up waiting for the control API at %s: %w", c.baseURL, ctx.Err())
			case <-time.After(controlProbeInterval):
			}
		}

		status, err := c.probe(ctx)
		switch {
		case err != nil:
			last = err
		case status == http.StatusOK || status == http.StatusNotFound:
			return nil
		default:
			last = fmt.Errorf("GET /healthz on %s returned %d", c.baseURL, status)
		}

		if !time.Now().Before(deadline) {
			return fmt.Errorf(
				"the control API at %s did not become ready within %s: %w. It must be reachable "+
					"before the first scenario, and must stay reachable even while the backend is "+
					"deliberately down, otherwise an outage cannot be ended",
				c.baseURL, timeout, last)
		}
	}
}

// probe performs one readiness request and returns its status code.
func (c *HTTPControl) probe(ctx context.Context) (int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/healthz", nil)
	if err != nil {
		return 0, fmt.Errorf("could not build a readiness request for %s: %w", c.baseURL, err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("control API not reachable at %s: %w", c.baseURL, err)
	}
	defer func() { _ = resp.Body.Close() }()

	_, _ = io.Copy(io.Discard, resp.Body)

	return resp.StatusCode, nil
}

// Description implements BackendControl.
func (c *HTTPControl) Description() string {
	return fmt.Sprintf("the backend at %s, driven over the control API", c.baseURL)
}

// BackendType implements BackendControl.
//
// Always BackendTypeHTTP: this type exists to speak the normative HTTP control
// API, and it has no other mode to be in.
func (c *HTTPControl) BackendType() BackendType { return BackendTypeHTTP }

// PrepareScenario implements BackendControl.
func (c *HTTPControl) PrepareScenario(ctx context.Context) error {
	c.mu.Lock()
	mustStart := c.backendMaybeDown || c.scenarioConfiguration != ""
	resetSupported := c.resetSupported
	c.mu.Unlock()

	// A backend that may be stopped has to be started; /reset is specified to
	// restore flag state, not to bring a stopped backend back up. So does one
	// left serving a configuration the last scenario asked for: /reset would
	// restore that configuration's baseline rather than take the backend off
	// it, and the next scenario would run against the wrong flag set.
	if mustStart || (resetSupported != nil && !*resetSupported) {
		// This suite's own configuration rather than the one in effect: taking
		// the backend back to it is what preparing a scenario means.
		if err := c.startWith(ctx, c.backendConfiguration); err != nil {
			return err
		}
		c.mu.Lock()
		// Cleared only now that the start succeeded, so a failure is retried by
		// the next scenario rather than downgraded to a /reset that would leave
		// the old configuration serving.
		c.scenarioConfiguration = ""
		c.mu.Unlock()
		return nil
	}

	status, err := c.call(ctx, "/reset", nil)
	if err != nil {
		return err
	}

	switch {
	case status == http.StatusNotFound || status == http.StatusNotImplemented:
		// Documented fallback. Remembered so the probe happens once per suite.
		c.mu.Lock()
		unsupported := false
		c.resetSupported = &unsupported
		c.mu.Unlock()
		return c.start(ctx)
	case status >= 200 && status < 300:
		c.mu.Lock()
		supported := true
		c.resetSupported = &supported
		c.mu.Unlock()
		return nil
	default:
		return fmt.Errorf("POST /reset on %s returned %d", c.baseURL, status)
	}
}

// ChangeFlag implements BackendControl.
func (c *HTTPControl) ChangeFlag(ctx context.Context) error {
	return c.require(ctx, "/change", nil)
}

// Disconnect implements ConnectionControl.
//
// It makes the backend unreachable without touching any container: the backend
// process inside the still-running container is stopped. See the type
// documentation.
func (c *HTTPControl) Disconnect(ctx context.Context) error {
	c.mu.Lock()
	c.backendMaybeDown = true
	c.mu.Unlock()

	return c.require(ctx, "/stop", nil)
}

// Reconnect implements ConnectionControl.
//
// Restarting with the configuration already in effect restores the same
// baseline flag state, so the provider observes a change in availability and
// never a change in flag values. In effect means the scenario's own, where one
// served a configuration: losing the connection is not how a scenario changes
// what the backend serves.
func (c *HTTPControl) Reconnect(ctx context.Context) error {
	return c.start(ctx)
}

// ServeConfiguration implements ConfigurationControl.
//
// POST /start?config=<name>, which also seeds that configuration's flag state.
// The name is remembered for as long as the scenario lasts, so that a Reconnect
// within it restores this configuration and the next PrepareScenario knows it
// has to start rather than reset.
func (c *HTTPControl) ServeConfiguration(ctx context.Context, configuration string) error {
	if err := c.startWith(ctx, configuration); err != nil {
		return err
	}
	c.mu.Lock()
	c.scenarioConfiguration = configuration
	c.mu.Unlock()
	return nil
}

// start brings the backend up on the configuration in effect: the one a
// scenario served, else this suite's own.
func (c *HTTPControl) start(ctx context.Context) error {
	c.mu.Lock()
	configuration := c.scenarioConfiguration
	c.mu.Unlock()

	if configuration == "" {
		configuration = c.backendConfiguration
	}
	return c.startWith(ctx, configuration)
}

// startWith brings the backend up on one named configuration.
//
// A /start that returned 2xx is a backend that is up, so this is also where the
// outage flag is cleared — any other arrangement leaves a path that starts the
// backend without recording it and costs every later scenario a restart.
func (c *HTTPControl) startWith(ctx context.Context, configuration string) error {
	if err := c.require(ctx, "/start", url.Values{"config": []string{configuration}}); err != nil {
		return err
	}
	c.mu.Lock()
	c.backendMaybeDown = false
	c.mu.Unlock()
	return nil
}

// require performs a control call and fails on any non-2xx response.
func (c *HTTPControl) require(ctx context.Context, path string, query url.Values) error {
	status, err := c.call(ctx, path, query)
	if err != nil {
		return err
	}
	if status < 200 || status >= 300 {
		return fmt.Errorf("POST %s on %s returned %d", path, c.baseURL, status)
	}
	return nil
}

// call performs one control-API request and returns its status code.
//
// The response body is drained and discarded: the control API's bodies are
// human-readable messages that the TCK is specified never to interpret, and
// draining lets the connection be reused.
func (c *HTTPControl) call(ctx context.Context, path string, query url.Values) (int, error) {
	target := c.baseURL + path
	if len(query) > 0 {
		target += "?" + query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, target, nil)
	if err != nil {
		return 0, fmt.Errorf("could not build a control request for %s: %w", target, err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("control request POST %s failed: %w. The control API must stay "+
			"reachable even while the backend is deliberately down, otherwise an outage cannot be ended",
			target, err)
	}
	defer func() { _ = resp.Body.Close() }()

	_, _ = io.Copy(io.Discard, resp.Body)

	return resp.StatusCode, nil
}
