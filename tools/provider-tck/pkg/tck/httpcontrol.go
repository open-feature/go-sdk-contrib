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

// DefaultConfiguration is the configuration name every backend under test must
// support, and the one that serves the canonical flag set.
const DefaultConfiguration = "default"

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
// It never stops, kills or recreates a container. Unavailability is simulated
// inside the running stack, through POST /stop, because container
// orchestrators assign host ports dynamically and cannot reliably preserve
// them across a restart — a restarted backend generally comes back on a
// different host port, silently invalidating every provider already pointed at
// the old one, and the resulting failure looks like a flaky provider. Starting
// and stopping the stack itself belongs to the adopting test, once per suite.
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
// and uses /start for the scenario following any disconnect.
type HTTPControl struct {
	baseURL       string
	configuration string
	client        *http.Client

	mu sync.Mutex
	// resetSupported is nil until the first /reset call tells us.
	resetSupported *bool
	// backendMaybeDown records that a disconnect happened, so the next
	// PrepareScenario starts the backend rather than merely resetting flags.
	backendMaybeDown bool
}

var (
	_ BackendControl    = (*HTTPControl)(nil)
	_ ConnectionControl = (*HTTPControl)(nil)
)

// HTTPControlOptions configures an HTTPControl.
type HTTPControlOptions struct {
	// BaseURL is the root of the control API, for example
	// http://localhost:32768. Required.
	//
	// It must be built from the dynamically mapped host port of the control
	// service, discovered after the stack is up. A stack under test must not
	// pin host ports.
	BaseURL string

	// Configuration is the named flag configuration to seed. Defaults to
	// DefaultConfiguration, which is the only name every backend must support
	// and the one serving the canonical flag set.
	Configuration string

	// Client is the HTTP client to use. Defaults to one with a 30 second
	// timeout.
	Client *http.Client
}

// NewHTTPControl returns a control that drives the backend at opts.BaseURL.
func NewHTTPControl(opts HTTPControlOptions) (*HTTPControl, error) {
	if opts.BaseURL == "" {
		return nil, errors.New("HTTPControlOptions.BaseURL is required: it is the root of the " +
			"control API, built from the dynamically mapped host port of the control service")
	}
	if _, err := url.Parse(opts.BaseURL); err != nil {
		return nil, fmt.Errorf("HTTPControlOptions.BaseURL %q is not a valid URL: %w", opts.BaseURL, err)
	}

	configuration := opts.Configuration
	if configuration == "" {
		configuration = DefaultConfiguration
	}

	client := opts.Client
	if client == nil {
		client = &http.Client{Timeout: defaultControlTimeout}
	}

	return &HTTPControl{
		baseURL:       strings.TrimSuffix(opts.BaseURL, "/"),
		configuration: configuration,
		client:        client,
	}, nil
}

// Description implements BackendControl.
func (c *HTTPControl) Description() string {
	return fmt.Sprintf("the backend at %s, driven over the control API", c.baseURL)
}

// PrepareScenario implements BackendControl.
func (c *HTTPControl) PrepareScenario(ctx context.Context) error {
	c.mu.Lock()
	mustStart := c.backendMaybeDown
	resetSupported := c.resetSupported
	c.mu.Unlock()

	// A backend that may be stopped has to be started; /reset is specified to
	// restore flag state, not to bring a stopped backend back up.
	if mustStart || (resetSupported != nil && !*resetSupported) {
		if err := c.start(ctx); err != nil {
			return err
		}
		c.mu.Lock()
		c.backendMaybeDown = false
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
// documentation for why that distinction is a requirement rather than a
// preference.
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
// never a change in flag values.
func (c *HTTPControl) Reconnect(ctx context.Context) error {
	if err := c.start(ctx); err != nil {
		return err
	}
	c.mu.Lock()
	c.backendMaybeDown = false
	c.mu.Unlock()
	return nil
}

func (c *HTTPControl) start(ctx context.Context) error {
	return c.require(ctx, "/start", url.Values{"config": []string{c.configuration}})
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
	defer resp.Body.Close()

	_, _ = io.Copy(io.Discard, resp.Body)

	return resp.StatusCode, nil
}
