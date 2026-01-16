package process

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/open-feature/flagd/core/pkg/logger"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
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
	if retryPolicy["InitialBackoff"].(string) != "0.1s" {
		t.Errorf("InitialBackoff = %v; want 0.1s", retryPolicy["InitialBackoff"])
	}
	if retryPolicy["MaxBackoff"].(string) != "0.5s" {
		t.Errorf("MaxBackoff = %v; want 0.5s", retryPolicy["MaxBackoff"])
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
	if !strings.Contains(result, `"InitialBackoff":"0.1s"`) {
		t.Error("Result does not contain InitialBackoff")
	}
	if !strings.Contains(result, `"MaxBackoff":"0.5s"`) {
		t.Error("Result does not contain MaxBackoff")
	}
	if !strings.Contains(result, `"RetryableStatusCodes":["UNKNOWN","UNAVAILABLE"]`) {
		t.Error("Result does not contain RetryableStatusCodes")
	}
}

// TestBuildRetryPolicyDefaults verifies that default values are applied per spec
func TestBuildRetryPolicyDefaults(t *testing.T) {
	g := &Sync{
		RetryBackOffMs:    0, // Should use default 1000ms
		RetryBackOffMaxMs: 0, // Should use default 120000ms
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

	// time.Duration.String() formats milliseconds as "1s", "120s", etc.
	if retryPolicy["InitialBackoff"].(string) != "1s" {
		t.Errorf("InitialBackoff = %v; want 1s (default)", retryPolicy["InitialBackoff"])
	}
	if retryPolicy["MaxBackoff"].(string) != "12s" {
		t.Errorf("MaxBackoff = %v; want 120s (gRPC format for 120000ms)", retryPolicy["MaxBackoff"])
	}
}

// TestBuildRetryPolicyFractionalSeconds verifies fractional second durations are formatted correctly
func TestBuildRetryPolicyFractionalSeconds(t *testing.T) {
	g := &Sync{
		RetryBackOffMs:    1500, // 1.5 seconds
		RetryBackOffMaxMs: 2500, // 2.5 seconds
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

	// Verify fractional seconds are preserved in gRPC format
	if retryPolicy["InitialBackoff"].(string) != "1.5s" {
		t.Errorf("InitialBackoff = %v; want 1.5s", retryPolicy["InitialBackoff"])
	}
	if retryPolicy["MaxBackoff"].(string) != "2.5s" {
		t.Errorf("MaxBackoff = %v; want 2.5s", retryPolicy["MaxBackoff"])
	}
}

func TestSync_initNonRetryableStatusCodesSet(t *testing.T) {
	tests := []struct {
		name             string
		fatalStatusCodes []string
		expectedCodes    []codes.Code
		notExpectedCodes []codes.Code
	}{
		{
			name:             "valid status codes",
			fatalStatusCodes: []string{"UNAVAILABLE", "INTERNAL", "DEADLINE_EXCEEDED"},
			expectedCodes:    []codes.Code{codes.Unavailable, codes.Internal, codes.DeadlineExceeded},
			notExpectedCodes: []codes.Code{codes.OK, codes.Unknown},
		},
		{
			name:             "empty array",
			fatalStatusCodes: []string{},
			expectedCodes:    []codes.Code{},
			notExpectedCodes: []codes.Code{codes.Unavailable, codes.Internal},
		},
		{
			name:             "invalid status codes",
			fatalStatusCodes: []string{"INVALID_CODE", "UNKNOWN_STATUS"},
			expectedCodes:    []codes.Code{},
			notExpectedCodes: []codes.Code{codes.Unavailable, codes.Internal},
		},
		{
			name:             "mixed valid and invalid codes",
			fatalStatusCodes: []string{"UNAVAILABLE", "INVALID_CODE", "INTERNAL"},
			expectedCodes:    []codes.Code{codes.Unavailable, codes.Internal},
			notExpectedCodes: []codes.Code{codes.OK, codes.Unknown},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset the global map before each test
			nonRetryableCodes = nil

			s := &Sync{
				FatalStatusCodes: tt.fatalStatusCodes,
				Logger: &logger.Logger{
					Logger: zap.NewNop(),
				},
			}

			s.initNonRetryableStatusCodesSet()

			// Verify expected codes are present
			for _, code := range tt.expectedCodes {
				if _, exists := nonRetryableCodes[code]; !exists {
					t.Errorf("expected code %v to be in nonRetryableCodes, but it was not found", code)
				}
			}

			// Verify not expected codes are absent
			for _, code := range tt.notExpectedCodes {
				if _, exists := nonRetryableCodes[code]; exists {
					t.Errorf("did not expect code %v to be in nonRetryableCodes, but it was found", code)
				}
			}

			// Verify the map size matches expected
			if len(nonRetryableCodes) != len(tt.expectedCodes) {
				t.Errorf("expected map size %d, got %d", len(tt.expectedCodes), len(nonRetryableCodes))
			}
		})
	}
}
