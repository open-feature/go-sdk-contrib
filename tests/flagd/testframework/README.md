# flagd E2E Test Framework

A comprehensive, reusable end-to-end testing framework for flagd providers across different resolver types (RPC, InProcess, File) with testcontainer integration, comprehensive debugging, and race condition fixes.

## Overview

This framework provides a complete testing infrastructure that supports all flagd resolver types with unified step definitions, centralized container management, and comprehensive debugging capabilities. The framework has been completely overhauled to fix race conditions, improve event handling, and provide a more maintainable testing architecture.

## Architecture Overview

### 🏗️ Design Principles
- **Provider Agnostic**: Same step definitions work across RPC, InProcess, and File providers
- **Generic Wildcard Patterns**: Future-proof step definitions using regex patterns (`^a ([^\\s]+) flagd provider$`)
- **Centralized Types**: All types consolidated in `types.go` for consistency
- **Comprehensive Debugging**: Built-in diagnostics infrastructure with `FLAGD_E2E_DEBUG` support
- **Testcontainer Integration**: Full container lifecycle management with health checks
- **Separation of Concerns**: Each file focuses on specific test aspects
- **Race Condition Free**: Fixed RPC service initialization and event handling races

### 🐳 Container Integration
- **Multi-port support**: RPC (8013), InProcess (8015), Launchpad (8080), Health (8014)
- **Version synchronization**: Automatic testbed version detection from submodule
- **Launchpad API integration**: Dynamic configuration changes and flag updates
- **Health monitoring**: Real-time container state and connectivity validation

## File Organization

```
testframework/
├── types.go              # All type definitions and TestState
├── utils.go              # Centralized utilities and ValueConverter
├── step_definitions.go   # Main scenario initialization with cleanup
├── config_steps.go       # Provider configuration testing
├── provider_steps.go     # Generic provider lifecycle with supplier pattern
├── flag_steps.go         # Flag evaluation with comprehensive validation
├── context_steps.go      # Context management with targeting keys
├── event_steps.go        # Event handling with Go channels (not arrays)
├── testbed_runner.go     # Centralized test orchestration and container management
├── testcontainer.go      # Testcontainer abstraction with lifecycle management
├── debug_helper.go       # Main debug coordinator
├── container_diagnostics.go  # Container health and port diagnostics
├── network_diagnostics.go    # Endpoint testing and connectivity validation
├── flag_data_inspector.go    # JSON validation and flag enumeration
├── README.md             # This documentation
└── DEBUG_UTILS.md        # Comprehensive debugging guide
```

## Core Architecture

### TestState
Central state object with enhanced event management:
```go
type TestState struct {
    // Provider configuration
    EnvVars      map[string]string
    ProviderType ProviderType
    Provider     openfeature.FeatureProvider
    Client       *openfeature.Client
    
    // Evaluation state with comprehensive tracking
    LastEvaluation EvaluationResult
    EvalContext    map[string]interface{}
    FlagKey        string
    FlagType       string
    DefaultValue   interface{}
    
    // Enhanced event tracking with Go channels
    EventChannel  chan EventRecord  // Replaced Java-style polling
    LastEvent     *EventRecord      // Multi-step verification support
    EventHandlers map[string]func(openfeature.EventDetails)
    
    // Container/testbed state
    Container    TestContainer
    LaunchpadURL string
    
    // Debug infrastructure
    DebugHelper  *DebugHelper
}
```

### Provider Types with Supplier Pattern
```go
type ProviderType int
const (
    RPC ProviderType = iota    // gRPC-based provider
    InProcess                  // HTTP sync-based provider  
    File                      // Offline file-based provider
)

// Provider supplier pattern for clean abstraction
type ProviderSupplier func(state TestState) (openfeature.FeatureProvider, error)

var (
    RPCProviderSupplier       ProviderSupplier
    InProcessProviderSupplier ProviderSupplier  
    FileProviderSupplier      ProviderSupplier
)
```

## Framework Components

### Central Test Orchestration

#### `testbed_runner.go`
Main test orchestration with enhanced container management:

**Key Responsibilities:**
- Testcontainer lifecycle management with health checks
- Container configuration and multi-port mapping
- Network setup and connectivity validation
- Integration with all provider types and configurations
- Launchpad API communication for dynamic flag management

