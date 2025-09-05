package testframework

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// FlagdTestContainer implements the TestContainer interface using testcontainers-go
type FlagdTestContainer struct {
	container     testcontainers.Container
	host          string
	launchpadURL  string
	rpcPort       int
	inProcessPort int
	launchpadPort int
	healthPort    int
}

// Container config type moved to types.go

// NewFlagdContainer creates a new flagd testbed container
func NewFlagdContainer(ctx context.Context, config FlagdContainerConfig) (*FlagdTestContainer, error) {
	// Build the image name
	image := "ghcr.io/open-feature/flagd-testbed"
	if config.Image != "" {
		image = config.Image
	}
	if config.Feature != "" {
		image = fmt.Sprintf("%s-%s", image, config.Feature)
	}
	tag := config.Tag
	if tag == "" {
		versionTag, err := readTestbedVersion(config)
		if err != nil {
			return nil, err
		}
		tag = versionTag
	}
	image = fmt.Sprintf("%s:%s", image, tag)

	// Define ports
	rpcPort := 8013
	inProcessPort := 8015
	launchpadPort := 8080
	healthPort := 8014

	// Create container request
	req := testcontainers.ContainerRequest{
		Image: image,
		ExposedPorts: []string{
			fmt.Sprintf("%d/tcp", rpcPort),
			fmt.Sprintf("%d/tcp", inProcessPort),
			fmt.Sprintf("%d/tcp", launchpadPort),
			fmt.Sprintf("%d/tcp", healthPort),
		},
		WaitingFor: wait.ForAll(
			// Wait for the container to start and launchpad to be ready
			wait.ForListeningPort("8080/tcp"),
		).WithDeadline(60 * time.Second),
		Networks: config.Networks,
	}

	// Add volume binding for flags directory if specified
	if config.FlagsDir != "" {
		absPath, err := filepath.Abs(config.FlagsDir)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve flags directory path: %w", err)
		}
		req.Mounts = testcontainers.Mounts(testcontainers.BindMount(absPath, "/flags"))
	}

	// Start the container
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start flagd container: %w", err)
	}

	// Get the host
	host, err := container.Host(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get container host: %w", err)
	}

	// Get mapped ports
	mappedLaunchpadPort, err := container.MappedPort(ctx, "8080")
	if err != nil {
		return nil, fmt.Errorf("failed to get launchpad port: %w", err)
	}

	mappedRPCPort, err := container.MappedPort(ctx, "8013")
	if err != nil {
		return nil, fmt.Errorf("failed to get RPC port: %w", err)
	}

	mappedInProcessPort, err := container.MappedPort(ctx, "8015")
	if err != nil {
		return nil, fmt.Errorf("failed to get in-process port: %w", err)
	}

	mappedHealthPort, err := container.MappedPort(ctx, "8014")
	if err != nil {
		return nil, fmt.Errorf("failed to get health port: %w", err)
	}

	flagdContainer := &FlagdTestContainer{
		container:     container,
		host:          host,
		rpcPort:       mappedRPCPort.Int(),
		inProcessPort: mappedInProcessPort.Int(),
		launchpadPort: mappedLaunchpadPort.Int(),
		healthPort:    mappedHealthPort.Int(),
		launchpadURL:  fmt.Sprintf("http://%s:%d", host, mappedLaunchpadPort.Int()),
	}

	// Additional wait time if configured
	if config.ExtraWaitTime > 0 {
		time.Sleep(config.ExtraWaitTime)
	}

	// Note: We don't check health here because flagd needs to be started via launchpad API first

	return flagdContainer, nil
}

func readTestbedVersion(config FlagdContainerConfig) (string, error) {
	wd, _ := os.Getwd()
	fileName := "version.txt"
	path := "../flagd-testbed"
	if config.TestbedDir != "" {
		path = config.TestbedDir
	}

	content, err := os.ReadFile(fmt.Sprintf("%s/%s", wd, filepath.Join(path, fileName)))
	if err != nil {
		fmt.Printf("Failed to read file: %s", fileName)
		return "", err
	}

	return "v" + strings.TrimSuffix(string(content), "\n"), nil
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

	return f.container.Start(context.Background())
}

// Stop stops the container
func (f *FlagdTestContainer) Stop() error {
	if f.container == nil {
		return fmt.Errorf("container not initialized")
	}

	timeout := 30 * time.Second
	return f.container.Stop(context.Background(), &timeout)
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

	return f.container.Terminate(context.Background())
}

// GetContainerLogs returns the container logs for debugging
func (f *FlagdTestContainer) GetContainerLogs(ctx context.Context) (string, error) {
	if f.container == nil {
		return "", fmt.Errorf("container not initialized")
	}

	logs, err := f.container.Logs(ctx)
	if err != nil {
		return "", err
	}
	defer logs.Close()

	// Read logs (implementation would need to read from the io.Reader)
	return "logs would be read here", nil
}

// GetRPCAddress returns the RPC endpoint address
func (f *FlagdTestContainer) GetRPCAddress() string {
	return fmt.Sprintf("%s:%d", f.host, f.rpcPort)
}

// GetInProcessAddress returns the in-process endpoint address
func (f *FlagdTestContainer) GetInProcessAddress() string {
	return fmt.Sprintf("%s:%d", f.host, f.inProcessPort)
}

// GetHealthAddress returns the health check endpoint address
func (f *FlagdTestContainer) GetHealthAddress() string {
	return fmt.Sprintf("%s:%d", f.host, f.healthPort)
}

// Container info type moved to types.go

// GetInfo returns detailed information about the container
func (f *FlagdTestContainer) GetInfo(ctx context.Context) (*ContainerInfo, error) {
	if f.container == nil {
		return nil, fmt.Errorf("container not initialized")
	}

	state, err := f.container.State(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get container state: %w", err)
	}

	return &ContainerInfo{
		ID:            f.container.GetContainerID(),
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
