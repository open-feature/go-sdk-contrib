package process

import (
	"encoding/json"
	"strings"
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
					InitialBackoff:		  (time.Duration(g.RetryBackOffMs) * time.Millisecond).String(),
					MaxBackoff: 		  (time.Duration(g.RetryBackOffMaxMs) * time.Millisecond).String(),
					BackoffMultiplier:    2.0,
					RetryableStatusCodes: []string{"UNKNOWN","UNAVAILABLE"},
				},
			},
		},
	}
	retryPolicyBytes, _ := json.Marshal(policy)
	retryPolicy := string(retryPolicyBytes)

	return retryPolicy
}

// Set of non-retryable gRPC status codes for faster lookup
var nonRetryableCodes map[string]struct{}

// initNonRetryableStatusCodesSet initializes the set of non-retryable gRPC status codes for quick lookup
func (g *Sync) initNonRetryableStatusCodesSet()  {
	nonRetryableCodes = make(map[string]struct{})
	for _, code := range g.FatalStatusCodes {
		normalized := toCamelCase(code)
		nonRetryableCodes[normalized] = struct{}{}
	}
}

// toCamelCase converts a SNAKE_CASE string to CamelCase
func toCamelCase(s string) string {
	parts := strings.Split(strings.ToLower(s), "_")
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[:1]) + part[1:]
		}
	}
	return strings.Join(parts, "")
}