**Usage Example:**
```go
runner := NewTestbedRunner(TestbedConfig{
    ProviderType: RPC,
    Tags:         []string{"~@reconnect", "~@events"},
})
defer runner.Cleanup()

// Enhanced provider supplier setup
runner.SetProviderSuppliers(
    rpcSupplier,
    inProcessSupplier, 
    fileSupplier,
)

err := runner.RunGherkinTestsWithSubtests(t, "RPC Provider Tests")
```

### Step Definition Categories

#### Configuration Steps (`config_steps.go`)
Test provider configuration validation and environment variable handling.

**Key Steps:**
- `a config was initialized`
- `an environment variable "([^"]*)" with value "([^"]*)"`
- `an option "([^"]*)" of type "([^"]*)" with value "([^"]*)"`
- `the option "([^"]*)" of type "([^"]*)" should have the value "([^"]*)"`
- `we should have an error`

**Enhanced Features:**
- Environment variable management with automatic cleanup
- Provider option validation with reflection
- Configuration error testing with detailed reporting
- Type conversion with comprehensive error handling

#### Provider Lifecycle Steps (`provider_steps.go`)
Enhanced provider management with supplier pattern and race condition fixes.

**Generic Wildcard Steps:**
- `^a ([^\\s]+) flagd provider$` - Universal provider creation
- `a stable flagd provider`
- `a ready event handler`
- `a error event handler`
- `a stale event handler`
- `the ready event handler should have been executed`
- `the connection is lost for (\d+)s`

**Race Condition Fixes:**
- **RPC service initialization**: Fixed race where service reported "initialized" before event stream was ready
- **Event stream connection**: Proper `streamReady chan error` signaling
- **Provider cleanup**: Enhanced cleanup between scenarios prevents contamination

#### Flag Evaluation Steps (`flag_steps.go`)
Comprehensive flag evaluation with metadata support and enhanced validation.

**Key Steps:**
- `a ([^-]*)-flag with key "([^"]*)" and a default value "([^"]*)"`
- `the flag was evaluated with details`
- `the resolved details value should be "([^"]*)"`
- `the reason should be "([^"]*)"`
- `the error-code should be "([^"]*)"`
- `the variant should be "([^"]*)"`
- `the metadata should contain key "([^"]*)" with value "([^"]*)"`

**Enhanced Features:**
- Multi-type flag support (Boolean, String, Integer, Float, Object)
- Detailed evaluation result validation with comprehensive assertions
- Metadata testing and validation
- Error handling with specific reason codes
- Default value management with type safety

#### Context Management Steps (`context_steps.go`)
Enhanced context handling with targeting key support.

**Key Steps:**
- `a context containing a key "([^"]*)", with type "([^"]*)" and with value "([^"]*)"`
- `a context containing a key "([^"]*)" with value "([^"]*)"`
- `an empty context`
- `a context with the following keys:` (data table support)
- `the targeting key is set to "([^"]*)"`

**Enhanced Features:**
- Dynamic context building with type validation
- Targeting key management and validation
- Nested context properties support
- Data table support for complex scenarios
- Context validation helpers with comprehensive error reporting

#### Event Handling Steps (`event_steps.go`)
Completely reworked event system using Go channels instead of Java-style arrays.

**Key Steps:**
- `a change event handler`
- `a change event was fired`
- `the flag should be part of the event payload`
- `a stale event was fired`
- `^I receive a ([^\\s]+) event$` - Generic event handling

**Major Improvements:**
- **Go channels**: `EventChannel chan EventRecord` instead of polling arrays
- **Non-blocking**: Prevents deadlocks during event handling
- **Event isolation**: Fresh channels per scenario prevent cross-contamination
- **LastEvent tracking**: Multi-step event verification support
- **Timeout handling**: Proper event waiting with configurable timeouts

### Enhanced Utility Infrastructure

#### `utils.go` - ValueConverter
Centralized type conversion with comprehensive error handling:

```go
type ValueConverter struct{}

// Enhanced conversion with better error handling
func (vc *ValueConverter) ConvertForSteps(value string, valueType string) (interface{}, error)

// Configuration reflection with validation
func (vc *ValueConverter) ConvertToReflectValue(valueType, value string, fieldType reflect.Type) reflect.Value
```

