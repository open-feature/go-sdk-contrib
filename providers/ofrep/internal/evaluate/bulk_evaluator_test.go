package evaluate

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/open-feature/go-sdk-contrib/providers/ofrep/internal/outbound"
	of "github.com/open-feature/go-sdk/openfeature"
)

func TestBulkSuccess200(t *testing.T) {
	t.Run("success evaluation response", func(t *testing.T) {
		seed := bulkEvaluationSuccess{
			Flags: []bulkEvaluationValue{
				{successDto: successDto{Value: true}, evaluationError: evaluationError{Key: "flag-bool"}},
				{evaluationError: evaluationError{Key: "flag-error", ErrorCode: "GENERAL", ErrorDetails: "something wrong"}},
			},
		}

		successBytes, err := json.Marshal(seed)
		if err != nil {
			t.Fatal(err)
		}

		evalCtx := of.FlattenedContext{}
		resolver := NewBulkEvaluator(mockOutbound{
			rsp: outbound.Resolution{
				Status: http.StatusOK,
				Data:   successBytes,
			},
		}, evalCtx)

		ctx := context.Background()
		err = resolver.Fetch(ctx)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		value, errRes := resolver.resolveSingle(ctx, "flag-bool", evalCtx)
		if errRes != nil {
			t.Fatalf("expected no error, got %v", errRes)
		}
		if !reflect.DeepEqual(value, &seed.Flags[0].successDto) {
			t.Fatalf("expected %v, got %v", seed.Flags[0].successDto, value)
		}

		_, errRes = resolver.resolveSingle(context.Background(), "flag-error", evalCtx)
		if errRes == nil {
			t.Fatal("expected error, but got nil")
		}
		if errRes.Error() != "GENERAL: something wrong" {
			t.Errorf("expected %v, but got %v", "GENERAL: something wrong", errRes.Error())
		}

		_, errRes = resolver.resolveSingle(context.Background(), "flag-unknown", evalCtx)
		if errRes == nil {
			t.Fatalf("expected error, but got nil")
		}
		if errRes.Error() != "FLAG_NOT_FOUND: flag for key 'flag-unknown' does not exist" {
			t.Errorf("expected %v, but got %v", "FLAG_NOT_FOUND: flag fork key 'flag-unknown' does not exist", errRes.Error())
		}
	})
}

func TestBulkErrors(t *testing.T) {
	emptyHeaders := map[string][]string{}
	tests := []struct {
		name             string
		statusCode       int
		headers          http.Header
		expectedErroCode of.ErrorCode
	}{
		{"internal server error", http.StatusInternalServerError, emptyHeaders, of.GeneralCode},
		{"server error", http.StatusServiceUnavailable, emptyHeaders, of.GeneralCode},
		{"bad request", http.StatusBadRequest, emptyHeaders, of.GeneralCode},
		{"unauthorized", http.StatusUnauthorized, emptyHeaders, of.GeneralCode},
		{"forbidden", http.StatusForbidden, emptyHeaders, of.GeneralCode},
		{"too many requests", http.StatusTooManyRequests, emptyHeaders, of.GeneralCode},
		{"too many requests with header", http.StatusTooManyRequests, map[string][]string{"Retry-After": {"100"}}, of.GeneralCode},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver := NewBulkEvaluator(mockOutbound{
				rsp: outbound.Resolution{
					Status:  tt.statusCode,
					Data:    []byte{},
					Headers: map[string][]string{"Retry-After": {"1"}},
				},
			}, of.FlattenedContext{})

			err := resolver.Fetch(context.Background())

			if err == nil {
				t.Fatal("expected non nil error, but got empty")
			}

			if !strings.Contains(err.Error(), string(tt.expectedErroCode)) {
				t.Errorf("expected error to contain error code %v, got %v", tt.expectedErroCode, err.Error())
			}
		})
	}
}
