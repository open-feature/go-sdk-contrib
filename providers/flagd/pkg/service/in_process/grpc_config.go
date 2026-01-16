package process

import (
	"encoding/json"
	"fmt"
	"time"

	"google.golang.org/grpc/codes"
)

const (
	// Default timeouts for keepalive settings
	defaultKeepaliveTime    = 30 * time.Second
	defaultKeepaliveTimeout = 5 * time.Second
	// Default retry intervals per https://flagd.dev/reference/specifications/providers/#configuration
	DefaultRetryBackoffMs    = 1000  // 1 second
	DefaultRetryBackoffMaxMs = 12000 // 12 seconds
)

type RetryPolicy struct {
	MaxAttempts          int      `json:"MaxAttempts"`
	InitialBackoff       string   `json:"InitialBackoff"`
	MaxBackoff           string   `json:"MaxBackoff"`
	BackoffMultiplier    float64  `json:"BackoffMultiplier"`
	RetryableStatusCodes []string `json:"RetryableStatusCodes"`
}

func (g *Sync) buildRetryPolicy() string {
	// Use default values if not configured (per https://flagd.dev/reference/specifications/providers/#configuration)
	initialBackoffMs := g.RetryBackOffMs
	if initialBackoffMs <= 0 {
		initialBackoffMs = DefaultRetryBackoffMs
	}

	maxBackoffMs := g.RetryBackOffMaxMs
	if maxBackoffMs <= 0 {
		maxBackoffMs = DefaultRetryBackoffMaxMs
	}

	// Format durations for gRPC service config (requires seconds-only format like "1s", "0.1s", "120s")
	initialDur := time.Duration(initialBackoffMs) * time.Millisecond
	maxDur := time.Duration(maxBackoffMs) * time.Millisecond

	// Convert to seconds and format as gRPC expects (no compound formats like "2m0s")
	initialBackoff := fmt.Sprint(initialDur.Seconds()) + "s"
	maxBackoff := fmt.Sprint(maxDur.Seconds()) + "s"

	var policy = map[string]interface{}{
		"methodConfig": []map[string]interface{}{
			{
				"name": []map[string]string{
					{"service": "flagd.sync.v1.FlagSyncService"},
				},
				"retryPolicy": RetryPolicy{
					MaxAttempts:          3,
					InitialBackoff:       initialBackoff,
					MaxBackoff:           maxBackoff,
					BackoffMultiplier:    2.0,
					RetryableStatusCodes: []string{"UNKNOWN", "UNAVAILABLE"},
				},
			},
		},
	}
	retryPolicyBytes, _ := json.Marshal(policy)
	retryPolicy := string(retryPolicyBytes)

	return retryPolicy
}

// Set of non-retryable gRPC status codes for faster lookup
var nonRetryableCodes map[codes.Code]struct{}

// initNonRetryableStatusCodesSet initializes the set of non-retryable gRPC status codes for quick lookup
func (g *Sync) initNonRetryableStatusCodesSet() {
	nonRetryableCodes = make(map[codes.Code]struct{})

	for _, codeStr := range g.FatalStatusCodes {
		// Wrap the string in quotes to match the expected JSON format
		jsonStr := fmt.Sprintf(`"%s"`, codeStr)

		var code codes.Code
		if err := code.UnmarshalJSON([]byte(jsonStr)); err != nil {
			g.Logger.Warn(fmt.Sprintf("unknown status code: %s, error: %v", codeStr, err))
			continue
		}

		nonRetryableCodes[code] = struct{}{}
	}
}

