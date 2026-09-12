package tck

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	"github.com/testcontainers/testcontainers-go/modules/compose"
	"github.com/testcontainers/testcontainers-go/wait"
)

// Defaults for the Compose harness. They are the same four defaults in every
// language's suite, so an adoption that says nothing about them behaves the
// same way everywhere.
const (
	// defaultBackendService is the Compose service expected to host both the
	// control API and the backend the provider connects to.
	defaultBackendService = "backend"

	// defaultControlPort is the container-internal port the control API listens
	// on.
	defaultControlPort = 8080

	// defaultStartupTimeout bounds the whole of bringing the stack up: the
	// Compose start, the port waits, and the control API becoming willing to
	// accept commands.
	defaultStartupTimeout = 60 * time.Second

	// controlProbeInterval is how often the control API is polled while waiting
	// for it.
	controlProbeInterval = 200 * time.Millisecond
)

// composeConfig describes the Docker Compose stack the suite owns.
//
// Every field is set by exactly one Option. The zero value is not usable: the
// Compose file has no default, because there is nothing sensible to guess.
type composeConfig struct {
	file            string
	backendService  string
	backendPorts    []int
	controlPort     int
	additionalPorts map[string][]int
	// backendConfiguration is the backend's named flag configuration, not the
	// provider's mode. See WithBackendConfiguration.
	backendConfiguration string
	startupTimeout       time.Duration
}

// WithComposeFile hands the suite the Docker Compose file describing the
// backend stack, and with it the whole container lifecycle.
//
// This is the default adoption path for a provider that talks to something. The
// suite starts the stack once, discovers the dynamically mapped host ports,
// builds the HTTP control against the stack's control API, waits until it
// accepts commands, constructs a provider per scenario and tears the stack down
// after the last one. An adopter names the file, says which service and ports to
// expose, and supplies a factory — see WithProviderFromEndpoint — and writes no
// container code at all.
//
// The path is resolved relative to the package directory, which is where `go
// test` runs, so "testdata/tck/docker-compose.yaml" is the idiomatic form.
//
//	tck.Run(t,
//	    tck.WithName("my-provider"),
//	    tck.WithComposeFile("testdata/tck/docker-compose.yaml"),
//	    tck.WithBackendPorts(8013),
//	    tck.WithProviderFromEndpoint(func(_ context.Context, e tck.BackendEndpoint) (openfeature.FeatureProvider, error) {
//	        return myprovider.New(e.Host(), e.Port(8013)), nil
//	    }),
//	    tck.WithUnavailableProvider(func(context.Context) (openfeature.FeatureProvider, error) {
//	        return myprovider.New("localhost", 9999), nil
//	    }),
//	)
//
// # The stack must not pin host ports
//
// Docker assigns them dynamically and the suite discovers them after startup. A
// pinned host port makes the suite unrunnable in parallel and collides with
// whatever the developer already has listening.
//
// # The stack starts once and is never restarted
//
// Not a matter of preference. Testcontainers cannot reliably preserve
// dynamically mapped host ports across a container restart, so a restart would
// silently invalidate every provider already pointed at the old port and the
// resulting failure would look like a flaky provider. Backend unavailability is
// always simulated inside the running stack, through the control API. See
// HTTPControl.
//
// # This does not replace WithControl
//
// A provider with no backend — in-memory, in-process — keeps supplying its own
// BackendControl through WithControl. Compose is an additional path, for
// providers that talk to something, and passing both is refused rather than
// silently preferring one.
func WithComposeFile(path string) Option {
	return func(c *config) { c.composeConfig().file = path }
}

// WithBackendService names the Compose service hosting both the control API and
// the backend the provider connects to.
//
// Defaults to "backend". Set it when the stack under test reuses a Compose file
// whose service is called something else.
func WithBackendService(service string) Option {
	return func(c *config) { c.composeConfig().backendService = service }
}

// WithBackendPorts declares the container-internal ports on the backend service
// that the *provider* connects to, so they are exposed and mapped.
//
// Required with WithComposeFile: a stack whose provider ports are not exposed
// has nothing for a provider to reach, and the failure surfaces as a connection
// refused inside the first scenario rather than as the configuration mistake it
// is.
//
// The control port is exposed automatically and must not be listed here. Resolve
// the mapped ports through BackendEndpoint.Port.
func WithBackendPorts(ports ...int) Option {
	return func(c *config) { c.composeConfig().backendPorts = ports }
}

