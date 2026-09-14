package gofeatureflag

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/open-feature/go-sdk-contrib/providers/go-feature-flag/pkg/model"
	"github.com/open-feature/go-sdk/openfeature"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockRoundTripper captures the last request body for inspection.
type mockRoundTripper struct {
	lastBody []byte
	status   int
	err      error
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Body != nil {
		m.lastBody, _ = io.ReadAll(req.Body)
	}
	if m.err != nil {
		return nil, m.err
	}
	return &http.Response{
		StatusCode: m.status,
		Body:       io.NopCloser(strings.NewReader("{}")),
	}, nil
}

// capturedRequest is used to unmarshal the raw events array before asserting on
// individual event types (CollectableEvent is an interface and cannot be unmarshalled
// directly).
type capturedRequest struct {
	Events []json.RawMessage `json:"events"`
	Meta   map[string]any    `json:"meta"`
}

func newTestProvider(t *testing.T, mrt *mockRoundTripper) *Provider {
	t.Helper()
	p, err := NewProviderWithContext(context.Background(), ProviderOptions{
		Endpoint:   "http://localhost:1031",
		HTTPClient: &http.Client{Transport: mrt},
	})
	require.NoError(t, err)
	return p
}

func newProviderHookContext(targetingKey string, attributes map[string]any) openfeature.HookContext {
	return openfeature.NewHookContext(
		"test-flag",
		openfeature.Boolean,
		false,
		openfeature.NewClientMetadata(""),
		openfeature.Metadata{Name: "test-provider"},
		openfeature.NewEvaluationContext(targetingKey, attributes),
	)
}

// MockRoundTripper is a mock implementation of http.RoundTripper used by evaluation tests.
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

// evalMockClient records HTTP calls and serves flag responses from testdata files.
type evalMockClient struct {
	callCount         int
	collectorRequests []string
	requestBodies     []string
}

func (m *evalMockClient) roundTripFunc(req *http.Request) *http.Response {
	var bodyBytes []byte
	if req.Body != nil {
		bodyBytes, _ = io.ReadAll(req.Body)
		req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		m.requestBodies = append(m.requestBodies, string(bodyBytes))
	}

	if req.URL.Path == "/v1/data/collector" {
		m.collectorRequests = append(m.collectorRequests, string(bodyBytes))
		return &http.Response{
			StatusCode: http.StatusOK,
		}
	}

	m.callCount++
	mockPath := "./testdata/mock_responses/%s.json"
	flagName := strings.ReplaceAll(req.URL.Path, "/ofrep/v1/evaluate/flags/", "")

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

	return &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(bytes.NewReader(content)),
	}
}

func defaultEvaluationCtx() openfeature.EvaluationContext {
	return openfeature.NewEvaluationContext(
		"d45e303a-38c2-11ed-a261-0242ac120002",
		map[string]any{
			"email":        "john.doe@gofeatureflag.org",
			"firstname":    "john",
			"lastname":     "doe",
			"anonymous":    false,
			"professional": true,
			"rate":         3.14,
			"age":          30,
			"admin":        true,
			"company_info": map[string]any{
				"name": "my_company",
				"size": 120,
			},
			"labels": []string{
				"pro", "beta",
			},
		},
	)
}

func TestNewProviderWithContext(t *testing.T) {
	tests := []struct {
		name    string
		options ProviderOptions
		wantErr bool
	}{
		{
			name:    "valid options",
			options: ProviderOptions{Endpoint: "http://localhost:1031"},
			wantErr: false,
		},
		{
			name:    "missing endpoint",
			options: ProviderOptions{Endpoint: ""},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p, err := NewProviderWithContext(context.Background(), tc.options)
			if tc.wantErr {
				require.Error(t, err)
				assert.Nil(t, p)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, p)
			}
		})
	}
}

func TestProvider_Track(t *testing.T) {
	tests := []struct {
		name             string
		trackingName     string
		evalCtx          openfeature.EvaluationContext
		details          openfeature.TrackingEventDetails
		wantContextKind  string
		wantUserKey      string
		wantEvalCtxKey   string
		wantEvalCtxValue any
		wantDetailKey    string
		wantDetailValue  any
	}{
		{
			name:            "regular user",
			trackingName:    "checkout",
			evalCtx:         openfeature.NewEvaluationContext("user-123", map[string]any{"anonymous": false}),
			details:         openfeature.NewTrackingEventDetails(0),
			wantContextKind: "user",
			wantUserKey:     "user-123",
		},
		{
			name:            "anonymous user",
			trackingName:    "checkout",
			evalCtx:         openfeature.NewEvaluationContext("anon-key", map[string]any{"anonymous": true}),
			details:         openfeature.NewTrackingEventDetails(0),
			wantContextKind: "anonymousUser",
			wantUserKey:     "anon-key",
		},
		{
			name:            "empty targeting key defaults to undefined",
			trackingName:    "checkout",
			evalCtx:         openfeature.NewEvaluationContext("", nil),
			details:         openfeature.NewTrackingEventDetails(0),
			wantContextKind: "user",
			wantUserKey:     "undefined",
		},
		{
			name:            "custom targeting key",
			trackingName:    "checkout",
			evalCtx:         openfeature.NewEvaluationContext("my-key", nil),
			details:         openfeature.NewTrackingEventDetails(0),
			wantContextKind: "user",
			wantUserKey:     "my-key",
		},
		{
			name:             "eval context attributes are preserved",
			trackingName:     "checkout",
			evalCtx:          openfeature.NewEvaluationContext("u1", map[string]any{"env": "prod"}),
			details:          openfeature.NewTrackingEventDetails(0),
			wantContextKind:  "user",
			wantUserKey:      "u1",
			wantEvalCtxKey:   "env",
			wantEvalCtxValue: "prod",
		},
		{
			name:            "tracking details are preserved",
			trackingName:    "purchase",
			evalCtx:         openfeature.NewEvaluationContext("u2", nil),
			details:         openfeature.NewTrackingEventDetails(0).Add("price", 9.99),
			wantContextKind: "user",
			wantUserKey:     "u2",
			wantDetailKey:   "price",
			wantDetailValue: 9.99,
		},
		{
			name:            "event name matches tracking event name",
			trackingName:    "my-event",
			evalCtx:         openfeature.NewEvaluationContext("u3", nil),
			details:         openfeature.NewTrackingEventDetails(0),
			wantContextKind: "user",
			wantUserKey:     "u3",
		},
		{
			name:            "kind is always tracking",
			trackingName:    "any-event",
			evalCtx:         openfeature.NewEvaluationContext("u4", nil),
			details:         openfeature.NewTrackingEventDetails(0),
			wantContextKind: "user",
			wantUserKey:     "u4",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mrt := &mockRoundTripper{status: http.StatusOK}
			p := newTestProvider(t, mrt)

			p.Track(context.Background(), tc.trackingName, tc.evalCtx, tc.details)

			err := p.dataCollectorMgr.SendData(context.Background())
			require.NoError(t, err)

			require.NotNil(t, mrt.lastBody, "expected HTTP request to be made")

			var captured capturedRequest
			require.NoError(t, json.Unmarshal(mrt.lastBody, &captured))
			require.Len(t, captured.Events, 1)

			var event model.TrackingEvent
			require.NoError(t, json.Unmarshal(captured.Events[0], &event))

			assert.Equal(t, "tracking", event.Kind)
			assert.Equal(t, tc.wantContextKind, event.ContextKind)
			assert.Equal(t, tc.wantUserKey, event.UserKey)
			assert.Equal(t, tc.trackingName, event.Key)
			assert.NotZero(t, event.CreationDate)

			if tc.wantEvalCtxKey != "" {
				assert.Equal(t, tc.wantEvalCtxValue, event.EvaluationContext[tc.wantEvalCtxKey])
			}
			if tc.wantDetailKey != "" {
				assert.InDelta(t, tc.wantDetailValue, event.TrackingDetails[tc.wantDetailKey], 0.001)
			}
		})
	}
}

