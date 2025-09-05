# Flagd E2E Debug Utils

A comprehensive debugging infrastructure for flagd provider end-to-end tests that helps developers troubleshoot test failures and understand test execution.

## Overview

The debug utils provide structured, conditional debugging output that can be enabled via environment variables. These utilities help diagnose:

- Container health and connectivity issues
- Flag data problems and validation errors
- Network connectivity issues  
- Gherkin scenario failures and test state
- Provider configuration problems

## Quick Start

### Enable Debug Mode

```bash
export FLAGD_E2E_DEBUG=true
go test -tags=e2e ./e2e/...
```

### Basic Usage

```go
import "github.com/open-feature/go-sdk-contrib/tests/flagd/testframework"

// Create debug helper with container and flags directory
debugHelper := testframework.NewDebugHelper(container, flagsDir)

// Run comprehensive diagnostics
results := debugHelper.FullDiagnostics()

// Access individual components
containerDiag := debugHelper.GetContainerDiagnostics()
flagInspector := debugHelper.GetFlagDataInspector()
networkDiag := debugHelper.GetNetworkDiagnostics()
scenarioDebug := debugHelper.GetScenarioDebugger()
```

## Core Components

### 1. DebugHelper

The main orchestrator that provides access to all debugging components.

```go
// Create a debug helper
debugHelper := testframework.NewDebugHelper(container, flagsDir)

// Run all diagnostics at once
results := debugHelper.FullDiagnostics()

// Access individual components
containerDiag := debugHelper.GetContainerDiagnostics()
```

**Methods:**
- `FullDiagnostics()` - Runs all available diagnostics and returns results
- `GetContainerDiagnostics()` - Returns container diagnostic tools
- `GetFlagDataInspector()` - Returns flag data inspection tools
- `GetNetworkDiagnostics()` - Returns network diagnostic tools  
- `GetScenarioDebugger()` - Returns scenario debugging tools

### 2. ContainerDiagnostics

Provides container health monitoring and debugging.

```go
containerDiag := testframework.NewContainerDiagnostics(container)

// Print comprehensive container information
containerDiag.PrintContainerInfo()

// Perform health checks
healthResults := containerDiag.HealthCheck()

// Display recent container logs
containerDiag.PrintContainerLogs(50) // last 50 lines
```

**Features:**
- Container port mapping display
- Health endpoint validation
- Launchpad connectivity testing
- Container log streaming (placeholder for future implementation)

### 3. FlagDataInspector

Helps debug flag-related issues and validates flag data.

```go
flagInspector := testframework.NewFlagDataInspector(flagsDir)

// List all flag files in directory
files := flagInspector.ListFlagFiles()

// Inspect and validate allFlags.json content
flags := flagInspector.InspectAllFlags()
```

**Features:**
- Flag file discovery and listing
- JSON validation and parsing
- Flag enumeration and display
- Error reporting for malformed flag data

### 4. NetworkDiagnostics

Tests network connectivity to all flagd endpoints.

```go
networkDiag := testframework.NewNetworkDiagnostics(container)

// Test connectivity to all endpoints
results := networkDiag.TestConnectivity()
```

**Features:**
- HTTP endpoint testing (launchpad, health)
- gRPC endpoint validation
- Connection timeout handling
- Structured results reporting

### 5. ScenarioDebugger

Provides context for Gherkin scenario failures.

```go
scenarioDebug := testframework.NewScenarioDebugger()

// Debug a failed scenario
scenarioDebug.DebugScenarioFailure(
    "Flag evaluation with context", 
    err, 
    testState,
)
```

**Features:**
- Scenario failure context
- Test state serialization
- Error details reporting
- JSON pretty-printing for complex objects

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `FLAGD_E2E_DEBUG` | Enable verbose debug output | `false` |

## Output Examples

### Container Information
```
[DEBUG:CONTAINER] === Container Information ===
[DEBUG:CONTAINER] Host: localhost
[DEBUG:CONTAINER] RPC Port: 8013
[DEBUG:CONTAINER] InProcess Port: 8015
[DEBUG:CONTAINER] Launchpad Port: 8080
[DEBUG:CONTAINER] Health Port: 8014
[DEBUG:CONTAINER] Launchpad URL: http://localhost:8080
[DEBUG:CONTAINER] Healthy: true
```

### Health Check Results
```json
[DEBUG:CONTAINER] Health Check Results:
{
  "container_healthy": true,
  "host": "localhost",
  "launchpad_url": "http://localhost:8080",
  "launchpad_status": 200,
  "flagd_health_status": 200
}
```

### Flag File Inspection
```
[DEBUG:FLAGS] === allFlags.json Content ===
[DEBUG:FLAGS] File path: /tmp/flagd-e2e-12345/allFlags.json
[DEBUG:FLAGS] ✅ File exists and is valid JSON
[DEBUG:FLAGS] Flag count: 3
[DEBUG:FLAGS] Available flags:
[DEBUG:FLAGS]   - simple-flag
[DEBUG:FLAGS]   - context-aware-flag
[DEBUG:FLAGS]   - complex-evaluation-flag
```

