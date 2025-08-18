# flagd E2E Test Framework Step Definitions

This package provides reusable Gherkin step definitions for testing flagd providers across different resolver types (RPC, InProcess, File) with comprehensive debugging and testcontainer integration.

## Architecture Overview

### Design Principles
- **Provider Agnostic**: Same steps work with RPC, InProcess, and File providers
- **Generic Wildcard Patterns**: Future-proof step definitions using regex patterns
- **Centralized Types**: All types consolidated in `types.go`
- **Comprehensive Debugging**: Built-in diagnostics and debugging infrastructure
- **Testcontainer Integration**: Full container lifecycle management
- **Separation of Concerns**: Each file focuses on specific test aspects

## File Organization

```
testframework/
├── types.go              # All type definitions
├── utils.go              # Centralized utilities and converters
├── step_definitions.go   # Main scenario initialization
├── config_steps.go       # Provider configuration testing
├── provider_steps.go     # Generic provider lifecycle with wildcards
├── flag_steps.go         # Flag evaluation with metadata support
├── context_steps.go      # Context management with targeting keys
├── event_steps.go        # Generic event handling with wildcards
├── testcontainer.go      # Testcontainer management
├── debug_utils.go        # Comprehensive debugging infrastructure
├── README.md             # This documentation
└── DEBUG_UTILS.md        # Debugging guide
```

## Core Types

### TestState
Central state object passed between all steps:
```go
type TestState struct {
    // Provider configuration
    EnvVars      map[string]string
    ProviderType ProviderType
    Provider     openfeature.FeatureProvider
    Client       *openfeature.Client
    
    // Evaluation state
    LastEvaluation EvaluationResult
    EvalContext    map[string]interface{}
    FlagKey        string
    FlagType       string
    DefaultValue   interface{}
    
    // Event tracking
    Events        []EventRecord
    EventHandlers map[string]func(openfeature.EventDetails)
    
    // Container/testbed state
    Container    TestContainer
    LaunchpadURL string
}
```

### Provider Types
```go
type ProviderType int
const (
    RPC ProviderType = iota    // gRPC-based provider
    InProcess                  // HTTP sync-based provider  
    File                      // Offline file-based provider
)
```

## Step Definition Categories

### Configuration Steps (`config_steps.go`)
Test provider configuration validation and environment variable handling.

**Key Steps:**
- `a config was initialized`
- `an environment variable "([^"]*)" with value "([^"]*)"`
- `an option "([^"]*)" of type "([^"]*)" with value "([^"]*)"`
- `the option "([^"]*)" of type "([^"]*)" should have the value "([^"]*)"`
- `we should have an error`

**Features:**
- Environment variable management with cleanup
- Provider option validation
- Configuration error testing
- Type conversion with reflection

### Provider Lifecycle Steps (`provider_steps.go`)
Manage provider creation, initialization, and state transitions.

**Key Steps:**
- `a stable flagd provider`
- `a ready event handler`
- `a error event handler` 
- `a stale event handler`
- `the ready event handler should have been executed`
- `the connection is lost for (\d+)s`

**Features:**
- Provider supplier abstraction
- Event handler management
- Connection simulation
- State validation

### Flag Evaluation Steps (`flag_steps.go`)
Test flag evaluation across all data types and scenarios.

**Key Steps:**
- `a ([^-]*)-flag with key "([^"]*)" and a default value "([^"]*)"`
- `the flag was evaluated with details`
- `the resolved details value should be "([^"]*)"`
- `the reason should be "([^"]*)"`
- `the error-code should be "([^"]*)"`
- `the variant should be "([^"]*)"`

**Features:**
- Multi-type flag support (Boolean, String, Integer, Float, Object)
- Detailed evaluation result validation
- Error handling and reason checking
- Default value management

### Context Management Steps (`context_steps.go`)
Handle evaluation context for flag targeting and personalization.

