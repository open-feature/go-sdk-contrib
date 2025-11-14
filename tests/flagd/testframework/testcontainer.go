package testframework

import (
	"context"
	"fmt"
	"github.com/docker/go-connections/nat"
	"net/http"
	"path/filepath"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/compose"
	"github.com/testcontainers/testcontainers-go/wait"
)

// FlagdTestContainer implements the TestContainer interface using testcontainers-go
type FlagdTestContainer struct {
	container     compose.ComposeStack
	host          string
	launchpadURL  string
	rpcPort       int
	inProcessPort int
	launchpadPort int
	healthPort    int
	envoyPort     int
	forbiddenPort int
}

// Container config type moved to types.go

// NewFlagdContainer creates a new flagd testbed container
func NewFlagdContainer(ctx context.Context, config FlagdContainerConfig) (*FlagdTestContainer, error) {
	// Create compose stack
	composeStack, err := compose.NewDockerCompose(filepath.Join(config.TestbedDir, "docker-compose.yaml"))
	if err != nil {
		return nil, fmt.Errorf("failed to create compose stack: %w", err)
	}

	// Build environment variables
	env := make(map[string]string)
	// for file tests
	env["FLAGS_DIR"] = config.FlagsDir
	if config.Image != "" {
		env["IMAGE"] = config.Image
	}
	if config.Tag != "" {
		env["VERSION"] = config.Tag
	}
	composeStack.WithEnv(env)

	// Configure wait strategies
	const flagdServiceName = "flagd"
	composeStack.WaitForService(flagdServiceName,
		wait.ForListeningPort("8080/tcp").WithStartupTimeout(60*time.Second))

	// Start the services
	err = composeStack.Up(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to start compose stack: %w", err)
	}

	// Get the host (always localhost for docker-compose)
	host := "localhost"

	flagdService, err := composeStack.ServiceContainer(ctx, flagdServiceName)
	if err != nil {
		composeStack.Down(ctx)
		return nil, fmt.Errorf("failed to start compose stack: %w", err)
	}
	rpcPort, err := getMappedPort(ctx, composeStack, flagdService, "8013")
	if err != nil {
		return nil, err
	}
	inProcessPort, err := getMappedPort(ctx, composeStack, flagdService, "8015")
	if err != nil {
		return nil, err
	}
	healthPort, err := getMappedPort(ctx, composeStack, flagdService, "8014")
	if err != nil {
		return nil, err
	}
	launchpadPort, err := getMappedPort(ctx, composeStack, flagdService, "8080")
	if err != nil {
		return nil, err
	}
	envoy, err := composeStack.ServiceContainer(ctx, "envoy")
	envoyPort, err := getMappedPort(ctx, composeStack, envoy, "9211")
	if err != nil {
		return nil, err
	}
	forbiddenPort, err := getMappedPort(ctx, composeStack, envoy, "9212")
	if err != nil {
		return nil, err
	}

	flagdContainer := &FlagdTestContainer{
		container:     composeStack,
		host:          host,
		rpcPort:       rpcPort,
		inProcessPort: inProcessPort,
		healthPort:    healthPort,
		envoyPort:     envoyPort,
		launchpadURL:  fmt.Sprintf("http://%s:%d", host, launchpadPort),
		forbiddenPort: forbiddenPort,
	}

	// Additional wait time if configured
	if config.ExtraWaitTime > 0 {
		time.Sleep(config.ExtraWaitTime)
	}

	// Note: We don't check health here because flagd needs to be started via launchpad API first
	return flagdContainer, nil
}

func getMappedPort(ctx context.Context, stack *compose.DockerCompose, container *testcontainers.DockerContainer, port nat.Port) (int, error) {
	rpcPort, err := container.MappedPort(ctx, port)
	if err != nil {
		stack.Down(ctx)
		return 0, fmt.Errorf("failed to fetch mapped port %s for %s: %w", port, container.ID, err)
	}
	return rpcPort.Int(), nil
}

// GetHost returns the container host
func (f *FlagdTestContainer) GetHost() string {
	return f.host
}

// GetPort returns the mapped port for a specific service
func (f *FlagdTestContainer) GetPort(service string) int {
	switch service {
	case "rpc":
		return f.rpcPort
	case "in-process":
		return f.inProcessPort
	case "launchpad":
		return f.launchpadPort
	case "health":
		return f.healthPort
	default:
		return 0
	}
}

// GetLaunchpadURL returns the full URL for the launchpad API
func (f *FlagdTestContainer) GetLaunchpadURL() string {
	return f.launchpadURL
}