func TestProvider_DataCollectorHookUsesProviderManager(t *testing.T) {
	mrt := &mockRoundTripper{status: http.StatusOK}
	p := newTestProvider(t, mrt)

	hooks := p.Hooks()
	require.Len(t, hooks, 2)

	hookCtx := newProviderHookContext("user-123", map[string]any{})
	evalDetails := openfeature.InterfaceEvaluationDetails{
		Value: "enabled-value",
		EvaluationDetails: openfeature.EvaluationDetails{
			FlagKey:  "test-flag",
			FlagType: openfeature.Object,
		},
	}
	evalDetails.Variant = "variant-A"
	evalDetails.Reason = openfeature.TargetingMatchReason

	for _, hook := range hooks {
		err := hook.After(context.Background(), hookCtx, evalDetails, openfeature.HookHints{})
		require.NoError(t, err)
	}

	err := p.dataCollectorMgr.SendData(context.Background())
	require.NoError(t, err)
	require.NotNil(t, mrt.lastBody, "expected provider manager flush to send hook-collected event")

	var captured capturedRequest
	require.NoError(t, json.Unmarshal(mrt.lastBody, &captured))
	require.Len(t, captured.Events, 1)

	var event model.FeatureEvent
	require.NoError(t, json.Unmarshal(captured.Events[0], &event))
	assert.Equal(t, "feature", event.Kind)
	assert.Equal(t, "user", event.ContextKind)
	assert.Equal(t, "user-123", event.UserKey)
	assert.Equal(t, "test-flag", event.Key)
	assert.Equal(t, "variant-A", event.Variation)
	assert.Equal(t, "enabled-value", event.Value)
	assert.False(t, event.Default)
	assert.Equal(t, "INPROCESS", event.Source)
	assert.Greater(t, event.CreationDate, int64(0))
}

func TestProvider_InitShutdown(t *testing.T) {
	mrt := &mockRoundTripper{status: http.StatusOK}
	p := newTestProvider(t, mrt)

	require.NoError(t, p.Init(openfeature.NewEvaluationContext("", nil)))
	p.Shutdown() // must not block
}

