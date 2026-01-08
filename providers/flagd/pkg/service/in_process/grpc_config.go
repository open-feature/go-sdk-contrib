package process

import (
	"encoding/json"
	"fmt"
	"google.golang.org/grpc/codes"
	"time"
)

const (
	// Default timeouts and retry intervals
	defaultKeepaliveTime    = 30 * time.Second
	defaultKeepaliveTimeout = 5 * time.Second
)

type RetryPolicy struct {
	MaxAttempts          int      `json:"MaxAttempts"`
	InitialBackoff       string   `json:"InitialBackoff"`
	MaxBackoff           string   `json:"MaxBackoff"`
	BackoffMultiplier    float64  `json:"BackoffMultiplier"`
	RetryableStatusCodes []string `json:"RetryableStatusCodes"`
}

func (g *Sync) buildRetryPolicy() string {
	var policy = map[string]interface{}{
		"methodConfig": []map[string]interface{}{
			{
				"name": []map[string]string{
					{"service": "flagd.sync.v1.FlagSyncService"},
				},
				"retryPolicy": RetryPolicy{
					MaxAttempts:          3,
					InitialBackoff:       (time.Duration(g.RetryBackOffMs) * time.Millisecond).String(),
					MaxBackoff:           (time.Duration(g.RetryBackOffMaxMs) * time.Millisecond).String(),
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