// Start starts the container (if not already running)
func (f *FlagdTestContainer) Start() error {
	if f.container == nil {
		return fmt.Errorf("container not initialized")
	}

	return f.container.Up(context.Background())
}

// Stop stops the container
func (f *FlagdTestContainer) Stop() error {
	if f.container == nil {
		return fmt.Errorf("container not initialized")
	}

	return f.container.Down(context.Background())
}

// Restart restarts the flagd service after a delay using the launchpad API
func (f *FlagdTestContainer) Restart(delaySeconds int) error {
	url := fmt.Sprintf("%s/restart?seconds=%d", f.launchpadURL, delaySeconds)

	client := &http.Client{Timeout: 10*time.Second + time.Duration(delaySeconds)}
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("failed to trigger restart: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("restart request failed with status: %d", resp.StatusCode)
	}

	return nil
}

// IsHealthy checks if the flagd service is healthy
func (f *FlagdTestContainer) IsHealthy() bool {
	healthURL := fmt.Sprintf("http://%s:%d/readyz", f.host, f.healthPort)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(healthURL)
	if err != nil {
		fmt.Printf("DEBUG: Health check failed: %v (URL: %s, host: %s, healthPort: %d)\n", err, healthURL, f.host, f.healthPort)
		return false
	}
	defer resp.Body.Close()

	healthy := resp.StatusCode == http.StatusOK
	if !healthy {
		fmt.Printf("DEBUG: Health check returned status: %d (URL: %s, host: %s, healthPort: %d)\n", resp.StatusCode, healthURL, f.host, f.healthPort)
	}
	return healthy
}

// StartFlagdWithConfig starts flagd with a specific configuration using launchpad
func (f *FlagdTestContainer) StartFlagdWithConfig(config string) error {
	url := fmt.Sprintf("%s/start?config=%s", f.launchpadURL, config)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("failed to start flagd with config %s: %w", config, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("start request failed with status: %d", resp.StatusCode)
	}

	// Wait for flagd to be ready - increased timeout for stability
	return f.waitForHealthy(15 * time.Second)
}

// StopFlagd stops the flagd service using launchpad
func (f *FlagdTestContainer) StopFlagd() error {
	url := fmt.Sprintf("%s/stop", f.launchpadURL)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("failed to stop flagd: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("stop request failed with status: %d", resp.StatusCode)
	}

	return nil
}

// TriggerFlagChange triggers a flag configuration change using launchpad
func (f *FlagdTestContainer) TriggerFlagChange() error {
	url := fmt.Sprintf("%s/change", f.launchpadURL)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("failed to trigger flag change: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("change request failed with status: %d", resp.StatusCode)
	}

	return nil
}

// Note: allFlags.json is automatically generated by the launchpad during startup

// waitForHealthy waits for the flagd service to become healthy
func (f *FlagdTestContainer) waitForHealthy(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		if f.IsHealthy() {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("flagd did not become healthy within %v", timeout)
}

// Terminate terminates and removes the container
func (f *FlagdTestContainer) Terminate() error {
	if f.container == nil {
		return nil
	}

	return f.container.Down(context.Background())
}

// GetContainerLogs returns the container logs for debugging
func (f *FlagdTestContainer) GetContainerLogs(ctx context.Context) (string, error) {
	if f.container == nil {
		return "", fmt.Errorf("container not initialized")
	}

	container, err := f.container.ServiceContainer(ctx, "flagd")
	if err != nil {
		return "", fmt.Errorf("failed to get flagd container: %w", err)
	}
	logs, err := container.Logs(ctx)
	if err != nil {
		return "", err
	}
	defer logs.Close()

	// Read logs (implementation would need to read from the io.Reader)
	return "logs would be read here", nil
}

// Container info type moved to types.go

// GetInfo returns detailed information about the container
func (f *FlagdTestContainer) GetInfo(ctx context.Context) (*ContainerInfo, error) {
	if f.container == nil {
		return nil, fmt.Errorf("container not initialized")
	}

	container, err := f.container.ServiceContainer(ctx, "flagd")
	if err != nil {
		return nil, fmt.Errorf("failed to get flagd container: %w", err)
	}
	state, err := container.State(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get container state: %w", err)
	}

	return &ContainerInfo{
		ID:            container.GetContainerID(),
		Host:          f.host,
		RPCPort:       f.rpcPort,
		InProcessPort: f.inProcessPort,
		LaunchpadPort: f.launchpadPort,
		HealthPort:    f.healthPort,
		LaunchpadURL:  f.launchpadURL,
		IsRunning:     state.Running,
		IsHealthy:     f.IsHealthy(),
	}, nil
}