**Supported Types with Validation:**
- `Boolean` → `bool` (with string validation)
- `Integer/Long` → `int64` (with overflow checks)
- `Float` → `float64` (with precision validation)
- `String` → `string` (with encoding validation)
- `Object` → `interface{}` (with JSON validation)
- `ResolverType` → `ProviderType` (with enum validation)
- `CacheType` → `string` (with supported type validation)

#### Helper Functions
- `CamelToSnake()` - Convert naming conventions with validation
- `ToFieldName()` - Convert to Go field names with reflection
- `ValueToString()` - Generic string conversion with type safety
- `StringToInt/Boolean()` - Type-safe conversions with error handling

### Container Integration (`testcontainer.go`)

#### Enhanced FlagdTestContainer
Comprehensive container management with health monitoring:

```go
type FlagdTestContainer struct {
    container     testcontainers.Container
    host          string
    launchpadURL  string
    rpcPort       int
    inProcessPort int
    launchpadPort int
    healthPort    int
    
    // Enhanced monitoring
    healthChecker *HealthChecker
    diagnostics   *ContainerDiagnostics
}
```

**Enhanced Methods:**
- `GetHost()` - Container host with connectivity validation
- `GetPort(service)` - Service-specific ports with health checks
- `StartFlagdWithConfig(config)` - Enhanced configuration via launchpad
- `TriggerFlagChange()` - Simulate flag updates with verification
- `IsHealthy()` - Comprehensive health validation
- `GetDiagnostics()` - Real-time container diagnostics

## Debug Infrastructure

### 🚀 Comprehensive Debugging (5 Specialized Components)

Enable with: `FLAGD_E2E_DEBUG=true`

#### `debug_helper.go`
Main debug coordinator with environment-controlled output:
- Centralized debug state management
- Component coordination and output formatting
- Debug level control and filtering

#### `container_diagnostics.go`
Container health and lifecycle monitoring:
- Real-time container state monitoring
- Port mapping validation and connectivity tests
- Resource usage and performance metrics
- Container log streaming and analysis

#### `network_diagnostics.go`
Endpoint testing and connectivity validation:
- Port accessibility testing
- Network latency and connectivity monitoring
- Service endpoint validation
- Connection troubleshooting utilities

#### `flag_data_inspector.go`
JSON validation and flag enumeration:
- Flag configuration validation and parsing
- JSON schema validation
- Flag enumeration and relationship mapping
- Configuration diff analysis

#### Event Tracking
Complete event lifecycle monitoring:
- Event stream connection monitoring
- Event sequence validation and timing
- Event payload inspection and validation
- Event history tracking and analysis

## Race Condition Fixes

### RPC Service Initialization
**Problem Fixed:** RPC service reported "initialized" before event stream was ready.

**Solution:**
```go
// Added streamReady channel for proper synchronization
type Service struct {
    streamReady chan error
    // ... other fields
}

func (s *Service) Init() error {
    // ... setup code
    return <-s.streamReady  // Wait for actual stream connection
}
```

### Event System Overhaul
**Problem Fixed:** Java-style array polling caused race conditions and missed events.

**Solution:**
```go
// Before: Events []EventRecord (polling)
// After: EventChannel chan EventRecord (non-blocking)
type TestState struct {
    EventChannel chan EventRecord
    LastEvent    *EventRecord
}
```

### Provider Cleanup Enhancement
**Problem Fixed:** State contamination between test scenarios.

**Solution:**
- Enhanced `cleanupProvider()` with proper shutdown
- Clear event channels in After hooks
- Isolated state management per scenario

## Usage Examples

### Basic Framework Usage
```go
import "github.com/open-feature/go-sdk-contrib/tests/flagd/testframework"

func TestRPCProviderE2E(t *testing.T) {
    // Create runner with enhanced configuration
    runner := NewTestbedRunner(TestbedConfig{
        ProviderType: RPC,
        Tags:         []string{"~@reconnect", "~@events"},
        DebugMode:    os.Getenv("FLAGD_E2E_DEBUG") == "true",
    })
    defer runner.Cleanup()
    
    // Enhanced provider supplier setup with validation
    runner.SetProviderSuppliers(
        runner.createRPCProviderSupplier(),
        runner.createInProcessProviderSupplier(), 
        runner.createFileProviderSupplier(),
    )
    
    // Run with enhanced error handling
    err := runner.RunGherkinTestsWithSubtests(t, "RPC Provider Tests")
    require.NoError(t, err)
}
```