// WithControlPort sets the container-internal port the control API listens on.
//
// Defaults to 8080. The port is exposed and waited for automatically.
func WithControlPort(port int) Option {
	return func(c *config) { c.composeConfig().controlPort = port }
}

// WithAdditionalPorts declares extra container-internal ports on a service
// other than the backend one, for a stack that contains more than the backend:
// a proxy, an edge service, a sidecar.
//
// Resolve them through BackendEndpoint.ServicePort, which takes the service
// name. Calling it more than once accumulates, so one call per service reads
// naturally.
//
// Empty by default.
func WithAdditionalPorts(service string, ports ...int) Option {
	return func(c *config) {
		cc := c.composeConfig()
		if cc.additionalPorts == nil {
			cc.additionalPorts = map[string][]int{}
		}
		cc.additionalPorts[service] = append(cc.additionalPorts[service], ports...)
	}
}

// WithBackendConfiguration sets the name of the backend's flag configuration
// the suite asks the control API to start, which is what seeds the canonical
// flag set.
//
// Defaults to DefaultBackendConfiguration, the only name every backend under
// test must support.
//
// It is not the provider's configuration. The conformance report's
// "configuration" field is the provider's own mode — flagd RPC against flagd
// in-process — and that one comes from WithName. This is the backend's config
// file, and the two are named apart because they were confused once already.
func WithBackendConfiguration(name string) Option {
	return func(c *config) { c.composeConfig().backendConfiguration = name }
}

// WithStartupTimeout sets how long to wait for the Compose stack and its control
// API to become reachable.
//
// Defaults to 60 seconds. It bounds the port waits and the control API readiness
// check together, not each of them separately.
func WithStartupTimeout(timeout time.Duration) Option {
	return func(c *config) { c.composeConfig().startupTimeout = timeout }
}

// composeConfig returns the stack description, creating it on first use.
//
// Any compose option creates it, so a configuration that describes ports and
// services but forgets the file is reported as a missing file rather than as a
// stray option — which is the mistake that actually happens.
func (c *config) composeConfig() *composeConfig {
	if c.compose == nil {
		c.compose = &composeConfig{}
	}
	return c.compose
}

func (cc *composeConfig) service() string {
	if cc.backendService == "" {
		return defaultBackendService
	}
	return cc.backendService
}

func (cc *composeConfig) control() int {
	if cc.controlPort == 0 {
		return defaultControlPort
	}
	return cc.controlPort
}

func (cc *composeConfig) backendConfig() string {
	if cc.backendConfiguration == "" {
		return DefaultBackendConfiguration
	}
	return cc.backendConfiguration
}

func (cc *composeConfig) timeout() time.Duration {
	if cc.startupTimeout <= 0 {
		return defaultStartupTimeout
	}
	return cc.startupTimeout
}

// validate reports what a Compose adoption is missing, before anything is
// started.
func (cc *composeConfig) validate() []error {
	var problems []error

	if cc.file == "" {
		problems = append(problems, errors.New(
			"tck.WithComposeFile is required to let the suite own the stack: it is the Compose file "+
				"describing the backend, resolved relative to the package directory"))
	}
	if len(cc.backendPorts) == 0 {
		problems = append(problems, errors.New(
			"tck.WithBackendPorts is required with tck.WithComposeFile: it is the container-internal "+
				"port or ports the provider connects to, and without them the stack publishes nothing "+
				"the provider can reach. The control port is exposed automatically and does not belong here"))
	}
	for _, port := range cc.backendPorts {
		if port == cc.control() {
			problems = append(problems, fmt.Errorf(
				"tck.WithBackendPorts lists %d, which is the control API port: the control port is "+
					"exposed automatically, and listing it here says the provider evaluates flags "+
					"through the control API. Remove it, or move the control API with tck.WithControlPort",
				port))
		}
	}
	if _, clash := cc.additionalPorts[cc.service()]; clash {
		problems = append(problems, fmt.Errorf(
			"tck.WithAdditionalPorts names %q, which is the backend service: its ports belong in "+
				"tck.WithBackendPorts, and declaring them twice makes which of the two the provider "+
				"should use a matter of guesswork", cc.service()))
	}

	return problems
}

