package process

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)


func TestBuildRetryPolicy(t *testing.T) {
	g := &Sync{
		RetryBackOffMs:    100,
		RetryBackOffMaxMs: 500,
	}

	result := g.buildRetryPolicy()

	// Unmarshal to check structure
	var policy map[string]interface{}
	if err := json.Unmarshal([]byte(result), &policy); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	methodConfig, ok := policy["methodConfig"].([]interface{})
	if !ok || len(methodConfig) == 0 {
		t.Fatalf("methodConfig missing or empty")
	}

	config := methodConfig[0].(map[string]interface{})
	retryPolicy, ok := config["retryPolicy"].(map[string]interface{})
	if !ok {
		t.Fatalf("retryPolicy missing")
	}

	if retryPolicy["MaxAttempts"].(float64) != 3 {
		t.Errorf("MaxAttempts = %v; want 3", retryPolicy["MaxAttempts"])
	}
	if retryPolicy["InitialBackoff"].(string) != "100ms" {
		t.Errorf("InitialBackoff = %v; want 100ms", retryPolicy["InitialBackoff"])
	}
	if retryPolicy["MaxBackoff"].(string) != "500ms" {
		t.Errorf("MaxBackoff = %v; want 500ms", retryPolicy["MaxBackoff"])
	}
	if retryPolicy["BackoffMultiplier"].(float64) != 2.0 {
		t.Errorf("BackoffMultiplier = %v; want 2.0", retryPolicy["BackoffMultiplier"])
	}
	codes := retryPolicy["RetryableStatusCodes"].([]interface{})
	expectedCodes := []string{"UNKNOWN", "UNAVAILABLE"}
	for i, code := range expectedCodes {
		if codes[i].(string) != code {
			t.Errorf("RetryableStatusCodes[%d] = %v; want %v", i, codes[i], code)
		}
	}

	// Also check that the result is valid JSON and contains expected substrings
	if !strings.Contains(result, `"MaxAttempts":3`) {
		t.Error("Result does not contain MaxAttempts")
	}
	if !strings.Contains(result, `"InitialBackoff":"100ms"`) {
		t.Error("Result does not contain InitialBackoff")
	}
	if !strings.Contains(result, `"MaxBackoff":"500ms"`) {
		t.Error("Result does not contain MaxBackoff")
	}
	if !strings.Contains(result, `"RetryableStatusCodes":["UNKNOWN","UNAVAILABLE"]`) {
		t.Error("Result does not contain RetryableStatusCodes")
	}
}

type syncTestCase struct {
	input    []string
	expected map[string]struct{}
}

func TestInitNonRetryableStatusCodesSet(t *testing.T) {
	testCases := []syncTestCase{
		{
			input:    []string{"PERMISSION_DENIED", "UNKNOWN"},
			expected: map[string]struct{}{"PermissionDenied": {}, "Unknown": {}},
		},
		{
			input:    []string{"ALREADY_EXISTS"},
			expected: map[string]struct{}{"AlreadyExists": {}},
		},
		{
			input:    []string{},
			expected: map[string]struct{}{},
		},
	}

	for _, tc := range testCases {
		g := &Sync{FatalStatusCodes: tc.input}
		nonRetryableCodes = nil // reset global
		g.initNonRetryableStatusCodesSet()
		if !reflect.DeepEqual(nonRetryableCodes, tc.expected) {
			t.Errorf("input: %v, got: %v, want: %v", tc.input, nonRetryableCodes, tc.expected)
		}
	}
}

func TestToCamelCase(t *testing.T) {
 testCases := []struct {
  input    string
  expected string
 }{
  {"INVALID_ARGUMENT", "InvalidArgument"},
  {"NOT_FOUND", "NotFound"},
  {"ALREADY_EXISTS", "AlreadyExists"},
  {"UNKNOWN", "Unknown"},
  {"", ""},
  {"SINGLE", "Single"},
  {"MULTI_WORD_EXAMPLE", "MultiWordExample"},
  {"_LEADING_UNDERSCORE", "LeadingUnderscore"},
  {"TRAILING_UNDERSCORE_", "TrailingUnderscore"},
  {"__DOUBLE__UNDERSCORES__", "DoubleUnderscores"},
 }

 for _, tc := range testCases {
  got := toCamelCase(tc.input)
  if got != tc.expected {
   t.Errorf("toCamelCase(%q) = %q; want %q", tc.input, got, tc.expected)
  }
 }
}