package testframework

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"unicode"

	flagd "github.com/open-feature/go-sdk-contrib/providers/flagd/pkg"
)

// ValueConverter provides unified type conversion for test steps
type ValueConverter struct{}

// NewValueConverter creates a new value converter
func NewValueConverter() *ValueConverter {
	return &ValueConverter{}
}

// ConvertForSteps converts string values to appropriate types for step definitions
// This is the main conversion function used by step definitions
func (vc *ValueConverter) ConvertForSteps(value string, valueType string) (interface{}, error) {
	// Handle empty values (no default value specified)
	if value == "" {
		return vc.getDefaultValue(valueType), nil
	}

	switch valueType {
	case "Boolean":
		return strconv.ParseBool(strings.ToLower(value))
	case "Integer", "Long":
		// Return int64 to match OpenFeature IntValueDetails return type
		return strconv.ParseInt(value, 10, 64)
	case "Float":
		return strconv.ParseFloat(value, 64)
	case "String":
		if value == "null" {
			return nil, nil
		}
		return value, nil
	case "Object":
		var obj interface{}
		err := json.Unmarshal([]byte(value), &obj)
		return obj, err
	case "ResolverType":
		return vc.parseResolverType(value)
	case "CacheType":
		return vc.parseCacheType(value)
	default:
		return value, nil
	}
}

// ConvertToReflectValue converts string values to reflect.Value for configuration
func (vc *ValueConverter) ConvertToReflectValue(valueType, value string, fieldType reflect.Type) reflect.Value {
	switch valueType {
	case "Integer":
		intVal, err := strconv.Atoi(value)
		if err != nil {
			panic(fmt.Errorf("failed to convert %s to integer: %w", value, err))
		}
		return reflect.ValueOf(intVal).Convert(fieldType)
	case "Boolean":
		boolVal, err := strconv.ParseBool(value)
		if err != nil {
			panic(fmt.Errorf("failed to convert %s to boolean: %w", value, err))
		}
		return reflect.ValueOf(boolVal).Convert(fieldType)
	case "ResolverType":
		resolverVal := flagd.ResolverType(strings.ToLower(value))
		return reflect.ValueOf(resolverVal).Convert(fieldType)
	case "Long":
		longVal, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			panic(fmt.Errorf("failed to convert %s to long: %w", value, err))
		}
		return reflect.ValueOf(longVal).Convert(fieldType)
	case "StringList":
		splitVal := strings.Split(value, ",")
		for i, v := range splitVal {
			splitVal[i] = strings.TrimSpace(v)
		}
		return reflect.ValueOf(splitVal).Convert(fieldType)
	default:
		return reflect.ValueOf(value).Convert(fieldType)
	}
}

// getDefaultValue returns the default value for a given type when empty string is provided
func (vc *ValueConverter) getDefaultValue(valueType string) interface{} {
	switch valueType {
	case "Boolean":
		return false
	case "Integer", "Long":
		return int64(0)
	case "Float":
		return 0.0
	case "String":
		return ""
	case "Object":
		return nil
	default:
		return nil
	}
}

// parseResolverType converts string to ProviderType
func (vc *ValueConverter) parseResolverType(value string) (ProviderType, error) {
	switch strings.ToLower(value) {
	case "rpc":
		return RPC, nil
	case "in-process":
		return InProcess, nil
	case "file":
		return File, nil
	default:
		return RPC, fmt.Errorf("unknown resolver type: %s", value)
	}
}

// parseCacheType handles cache type conversion
func (vc *ValueConverter) parseCacheType(value string) (string, error) {
	switch strings.ToLower(value) {
	case "lru", "disabled":
		return strings.ToLower(value), nil
	default:
		return "", fmt.Errorf("unknown cache type: %s", value)
	}
}

// Utility functions for string formatting and naming

// CamelToSnake converts CamelCase to snake_case
func CamelToSnake(input string) string {
	var result strings.Builder
	for i, r := range input {
		if i > 0 && 'A' <= r && r <= 'Z' {
			result.WriteRune('_')
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}

// ToFieldName converts option name to Go field name (capitalize first letter)
func ToFieldName(option string) string {
	r := []rune(option)
	return string(append([]rune{unicode.ToUpper(r[0])}, r[1:]...))
}

// ValueToString converts any value to its string representation
func ValueToString(value interface{}) string {
	return fmt.Sprintf("%v", value)
}

// StringToInt converts string to int with error handling
func StringToInt(str string) int {
	i, err := strconv.Atoi(str)
	if err != nil {
		panic(fmt.Errorf("failed to convert %s to int: %w", str, err))
	}
	return i
}

// StringToBoolean converts string to boolean with error handling
func StringToBoolean(str string) bool {
	b, err := strconv.ParseBool(str)
	if err != nil {
		panic(fmt.Errorf("failed to convert %s to boolean: %w", str, err))
	}
	return b
}

// Global converter instance
var DefaultConverter = NewValueConverter()

// Helper functions for wrapping state methods
func withStateNoArgs(fn func(*TestState, context.Context) error) func(context.Context) error {
	return func(ctx context.Context) error {
		state := GetStateFromContext(ctx)
		if state == nil {
			return fmt.Errorf("test state not found in context")
		}
		return fn(state, ctx)
	}
}

func withState2Args(fn func(*TestState, context.Context, string, string) error) func(context.Context, string, string) error {
	return func(ctx context.Context, arg1, arg2 string) error {
		state := GetStateFromContext(ctx)
		if state == nil {
			return fmt.Errorf("test state not found in context")
		}
		return fn(state, ctx, arg1, arg2)
	}
}

func withState3Args(fn func(*TestState, context.Context, string, string, string) error) func(context.Context, string, string, string) error {
	return func(ctx context.Context, arg1, arg2, arg3 string) error {
		state := GetStateFromContext(ctx)
		if state == nil {
			return fmt.Errorf("test state not found in context")
		}
		return fn(state, ctx, arg1, arg2, arg3)
	}
}

func withState3ArgsReturningContext(fn func(*TestState, context.Context, string, string, string) (context.Context, error)) func(context.Context, string, string, string) (context.Context, error) {
	return func(ctx context.Context, arg1, arg2, arg3 string) (context.Context, error) {
		state := GetStateFromContext(ctx)
		if state == nil {
			return ctx, fmt.Errorf("test state not found in context")
		}
		return fn(state, ctx, arg1, arg2, arg3)
	}
}

func withStateNoArgsReturningContext(fn func(*TestState, context.Context) (context.Context, error)) func(context.Context) (context.Context, error) {
	return func(ctx context.Context) (context.Context, error) {
		state := GetStateFromContext(ctx)
		if state == nil {
			return ctx, fmt.Errorf("test state not found in context")
		}
		return fn(state, ctx)
	}
}

func GetStateFromContext(ctx context.Context) *TestState {
	if state, ok := ctx.Value(TestStateKey{}).(*TestState); ok {
		return state
	}
	return nil
}
