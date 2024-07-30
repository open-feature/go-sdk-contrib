package provider_v2_test

import (
	"bytes"
	"context"
	"fmt"
	"github.com/open-feature/go-sdk-contrib/providers/go-feature-flag/pkg/provider_v2"
	of "github.com/open-feature/go-sdk/openfeature"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

// MockRoundTripper is a mock implementation of http.RoundTripper.
type MockRoundTripper struct {
	RoundTripFunc func(req *http.Request) *http.Response
}

func (m *MockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.RoundTripFunc(req), nil
}

// NewMockClient creates a new http.Client with the mock RoundTripper.
func NewMockClient(roundTripFunc func(req *http.Request) *http.Response) *http.Client {
	return &http.Client{
		Transport: &MockRoundTripper{RoundTripFunc: roundTripFunc},
	}
}

type mockClient struct {
	callCount           int
	collectorCallCount  int
	flagChangeCallCount int
}

func (m *mockClient) roundTripFunc(req *http.Request) *http.Response {
	if req.URL.Path == "/v1/data/collector" {
		m.collectorCallCount++
		return &http.Response{
			StatusCode: http.StatusOK,
		}
	}

	if req.URL.Path == "/v1/flag/change" {
		m.flagChangeCallCount++
		if req.Header.Get("If-None-Match") == "123456" {
			resp := &http.Response{
				StatusCode: http.StatusOK,
				Header:     map[string][]string{},
			}
			resp.Header.Set("ETag", "78910")
			return resp
		}
		resp := &http.Response{
			StatusCode: http.StatusOK,
			Header:     map[string][]string{},
		}
		resp.Header.Set("ETag", "123456")
		return resp
	}

	m.callCount++
	mockPath := "../../testutils/mock_responses/%s.json"
	flagName := strings.Replace(req.URL.Path, "/ofrep/v1/evaluate/flags/", "", -1)

	if flagName == "unauthorized" {
		return &http.Response{
			StatusCode: http.StatusUnauthorized,
			Body:       io.NopCloser(bytes.NewReader([]byte(""))),
		}
	}

	content, err := os.ReadFile(fmt.Sprintf(mockPath, flagName))
	if err != nil {
		content, _ = os.ReadFile(fmt.Sprintf(mockPath, "flag_not_found"))
	}
	statusCode := http.StatusOK

	if strings.Contains(string(content), "errorCode") {
		statusCode = http.StatusBadRequest
	}
	if strings.Contains(string(content), "FLAG_NOT_FOUND") {
		statusCode = http.StatusNotFound
	}

	body := io.NopCloser(bytes.NewReader(content))
	return &http.Response{
		StatusCode: statusCode,
		Body:       body,
	}
}

func defaultEvaluationCtx() of.EvaluationContext {
	return of.NewEvaluationContext(
		"d45e303a-38c2-11ed-a261-0242ac120002",
		map[string]interface{}{
			"email":        "john.doe@gofeatureflag.org",
			"firstname":    "john",
			"lastname":     "doe",
			"anonymous":    false,
			"professional": true,
			"rate":         3.14,
			"age":          30,
			"admin":        true,
			"company_info": map[string]interface{}{
				"name": "my_company",
				"size": 120,
			},
			"labels": []string{
				"pro", "beta",
			},
		},
	)
}

func TestProvider_BooleanEvaluation(t *testing.T) {
	type args struct {
		flag         string
		defaultValue bool
		evalCtx      of.EvaluationContext
	}
	tests := []struct {
		name string
		args args
		want of.BooleanEvaluationDetails
	}{
		{
			name: "unauthorized flag",
			args: args{
				flag:         "unauthorized",
				defaultValue: false,
				evalCtx:      defaultEvaluationCtx(),
			},
			want: of.BooleanEvaluationDetails{
				Value: false,
				EvaluationDetails: of.EvaluationDetails{
					FlagKey:  "unauthorized",
					FlagType: of.Boolean,
					ResolutionDetail: of.ResolutionDetail{
						Variant:      "",
						Reason:       of.ErrorReason,
						ErrorCode:    of.GeneralCode,
						ErrorMessage: "authentication/authorization error",
						FlagMetadata: map[string]interface{}{},
					},
				},
			},
		},
		{
			name: "should resolve a valid boolean flag with TARGETING_MATCH reason",
			args: args{
				flag:         "bool_targeting_match",
				defaultValue: false,
				evalCtx:      defaultEvaluationCtx(),
			},
			want: of.BooleanEvaluationDetails{
				Value: true,
				EvaluationDetails: of.EvaluationDetails{
					FlagKey:  "bool_targeting_match",
					FlagType: of.Boolean,
					ResolutionDetail: of.ResolutionDetail{
						Variant:      "True",
						Reason:       of.TargetingMatchReason,
						ErrorCode:    "",
						ErrorMessage: "",
						FlagMetadata: map[string]interface{}{
							"gofeatureflag_cacheable": true,
						},
					},
				},
			},
		},
		{
			name: "should use boolean default value if the flag is disabled",
			args: args{
				flag:         "disabled_bool",
				defaultValue: false,
				evalCtx:      defaultEvaluationCtx(),
			},
			want: of.BooleanEvaluationDetails{
				Value: false,
				EvaluationDetails: of.EvaluationDetails{
					FlagKey:  "disabled_bool",
					FlagType: of.Boolean,
					ResolutionDetail: of.ResolutionDetail{
						Variant:      "SdkDefault",
						Reason:       of.DisabledReason,
						ErrorCode:    "",
						ErrorMessage: "",
						FlagMetadata: map[string]interface{}{},
					},
				},
			},
		},
		{
			name: "should error if we expect a boolean and got another type",
			args: args{
				flag:         "string_key",
				defaultValue: false,
				evalCtx:      defaultEvaluationCtx(),
			},
			want: of.BooleanEvaluationDetails{
				Value: false,
				EvaluationDetails: of.EvaluationDetails{
					FlagKey:  "string_key",
					FlagType: of.Boolean,
					ResolutionDetail: of.ResolutionDetail{
						Variant:      "",
						Reason:       of.ErrorReason,
						ErrorCode:    of.TypeMismatchCode,
						ErrorMessage: "resolved value CC0000 is not of boolean type",
						FlagMetadata: map[string]interface{}{},
					},
				},
			},
		},
		{
			name: "should error if flag does not exists",
			args: args{
				flag:         "does_not_exists",
				defaultValue: false,
				evalCtx:      defaultEvaluationCtx(),
			},
			want: of.BooleanEvaluationDetails{
				Value: false,
				EvaluationDetails: of.EvaluationDetails{
					FlagKey:  "does_not_exists",
					FlagType: of.Boolean,
					ResolutionDetail: of.ResolutionDetail{
						Variant:      "",
						Reason:       of.ErrorReason,
						ErrorCode:    of.FlagNotFoundCode,
						ErrorMessage: "flag for key 'does_not_exists' does not exist",
						FlagMetadata: map[string]interface{}{},
					},
				},
			},
		},
		{
			name: "should return custom reason if returned by relay proxy",
			args: args{
				flag:         "unknown_reason",
				defaultValue: false,
				evalCtx:      defaultEvaluationCtx(),
			},
			want: of.BooleanEvaluationDetails{
				Value: true,
				EvaluationDetails: of.EvaluationDetails{
					FlagKey:  "unknown_reason",
					FlagType: of.Boolean,
					ResolutionDetail: of.ResolutionDetail{
						Variant:      "True",
						Reason:       "CUSTOM_REASON",
						ErrorCode:    "",
						ErrorMessage: "",
						FlagMetadata: map[string]interface{}{},
					},
				},
			},
		},
		{
			name: "should return error if no targeting key",
			args: args{
				flag:         "bool_targeting_match",
				defaultValue: false,
				evalCtx:      of.EvaluationContext{},
			},
			want: of.BooleanEvaluationDetails{
				Value: false,
				EvaluationDetails: of.EvaluationDetails{
					FlagKey:  "bool_targeting_match",
					FlagType: of.Boolean,
					ResolutionDetail: of.ResolutionDetail{
						Reason:       of.ErrorReason,
						ErrorCode:    of.TargetingKeyMissingCode,
						ErrorMessage: "no targetingKey provided in the evaluation context",
						FlagMetadata: map[string]interface{}{},
					},
				},
			},
		},
		{
			name: "should return an error if invalid json body",
			args: args{
				flag:         "invalid_json_body",
				defaultValue: false,
				evalCtx:      defaultEvaluationCtx(),
			},
			want: of.BooleanEvaluationDetails{
				Value: false,
				EvaluationDetails: of.EvaluationDetails{
					FlagKey:  "invalid_json_body",
					FlagType: of.Boolean,
					ResolutionDetail: of.ResolutionDetail{
						Reason:       of.ErrorReason,
						ErrorCode:    of.ParseErrorCode,
						ErrorMessage: "error parsing the response: unexpected end of JSON input",
						FlagMetadata: map[string]interface{}{},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		cli := mockClient{}
		t.Run(tt.name, func(t *testing.T) {
			options := provider_v2.ProviderOptions{
				Endpoint:            "https://gofeatureflag.org/",
				HTTPClient:          NewMockClient(cli.roundTripFunc),
				GOFeatureFlagConfig: nil,
				DisableCache:        true,
			}
			provider, err := provider_v2.NewProvider(options)
			assert.NoError(t, err)

			err = of.SetProviderAndWait(provider)
			require.NoError(t, err)
			client := of.NewClient("test-app")
			value, err := client.BooleanValueDetails(context.TODO(), tt.args.flag, tt.args.defaultValue, tt.args.evalCtx)

			if tt.want.ErrorCode != "" {
				require.Error(t, err)
				want := fmt.Sprintf("error code: %s: %s", tt.want.ErrorCode, tt.want.ErrorMessage)
				assert.Equal(t, want, err.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.want, value)
		})
	}
}

func TestProvider_StringEvaluation(t *testing.T) {
	type args struct {
		flag         string
		defaultValue string
		evalCtx      of.EvaluationContext
	}
	tests := []struct {
		name string
		args args
		want of.StringEvaluationDetails
	}{
		{
			name: "should resolve a valid string flag with TARGETING_MATCH reason",
			args: args{
				flag:         "string_key",
				defaultValue: "default",
				evalCtx:      defaultEvaluationCtx(),
			},
			want: of.StringEvaluationDetails{
				Value: "CC0000",
				EvaluationDetails: of.EvaluationDetails{
					FlagKey:  "string_key",
					FlagType: of.String,
					ResolutionDetail: of.ResolutionDetail{
						Variant:      "True",
						Reason:       of.TargetingMatchReason,
						ErrorCode:    "",
						ErrorMessage: "",
						FlagMetadata: map[string]interface{}{},
					},
				},
			},
		},
		{
			name: "should use string default value if the flag is disabled",
			args: args{
				flag:         "disabled_string",
				defaultValue: "default",
				evalCtx:      defaultEvaluationCtx(),
			},
			want: of.StringEvaluationDetails{
				Value: "default",
				EvaluationDetails: of.EvaluationDetails{
					FlagKey:  "disabled_string",
					FlagType: of.String,
					ResolutionDetail: of.ResolutionDetail{
						Variant:      "SdkDefault",
						Reason:       of.DisabledReason,
						ErrorCode:    "",
						ErrorMessage: "",
						FlagMetadata: map[string]interface{}{},
					},
				},
			},
		},
		{
			name: "should error if we expect a string and got another type",
			args: args{
				flag:         "bool_targeting_match",
				defaultValue: "default",
				evalCtx:      defaultEvaluationCtx(),
			},
			want: of.StringEvaluationDetails{
				Value: "default",
				EvaluationDetails: of.EvaluationDetails{
					FlagKey:  "bool_targeting_match",
					FlagType: of.String,
					ResolutionDetail: of.ResolutionDetail{
						Variant:      "",
						Reason:       of.ErrorReason,
						ErrorCode:    of.TypeMismatchCode,
						ErrorMessage: "resolved value true is not of string type",
						FlagMetadata: map[string]interface{}{},
					},
				},
			},
		},
		{
			name: "should error if flag does not exists",
			args: args{
				flag:         "does_not_exists",
				defaultValue: "default",
				evalCtx:      defaultEvaluationCtx(),
			},
			want: of.StringEvaluationDetails{
				Value: "default",
				EvaluationDetails: of.EvaluationDetails{
					FlagKey:  "does_not_exists",
					FlagType: of.String,
					ResolutionDetail: of.ResolutionDetail{
						Variant:      "",
						Reason:       of.ErrorReason,
						ErrorCode:    of.FlagNotFoundCode,
						ErrorMessage: "flag for key 'does_not_exists' does not exist",
						FlagMetadata: map[string]interface{}{},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cli := mockClient{}
			options := provider_v2.ProviderOptions{
				Endpoint:            "https://gofeatureflag.org/",
				HTTPClient:          NewMockClient(cli.roundTripFunc),
				GOFeatureFlagConfig: nil,
				DisableCache:        true,
			}
			provider, err := provider_v2.NewProvider(options)
			assert.NoError(t, err)

			err = of.SetProviderAndWait(provider)
			assert.NoError(t, err)
			client := of.NewClient("test-app")
			value, err := client.StringValueDetails(context.TODO(), tt.args.flag, tt.args.defaultValue, tt.args.evalCtx)

			if tt.want.ErrorCode != "" {
				assert.Error(t, err)
				want := fmt.Sprintf("error code: %s: %s", tt.want.ErrorCode, tt.want.ErrorMessage)
				assert.Equal(t, want, err.Error())
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.want, value)
		})
	}
}

func TestProvider_FloatEvaluation(t *testing.T) {
	type args struct {
		flag         string
		defaultValue float64
		evalCtx      of.EvaluationContext
	}
	tests := []struct {
		name string
		args args
		want of.FloatEvaluationDetails
	}{
		{
			name: "should resolve a valid float flag with TARGETING_MATCH reason",
			args: args{
				flag:         "double_key",
				defaultValue: 123.45,
				evalCtx:      defaultEvaluationCtx(),
			},
			want: of.FloatEvaluationDetails{
				Value: 100.25,
				EvaluationDetails: of.EvaluationDetails{
					FlagKey:  "double_key",
					FlagType: of.Float,
					ResolutionDetail: of.ResolutionDetail{
						Variant:      "True",
						Reason:       of.TargetingMatchReason,
						ErrorCode:    "",
						ErrorMessage: "",
						FlagMetadata: map[string]interface{}{},
					},
				},
			},
		},
		{
			name: "should use float default value if the flag is disabled",
			args: args{
				flag:         "disabled_float",
				defaultValue: 123.45,
				evalCtx:      defaultEvaluationCtx(),
			},
			want: of.FloatEvaluationDetails{
				Value: 123.45,
				EvaluationDetails: of.EvaluationDetails{
					FlagKey:  "disabled_float",
					FlagType: of.Float,
					ResolutionDetail: of.ResolutionDetail{
						Variant:      "SdkDefault",
						Reason:       of.DisabledReason,
						ErrorCode:    "",
						ErrorMessage: "",
						FlagMetadata: map[string]interface{}{},
					},
				},
			},
		},
		{
			name: "should error if we expect a string and got another type",
			args: args{
				flag:         "bool_targeting_match",
				defaultValue: 123.45,
				evalCtx:      defaultEvaluationCtx(),
			},
			want: of.FloatEvaluationDetails{
				Value: 123.45,
				EvaluationDetails: of.EvaluationDetails{
					FlagKey:  "bool_targeting_match",
					FlagType: of.Float,
					ResolutionDetail: of.ResolutionDetail{
						Variant:      "",
						Reason:       of.ErrorReason,
						ErrorCode:    of.TypeMismatchCode,
						ErrorMessage: "resolved value true is not of float type",
						FlagMetadata: map[string]interface{}{},
					},
				},
			},
		},
		{
			name: "should error if flag does not exists",
			args: args{
				flag:         "does_not_exists",
				defaultValue: 123.45,
				evalCtx:      defaultEvaluationCtx(),
			},
			want: of.FloatEvaluationDetails{
				Value: 123.45,
				EvaluationDetails: of.EvaluationDetails{
					FlagKey:  "does_not_exists",
					FlagType: of.Float,
					ResolutionDetail: of.ResolutionDetail{
						Variant:      "",
						Reason:       of.ErrorReason,
						ErrorCode:    of.FlagNotFoundCode,
						ErrorMessage: "flag for key 'does_not_exists' does not exist",
						FlagMetadata: map[string]interface{}{},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cli := mockClient{}
			options := provider_v2.ProviderOptions{
				Endpoint:            "https://gofeatureflag.org/",
				HTTPClient:          NewMockClient(cli.roundTripFunc),
				GOFeatureFlagConfig: nil,
				DisableCache:        true,
			}
			provider, err := provider_v2.NewProvider(options)
			assert.NoError(t, err)

			err = of.SetProviderAndWait(provider)
			assert.NoError(t, err)
			client := of.NewClient("test-app")
			value, err := client.FloatValueDetails(context.TODO(), tt.args.flag, tt.args.defaultValue, tt.args.evalCtx)

			if tt.want.ErrorCode != "" {
				assert.Error(t, err)
				want := fmt.Sprintf("error code: %s: %s", tt.want.ErrorCode, tt.want.ErrorMessage)
				assert.Equal(t, want, err.Error())
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.want, value)
		})
	}
}

func TestProvider_IntEvaluation(t *testing.T) {
	type args struct {
		flag         string
		defaultValue int64
		evalCtx      of.EvaluationContext
	}
	tests := []struct {
		name string
		args args
		want of.IntEvaluationDetails
	}{
		{
			name: "should resolve a valid float flag with TARGETING_MATCH reason",
			args: args{
				flag:         "integer_key",
				defaultValue: 123,
				evalCtx:      defaultEvaluationCtx(),
			},
			want: of.IntEvaluationDetails{
				Value: 100,
				EvaluationDetails: of.EvaluationDetails{
					FlagKey:  "integer_key",
					FlagType: of.Int,
					ResolutionDetail: of.ResolutionDetail{
						Variant:      "True",
						Reason:       of.TargetingMatchReason,
						ErrorCode:    "",
						ErrorMessage: "",
						FlagMetadata: map[string]interface{}{},
					},
				},
			},
		},
		{
			name: "should use float default value if the flag is disabled",
			args: args{
				flag:         "disabled_int",
				defaultValue: 123,
				evalCtx:      defaultEvaluationCtx(),
			},
			want: of.IntEvaluationDetails{
				Value: 123,
				EvaluationDetails: of.EvaluationDetails{
					FlagKey:  "disabled_int",
					FlagType: of.Int,
					ResolutionDetail: of.ResolutionDetail{
						Variant:      "SdkDefault",
						Reason:       of.DisabledReason,
						ErrorCode:    "",
						ErrorMessage: "",
						FlagMetadata: map[string]interface{}{},
					},
				},
			},
		},
		{
			name: "should error if we expect a string and got another type",
			args: args{
				flag:         "bool_targeting_match",
				defaultValue: 123,
				evalCtx:      defaultEvaluationCtx(),
			},
			want: of.IntEvaluationDetails{
				Value: 123,
				EvaluationDetails: of.EvaluationDetails{
					FlagKey:  "bool_targeting_match",
					FlagType: of.Int,
					ResolutionDetail: of.ResolutionDetail{
						Variant:      "",
						Reason:       of.ErrorReason,
						ErrorCode:    of.TypeMismatchCode,
						ErrorMessage: "resolved value true is not of integer type",
						FlagMetadata: map[string]interface{}{},
					},
				},
			},
		},
		{
			name: "should error if flag does not exists",
			args: args{
				flag:         "does_not_exists",
				defaultValue: 123,
				evalCtx:      defaultEvaluationCtx(),
			},
			want: of.IntEvaluationDetails{
				Value: 123,
				EvaluationDetails: of.EvaluationDetails{
					FlagKey:  "does_not_exists",
					FlagType: of.Int,
					ResolutionDetail: of.ResolutionDetail{
						Variant:      "",
						Reason:       of.ErrorReason,
						ErrorCode:    of.FlagNotFoundCode,
						ErrorMessage: "flag for key 'does_not_exists' does not exist",
						FlagMetadata: map[string]interface{}{},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cli := mockClient{}
			options := provider_v2.ProviderOptions{
				Endpoint:            "https://gofeatureflag.org/",
				HTTPClient:          NewMockClient(cli.roundTripFunc),
				GOFeatureFlagConfig: nil,
				DisableCache:        true,
			}
			provider, err := provider_v2.NewProvider(options)
			assert.NoError(t, err)

			err = of.SetProviderAndWait(provider)
			assert.NoError(t, err)
			client := of.NewClient("test-app")
			value, err := client.IntValueDetails(context.TODO(), tt.args.flag, tt.args.defaultValue, tt.args.evalCtx)

			if tt.want.ErrorCode != "" {
				assert.Error(t, err)
				want := fmt.Sprintf("error code: %s: %s", tt.want.ErrorCode, tt.want.ErrorMessage)
				assert.Equal(t, want, err.Error())
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.want, value)
		})
	}
}

func TestProvider_ObjectEvaluation(t *testing.T) {
	type args struct {
		flag         string
		defaultValue interface{}
		evalCtx      of.EvaluationContext
	}
	tests := []struct {
		name string
		args args
		want of.InterfaceEvaluationDetails
	}{
		{
			name: "should resolve a valid interface flag with TARGETING_MATCH reason",
			args: args{
				flag:         "object_key",
				defaultValue: nil,
				evalCtx:      defaultEvaluationCtx(),
			},
			want: of.InterfaceEvaluationDetails{
				Value: map[string]interface{}{
					"test":  "test1",
					"test2": false,
					"test3": 123.3,
					"test4": float64(1),
					"test5": nil,
				},
				EvaluationDetails: of.EvaluationDetails{
					FlagKey:  "object_key",
					FlagType: of.Object,
					ResolutionDetail: of.ResolutionDetail{
						Variant:      "True",
						Reason:       of.TargetingMatchReason,
						ErrorCode:    "",
						ErrorMessage: "",
						FlagMetadata: map[string]interface{}{},
					},
				},
			},
		},
		{
			name: "should use interface default value if the flag is disabled",
			args: args{
				flag:         "disabled_int",
				defaultValue: nil,
				evalCtx:      defaultEvaluationCtx(),
			},
			want: of.InterfaceEvaluationDetails{
				Value: nil,
				EvaluationDetails: of.EvaluationDetails{
					FlagKey:  "disabled_int",
					FlagType: of.Object,
					ResolutionDetail: of.ResolutionDetail{
						Variant:      "SdkDefault",
						Reason:       of.DisabledReason,
						ErrorCode:    "",
						ErrorMessage: "",
						FlagMetadata: map[string]interface{}{},
					},
				},
			},
		},
		{
			name: "should error if flag does not exists",
			args: args{
				flag:         "does_not_exists",
				defaultValue: nil,
				evalCtx:      defaultEvaluationCtx(),
			},
			want: of.InterfaceEvaluationDetails{
				Value: nil,
				EvaluationDetails: of.EvaluationDetails{
					FlagKey:  "does_not_exists",
					FlagType: of.Object,
					ResolutionDetail: of.ResolutionDetail{
						Variant:      "",
						Reason:       of.ErrorReason,
						ErrorCode:    of.FlagNotFoundCode,
						ErrorMessage: "flag for key 'does_not_exists' does not exist",
						FlagMetadata: map[string]interface{}{},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cli := mockClient{}
			options := provider_v2.ProviderOptions{
				Endpoint:            "https://gofeatureflag.org/",
				HTTPClient:          NewMockClient(cli.roundTripFunc),
				GOFeatureFlagConfig: nil,
				DisableCache:        true,
			}
			provider, err := provider_v2.NewProvider(options)
			assert.NoError(t, err)

			err = of.SetProviderAndWait(provider)
			assert.NoError(t, err)
			client := of.NewClient("test-app")
			value, err := client.ObjectValueDetails(context.TODO(), tt.args.flag, tt.args.defaultValue, tt.args.evalCtx)

			if tt.want.ErrorCode != "" {
				require.Error(t, err)
				want := fmt.Sprintf("error code: %s: %s", tt.want.ErrorCode, tt.want.ErrorMessage)
				assert.Equal(t, want, err.Error())
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.want, value)
		})
	}
}

func TestProvider_Cache(t *testing.T) {
	t.Run("Call flag multiple times with the same user", func(t *testing.T) {
		cli := mockClient{}
		options := provider_v2.ProviderOptions{
			Endpoint:            "https://gofeatureflag.org/",
			HTTPClient:          NewMockClient(cli.roundTripFunc),
			GOFeatureFlagConfig: nil,
			DisableCache:        false,
			FlagCacheTTL:        5 * time.Minute,
			FlagCacheSize:       5,
		}

		provider, err := provider_v2.NewProvider(options)
		defer provider.Shutdown()
		assert.NoError(t, err)

		err = of.SetProviderAndWait(provider)
		assert.NoError(t, err)
		client := of.NewClient("test-app")
		got1, err := client.BooleanValueDetails(context.TODO(), "bool_targeting_match", false, defaultEvaluationCtx())
		assert.NoError(t, err)
		assert.Equal(t, got1.Reason, of.TargetingMatchReason)
		got2, err := client.BooleanValueDetails(context.TODO(), "bool_targeting_match", false, defaultEvaluationCtx())
		assert.NoError(t, err)
		assert.Equal(t, got2.Reason, of.CachedReason)
		got3, err := client.BooleanValueDetails(context.TODO(), "bool_targeting_match", false, defaultEvaluationCtx())
		assert.NoError(t, err)
		assert.Equal(t, got3.Reason, of.CachedReason)
		got4, err := client.BooleanValueDetails(context.TODO(), "bool_targeting_match", false, defaultEvaluationCtx())
		assert.NoError(t, err)
		assert.Equal(t, got4.Reason, of.CachedReason)
		assert.Equal(t, 1, cli.callCount)
	})

	t.Run("Call flag multiple times with different evaluation context", func(t *testing.T) {
		cli := mockClient{}
		options := provider_v2.ProviderOptions{
			Endpoint:            "https://gofeatureflag.org/",
			HTTPClient:          NewMockClient(cli.roundTripFunc),
			GOFeatureFlagConfig: nil,
			DisableCache:        false,
			FlagCacheTTL:        5 * time.Minute,
			FlagCacheSize:       5,
		}

		provider, err := provider_v2.NewProvider(options)
		defer provider.Shutdown()
		assert.NoError(t, err)

		err = of.SetProviderAndWait(provider)
		assert.NoError(t, err)
		client := of.NewClient("test-app")
		ctx1 := of.NewEvaluationContext("ffbe55ca-2150-4f15-a842-af6efb3a1391", map[string]interface{}{})
		ctx2 := of.NewEvaluationContext("316d4ac7-6072-472d-8a33-e35ed1702337", map[string]interface{}{})
		ctx3 := of.NewEvaluationContext("2b31904a-bfb0-46b8-8923-6bf32925de05", map[string]interface{}{})
		ctx4 := of.NewEvaluationContext("5d1d5245-23fd-466e-96a1-101e5088396e", map[string]interface{}{})
		got1, err := client.BooleanValueDetails(context.TODO(), "bool_targeting_match", false, ctx1)
		assert.NoError(t, err)
		assert.NotEqual(t, got1.Reason, of.CachedReason)
		got2, err := client.BooleanValueDetails(context.TODO(), "bool_targeting_match", false, ctx2)
		assert.NoError(t, err)
		assert.NotEqual(t, got2.Reason, of.CachedReason)
		got3, err := client.BooleanValueDetails(context.TODO(), "bool_targeting_match", false, ctx3)
		assert.NoError(t, err)
		assert.NotEqual(t, got3.Reason, of.CachedReason)
		got4, err := client.BooleanValueDetails(context.TODO(), "bool_targeting_match", false, ctx4)
		assert.NotEqual(t, got4.Reason, of.CachedReason)
		assert.NoError(t, err)
		assert.Equal(t, 4, cli.callCount)
	})

	t.Run("Cache fill all cache", func(t *testing.T) {
		mockedHttpClient := mockClient{}
		options := provider_v2.ProviderOptions{
			Endpoint:            "https://gofeatureflag.org/",
			HTTPClient:          NewMockClient(mockedHttpClient.roundTripFunc),
			GOFeatureFlagConfig: nil,
			DisableCache:        false,
			FlagCacheTTL:        5 * time.Minute,
			FlagCacheSize:       2,
		}

		provider, err := provider_v2.NewProvider(options)
		defer provider.Shutdown()
		assert.NoError(t, err)

		err = of.SetProviderAndWait(provider)
		assert.NoError(t, err)
		client := of.NewClient("test-app")
		ctx1 := of.NewEvaluationContext("ffbe55ca-2150-4f15-a842-af6efb3a1391", map[string]interface{}{})
		ctx2 := of.NewEvaluationContext("316d4ac7-6072-472d-8a33-e35ed1702337", map[string]interface{}{})
		ctx3 := of.NewEvaluationContext("2b31904a-bfb0-46b8-8923-6bf32925de05", map[string]interface{}{})
		r, err := client.BooleanValueDetails(context.TODO(), "bool_targeting_match", false, ctx1)
		assert.NoError(t, err)
		assert.Equal(t, of.TargetingMatchReason, r.Reason)
		r, err = client.BooleanValueDetails(context.TODO(), "bool_targeting_match", false, ctx1)
		assert.NoError(t, err)
		assert.Equal(t, of.CachedReason, r.Reason)
		r, err = client.BooleanValueDetails(context.TODO(), "bool_targeting_match", false, ctx2)
		assert.NoError(t, err)
		assert.Equal(t, of.TargetingMatchReason, r.Reason)
		r, err = client.BooleanValueDetails(context.TODO(), "bool_targeting_match", false, ctx2)
		assert.NoError(t, err)
		assert.Equal(t, of.CachedReason, r.Reason)
		r, err = client.BooleanValueDetails(context.TODO(), "bool_targeting_match", false, ctx3)
		assert.NoError(t, err)
		assert.Equal(t, of.TargetingMatchReason, r.Reason)
		r, err = client.BooleanValueDetails(context.TODO(), "bool_targeting_match", false, ctx3)
		assert.NoError(t, err)
		assert.Equal(t, of.CachedReason, r.Reason)
		r, err = client.BooleanValueDetails(context.TODO(), "bool_targeting_match", false, ctx1)
		assert.NoError(t, err)
		assert.Equal(t, of.TargetingMatchReason, r.Reason)
		assert.Equal(t, 4, mockedHttpClient.callCount)
	})

	t.Run("Cache TTL reached", func(t *testing.T) {
		mockedHttpClient := mockClient{}
		options := provider_v2.ProviderOptions{
			Endpoint:            "https://gofeatureflag.org/",
			HTTPClient:          NewMockClient(mockedHttpClient.roundTripFunc),
			GOFeatureFlagConfig: nil,
			DisableCache:        false,
			FlagCacheTTL:        500 * time.Millisecond,
			FlagCacheSize:       200,
		}

		provider, err := provider_v2.NewProvider(options)
		defer provider.Shutdown()
		assert.NoError(t, err)

		err = of.SetProviderAndWait(provider)
		assert.NoError(t, err)
		client := of.NewClient("test-app")
		_, err = client.BooleanValueDetails(context.TODO(), "bool_targeting_match", false, defaultEvaluationCtx())
		assert.NoError(t, err)
		time.Sleep(700 * time.Millisecond)
		_, err = client.BooleanValueDetails(context.TODO(), "bool_targeting_match", false, defaultEvaluationCtx())
		assert.NoError(t, err)
		assert.Equal(t, 2, mockedHttpClient.callCount)
	})
}

func TestProvider_DataCollectorHook(t *testing.T) {
	t.Run("DataCollectorHook is called for success and call API", func(t *testing.T) {
		cli := mockClient{}
		options := provider_v2.ProviderOptions{
			Endpoint:             "https://gofeatureflag.org/",
			HTTPClient:           NewMockClient(cli.roundTripFunc),
			DisableCache:         false,
			DataFlushInterval:    100 * time.Millisecond,
			DisableDataCollector: false,
		}
		provider, err := provider_v2.NewProvider(options)
		defer provider.Shutdown()
		assert.NoError(t, err)
		err = of.SetProviderAndWait(provider)
		assert.NoError(t, err)
		client := of.NewClient("test-app")

		_ = client.Boolean(context.TODO(), "bool_targeting_match", false, defaultEvaluationCtx())
		_ = client.Boolean(context.TODO(), "bool_targeting_match", false, defaultEvaluationCtx())
		_ = client.Boolean(context.TODO(), "bool_targeting_match", false, defaultEvaluationCtx())
		_ = client.Boolean(context.TODO(), "bool_targeting_match", false, defaultEvaluationCtx())
		_ = client.Boolean(context.TODO(), "bool_targeting_match", false, defaultEvaluationCtx())
		_ = client.Boolean(context.TODO(), "bool_targeting_match", false, defaultEvaluationCtx())
		_ = client.Boolean(context.TODO(), "bool_targeting_match", false, defaultEvaluationCtx())
		_ = client.Boolean(context.TODO(), "bool_targeting_match", false, defaultEvaluationCtx())
		_ = client.Boolean(context.TODO(), "bool_targeting_match", false, defaultEvaluationCtx())

		time.Sleep(500 * time.Millisecond)
		assert.Equal(t, 1, cli.callCount)
		assert.Equal(t, 1, cli.collectorCallCount)
	})

	t.Run("DataCollectorHook is called for errors and call API", func(t *testing.T) {
		cli := mockClient{}
		options := provider_v2.ProviderOptions{
			Endpoint:             "https://gofeatureflag.org/",
			HTTPClient:           NewMockClient(cli.roundTripFunc),
			DisableCache:         false,
			DataFlushInterval:    100 * time.Millisecond,
			DisableDataCollector: false,
		}
		provider, err := provider_v2.NewProvider(options)
		defer provider.Shutdown()
		assert.NoError(t, err)
		err = of.SetProviderAndWait(provider)
		assert.NoError(t, err)
		client := of.NewClient("test-app")

		_ = client.String(context.TODO(), "bool_targeting_match", "false", defaultEvaluationCtx())

		time.Sleep(1000 * time.Millisecond)
		assert.Equal(t, 1, cli.callCount)
		assert.Equal(t, 1, cli.collectorCallCount)
	})
}

func TestProvider_FlagChangePolling(t *testing.T) {
	t.Run("Should purge the cache if configuration has changed", func(t *testing.T) {
		cli := mockClient{}
		options := provider_v2.ProviderOptions{
			Endpoint:                  "https://gofeatureflag.org/",
			HTTPClient:                NewMockClient(cli.roundTripFunc),
			DisableCache:              false,
			FlagCacheTTL:              10 * time.Minute,
			DisableDataCollector:      true,
			FlagChangePollingInterval: 100 * time.Millisecond,
		}
		provider, err := provider_v2.NewProvider(options)
		defer provider.Shutdown()
		assert.NoError(t, err)
		err = of.SetProviderAndWait(provider)
		assert.NoError(t, err)
		client := of.NewClient("test-app")

		details, err := client.BooleanValueDetails(context.TODO(), "bool_targeting_match", false, defaultEvaluationCtx())
		require.NoError(t, err)
		assert.Equal(t, of.TargetingMatchReason, details.Reason)

		details, err = client.BooleanValueDetails(context.TODO(), "bool_targeting_match", false, defaultEvaluationCtx())
		require.NoError(t, err)
		assert.Equal(t, of.CachedReason, details.Reason)

		details, err = client.BooleanValueDetails(context.TODO(), "bool_targeting_match", false, defaultEvaluationCtx())
		require.NoError(t, err)
		assert.Equal(t, of.CachedReason, details.Reason)

		time.Sleep(220 * time.Millisecond) // Waiting > 200ms to trigger the polling in the mock

		details, err = client.BooleanValueDetails(context.TODO(), "bool_targeting_match", false, defaultEvaluationCtx())
		require.NoError(t, err)
		assert.Equal(t, of.TargetingMatchReason, details.Reason)
	})

}
