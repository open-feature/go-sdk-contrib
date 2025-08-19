package testframework

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// DebugMode controls verbose debugging output
var DebugMode = os.Getenv("FLAGD_E2E_DEBUG") == "true"

// DebugLogger provides structured debugging output
type DebugLogger struct {
	prefix string
}

func NewDebugLogger(prefix string) *DebugLogger {
	return &DebugLogger{prefix: prefix}
}

func (d *DebugLogger) Printf(format string, args ...interface{}) {
	if DebugMode {
		fmt.Printf("[DEBUG:%s] %s\n", d.prefix, fmt.Sprintf(format, args...))
	}
}

func (d *DebugLogger) PrintJSON(obj interface{}, label string) {
	if DebugMode {
		if data, err := json.MarshalIndent(obj, "", "  "); err == nil {
			fmt.Printf("[DEBUG:%s] %s:\n%s\n", d.prefix, label, string(data))
		}
	}
}

// ContainerDiagnostics provides comprehensive container debugging
type ContainerDiagnostics struct {
	container TestContainer
	logger    *DebugLogger
}

func NewContainerDiagnostics(container TestContainer) *ContainerDiagnostics {
	return &ContainerDiagnostics{
		container: container,
		logger:    NewDebugLogger("CONTAINER"),
	}
}

// PrintContainerInfo displays comprehensive container information
func (cd *ContainerDiagnostics) PrintContainerInfo() {
	if !DebugMode {
		return
	}

	cd.logger.Printf("=== Container Information ===")
	if cd.container != nil {
		cd.logger.Printf("Host: %s", cd.container.GetHost())
		cd.logger.Printf("RPC Port: %d", cd.container.GetPort("rpc"))
		cd.logger.Printf("InProcess Port: %d", cd.container.GetPort("in-process"))
		cd.logger.Printf("Launchpad Port: %d", cd.container.GetPort("launchpad"))
		cd.logger.Printf("Health Port: %d", cd.container.GetPort("health"))
		cd.logger.Printf("Launchpad URL: %s", cd.container.GetLaunchpadURL())
		cd.logger.Printf("Healthy: %t", cd.container.IsHealthy())
	} else {
		cd.logger.Printf("Container not initialized")
	}
}

// PrintContainerLogs streams recent container logs
func (cd *ContainerDiagnostics) PrintContainerLogs(lines int) {
	if !DebugMode {
		return
	}

	cd.logger.Printf("=== Container Logs (last %d lines) ===", lines)
	if cd.container != nil {
		// This is a placeholder - actual implementation would depend on testcontainers API
		cd.logger.Printf("Container logs would be displayed here")
		// TODO: Implement actual log streaming when testcontainers supports it
	}
}

// HealthCheck performs comprehensive health diagnostics
func (cd *ContainerDiagnostics) HealthCheck() map[string]interface{} {
	results := make(map[string]interface{})
	
	if cd.container == nil {
		results["container"] = "not_initialized"
		return results
	}

	// Basic container health
	results["container_healthy"] = cd.container.IsHealthy()
	results["host"] = cd.container.GetHost()

	// Test launchpad connectivity
	launchpadURL := cd.container.GetLaunchpadURL()
	results["launchpad_url"] = launchpadURL
	
	client := &http.Client{Timeout: 5 * time.Second}
	if resp, err := client.Get(launchpadURL + "/health"); err == nil {
		results["launchpad_status"] = resp.StatusCode
		resp.Body.Close()
	} else {
		results["launchpad_error"] = err.Error()
	}

	// Test flagd health endpoint
	healthURL := fmt.Sprintf("http://%s:%d/readyz", 
		cd.container.GetHost(), 
		cd.container.GetPort("health"))
	if resp, err := client.Get(healthURL); err == nil {
		results["flagd_health_status"] = resp.StatusCode
		resp.Body.Close()
	} else {
		results["flagd_health_error"] = err.Error()
	}

	if DebugMode {
		cd.logger.PrintJSON(results, "Health Check Results")
	}

	return results
}

// FlagDataInspector helps debug flag-related issues
type FlagDataInspector struct {
	flagsDir string
	logger   *DebugLogger
}

func NewFlagDataInspector(flagsDir string) *FlagDataInspector {
	return &FlagDataInspector{
		flagsDir: flagsDir,
		logger:   NewDebugLogger("FLAGS"),
	}
}

// ListFlagFiles shows all flag files in the directory
func (fdi *FlagDataInspector) ListFlagFiles() []string {
	if fdi.flagsDir == "" {
		return nil
	}

	var files []string
	if entries, err := os.ReadDir(fdi.flagsDir); err == nil {
		for _, entry := range entries {
			if strings.HasSuffix(entry.Name(), ".json") {
				files = append(files, entry.Name())
				if DebugMode {
					fdi.logger.Printf("Found flag file: %s", entry.Name())
				}
			}
		}
	}
	return files
}