### Debug Mode Usage
```bash
# Enable comprehensive debugging
FLAGD_E2E_DEBUG=true go test -v ./your-test-package

# This provides:
# - Container lifecycle monitoring
# - Network connectivity validation  
# - Flag configuration inspection
# - Event stream monitoring
# - Provider state validation
```

### Provider-Specific Configuration
```go
// RPC Provider with enhanced options
config := TestbedConfig{
    ProviderType: RPC,
    Tags:         []string{"~@reconnect", "~@events", "~@grace", "~@sync", "~@metadata"},
    Timeout:      30 * time.Second,
    RetryPolicy:  &RetryPolicy{MaxAttempts: 3, BackoffMs: 1000},
}

// InProcess Provider with sync configuration
config := TestbedConfig{
    ProviderType: InProcess,
    Tags:         []string{"~@grace", "~@reconnect", "~@events", "~@sync"},
    SyncEndpoint: "http://localhost:8015",
}

// File Provider with offline configuration
config := TestbedConfig{
    ProviderType: File,
    Tags:         []string{"~@reconnect", "~@sync", "~@grace", "~@events"},
    FilePath:     "/tmp/allFlags.json",
    WatchMode:    true,
}
```

## Extension Points

### Adding New Step Definitions
1. **Choose appropriate file**: Add to relevant `*_steps.go` file based on functionality
2. **Use generic patterns**: Implement wildcard regex patterns for future-proofing
3. **Register step**: Add to `InitializeScenario()` function in `step_definitions.go`
4. **Update types**: Add new types to `types.go` if needed
5. **Use utilities**: Leverage `ValueConverter` and helpers from `utils.go`
6. **Add debugging**: Include appropriate debug output for troubleshooting

### Adding New Provider Types
1. **Update types**: Add new `ProviderType` constant to enum
2. **Implement supplier**: Add `ProviderSupplier` function with validation
3. **Update configuration**: Add provider-specific config to `TestbedConfig`
4. **Test integration**: Ensure container integration works with new provider
5. **Add debug support**: Include provider-specific debugging capabilities

### Testing New Scenarios
1. **Add feature files**: Create new `.feature` files in testbed repository
2. **Implement steps**: Add missing step definitions following generic patterns
3. **Update tag filtering**: Add appropriate tags for scenario organization
4. **Test across providers**: Verify scenarios work with all provider types
5. **Add documentation**: Update this README with new capabilities

## Performance Considerations

- **Container reuse**: Efficient container lifecycle management reduces test startup time
- **Parallel execution**: Framework supports parallel test execution across provider types
- **Resource cleanup**: Proper cleanup prevents resource leaks and test interference
- **Optimized startup**: Fast container initialization with intelligent health checks
- **Event efficiency**: Non-blocking event handling prevents deadlocks and timeouts
- **Debug overhead**: Debug mode can be disabled for performance-critical testing

## Migration Guide

### From Old Integration Package
The framework replaces the previous `pkg/integration/*` package:

1. **Update imports**: Change to `tests/flagd/testframework`
2. **Use new runner**: Replace old test runners with `NewTestbedRunner`
3. **Update step registration**: Steps are automatically available through `InitializeScenario()`
4. **Migrate suppliers**: Update provider suppliers to new signature
5. **Enable debugging**: Use `FLAGD_E2E_DEBUG=true` for enhanced diagnostics

### Benefits of Migration
- **Enhanced reliability**: Fixed race conditions and improved error handling
- **Better debugging**: Comprehensive debug infrastructure for troubleshooting
- **Improved maintainability**: Centralized types and utilities
- **Future-proof design**: Generic patterns support easy extension
- **Performance improvements**: Optimized container management and event handling

This framework provides a solid foundation for comprehensive flagd provider testing while maintaining clean separation between test infrastructure and business logic. The enhanced debugging capabilities and race condition fixes significantly improve developer experience and test reliability.