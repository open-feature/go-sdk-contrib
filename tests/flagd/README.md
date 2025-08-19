# Flagd Integration Testing Framework

This directory contains a comprehensive testing framework for flagd providers that enables running all gherkin scenarios from the flagd-testbed against different resolver types (RPC, in-process, file) using testcontainers and a unified set of step definitions.

## Architecture

### Key Components

1. **Unified Step Definitions (`testframework/`)**: Single source of step definitions that work across all resolver types
2. **Testcontainer Integration**: Uses testcontainers-go to manage flagd-testbed instances
3. **Provider Abstraction**: Supports all flagd resolver types through a common interface
4. **Gherkin Compatibility**: Runs all flagd-testbed gherkin scenarios with appropriate tagging
5. **Debug Utils**: Comprehensive debugging infrastructure for troubleshooting test failures

### Directory Structure

```
tests/flagd/
├── testframework/            # Unified step definitions and test framework
│   ├── step_definitions.go    # Main initialization and shared state
│   ├── config_steps.go        # Configuration step definitions
│   ├── provider_steps.go      # Provider lifecycle management
│   ├── flag_steps.go         # Flag evaluation steps
│   ├── context_steps.go      # Evaluation context management
│   ├── event_steps.go        # Event handling steps
│   ├── testcontainer.go      # Testcontainers implementation
│   ├── debug_utils.go        # Comprehensive debugging infrastructure
│   ├── types.go              # Shared types and interfaces
│   ├── utils.go              # Utility functions
│   └── DEBUG_UTILS.md        # Debug utils documentation
├── go.mod                    # Module dependencies
└── README.md                 # This file
```

## Step Definition Organization

Following the patterns from Java and Python implementations, step definitions are organized by domain:

- **Configuration**: Reuses existing `config.go` with enhancements for TestState integration
- **Provider Lifecycle**: Generic wildcard patterns for provider creation (`^a (\w+) flagd provider$`)
- **Flag Evaluation**: All flag evaluation scenarios (boolean, string, integer, float, object)
- **Context Management**: Evaluation context setup with targeting key support
- **Event Handling**: Consolidated event handlers with generic wildcard patterns (`^a (\w+) event handler$`)

## Usage in Provider Tests

The framework is designed to be used from provider-specific test files in `providers/flagd/e2e/`:

### Example RPC Provider Tests

```go
func TestRPCProviderE2E(t *testing.T) {
    runner := NewTestbedRunner(TestbedConfig{
        ResolverType:  testframework.RPC,
        TestbedConfig: "default",
    })
    defer runner.Cleanup()
    
    ctx := context.Background()
    if err := runner.SetupContainer(ctx); err != nil {
        t.Fatalf("Failed to setup container: %v", err)
    }
    
    featurePaths := []string{
        "../../flagd-testbed/gherkin",
    }
    
    tags := "@rpc && ~@targetURI && ~@unixsocket"
    
    if err := runner.RunGherkinTests(featurePaths, tags); err != nil {
        t.Fatalf("Tests failed: %v", err)
    }
}
```

### Example In-Process Provider Tests

```go
func TestInProcessProviderE2E(t *testing.T) {
    runner := NewTestbedRunner(TestbedConfig{
        ResolverType:  testframework.InProcess,
        TestbedConfig: "default",
    })
    defer runner.Cleanup()
    
    // ... similar setup
    tags := "@in-process && ~@rpc && ~@file"
    // ... run tests
}
```

### Example File Provider Tests

```go
func TestFileProviderE2E(t *testing.T) {
    tempDir := t.TempDir()
    createTestFlagFile(tempDir) // Create flag configuration
    
    runner := NewTestbedRunner(TestbedConfig{
        ResolverType: testframework.File,
        FlagsDir:     tempDir,
    })
    defer runner.Cleanup()
    
    // No container needed for file provider
    tags := "@file && ~@rpc && ~@in-process && ~@events"
    // ... run tests
}
```

## Test State Management

The framework uses a unified `TestState` struct that maintains:

- **Provider Configuration**: Options, environment variables, resolver type
- **Evaluation State**: Last evaluation results, context, flag information
- **Event Tracking**: All provider events with timestamps
- **Container State**: Testcontainer management and launchpad integration

## Testcontainer Integration

The `FlagdTestContainer` provides:

- **Lifecycle Management**: Start, stop, restart flagd services
- **Health Checks**: Wait for flagd readiness
- **Launchpad API**: Trigger configuration changes and restarts
- **Multi-Port Support**: RPC (8013), in-process (8015), launchpad (8080), health (8014)

## Provider Supplier Pattern

Following the Java/Python pattern, provider creation is abstracted through supplier functions:

```go
testframework.SetProviderSuppliers(
    createRPCProviderSupplier(),
    createInProcessProviderSupplier(), 
    createFileProviderSupplier(),
)
```

This allows the step definitions to create appropriate providers without knowing the specific resolver type.

## Gherkin Tag Strategy

The framework uses Gherkin tags to run appropriate scenarios for each resolver type:

- `@rpc`: RPC provider scenarios
- `@in-process`: In-process provider scenarios  
- `@file`: File-based provider scenarios
- `@events`: Event-related scenarios
- `@targeting`: Targeting/context scenarios
- `@caching`: Cache-related scenarios (RPC only)
- `@ssl`: SSL/TLS scenarios

### Tag Filtering Examples

```bash
# RPC provider tests excluding advanced features
@rpc && ~@targetURI && ~@unixsocket && ~@sync

# In-process provider with sync features
@in-process && @sync

# File provider basics
@file && ~@events && ~@reconnect

# SSL/TLS certificate tests specifically
@customCert
```

## Debugging E2E Tests

The framework includes comprehensive debugging utilities to help troubleshoot test failures. See [DEBUG_UTILS.md](testframework/DEBUG_UTILS.md) for complete documentation.

### Quick Debug Mode

```bash
# Enable debug output for all tests
export FLAGD_E2E_DEBUG=true
go test -v ./e2e

# Debug specific test with verbose output
FLAGD_E2E_DEBUG=true go test -v -run TestRPCProvider ./e2e
```

### Debug Output Features

- **Container Health**: Port mapping, connectivity, health checks
- **Flag Data Validation**: JSON parsing, flag enumeration, file discovery
- **Network Diagnostics**: Endpoint testing, connectivity validation
- **Scenario Context**: Failure debugging with test state serialization

### Example Debug Output

```
🔍 Running Full E2E Diagnostics...
[DEBUG:CONTAINER] === Container Information ===
[DEBUG:CONTAINER] Host: localhost
[DEBUG:CONTAINER] RPC Port: 8013
[DEBUG:FLAGS] ✅ File exists and is valid JSON
[DEBUG:FLAGS] Available flags: simple-flag, context-aware-flag
[DEBUG:NETWORK] launchpad: HTTP 200
✅ Diagnostics complete
```

## Running Tests

### Prerequisites

1. Docker daemon running (for testcontainers)
2. Go 1.21+
3. Network access to pull `ghcr.io/open-feature/flagd-testbed` images

### Running Individual Test Suites

```bash
# Run RPC provider tests
cd providers/flagd && go test -v ./e2e -run TestRPCProvider

# Run in-process provider tests  
cd providers/flagd && go test -v ./e2e -run TestInProcess

# Run file provider tests
cd providers/flagd && go test -v ./e2e -run TestFileProvider

# Run all configuration tests
cd providers/flagd && go test -v ./e2e -run TestConfiguration
```

### Running All E2E Tests

```bash
cd providers/flagd && go test -v ./e2e
```

### Skip Long-Running Tests

```bash
go test -v -short ./e2e
```

## Integration with Existing Config System

The framework bridges with the existing `tests/flagd/pkg/integration/config.go` system through:

1. **Config Bridge**: `config_bridge.go` translates between the existing context-based approach and the new TestState
2. **Step Reuse**: Existing configuration step definitions are reused and enhanced
3. **Provider Options**: Configuration is converted to flagd provider options seamlessly

## Benefits

1. **Single Source of Truth**: One set of step definitions for all resolver types
2. **Comprehensive Coverage**: All flagd-testbed scenarios can be run (currently 108/130 scenarios passing)
3. **Real Integration**: Uses actual flagd instances via testcontainers
4. **Maintainability**: Centralized test logic with generic wildcard patterns reduces duplication
5. **Compatibility**: Works with existing configuration testing framework
6. **Flexibility**: Supports different testbed configurations and scenarios
7. **Enhanced Debugging**: Comprehensive debug utilities with container diagnostics
8. **Future-Proof**: Generic patterns adapt to new provider types and event handlers

## Future Enhancements

1. **Custom Gherkin Features**: Add provider-specific scenarios
2. **Performance Testing**: Benchmark scenarios using the same framework
3. **Parallel Execution**: Run different resolver types in parallel
4. **CI Integration**: Structured test reporting and artifact collection
5. **Mock Modes**: Support for offline testing without containers

## Contributing

When adding new step definitions:

1. Choose the appropriate domain file (config, provider, flag, context, event)
2. Follow existing patterns for error handling and state management
3. Add corresponding tests in `providers/flagd/e2e/`
4. Update documentation for new scenarios or configurations