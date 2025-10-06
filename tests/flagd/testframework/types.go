package testframework

import (
	"time"

	flagd "github.com/open-feature/go-sdk-contrib/providers/flagd/pkg"
	"github.com/open-feature/go-sdk/openfeature"
)

// ProviderType represents the type of provider being tested
type ProviderType int

const (
	RPC ProviderType = iota
	InProcess
	File
)

func (p ProviderType) String() string {
	switch p {
	case RPC:
		return "rpc"
	case InProcess:
		return "in-process"
	case File:
		return "file"
	default:
		return "unknown"
	}
}

// TestStateKey is the key used to pass TestState across context.Context
type TestStateKey struct{}

// EvaluationResult holds the result of flag evaluation in a generic way
type EvaluationResult struct {
	FlagKey      string
	Value        interface{}
	Reason       openfeature.Reason
	Variant      string
	ErrorCode    openfeature.ErrorCode
	ErrorMessage string
}

// EventRecord tracks events for verification
type EventRecord struct {
	Type      string
	Timestamp time.Time
	Details   openfeature.EventDetails
}

// TestState holds all test state shared across step definitions
type TestState struct {
	// Provider configuration
	EnvVars      map[string]string
	ProviderType ProviderType
	Provider     openfeature.FeatureProvider
	Client       *openfeature.Client
	ConfigError  error

	// Configuration testing state
	ProviderOptions []ProviderOption
	ProviderConfig  ErrorAwareProviderConfiguration

	// Evaluation state
	LastEvaluation EvaluationResult
	EvalContext    map[string]interface{}
	TargetingKey   string
	FlagKey        string
	FlagType       string
	DefaultValue   interface{}

	// Event tracking
	EventChannel chan EventRecord // Single channel for all events
	LastEvent    *EventRecord     // Store the last received event for multiple step access

	// Container/testbed state
	Container    TestContainer
	LaunchpadURL string
	FlagDir      string
}

// Configuration-related types

// ProviderOption is a struct to store the defined options between steps
type ProviderOption struct {
	Option    string
	ValueType string
	Value     string
}

// ErrorAwareProviderConfiguration contains a ProviderConfiguration and an error
type ErrorAwareProviderConfiguration struct {
	Configuration *flagd.ProviderConfiguration
	Error         error
}

// Context keys for passing data between steps (legacy support)
type ctxProviderOptionsKey struct{}
type ctxErrorAwareProviderConfigurationKey struct{}

// Container-related interfaces and types

// TestContainer interface abstracts container operations
type TestContainer interface {
	GetHost() string
	GetPort(service string) int
	GetLaunchpadURL() string
	Start() error
	Stop() error
	Restart(delaySeconds int) error
	IsHealthy() bool
}

// FlagdContainerConfig holds configuration for the flagd testbed container
type FlagdContainerConfig struct {
	Image         string
	Tag           string
	Feature       string
	FlagsDir      string
	Networks      []string
	ExtraWaitTime time.Duration
	TestbedDir    string
}

// ContainerInfo provides information about the running container
type ContainerInfo struct {
	ID            string
	Image         string
	Host          string
	RPCPort       int
	InProcessPort int
	LaunchpadPort int
	HealthPort    int
	LaunchpadURL  string
	IsRunning     bool
	IsHealthy     bool
}

// Provider supplier functions

// ProviderSupplier is a function type that creates providers
type ProviderSupplier func(state TestState) (openfeature.FeatureProvider, error)

// Global provider supplier variables
var (
	RPCProviderSupplier       ProviderSupplier
	InProcessProviderSupplier ProviderSupplier
	FileProviderSupplier      ProviderSupplier
)