// BackendEndpoint is the running stack, as a provider factory sees it: a host,
// and host ports resolved from container-internal ones.
//
// It exists because the external ports are only known *after* the stack has
// started. A stack under test must not pin host ports, so a provider cannot be
// configured until the stack is up — which is why WithProviderFromEndpoint takes
// a factory rather than a provider.
//
// The mapping is stable for the lifetime of the suite: the stack is started once
// and never restarted, so a provider built from this endpoint stays valid across
// every scenario.
type BackendEndpoint struct {
	defaultService string
	hosts          map[string]string
	ports          map[servicePort]int
}

// servicePort keys a resolved host port by the service and container-internal
// port it belongs to.
type servicePort struct {
	service string
	port    int
}

// Host returns the host the backend service is reachable on.
//
// This is not necessarily "localhost": with a remote Docker daemon, Docker
// Desktop on some platforms, or a rootless setup it can be an arbitrary address.
// Always use this rather than hard-coding a host.
func (e BackendEndpoint) Host() string {
	return e.ServiceHost(e.defaultService)
}

// ServiceHost returns the host a named service is reachable on.
func (e BackendEndpoint) ServiceHost(service string) string {
	if host, ok := e.hosts[service]; ok {
		return host
	}
	panic(undeclaredService(service, e.services()))
}

// Port returns the host port mapped to a container-internal port on the backend
// service.
//
// The port must have been declared through WithBackendPorts, which is what
// caused it to be exposed at all.
func (e BackendEndpoint) Port(internal int) int {
	return e.ServicePort(e.defaultService, internal)
}

// ServicePort returns the host port mapped to a container-internal port on a
// named service.
//
// Use it for multi-service stacks. The service and port must have been declared
// through WithAdditionalPorts, otherwise nothing exposed it.
//
// Asking for a port that was never declared panics rather than returning zero. A
// provider handed port 0 fails with a connection error inside the first
// scenario, which reads as a provider defect; the panic names the option that is
// missing instead.
func (e BackendEndpoint) ServicePort(service string, internal int) int {
	if port, ok := e.ports[servicePort{service: service, port: internal}]; ok {
		return port
	}
	panic(fmt.Sprintf(
		"tck: container port %d of service %q was never exposed, so it has no host port. "+
			"Declare it with tck.WithBackendPorts (for the backend service) or "+
			"tck.WithAdditionalPorts (for any other). Exposed: %s",
		internal, service, e.describe()))
}

func (e BackendEndpoint) services() []string {
	services := make([]string, 0, len(e.hosts))
	for service := range e.hosts {
		services = append(services, service)
	}
	sort.Strings(services)
	return services
}

// describe lists what the endpoint actually resolved, for a failure message.
func (e BackendEndpoint) describe() string {
	mappings := make([]string, 0, len(e.ports))
	for key, host := range e.ports {
		mappings = append(mappings, fmt.Sprintf("%s:%d->%d", key.service, key.port, host))
	}
	sort.Strings(mappings)
	if len(mappings) == 0 {
		return "nothing"
	}
	return fmt.Sprint(mappings)
}

func undeclaredService(service string, known []string) string {
	return fmt.Sprintf(
		"tck: service %q is not part of the stack this suite exposed, so it has no host. Exposed: %v",
		service, known)
}

// WithProviderFromEndpoint supplies the factory that creates the provider under
// test against the Compose stack the suite started.
//
// It is WithProvider with the discovered endpoint handed to it, and it is a
// factory for the same reason: the mapped host ports do not exist until the
// stack is up, and each scenario gets its own provider.
//
//	tck.WithProviderFromEndpoint(func(_ context.Context, e tck.BackendEndpoint) (openfeature.FeatureProvider, error) {
//	    return myprovider.New(e.Host(), e.Port(8013)), nil
//	})
//
// Return a configured but uninitialised provider; the TCK owns the lifecycle
// from there. Required with WithComposeFile, and refused without it — there
// would be no stack for the endpoint to describe.
func WithProviderFromEndpoint(factory EndpointProviderFactory) Option {
	return func(c *config) { c.NewProviderFromEndpoint = factory }
}