func TestProvider_BooleanEvaluation(t *testing.T) {
	type args struct {
		flag         string
		defaultValue bool
		evalCtx      openfeature.EvaluationContext
	}
	tests := []struct {
		name string
		args args
		want openfeature.BooleanEvaluationDetails
	}{
		{
			name: "unauthorized flag",
			args: args{
				flag:         "unauthorized",
				defaultValue: false,
				evalCtx:      defaultEvaluationCtx(),
			},
			want: openfeature.BooleanEvaluationDetails{
				Value: false,
				EvaluationDetails: openfeature.EvaluationDetails{
					FlagKey:  "unauthorized",
					FlagType: openfeature.Boolean,
					ResolutionDetail: openfeature.ResolutionDetail{
						Variant:      "",
						Reason:       openfeature.ErrorReason,
						ErrorCode:    openfeature.GeneralCode,
						ErrorMessage: "authentication/authorization error",
						FlagMetadata: map[string]any{},
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
			want: openfeature.BooleanEvaluationDetails{
				Value: true,
				EvaluationDetails: openfeature.EvaluationDetails{
					FlagKey:  "bool_targeting_match",
					FlagType: openfeature.Boolean,
					ResolutionDetail: openfeature.ResolutionDetail{
						Variant:      "True",
						Reason:       openfeature.TargetingMatchReason,
						ErrorCode:    "",
						ErrorMessage: "",
						FlagMetadata: map[string]any{
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
			want: openfeature.BooleanEvaluationDetails{
				Value: false,
				EvaluationDetails: openfeature.EvaluationDetails{
					FlagKey:  "disabled_bool",
					FlagType: openfeature.Boolean,
					ResolutionDetail: openfeature.ResolutionDetail{
						Variant:      "SdkDefault",
						Reason:       openfeature.DisabledReason,
						ErrorCode:    "",
						ErrorMessage: "",
						FlagMetadata: map[string]any{},
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
			want: openfeature.BooleanEvaluationDetails{
				Value: false,
				EvaluationDetails: openfeature.EvaluationDetails{
					FlagKey:  "string_key",
					FlagType: openfeature.Boolean,
					ResolutionDetail: openfeature.ResolutionDetail{
						Variant:      "",
						Reason:       openfeature.ErrorReason,
						ErrorCode:    openfeature.TypeMismatchCode,
						ErrorMessage: "resolved value CC0000 is not of boolean type",
						FlagMetadata: map[string]any{},
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
			want: openfeature.BooleanEvaluationDetails{
				Value: false,
				EvaluationDetails: openfeature.EvaluationDetails{
					FlagKey:  "does_not_exists",
					FlagType: openfeature.Boolean,
					ResolutionDetail: openfeature.ResolutionDetail{
						Variant:      "",
						Reason:       openfeature.ErrorReason,
						ErrorCode:    openfeature.FlagNotFoundCode,
						ErrorMessage: "flag for key 'does_not_exists' does not exist",
						FlagMetadata: map[string]any{},
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
			want: openfeature.BooleanEvaluationDetails{
				Value: true,
				EvaluationDetails: openfeature.EvaluationDetails{
					FlagKey:  "unknown_reason",
					FlagType: openfeature.Boolean,
					ResolutionDetail: openfeature.ResolutionDetail{
						Variant:      "True",
						Reason:       "CUSTOM_REASON",
						ErrorCode:    "",
						ErrorMessage: "",
						FlagMetadata: map[string]any{},
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
			want: openfeature.BooleanEvaluationDetails{
				Value: false,
				EvaluationDetails: openfeature.EvaluationDetails{
					FlagKey:  "invalid_json_body",
					FlagType: openfeature.Boolean,
					ResolutionDetail: openfeature.ResolutionDetail{
						Reason:       openfeature.ErrorReason,
						ErrorCode:    openfeature.ParseErrorCode,
						ErrorMessage: "error parsing the response: unexpected end of JSON input",
						FlagMetadata: map[string]any{},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cli := evalMockClient{}
			options := ProviderOptions{
				Endpoint:       "https://gofeatureflag.org/",
				HTTPClient:     NewMockClient(cli.roundTripFunc),
				EvaluationType: EvaluationTypeRemote,
			}
			provider, err := NewProvider(options)
			require.NoError(t, err)

			err = openfeature.SetProviderAndWait(provider)
			require.NoError(t, err)
			client := openfeature.NewClient("test-app")
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
		evalCtx      openfeature.EvaluationContext
	}
	tests := []struct {
		name string
		args args
		want openfeature.StringEvaluationDetails
	}{
		{
			name: "should resolve a valid string flag with TARGETING_MATCH reason",
			args: args{
				flag:         "string_key",
				defaultValue: "default",
				evalCtx:      defaultEvaluationCtx(),
			},
			want: openfeature.StringEvaluationDetails{
				Value: "CC0000",
				EvaluationDetails: openfeature.EvaluationDetails{
					FlagKey:  "string_key",
					FlagType: openfeature.String,
					ResolutionDetail: openfeature.ResolutionDetail{
						Variant:      "True",
						Reason:       openfeature.TargetingMatchReason,
						ErrorCode:    "",
						ErrorMessage: "",
						FlagMetadata: map[string]any{},
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
			want: openfeature.StringEvaluationDetails{
				Value: "default",
				EvaluationDetails: openfeature.EvaluationDetails{
					FlagKey:  "disabled_string",
					FlagType: openfeature.String,
					ResolutionDetail: openfeature.ResolutionDetail{
						Variant:      "SdkDefault",
						Reason:       openfeature.DisabledReason,
						ErrorCode:    "",
						ErrorMessage: "",
						FlagMetadata: map[string]any{},
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
			want: openfeature.StringEvaluationDetails{
				Value: "default",
				EvaluationDetails: openfeature.EvaluationDetails{
					FlagKey:  "bool_targeting_match",
					FlagType: openfeature.String,
					ResolutionDetail: openfeature.ResolutionDetail{
						Variant:      "",
						Reason:       openfeature.ErrorReason,
						ErrorCode:    openfeature.TypeMismatchCode,
						ErrorMessage: "resolved value true is not of string type",
						FlagMetadata: map[string]any{},
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
			want: openfeature.StringEvaluationDetails{
				Value: "default",
				EvaluationDetails: openfeature.EvaluationDetails{
					FlagKey:  "does_not_exists",
					FlagType: openfeature.String,
					ResolutionDetail: openfeature.ResolutionDetail{
						Variant:      "",
						Reason:       openfeature.ErrorReason,
						ErrorCode:    openfeature.FlagNotFoundCode,
						ErrorMessage: "flag for key 'does_not_exists' does not exist",
						FlagMetadata: map[string]any{},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cli := evalMockClient{}
			options := ProviderOptions{
				Endpoint:       "https://gofeatureflag.org/",
				HTTPClient:     NewMockClient(cli.roundTripFunc),
				EvaluationType: EvaluationTypeRemote,
			}
			provider, err := NewProvider(options)
			require.NoError(t, err)

			err = openfeature.SetProviderAndWait(provider)
			require.NoError(t, err)
			client := openfeature.NewClient("test-app")
			value, err := client.StringValueDetails(context.TODO(), tt.args.flag, tt.args.defaultValue, tt.args.evalCtx)

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

func TestProvider_FloatEvaluation(t *testing.T) {
	type args struct {
		flag         string
		defaultValue float64
		evalCtx      openfeature.EvaluationContext
	}
	tests := []struct {
		name string
		args args
		want openfeature.FloatEvaluationDetails
	}{
		{
			name: "should resolve a valid float flag with TARGETING_MATCH reason",
			args: args{
				flag:         "double_key",
				defaultValue: 123.45,
				evalCtx:      defaultEvaluationCtx(),
			},
			want: openfeature.FloatEvaluationDetails{
				Value: 100.25,
				EvaluationDetails: openfeature.EvaluationDetails{
					FlagKey:  "double_key",
					FlagType: openfeature.Float,
					ResolutionDetail: openfeature.ResolutionDetail{
						Variant:      "True",
						Reason:       openfeature.TargetingMatchReason,
						ErrorCode:    "",
						ErrorMessage: "",
						FlagMetadata: map[string]any{},
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
			want: openfeature.FloatEvaluationDetails{
				Value: 123.45,
				EvaluationDetails: openfeature.EvaluationDetails{
					FlagKey:  "disabled_float",
					FlagType: openfeature.Float,
					ResolutionDetail: openfeature.ResolutionDetail{
						Variant:      "SdkDefault",
						Reason:       openfeature.DisabledReason,
						ErrorCode:    "",
						ErrorMessage: "",
						FlagMetadata: map[string]any{},
					},
				},
			},
		},
		{
			name: "should error if we expect a float and got another type",
			args: args{
				flag:         "bool_targeting_match",
				defaultValue: 123.45,
				evalCtx:      defaultEvaluationCtx(),
			},
			want: openfeature.FloatEvaluationDetails{
				Value: 123.45,
				EvaluationDetails: openfeature.EvaluationDetails{
					FlagKey:  "bool_targeting_match",
					FlagType: openfeature.Float,
					ResolutionDetail: openfeature.ResolutionDetail{
						Variant:      "",
						Reason:       openfeature.ErrorReason,
						ErrorCode:    openfeature.TypeMismatchCode,
						ErrorMessage: "resolved value true is not of float type",
						FlagMetadata: map[string]any{},
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
			want: openfeature.FloatEvaluationDetails{
				Value: 123.45,
				EvaluationDetails: openfeature.EvaluationDetails{
					FlagKey:  "does_not_exists",
					FlagType: openfeature.Float,
					ResolutionDetail: openfeature.ResolutionDetail{
						Variant:      "",
						Reason:       openfeature.ErrorReason,
						ErrorCode:    openfeature.FlagNotFoundCode,
						ErrorMessage: "flag for key 'does_not_exists' does not exist",
						FlagMetadata: map[string]any{},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cli := evalMockClient{}
			options := ProviderOptions{
				Endpoint:       "https://gofeatureflag.org/",
				HTTPClient:     NewMockClient(cli.roundTripFunc),
				EvaluationType: EvaluationTypeRemote,
			}
			provider, err := NewProvider(options)
			require.NoError(t, err)

			err = openfeature.SetProviderAndWait(provider)
			require.NoError(t, err)
			client := openfeature.NewClient("test-app")
			value, err := client.FloatValueDetails(context.TODO(), tt.args.flag, tt.args.defaultValue, tt.args.evalCtx)

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

func TestProvider_IntEvaluation(t *testing.T) {
	type args struct {
		flag         string
		defaultValue int64
		evalCtx      openfeature.EvaluationContext
	}
	tests := []struct {
		name string
		args args
		want openfeature.IntEvaluationDetails
	}{
		{
			name: "should resolve a valid int flag with TARGETING_MATCH reason",
			args: args{
				flag:         "integer_key",
				defaultValue: 123,
				evalCtx:      defaultEvaluationCtx(),
			},
			want: openfeature.IntEvaluationDetails{
				Value: 100,
				EvaluationDetails: openfeature.EvaluationDetails{
					FlagKey:  "integer_key",
					FlagType: openfeature.Int,
					ResolutionDetail: openfeature.ResolutionDetail{
						Variant:      "True",
						Reason:       openfeature.TargetingMatchReason,
						ErrorCode:    "",
						ErrorMessage: "",
						FlagMetadata: map[string]any{},
					},
				},
			},
		},
		{
			name: "should use int default value if the flag is disabled",
			args: args{
				flag:         "disabled_int",
				defaultValue: 123,
				evalCtx:      defaultEvaluationCtx(),
			},
			want: openfeature.IntEvaluationDetails{
				Value: 123,
				EvaluationDetails: openfeature.EvaluationDetails{
					FlagKey:  "disabled_int",
					FlagType: openfeature.Int,
					ResolutionDetail: openfeature.ResolutionDetail{
						Variant:      "SdkDefault",
						Reason:       openfeature.DisabledReason,
						ErrorCode:    "",
						ErrorMessage: "",
						FlagMetadata: map[string]any{},
					},
				},
			},
		},
		{
			name: "should error if we expect an int and got another type",
			args: args{
				flag:         "bool_targeting_match",
				defaultValue: 123,
				evalCtx:      defaultEvaluationCtx(),
			},
			want: openfeature.IntEvaluationDetails{
				Value: 123,
				EvaluationDetails: openfeature.EvaluationDetails{
					FlagKey:  "bool_targeting_match",
					FlagType: openfeature.Int,
					ResolutionDetail: openfeature.ResolutionDetail{
						Variant:      "",
						Reason:       openfeature.ErrorReason,
						ErrorCode:    openfeature.TypeMismatchCode,
						ErrorMessage: "resolved value true is not of integer type",
						FlagMetadata: map[string]any{},
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
			want: openfeature.IntEvaluationDetails{
				Value: 123,
				EvaluationDetails: openfeature.EvaluationDetails{
					FlagKey:  "does_not_exists",
					FlagType: openfeature.Int,
					ResolutionDetail: openfeature.ResolutionDetail{
						Variant:      "",
						Reason:       openfeature.ErrorReason,
						ErrorCode:    openfeature.FlagNotFoundCode,
						ErrorMessage: "flag for key 'does_not_exists' does not exist",
						FlagMetadata: map[string]any{},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cli := evalMockClient{}
			options := ProviderOptions{
				Endpoint:       "https://gofeatureflag.org/",
				HTTPClient:     NewMockClient(cli.roundTripFunc),
				EvaluationType: EvaluationTypeRemote,
			}
			provider, err := NewProvider(options)
			require.NoError(t, err)

			err = openfeature.SetProviderAndWait(provider)
			require.NoError(t, err)
			client := openfeature.NewClient("test-app")
			value, err := client.IntValueDetails(context.TODO(), tt.args.flag, tt.args.defaultValue, tt.args.evalCtx)

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

func TestProvider_ObjectEvaluation(t *testing.T) {
	type args struct {
		flag         string
		defaultValue any
		evalCtx      openfeature.EvaluationContext
	}
	tests := []struct {
		name string
		args args
		want openfeature.InterfaceEvaluationDetails
	}{
		{
			name: "should resolve a valid interface flag with TARGETING_MATCH reason",
			args: args{
				flag:         "object_key",
				defaultValue: nil,
				evalCtx:      defaultEvaluationCtx(),
			},
			want: openfeature.InterfaceEvaluationDetails{
				Value: map[string]any{
					"test":  "test1",
					"test2": false,
					"test3": 123.3,
					"test4": float64(1),
					"test5": nil,
				},
				EvaluationDetails: openfeature.EvaluationDetails{
					FlagKey:  "object_key",
					FlagType: openfeature.Object,
					ResolutionDetail: openfeature.ResolutionDetail{
						Variant:      "True",
						Reason:       openfeature.TargetingMatchReason,
						ErrorCode:    "",
						ErrorMessage: "",
						FlagMetadata: map[string]any{},
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
			want: openfeature.InterfaceEvaluationDetails{
				Value: nil,
				EvaluationDetails: openfeature.EvaluationDetails{
					FlagKey:  "disabled_int",
					FlagType: openfeature.Object,
					ResolutionDetail: openfeature.ResolutionDetail{
						Variant:      "SdkDefault",
						Reason:       openfeature.DisabledReason,
						ErrorCode:    "",
						ErrorMessage: "",
						FlagMetadata: map[string]any{},
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
			want: openfeature.InterfaceEvaluationDetails{
				Value: nil,
				EvaluationDetails: openfeature.EvaluationDetails{
					FlagKey:  "does_not_exists",
					FlagType: openfeature.Object,
					ResolutionDetail: openfeature.ResolutionDetail{
						Variant:      "",
						Reason:       openfeature.ErrorReason,
						ErrorCode:    openfeature.FlagNotFoundCode,
						ErrorMessage: "flag for key 'does_not_exists' does not exist",
						FlagMetadata: map[string]any{},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cli := evalMockClient{}
			options := ProviderOptions{
				Endpoint:       "https://gofeatureflag.org/",
				HTTPClient:     NewMockClient(cli.roundTripFunc),
				EvaluationType: EvaluationTypeRemote,
			}
			provider, err := NewProvider(options)
			require.NoError(t, err)

			err = openfeature.SetProviderAndWait(provider)
			require.NoError(t, err)
			client := openfeature.NewClient("test-app")
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

// inprocessMockClient serves flag configuration for in-process evaluation tests.
type inprocessMockClient struct{}

func (m *inprocessMockClient) roundTripFunc(req *http.Request) *http.Response {
	if strings.HasSuffix(req.URL.Path, "/v1/data/collector") {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader([]byte("{}"))),
		}
	}
	if strings.HasSuffix(req.URL.Path, "/v1/flag/configuration") {
		content, err := os.ReadFile("./testdata/flag_config_responses/valid-all-types.json")
		if err != nil {
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Body:       io.NopCloser(bytes.NewReader([]byte(err.Error()))),
			}
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader(content)),
		}
	}
	return &http.Response{
		StatusCode: http.StatusNotFound,
		Body:       io.NopCloser(bytes.NewReader([]byte(""))),
	}
}

func newInprocessProvider(t *testing.T, cli *inprocessMockClient) *Provider {
	t.Helper()
	options := ProviderOptions{
		Endpoint:       "https://gofeatureflag.org/",
		HTTPClient:     NewMockClient(cli.roundTripFunc),
		EvaluationType: EvaluationTypeInProcess,
	}
	provider, err := NewProvider(options)
	require.NoError(t, err)
	return provider
}

func TestProvider_BooleanEvaluation_InProcess(t *testing.T) {
	type args struct {
		flag         string
		defaultValue bool
		evalCtx      openfeature.EvaluationContext
	}
	tests := []struct {
		name string
		args args
		want openfeature.BooleanEvaluationDetails
	}{
		{
			name: "should resolve a valid boolean flag with TARGETING_MATCH reason",
			args: args{
				flag:         "bool_targeting_match",
				defaultValue: false,
				evalCtx:      defaultEvaluationCtx(),
			},
			want: openfeature.BooleanEvaluationDetails{
				Value: true,
				EvaluationDetails: openfeature.EvaluationDetails{
					FlagKey:  "bool_targeting_match",
					FlagType: openfeature.Boolean,
					ResolutionDetail: openfeature.ResolutionDetail{
						Variant:      "enabled",
						Reason:       openfeature.TargetingMatchReason,
						ErrorCode:    "",
						ErrorMessage: "",
						FlagMetadata: map[string]any{
							"description":  "this is a test flag",
							"defaultValue": false,
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
			want: openfeature.BooleanEvaluationDetails{
				Value: false,
				EvaluationDetails: openfeature.EvaluationDetails{
					FlagKey:  "disabled_bool",
					FlagType: openfeature.Boolean,
					ResolutionDetail: openfeature.ResolutionDetail{
						Variant:      "SdkDefault",
						Reason:       openfeature.DisabledReason,
						ErrorCode:    "",
						ErrorMessage: "",
						FlagMetadata: map[string]any{
							"description":  "this is a test flag",
							"defaultValue": false,
						},
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
			want: openfeature.BooleanEvaluationDetails{
				Value: false,
				EvaluationDetails: openfeature.EvaluationDetails{
					FlagKey:  "string_key",
					FlagType: openfeature.Boolean,
					ResolutionDetail: openfeature.ResolutionDetail{
						Variant:      "",
						Reason:       openfeature.ErrorReason,
						ErrorCode:    openfeature.TypeMismatchCode,
						ErrorMessage: "wrong variation used for flag string_key",
						FlagMetadata: map[string]any{},
					},
				},
			},
		},
		{
			name: "should error if flag does not exist",
			args: args{
				flag:         "does_not_exists",
				defaultValue: false,
				evalCtx:      defaultEvaluationCtx(),
			},
			want: openfeature.BooleanEvaluationDetails{
				Value: false,
				EvaluationDetails: openfeature.EvaluationDetails{
					FlagKey:  "does_not_exists",
					FlagType: openfeature.Boolean,
					ResolutionDetail: openfeature.ResolutionDetail{
						Variant:      "",
						Reason:       openfeature.ErrorReason,
						ErrorCode:    openfeature.FlagNotFoundCode,
						ErrorMessage: "does_not_exists",
						FlagMetadata: map[string]any{},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cli := inprocessMockClient{}
			provider := newInprocessProvider(t, &cli)

			err := openfeature.SetProviderAndWait(provider)
			require.NoError(t, err)
			client := openfeature.NewClient("test-app")
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

func TestProvider_StringEvaluation_InProcess(t *testing.T) {
	type args struct {
		flag         string
		defaultValue string
		evalCtx      openfeature.EvaluationContext
	}
	tests := []struct {
		name string
		args args
		want openfeature.StringEvaluationDetails
	}{
		{
			name: "should resolve a valid string flag with STATIC reason",
			args: args{
				flag:         "string_key",
				defaultValue: "default",
				evalCtx:      defaultEvaluationCtx(),
			},
			want: openfeature.StringEvaluationDetails{
				Value: "CC0002",
				EvaluationDetails: openfeature.EvaluationDetails{
					FlagKey:  "string_key",
					FlagType: openfeature.String,
					ResolutionDetail: openfeature.ResolutionDetail{
						Variant:      "color1",
						Reason:       openfeature.Reason("STATIC"),
						ErrorCode:    "",
						ErrorMessage: "",
						FlagMetadata: map[string]any{
							"description":  "this is a test flag",
							"defaultValue": "CC0000",
						},
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
			want: openfeature.StringEvaluationDetails{
				Value: "default",
				EvaluationDetails: openfeature.EvaluationDetails{
					FlagKey:  "disabled_string",
					FlagType: openfeature.String,
					ResolutionDetail: openfeature.ResolutionDetail{
						Variant:      "SdkDefault",
						Reason:       openfeature.DisabledReason,
						ErrorCode:    "",
						ErrorMessage: "",
						FlagMetadata: map[string]any{
							"description":  "this is a test",
							"defaultValue": "CC0000",
						},
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
			want: openfeature.StringEvaluationDetails{
				Value: "default",
				EvaluationDetails: openfeature.EvaluationDetails{
					FlagKey:  "bool_targeting_match",
					FlagType: openfeature.String,
					ResolutionDetail: openfeature.ResolutionDetail{
						Variant:      "",
						Reason:       openfeature.ErrorReason,
						ErrorCode:    openfeature.TypeMismatchCode,
						ErrorMessage: "wrong variation used for flag bool_targeting_match",
						FlagMetadata: map[string]any{},
					},
				},
			},
		},
		{
			name: "should error if flag does not exist",
			args: args{
				flag:         "does_not_exists",
				defaultValue: "default",
				evalCtx:      defaultEvaluationCtx(),
			},
			want: openfeature.StringEvaluationDetails{
				Value: "default",
				EvaluationDetails: openfeature.EvaluationDetails{
					FlagKey:  "does_not_exists",
					FlagType: openfeature.String,
					ResolutionDetail: openfeature.ResolutionDetail{
						Variant:      "",
						Reason:       openfeature.ErrorReason,
						ErrorCode:    openfeature.FlagNotFoundCode,
						ErrorMessage: "does_not_exists",
						FlagMetadata: map[string]any{},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cli := inprocessMockClient{}
			provider := newInprocessProvider(t, &cli)

			err := openfeature.SetProviderAndWait(provider)
			require.NoError(t, err)
			client := openfeature.NewClient("test-app")
			value, err := client.StringValueDetails(context.TODO(), tt.args.flag, tt.args.defaultValue, tt.args.evalCtx)

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

func TestProvider_FloatEvaluation_InProcess(t *testing.T) {
	type args struct {
		flag         string
		defaultValue float64
		evalCtx      openfeature.EvaluationContext
	}
	tests := []struct {
		name string
		args args
		want openfeature.FloatEvaluationDetails
	}{
		{
			name: "should resolve a valid float flag with TARGETING_MATCH reason",
			args: args{
				flag:         "double_key",
				defaultValue: 123.45,
				evalCtx:      defaultEvaluationCtx(),
			},
			want: openfeature.FloatEvaluationDetails{
				Value: 101.25,
				EvaluationDetails: openfeature.EvaluationDetails{
					FlagKey:  "double_key",
					FlagType: openfeature.Float,
					ResolutionDetail: openfeature.ResolutionDetail{
						Variant:      "medium",
						Reason:       openfeature.TargetingMatchReason,
						ErrorCode:    "",
						ErrorMessage: "",
						FlagMetadata: map[string]any{
							"description":  "this is a test flag",
							"defaultValue": 100.25,
						},
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
			want: openfeature.FloatEvaluationDetails{
				Value: 123.45,
				EvaluationDetails: openfeature.EvaluationDetails{
					FlagKey:  "disabled_float",
					FlagType: openfeature.Float,
					ResolutionDetail: openfeature.ResolutionDetail{
						Variant:      "SdkDefault",
						Reason:       openfeature.DisabledReason,
						ErrorCode:    "",
						ErrorMessage: "",
						FlagMetadata: map[string]any{
							"description":  "this is a test",
							"defaultValue": 100.25,
						},
					},
				},
			},
		},
		{
			name: "should error if we expect a float and got another type",
			args: args{
				flag:         "bool_targeting_match",
				defaultValue: 123.45,
				evalCtx:      defaultEvaluationCtx(),
			},
			want: openfeature.FloatEvaluationDetails{
				Value: 123.45,
				EvaluationDetails: openfeature.EvaluationDetails{
					FlagKey:  "bool_targeting_match",
					FlagType: openfeature.Float,
					ResolutionDetail: openfeature.ResolutionDetail{
						Variant:      "",
						Reason:       openfeature.ErrorReason,
						ErrorCode:    openfeature.TypeMismatchCode,
						ErrorMessage: "wrong variation used for flag bool_targeting_match",
						FlagMetadata: map[string]any{},
					},
				},
			},
		},
		{
			name: "should error if flag does not exist",
			args: args{
				flag:         "does_not_exists",
				defaultValue: 123.45,
				evalCtx:      defaultEvaluationCtx(),
			},
			want: openfeature.FloatEvaluationDetails{
				Value: 123.45,
				EvaluationDetails: openfeature.EvaluationDetails{
					FlagKey:  "does_not_exists",
					FlagType: openfeature.Float,
					ResolutionDetail: openfeature.ResolutionDetail{
						Variant:      "",
						Reason:       openfeature.ErrorReason,
						ErrorCode:    openfeature.FlagNotFoundCode,
						ErrorMessage: "does_not_exists",
						FlagMetadata: map[string]any{},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cli := inprocessMockClient{}
			provider := newInprocessProvider(t, &cli)

			err := openfeature.SetProviderAndWait(provider)
			require.NoError(t, err)
			client := openfeature.NewClient("test-app")
			value, err := client.FloatValueDetails(context.TODO(), tt.args.flag, tt.args.defaultValue, tt.args.evalCtx)

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

func TestProvider_IntEvaluation_InProcess(t *testing.T) {
	type args struct {
		flag         string
		defaultValue int64
		evalCtx      openfeature.EvaluationContext
	}
	tests := []struct {
		name string
		args args
		want openfeature.IntEvaluationDetails
	}{
		{
			name: "should resolve a valid int flag with TARGETING_MATCH reason",
			args: args{
				flag:         "integer_key",
				defaultValue: 123,
				evalCtx:      defaultEvaluationCtx(),
			},
			want: openfeature.IntEvaluationDetails{
				Value: 101,
				EvaluationDetails: openfeature.EvaluationDetails{
					FlagKey:  "integer_key",
					FlagType: openfeature.Int,
					ResolutionDetail: openfeature.ResolutionDetail{
						Variant:      "medium",
						Reason:       openfeature.TargetingMatchReason,
						ErrorCode:    "",
						ErrorMessage: "",
						FlagMetadata: map[string]any{
							"defaultValue": float64(1000),
							"description":  "this is a test flag",
						},
					},
				},
			},
		},
		{
			name: "should use int default value if the flag is disabled",
			args: args{
				flag:         "disabled_int",
				defaultValue: 123,
				evalCtx:      defaultEvaluationCtx(),
			},
			want: openfeature.IntEvaluationDetails{
				Value: 123,
				EvaluationDetails: openfeature.EvaluationDetails{
					FlagKey:  "disabled_int",
					FlagType: openfeature.Int,
					ResolutionDetail: openfeature.ResolutionDetail{
						Variant:      "SdkDefault",
						Reason:       openfeature.DisabledReason,
						ErrorCode:    "",
						ErrorMessage: "",
						FlagMetadata: map[string]any{
							"description":  "this is a test",
							"defaultValue": float64(100),
						},
					},
				},
			},
		},
		{
			name: "should error if we expect an int and got another type",
			args: args{
				flag:         "bool_targeting_match",
				defaultValue: 123,
				evalCtx:      defaultEvaluationCtx(),
			},
			want: openfeature.IntEvaluationDetails{
				Value: 123,
				EvaluationDetails: openfeature.EvaluationDetails{
					FlagKey:  "bool_targeting_match",
					FlagType: openfeature.Int,
					ResolutionDetail: openfeature.ResolutionDetail{
						Variant:      "",
						Reason:       openfeature.ErrorReason,
						ErrorCode:    openfeature.TypeMismatchCode,
						ErrorMessage: "wrong variation used for flag bool_targeting_match",
						FlagMetadata: map[string]any{},
					},
				},
			},
		},
		{
			name: "should error if flag does not exist",
			args: args{
				flag:         "does_not_exists",
				defaultValue: 123,
				evalCtx:      defaultEvaluationCtx(),
			},
			want: openfeature.IntEvaluationDetails{
				Value: 123,
				EvaluationDetails: openfeature.EvaluationDetails{
					FlagKey:  "does_not_exists",
					FlagType: openfeature.Int,
					ResolutionDetail: openfeature.ResolutionDetail{
						Variant:      "",
						Reason:       openfeature.ErrorReason,
						ErrorCode:    openfeature.FlagNotFoundCode,
						ErrorMessage: "does_not_exists",
						FlagMetadata: map[string]any{},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cli := inprocessMockClient{}
			provider := newInprocessProvider(t, &cli)

			err := openfeature.SetProviderAndWait(provider)
			require.NoError(t, err)
			client := openfeature.NewClient("test-app")
			value, err := client.IntValueDetails(context.TODO(), tt.args.flag, tt.args.defaultValue, tt.args.evalCtx)

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

func TestProvider_ObjectEvaluation_InProcess(t *testing.T) {
	type args struct {
		flag         string
		defaultValue any
		evalCtx      openfeature.EvaluationContext
	}
	tests := []struct {
		name string
		args args
		want openfeature.InterfaceEvaluationDetails
	}{
		{
			name: "should resolve a valid object flag with TARGETING_MATCH reason",
			args: args{
				flag:         "object_key",
				defaultValue: nil,
				evalCtx:      defaultEvaluationCtx(),
			},
			want: openfeature.InterfaceEvaluationDetails{
				Value: map[string]any{
					"test": "false",
				},
				EvaluationDetails: openfeature.EvaluationDetails{
					FlagKey:  "object_key",
					FlagType: openfeature.Object,
					ResolutionDetail: openfeature.ResolutionDetail{
						Variant:      "varB",
						Reason:       openfeature.TargetingMatchReason,
						ErrorCode:    "",
						ErrorMessage: "",
						FlagMetadata: map[string]any{},
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
			want: openfeature.InterfaceEvaluationDetails{
				Value: nil,
				EvaluationDetails: openfeature.EvaluationDetails{
					FlagKey:  "disabled_int",
					FlagType: openfeature.Object,
					ResolutionDetail: openfeature.ResolutionDetail{
						Variant:      "SdkDefault",
						Reason:       openfeature.DisabledReason,
						ErrorCode:    "",
						ErrorMessage: "",
						FlagMetadata: map[string]any{
							"description":  "this is a test",
							"defaultValue": float64(100),
						},
					},
				},
			},
		},
		{
			name: "should error if flag does not exist",
			args: args{
				flag:         "does_not_exists",
				defaultValue: nil,
				evalCtx:      defaultEvaluationCtx(),
			},
			want: openfeature.InterfaceEvaluationDetails{
				Value: nil,
				EvaluationDetails: openfeature.EvaluationDetails{
					FlagKey:  "does_not_exists",
					FlagType: openfeature.Object,
					ResolutionDetail: openfeature.ResolutionDetail{
						Variant:      "",
						Reason:       openfeature.ErrorReason,
						ErrorCode:    openfeature.FlagNotFoundCode,
						ErrorMessage: "does_not_exists",
						FlagMetadata: map[string]any{},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cli := inprocessMockClient{}
			provider := newInprocessProvider(t, &cli)

			err := openfeature.SetProviderAndWait(provider)
			require.NoError(t, err)
			client := openfeature.NewClient("test-app")
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

func TestProvider_EvaluationEnrichmentHook(t *testing.T) {
	tests := []struct {
		name             string
		want             string
		evalCtx          openfeature.EvaluationContext
		exporterMetadata map[string]any
	}{
		{
			name:             "should add the metadata to the evaluation context",
			exporterMetadata: map[string]any{"toto": 123, "tata": "titi"},
			evalCtx:          defaultEvaluationCtx(),
			want:             `{"context":{"admin":true,"age":30,"anonymous":false,"company_info":{"name":"my_company","size":120},"email":"john.doe@gofeatureflag.org","firstname":"john","gofeatureflag":{"exporterMetadata":{"openfeature":true,"provider":"go","tata":"titi","toto":123}},"labels":["pro","beta"],"lastname":"doe","professional":true,"rate":3.14,"targetingKey":"d45e303a-38c2-11ed-a261-0242ac120002"}}`,
		},
		{
			name:             "should have the default metadata if not provided",
			exporterMetadata: nil,
			evalCtx:          defaultEvaluationCtx(),
			want:             `{"context":{"admin":true,"age":30,"anonymous":false,"company_info":{"name":"my_company","size":120},"email":"john.doe@gofeatureflag.org","firstname":"john","gofeatureflag":{"exporterMetadata":{"openfeature":true,"provider":"go"}},"labels":["pro","beta"],"lastname":"doe","professional":true,"rate":3.14,"targetingKey":"d45e303a-38c2-11ed-a261-0242ac120002"}}`,
		},
		{
			name:             "should not remove other gofeatureflag specific metadata",
			exporterMetadata: map[string]any{"toto": 123, "tata": "titi"},
			evalCtx:          openfeature.NewEvaluationContext("d45e303a-38c2-11ed-a261-0242ac120002", map[string]any{"age": 30, "gofeatureflag": map[string]any{"flags": []string{"flag1", "flag2"}}}),
			want:             `{"context":{"age":30,"gofeatureflag":{"flags":["flag1","flag2"], "exporterMetadata":{"openfeature":true,"provider":"go","tata":"titi","toto":123}}, "targetingKey":"d45e303a-38c2-11ed-a261-0242ac120002"}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cli := evalMockClient{}
			options := ProviderOptions{
				Endpoint:         "https://gofeatureflag.org/",
				HTTPClient:       NewMockClient(cli.roundTripFunc),
				EvaluationType:   EvaluationTypeRemote,
				ExporterMetadata: tt.exporterMetadata,
			}
			provider, err := NewProvider(options)
			require.NoError(t, err)

			err = openfeature.SetProviderAndWait(provider)
			require.NoError(t, err)
			client := openfeature.NewClient("test-app")

			_, err = client.BooleanValueDetails(context.TODO(), "bool_targeting_match", false, tt.evalCtx)
			assert.NoError(t, err)

			assert.JSONEq(t, tt.want, cli.requestBodies[len(cli.requestBodies)-1])
		})
	}
}

// --------------------------------------------------------------------------
// Remote cache test infrastructure
// --------------------------------------------------------------------------

// remoteCacheMockClient is a RoundTripper-based mock for Remote evaluation tests.
// It routes requests to the appropriate handler and tracks call counts.
type remoteCacheMockClient struct {
	mu                 sync.Mutex
	ofrepCallCount     int
	collectorCallCount int
	collectorBodies    []string
	cacheable          bool // controls gofeatureflag_cacheable in OFREP response
}

func (m *remoteCacheMockClient) roundTripFunc(req *http.Request) *http.Response {
	m.mu.Lock()
	defer m.mu.Unlock()

	switch {
	case req.URL.Path == "/v1/flag/configuration":
		// Return 304 so the polling goroutine never purges the cache during tests.
		return &http.Response{StatusCode: http.StatusNotModified, Body: http.NoBody}

	case strings.HasPrefix(req.URL.Path, "/ofrep/v1/evaluate/flags/"):
		m.ofrepCallCount++
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(bytes.NewReader(cacheableOfrepFlagResponse(true, m.cacheable))),
		}

	case req.URL.Path == "/v1/data/collector":
		body, _ := io.ReadAll(req.Body)
		m.collectorCallCount++
		m.collectorBodies = append(m.collectorBodies, string(body))
		return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}
	}

	return &http.Response{StatusCode: http.StatusNotFound, Body: http.NoBody}
}

// cacheableOfrepFlagResponse builds a minimal OFREP boolean response.
// When cacheable is true the response includes gofeatureflag_cacheable=true so the
// Remote evaluator stores it in the cache.
func cacheableOfrepFlagResponse(value any, cacheable bool) []byte {
	meta := map[string]any{}
	if cacheable {
		meta["gofeatureflag_cacheable"] = true
	}
	b, _ := json.Marshal(map[string]any{
		"key":      "test-flag",
		"reason":   "TARGETING_MATCH",
		"variant":  "default",
		"value":    value,
		"metadata": meta,
	})
	return b
}

// newRemoteProvider creates a provider with EvaluationTypeRemote wired to the given mock client.
func newRemoteProvider(t *testing.T, cli *remoteCacheMockClient, cacheTTL time.Duration, cacheSize int, disableCache bool) *Provider {
	t.Helper()
	p, err := NewProviderWithContext(context.Background(), ProviderOptions{
		Endpoint:                  "https://gofeatureflag.org/",
		HTTPClient:                NewMockClient(cli.roundTripFunc),
		EvaluationType:            EvaluationTypeRemote,
		FlagCacheTTL:              cacheTTL,
		FlagCacheSize:             cacheSize,
		DisableCache:              disableCache,
		FlagChangePollingInterval: 24 * time.Hour, // prevent polling from firing during tests
	})
	require.NoError(t, err)
	return p
}

// --------------------------------------------------------------------------
// TestProvider_Remote_Cache
// --------------------------------------------------------------------------

func TestProvider_Remote_Cache(t *testing.T) {
	t.Run("same user hits cache on second call", func(t *testing.T) {
		cli := &remoteCacheMockClient{cacheable: true}
		provider := newRemoteProvider(t, cli, 5*time.Minute, 100, false)
		err := openfeature.SetProviderAndWait(provider)
		require.NoError(t, err)
		defer provider.ShutdownWithContext(context.Background()) //nolint:errcheck
		client := openfeature.NewClient("test-app")

		r1, err := client.BooleanValueDetails(context.TODO(), "test-flag", false, defaultEvaluationCtx())
		require.NoError(t, err)
		assert.Equal(t, openfeature.TargetingMatchReason, r1.Reason)

		r2, err := client.BooleanValueDetails(context.TODO(), "test-flag", false, defaultEvaluationCtx())
		require.NoError(t, err)
		assert.Equal(t, openfeature.CachedReason, r2.Reason)

		assert.Equal(t, 1, cli.ofrepCallCount)
	})

	t.Run("different contexts each miss the cache", func(t *testing.T) {
		cli := &remoteCacheMockClient{cacheable: true}
		provider := newRemoteProvider(t, cli, 5*time.Minute, 100, false)
		err := openfeature.SetProviderAndWait(provider)
		require.NoError(t, err)
		defer provider.ShutdownWithContext(context.Background()) //nolint:errcheck
		client := openfeature.NewClient("test-app")

		contexts := []openfeature.EvaluationContext{
			openfeature.NewEvaluationContext("ctx-1", nil),
			openfeature.NewEvaluationContext("ctx-2", nil),
			openfeature.NewEvaluationContext("ctx-3", nil),
			openfeature.NewEvaluationContext("ctx-4", nil),
		}
		for _, ctx := range contexts {
			r, err := client.BooleanValueDetails(context.TODO(), "test-flag", false, ctx)
			require.NoError(t, err)
			assert.NotEqual(t, openfeature.CachedReason, r.Reason)
		}
		assert.Equal(t, 4, cli.ofrepCallCount)
	})

	t.Run("LRU eviction re-fetches evicted entry", func(t *testing.T) {
		cli := &remoteCacheMockClient{cacheable: true}
		provider := newRemoteProvider(t, cli, 5*time.Minute, 2, false) // cache size = 2
		err := openfeature.SetProviderAndWait(provider)
		require.NoError(t, err)
		defer provider.ShutdownWithContext(context.Background()) //nolint:errcheck
		client := openfeature.NewClient("test-app")

		ctx1 := openfeature.NewEvaluationContext("ctx-1", nil)
		ctx2 := openfeature.NewEvaluationContext("ctx-2", nil)
		ctx3 := openfeature.NewEvaluationContext("ctx-3", nil)

		// ctx1 — miss then hit
		r, _ := client.BooleanValueDetails(context.TODO(), "test-flag", false, ctx1)
		assert.Equal(t, openfeature.TargetingMatchReason, r.Reason)
		r, _ = client.BooleanValueDetails(context.TODO(), "test-flag", false, ctx1)
		assert.Equal(t, openfeature.CachedReason, r.Reason)

		// ctx2 — miss then hit
		r, _ = client.BooleanValueDetails(context.TODO(), "test-flag", false, ctx2)
		assert.Equal(t, openfeature.TargetingMatchReason, r.Reason)
		r, _ = client.BooleanValueDetails(context.TODO(), "test-flag", false, ctx2)
		assert.Equal(t, openfeature.CachedReason, r.Reason)

		// ctx3 — evicts ctx1 (LRU, cache size 2)
		r, _ = client.BooleanValueDetails(context.TODO(), "test-flag", false, ctx3)
		assert.Equal(t, openfeature.TargetingMatchReason, r.Reason)

		// ctx1 — evicted, must re-fetch
		r, _ = client.BooleanValueDetails(context.TODO(), "test-flag", false, ctx1)
		assert.Equal(t, openfeature.TargetingMatchReason, r.Reason)

		assert.Equal(t, 4, cli.ofrepCallCount)
	})

	t.Run("TTL expiration triggers re-fetch", func(t *testing.T) {
		cli := &remoteCacheMockClient{cacheable: true}
		provider := newRemoteProvider(t, cli, 300*time.Millisecond, 100, false)
		err := openfeature.SetProviderAndWait(provider)
		require.NoError(t, err)
		defer provider.ShutdownWithContext(context.Background()) //nolint:errcheck
		client := openfeature.NewClient("test-app")

		_, err = client.BooleanValueDetails(context.TODO(), "test-flag", false, defaultEvaluationCtx())
		require.NoError(t, err)

		time.Sleep(500 * time.Millisecond) // wait for TTL to expire

		_, err = client.BooleanValueDetails(context.TODO(), "test-flag", false, defaultEvaluationCtx())
		require.NoError(t, err)

		assert.Equal(t, 2, cli.ofrepCallCount)
	})

	t.Run("non-cacheable flag is never cached", func(t *testing.T) {
		cli := &remoteCacheMockClient{cacheable: false} // metadata does not include cacheable=true
		provider := newRemoteProvider(t, cli, 5*time.Minute, 100, false)
		err := openfeature.SetProviderAndWait(provider)
		require.NoError(t, err)
		defer provider.ShutdownWithContext(context.Background()) //nolint:errcheck
		client := openfeature.NewClient("test-app")

		r1, err := client.BooleanValueDetails(context.TODO(), "test-flag", false, defaultEvaluationCtx())
		require.NoError(t, err)
		assert.NotEqual(t, openfeature.CachedReason, r1.Reason)

		r2, err := client.BooleanValueDetails(context.TODO(), "test-flag", false, defaultEvaluationCtx())
		require.NoError(t, err)
		assert.NotEqual(t, openfeature.CachedReason, r2.Reason)

		assert.Equal(t, 2, cli.ofrepCallCount)
	})

	t.Run("disabled cache always calls remote", func(t *testing.T) {
		cli := &remoteCacheMockClient{cacheable: true}
		provider := newRemoteProvider(t, cli, 5*time.Minute, 100, true) // DisableCache=true
		err := openfeature.SetProviderAndWait(provider)
		require.NoError(t, err)
		defer provider.ShutdownWithContext(context.Background()) //nolint:errcheck
		client := openfeature.NewClient("test-app")

		r1, err := client.BooleanValueDetails(context.TODO(), "test-flag", false, defaultEvaluationCtx())
		require.NoError(t, err)
		assert.NotEqual(t, openfeature.CachedReason, r1.Reason)

		r2, err := client.BooleanValueDetails(context.TODO(), "test-flag", false, defaultEvaluationCtx())
		require.NoError(t, err)
		assert.NotEqual(t, openfeature.CachedReason, r2.Reason)

		assert.Equal(t, 2, cli.ofrepCallCount)
	})
}

// --------------------------------------------------------------------------
// TestProvider_Remote_DataCollector
// --------------------------------------------------------------------------

func TestProvider_Remote_DataCollector(t *testing.T) {
	t.Run("cached evaluation sends event to collector with PROVIDER_CACHE source", func(t *testing.T) {
		cli := &remoteCacheMockClient{cacheable: true}
		p, err := NewProviderWithContext(context.Background(), ProviderOptions{
			Endpoint:                  "https://gofeatureflag.org/",
			HTTPClient:                NewMockClient(cli.roundTripFunc),
			EvaluationType:            EvaluationTypeRemote,
			FlagCacheTTL:              5 * time.Minute,
			FlagCacheSize:             100,
			FlagChangePollingInterval: 24 * time.Hour,
		})
		require.NoError(t, err)
		err = openfeature.SetProviderAndWait(p)
		require.NoError(t, err)
		defer p.ShutdownWithContext(context.Background()) //nolint:errcheck
		client := openfeature.NewClient("test-app")

		// First call: miss → hook skips collection (non-cached reason)
		_, err = client.BooleanValueDetails(context.TODO(), "test-flag", false, defaultEvaluationCtx())
		require.NoError(t, err)

		// Second call: hit → hook collects event (CachedReason)
		_, err = client.BooleanValueDetails(context.TODO(), "test-flag", false, defaultEvaluationCtx())
		require.NoError(t, err)

		// Flush the data collector
		require.NoError(t, p.dataCollectorMgr.SendData(context.Background()))

		assert.Equal(t, 1, cli.collectorCallCount)

		var payload struct {
			Events []json.RawMessage `json:"events"`
		}
		require.NoError(t, json.Unmarshal([]byte(cli.collectorBodies[0]), &payload))
		require.Len(t, payload.Events, 1)

		var event model.FeatureEvent
		require.NoError(t, json.Unmarshal(payload.Events[0], &event))
		assert.Equal(t, "PROVIDER_CACHE", event.Source)
	})

	t.Run("non-cached remote evaluation sends no event", func(t *testing.T) {
		cli := &remoteCacheMockClient{cacheable: true}
		p, err := NewProviderWithContext(context.Background(), ProviderOptions{
			Endpoint:                  "https://gofeatureflag.org/",
			HTTPClient:                NewMockClient(cli.roundTripFunc),
			EvaluationType:            EvaluationTypeRemote,
			DisableCache:              true, // cache off → all calls are non-cached
			FlagChangePollingInterval: 24 * time.Hour,
		})
		require.NoError(t, err)
		err = openfeature.SetProviderAndWait(p)
		require.NoError(t, err)
		defer p.ShutdownWithContext(context.Background()) //nolint:errcheck
		client := openfeature.NewClient("test-app")

		_, err = client.BooleanValueDetails(context.TODO(), "test-flag", false, defaultEvaluationCtx())
		require.NoError(t, err)

		require.NoError(t, p.dataCollectorMgr.SendData(context.Background()))

		assert.Equal(t, 0, cli.collectorCallCount)
	})
}
