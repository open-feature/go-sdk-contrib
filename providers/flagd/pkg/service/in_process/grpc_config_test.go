package process

import (
	"github.com/open-feature/flagd/core/pkg/logger"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"testing"
)

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