### Network Connectivity
```
[DEBUG:NETWORK] === Network Connectivity Test ===
[DEBUG:NETWORK] launchpad: HTTP 200
[DEBUG:NETWORK] health: HTTP 200
[DEBUG:NETWORK] rpc: TCP connection test needed
```

### Scenario Failure
```
[DEBUG:SCENARIO] === Scenario Failure Debug ===
[DEBUG:SCENARIO] Scenario: Flag change event with caching
[DEBUG:SCENARIO] Error: expected reason STATIC, got CACHED
[DEBUG:SCENARIO] Test State at Failure:
{
  "flagKey": "test-flag",
  "expectedReason": "STATIC",
  "actualReason": "CACHED",
  "provider": "rpc",
  "cacheEnabled": true
}
```

## Integration Examples

### Basic E2E Test Setup

```go
package e2e

import (
    "github.com/open-feature/go-sdk-contrib/tests/flagd/testframework"
)

func TestFlagEvaluation(t *testing.T) {
    // Setup container
    container, err := testframework.NewFlagdContainer(ctx, config)
    require.NoError(t, err)
    
    // Create debug helper
    debugHelper := testframework.NewDebugHelper(container, flagsDir)
    
    // On test failure, run diagnostics
    t.Cleanup(func() {
        if t.Failed() && testframework.DebugMode {
            debugHelper.FullDiagnostics()
        }
    })
    
    // Your test logic here...
}
```

### Custom Gherkin Step Definitions

```go
func (ts *TestState) setupProvider() error {
    // Create provider...
    
    // Debug on setup failure
    if err != nil && testframework.DebugMode {
        debugHelper := testframework.NewDebugHelper(ts.Container, flagsDir)
        scenarioDebug := debugHelper.GetScenarioDebugger()
        scenarioDebug.DebugScenarioFailure("Provider Setup", err, ts)
    }
    
    return err
}
```

### CI/CD Integration

```yaml
# GitHub Actions example
- name: Run E2E Tests with Debug
  env:
    FLAGD_E2E_DEBUG: ${{ github.event_name == 'pull_request' }}
  run: go test -tags=e2e ./e2e/...
```

## Architecture

### Component Relationships

```
DebugHelper (Orchestrator)
├── ContainerDiagnostics
│   ├── Container health checks
│   ├── Port mapping display
│   └── Log streaming
├── FlagDataInspector
│   ├── Flag file discovery
│   ├── JSON validation
│   └── Content inspection
├── NetworkDiagnostics
│   ├── HTTP endpoint testing
│   ├── gRPC connectivity
│   └── Timeout handling
└── ScenarioDebugger
    ├── Failure context
    ├── State serialization
    └── Error reporting
```

### Design Principles

1. **Conditional Output**: All debug output respects `FLAGD_E2E_DEBUG` environment variable
2. **Structured Logging**: Consistent prefix-based logging with component identification
3. **Zero Performance Impact**: When debug mode is disabled, minimal overhead
4. **Comprehensive Coverage**: Addresses common debugging scenarios
5. **Reusable Components**: Can be used across different test suites
6. **Graceful Degradation**: Handles missing containers/data gracefully

## Troubleshooting Guide

### Common Issues

**Container not starting:**
```bash
FLAGD_E2E_DEBUG=true go test -tags=e2e -run TestContainer
# Look for container health check failures
```

**Flag evaluation errors:**
```bash
FLAGD_E2E_DEBUG=true go test -tags=e2e -run TestFlags
# Check flag file validation output
```

**Network connectivity issues:**
```bash
FLAGD_E2E_DEBUG=true go test -tags=e2e -run TestNetwork
# Review endpoint connectivity results
```

**Scenario timeouts:**
```bash
FLAGD_E2E_DEBUG=true go test -tags=e2e -run TestScenario -timeout=60s
# Check scenario failure debug output
```

### Performance Considerations

- Debug utils add ~0ms overhead when disabled
- JSON serialization only occurs in debug mode
- Network tests use 5-second timeouts
- File operations are lazy-loaded

### Future Enhancements

- [ ] Container log streaming implementation
- [ ] Real-time metrics collection
- [ ] Integration with observability tools
- [ ] Custom debug formatters
- [ ] Automated failure analysis

## Contributing

When adding new debug features:

1. Follow the conditional debug pattern (`if DebugMode`)
2. Use structured logging with component prefixes
3. Provide both human-readable and machine-readable output
4. Handle edge cases gracefully
5. Add corresponding documentation examples

## API Reference

See the Go documentation for complete API details:
```bash
go doc github.com/open-feature/go-sdk-contrib/tests/flagd/testframework
```