// InspectAllFlags reads and displays the contents of allFlags.json
func (fdi *FlagDataInspector) InspectAllFlags() map[string]interface{} {
	if fdi.flagsDir == "" {
		return nil
	}

	flagFile := filepath.Join(fdi.flagsDir, "allFlags.json")
	
	if !DebugMode {
		return nil
	}

	fdi.logger.Printf("=== allFlags.json Content ===")
	fdi.logger.Printf("File path: %s", flagFile)

	if _, err := os.Stat(flagFile); os.IsNotExist(err) {
		fdi.logger.Printf("❌ File does not exist")
		return nil
	}

	data, err := os.ReadFile(flagFile)
	if err != nil {
		fdi.logger.Printf("❌ Error reading file: %v", err)
		return nil
	}

	var flags map[string]interface{}
	if err := json.Unmarshal(data, &flags); err != nil {
		fdi.logger.Printf("❌ Error parsing JSON: %v", err)
		fdi.logger.Printf("Raw content (first 500 chars): %s", string(data[:min(len(data), 500)]))
		return nil
	}

	fdi.logger.Printf("✅ File exists and is valid JSON")
	fdi.logger.Printf("Flag count: %d", len(flags))
	
	if flagsData, ok := flags["flags"]; ok {
		if flagsMap, ok := flagsData.(map[string]interface{}); ok {
			fdi.logger.Printf("Available flags:")
			for flagKey := range flagsMap {
				fdi.logger.Printf("  - %s", flagKey)
			}
		}
	}

	return flags
}

// ScenarioDebugger helps debug individual Gherkin scenarios
type ScenarioDebugger struct {
	logger *DebugLogger
}

func NewScenarioDebugger() *ScenarioDebugger {
	return &ScenarioDebugger{
		logger: NewDebugLogger("SCENARIO"),
	}
}

// DebugScenarioFailure provides context when scenarios fail
func (sd *ScenarioDebugger) DebugScenarioFailure(scenarioName string, err error, testState interface{}) {
	if !DebugMode {
		return
	}

	sd.logger.Printf("=== Scenario Failure Debug ===")
	sd.logger.Printf("Scenario: %s", scenarioName)
	sd.logger.Printf("Error: %v", err)
	
	if testState != nil {
		sd.logger.PrintJSON(testState, "Test State at Failure")
	}
}

// NetworkDiagnostics helps debug connectivity issues
type NetworkDiagnostics struct {
	container TestContainer
	logger    *DebugLogger
}

func NewNetworkDiagnostics(container TestContainer) *NetworkDiagnostics {
	return &NetworkDiagnostics{
		container: container,
		logger:    NewDebugLogger("NETWORK"),
	}
}

// TestConnectivity tests all network endpoints
func (nd *NetworkDiagnostics) TestConnectivity() map[string]string {
	results := make(map[string]string)
	
	if nd.container == nil {
		results["error"] = "container not initialized"
		return results
	}

	host := nd.container.GetHost()
	client := &http.Client{Timeout: 5 * time.Second}

	// Test each endpoint
	endpoints := map[string]string{
		"launchpad": fmt.Sprintf("http://%s:%d/health", host, nd.container.GetPort("launchpad")),
		"health":    fmt.Sprintf("http://%s:%d/readyz", host, nd.container.GetPort("health")),
		"rpc":       fmt.Sprintf("%s:%d", host, nd.container.GetPort("rpc")),
	}

	for name, url := range endpoints {
		if strings.HasPrefix(url, "http") {
			if resp, err := client.Get(url); err == nil {
				results[name] = fmt.Sprintf("HTTP %d", resp.StatusCode)
				resp.Body.Close()
			} else {
				results[name] = fmt.Sprintf("Error: %v", err)
			}
		} else {
			// For non-HTTP endpoints like gRPC, just check if we can connect
			results[name] = "TCP connection test needed"
		}
	}

	if DebugMode {
		nd.logger.Printf("=== Network Connectivity Test ===")
		for endpoint, result := range results {
			nd.logger.Printf("%s: %s", endpoint, result)
		}
	}

	return results
}

// DebugHelper provides high-level debugging functions
type DebugHelper struct {
	container TestContainer
	flagsDir  string
	container_diag *ContainerDiagnostics
	flags     *FlagDataInspector
	network   *NetworkDiagnostics
	scenario  *ScenarioDebugger
}

func NewDebugHelper(container TestContainer, flagsDir string) *DebugHelper {
	return &DebugHelper{
		container: container,
		flagsDir:  flagsDir,
		container_diag: NewContainerDiagnostics(container),
		flags:     NewFlagDataInspector(flagsDir),
		network:   NewNetworkDiagnostics(container),
		scenario:  NewScenarioDebugger(),
	}
}

// FullDiagnostics runs all available diagnostics
func (dh *DebugHelper) FullDiagnostics() map[string]interface{} {
	if !DebugMode {
		return nil
	}

	fmt.Println("🔍 Running Full E2E Diagnostics...")

	results := make(map[string]interface{})
	
	// Container info
	dh.container_diag.PrintContainerInfo()
	results["health"] = dh.container_diag.HealthCheck()
	
	// Network connectivity
	results["connectivity"] = dh.network.TestConnectivity()
	
	// Flag data
	if dh.flagsDir != "" {
		results["flag_files"] = dh.flags.ListFlagFiles()
		results["all_flags"] = dh.flags.InspectAllFlags()
	}

	fmt.Println("✅ Diagnostics complete")
	return results
}

// GetContainerDiagnostics returns the container diagnostics instance
func (dh *DebugHelper) GetContainerDiagnostics() *ContainerDiagnostics {
	return dh.container_diag
}

// GetFlagDataInspector returns the flag data inspector instance
func (dh *DebugHelper) GetFlagDataInspector() *FlagDataInspector {
	return dh.flags
}

// GetNetworkDiagnostics returns the network diagnostics instance
func (dh *DebugHelper) GetNetworkDiagnostics() *NetworkDiagnostics {
	return dh.network
}

// GetScenarioDebugger returns the scenario debugger instance
func (dh *DebugHelper) GetScenarioDebugger() *ScenarioDebugger {
	return dh.scenario
}

// Utility function for min (Go 1.21+)
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}