// startCompose brings the stack up, resolves the endpoint and builds the
// control.
//
// It returns the endpoint and the control rather than storing them, so that the
// one place which mutates the configuration is Run.
func startCompose(ctx context.Context, cc *composeConfig) (BackendEndpoint, *HTTPControl, func(), error) {
	path, err := filepath.Abs(cc.file)
	if err != nil {
		path = cc.file
	}

	stack, err := compose.NewDockerCompose(cc.file)
	if err != nil {
		return BackendEndpoint{}, nil, nil, fmt.Errorf(
			"could not read the Compose file %s: %w. tck.WithComposeFile is resolved relative to the "+
				"package directory, which is where go test runs", path, err)
	}

	for service, ports := range cc.exposed() {
		stack.WaitForService(service, listeningOn(ports, cc.timeout()))
	}

	if err := stack.Up(ctx); err != nil {
		return BackendEndpoint{}, nil, nil, fmt.Errorf("could not start the Compose stack %s: %w", path, err)
	}

	stop := func() {
		// A fresh context: the suite's may already be cancelled by the time
		// teardown runs, and a stack that is not brought down leaks containers
		// for the rest of the session.
		if err := stack.Down(context.Background(), compose.RemoveOrphans(true)); err != nil {
			_ = err
		}
	}

	endpoint, err := resolveEndpoint(ctx, stack, cc)
	if err != nil {
		stop()
		return BackendEndpoint{}, nil, nil, err
	}

	control, err := NewHTTPControl(HTTPControlOptions{
		BaseURL:              fmt.Sprintf("http://%s:%d", endpoint.Host(), endpoint.Port(cc.control())),
		BackendConfiguration: cc.backendConfig(),
	})
	if err != nil {
		stop()
		return BackendEndpoint{}, nil, nil, err
	}

	if err := control.AwaitReady(ctx, cc.timeout()); err != nil {
		stop()
		return BackendEndpoint{}, nil, nil, err
	}

	return endpoint, control, stop, nil
}

// exposed is every service and the container-internal ports to publish for it,
// the control port included.
func (cc *composeConfig) exposed() map[string][]int {
	exposed := map[string][]int{}
	for service, ports := range cc.additionalPorts {
		exposed[service] = append(exposed[service], ports...)
	}
	exposed[cc.service()] = append([]int{cc.control()}, cc.backendPorts...)
	return exposed
}

// listeningOn builds the readiness strategy for one service.
//
// One strategy per service rather than one per port, because the compose module
// keeps a single wait strategy per service name and a second call for the same
// service replaces the first — so waiting for two ports of one service with two
// calls silently waits for only the last.
func listeningOn(ports []int, timeout time.Duration) wait.Strategy {
	strategies := make([]wait.Strategy, 0, len(ports))
	for _, port := range ports {
		strategies = append(strategies, wait.ForListeningPort(strconv.Itoa(port)+"/tcp"))
	}
	return wait.ForAll(strategies...).WithDeadline(timeout)
}

// resolveEndpoint reads the dynamically mapped host ports, once, after the stack
// is up.
func resolveEndpoint(ctx context.Context, stack *compose.DockerCompose, cc *composeConfig) (BackendEndpoint, error) {
	endpoint := BackendEndpoint{
		defaultService: cc.service(),
		hosts:          map[string]string{},
		ports:          map[servicePort]int{},
	}

	for service, ports := range cc.exposed() {
		container, err := stack.ServiceContainer(ctx, service)
		if err != nil {
			return BackendEndpoint{}, fmt.Errorf(
				"the Compose stack has no service %q: %w. Services are named in the Compose file; "+
					"the backend one defaults to %q and is set with tck.WithBackendService",
				service, err, defaultBackendService)
		}

		host, err := container.Host(ctx)
		if err != nil {
			return BackendEndpoint{}, fmt.Errorf("could not resolve the host of service %q: %w", service, err)
		}
		endpoint.hosts[service] = host

		for _, port := range ports {
			mapped, err := container.MappedPort(ctx, wireProtoPort(port))
			if err != nil {
				return BackendEndpoint{}, fmt.Errorf(
					"service %q published no host port for container port %d: %w. The Compose file "+
						"must list it under ports, and must not pin a host port for it",
					service, port, err)
			}
			endpoint.ports[servicePort{service: service, port: port}] = int(mapped.Num())
		}
	}

	return endpoint, nil
}

func wireProtoPort(port int) string {
	return strconv.Itoa(port) + "/tcp"
}
