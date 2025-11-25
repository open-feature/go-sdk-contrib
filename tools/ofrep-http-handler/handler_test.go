package ofrephandler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	ofrephandler "github.com/open-feature/go-sdk-contrib/tools/ofrep-http-handler"
)

func TestHandler_ServeHTTP(t *testing.T) {
	flags := map[string]mockValue{
		"test-bool-flag":   {value: true},
		"test-string-flag": {value: "hello"},
		"test-ctx-flag": {
			value:      "context-value",
			requireCtx: map[string]any{"userId": "456"},
		},
	}

	tests := []struct {
		name         string
		method       string
		path         string
		body         any
		expectedCode int
		expectedType string
		options      []ofrephandler.Option
	}{
		{
			name:         "successful boolean flag evaluation",
			method:       "POST",
			path:         "/ofrep/v1/evaluate/flags/test-bool-flag",
			expectedCode: http.StatusOK,
			expectedType: "bool",
		},
		{
			name:         "successful string flag evaluation",
			method:       "POST",
			path:         "/ofrep/v1/evaluate/flags/test-string-flag",
			expectedCode: http.StatusOK,
			expectedType: "string",
		},
		{
			name:         "successful matching context for flag",
			method:       "POST",
			path:         "/ofrep/v1/evaluate/flags/test-ctx-flag",
			body:         ofrephandler.EvaluationRequest{Context: map[string]any{"userId": "456"}},
			expectedCode: http.StatusOK,
		},
		{
			name:         "fails non-matching context for flag",
			method:       "POST",
			path:         "/ofrep/v1/evaluate/flags/test-ctx-flag",
			body:         ofrephandler.EvaluationRequest{Context: map[string]any{"userId": "123"}},
			expectedCode: http.StatusNotFound,
		},
		{
			name:         "successful flag evaluation on non-post with option",
			method:       "GET",
			path:         "/ofrep/v1/evaluate/flags/test-bool-flag",
			expectedCode: http.StatusOK,
			expectedType: "bool",
			options:      []ofrephandler.Option{ofrephandler.WithoutOnlyPOST()},
		},
		{
			name:         "successful flag evaluation on non-prefix path with option",
			method:       "POST",
			path:         "/test-bool-flag",
			expectedCode: http.StatusOK,
			expectedType: "bool",
			options:      []ofrephandler.Option{ofrephandler.WithoutPathPrefix()},
		},
		{
			name:         "flag not found",
			method:       "POST",
			path:         "/ofrep/v1/evaluate/flags/nonexistent-flag",
			expectedCode: http.StatusNotFound,
		},
		{
			name:         "method not allowed",
			method:       "GET",
			path:         "/ofrep/v1/evaluate/flags/test-bool-flag",
			expectedCode: http.StatusMethodNotAllowed,
		},
		{
			name:         "invalid path",
			method:       "POST",
			path:         "/invalid/path",
			expectedCode: http.StatusNotFound,
		},
		{
			name:         "empty flag key",
			method:       "POST",
			path:         "/ofrep/v1/evaluate/flags/",
			expectedCode: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body bytes.Buffer
			if tt.body != nil {
				json.NewEncoder(&body).Encode(tt.body)
			}

			req := httptest.NewRequest(tt.method, tt.path, &body)
			req.SetPathValue("key", strings.TrimPrefix(strings.TrimPrefix(tt.path, "/ofrep/v1/evaluate/flags/"), "/"))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			ofrephandler.New(newMockProvider(flags), tt.options...).ServeHTTP(w, req)

			if w.Code != tt.expectedCode {
				t.Errorf("expected status code %d, got %d", tt.expectedCode, w.Code)
			}

			if tt.expectedCode == http.StatusOK {
				var response ofrephandler.EvaluationResponse
				err := json.NewDecoder(w.Body).Decode(&response)
				if err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}

				switch tt.expectedType {
				case "bool":
					if _, ok := response.Value.(bool); !ok {
						t.Errorf("expected boolean value, got %T", response.Value)
					}
				case "string":
					if _, ok := response.Value.(string); !ok {
						t.Errorf("expected string value, got %T", response.Value)
					}
				}

				if response.Key == "" {
					t.Error("expected key in response")
				}
			}
		})
	}
}