**Key Steps:**
- `a context containing a key "([^"]*)", with type "([^"]*)" and with value "([^"]*)"`
- `a context containing a key "([^"]*)" with value "([^"]*)"`
- `an empty context`
- `a context with the following keys:` (data table)

**Features:**
- Dynamic context building
- Type-safe value conversion
- Data table support
- Context validation helpers

### Event Handling Steps (`event_steps.go`)
Test provider event lifecycle and change notifications.

**Key Steps:**
- `a change event handler`
- `a change event was fired`
- `the flag should be part of the event payload`
- `a stale event was fired`

**Features:**
- Event sequence validation
- Event payload inspection
- Timeout-based event waiting
- Event history tracking

## Utility Functions (`utils.go`)

### ValueConverter
Centralized type conversion for step parameters:
```go
type ValueConverter struct{}

// Main conversion method
func (vc *ValueConverter) ConvertForSteps(value string, valueType string) (interface{}, error)

// Configuration reflection method  
func (vc *ValueConverter) ConvertToReflectValue(valueType, value string, fieldType reflect.Type) reflect.Value
```

**Supported Types:**
- `Boolean` → `bool`
- `Integer/Long` → `int64` 
- `Float` → `float64`
- `String` → `string`
- `Object` → `interface{}` (JSON parsing)
- `ResolverType` → `ProviderType`
- `CacheType` → `string`

### Helper Functions
- `CamelToSnake()` - Convert naming conventions
- `ToFieldName()` - Convert to Go field names
- `ValueToString()` - Generic string conversion
- `StringToInt/Boolean()` - Type-safe conversions

## Container Integration (`testcontainer.go`)

### FlagdTestContainer
Manages flagd testbed container lifecycle:
```go
type FlagdTestContainer struct {
    container     testcontainers.Container
    host          string
    launchpadURL  string
    rpcPort       int
    inProcessPort int
    launchpadPort int
    healthPort    int
}
```

**Key Methods:**
- `GetHost()` - Container host address
- `GetPort(service)` - Service-specific ports
- `StartFlagdWithConfig(config)` - Configure flagd via launchpad
- `TriggerFlagChange()` - Simulate flag updates
- `IsHealthy()` - Health check validation

## Provider Supplier Pattern

The step definitions use a supplier pattern to create providers for different resolver types:

```go
type ProviderSupplier func(state TestState) (openfeature.FeatureProvider, error)

var (
    RPCProviderSupplier       ProviderSupplier
    InProcessProviderSupplier ProviderSupplier  
    FileProviderSupplier      ProviderSupplier
)
```

This allows the same step definitions to work with different provider implementations by injecting the appropriate supplier functions.

## Usage Example

```go
// In your test file (e.g., providers/flagd/e2e/rpc_test.go)
func TestRPCProviderE2E(t *testing.T) {
    runner := NewTestbedRunner(TestbedConfig{
        ResolverType: integration.RPC,
        TestbedConfig: "default",
    })
    defer runner.Cleanup()
    
    // Runner sets up provider suppliers
    integration.SetProviderSuppliers(
        runner.createRPCProviderSupplier(),
        runner.createInProcessProviderSupplier(), 
        runner.createFileProviderSupplier(),
    )
    
    // Steps are automatically available through integration.InitializeScenario()
    err := runner.RunGherkinTestsWithSubtests(t, featurePaths, tags)
}
```

## Extension Points

### Adding New Steps
1. Add step implementation to appropriate `*_steps.go` file
2. Register in `InitializeScenario()` function
3. Use `TestState` for state management
4. Use `ValueConverter` for type conversions

### Adding New Provider Types
1. Add new `ProviderType` constant
2. Implement `ProviderSupplier` function
3. Update provider creation logic
4. Add provider-specific configuration

### Testing New Scenarios
1. Add new `.feature` files to testbed
2. Implement missing step definitions
3. Update tag filtering as needed
4. Test across all provider types

This architecture provides a solid foundation for comprehensive flagd provider testing while maintaining clean separation between test infrastructure and business logic.