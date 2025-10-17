package gofeatureflag_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	gofeatureflag "github.com/open-feature/go-sdk-contrib/providers/go-feature-flag/pkg"
	"github.com/open-feature/go-sdk-contrib/providers/go-feature-flag/pkg/gofferror"
	"github.com/open-feature/go-sdk-contrib/providers/go-feature-flag/pkg/model"
	"github.com/open-feature/go-sdk-contrib/providers/go-feature-flag/pkg/testutils"
	"github.com/open-feature/go-sdk-contrib/providers/go-feature-flag/pkg/testutils/mock"
	"github.com/open-feature/go-sdk/openfeature"
)

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

func TestNewProvider_InvalidOptions(t *testing.T) {
	tests := []struct {
		name        string
		options     gofeatureflag.ProviderOptions
		wantErr     bool
		ctx         context.Context
		expectedErr gofferror.InvalidOption
	}{
		{
			name: "empty endpoint should return error",
			options: gofeatureflag.ProviderOptions{
				Endpoint: "",
			},
			wantErr:     true,
			expectedErr: gofferror.NewInvalidOption("invalid option: endpoint is required"),
		},
		{
			name: "endpoint without scheme should return error",
			options: gofeatureflag.ProviderOptions{
				Endpoint: "localhost:1031",
			},
			wantErr:     true,
			expectedErr: gofferror.NewInvalidOption("invalid option: endpoint must have http or https scheme"),
		},
		{
			name: "endpoint with invalid scheme should return error",
			options: gofeatureflag.ProviderOptions{
				Endpoint: "ftp://example.com",
			},
			wantErr:     true,
			expectedErr: gofferror.NewInvalidOption("invalid option: endpoint must have http or https scheme"),
		},
		{
			name: "endpoint without host should return error",
			options: gofeatureflag.ProviderOptions{
				Endpoint: "http://",
			},
			wantErr:     true,
			expectedErr: gofferror.NewInvalidOption("invalid option: endpoint must have a valid host"),
		},
		{
			name: "valid http endpoint should not return error",
			options: gofeatureflag.ProviderOptions{
				Endpoint: "http://localhost:1031",
			},
			wantErr: false,
		},
		{
			name: "valid https endpoint should not return error",
			options: gofeatureflag.ProviderOptions{
				Endpoint: "https://example.com",
			},
			wantErr: false,
		},
		{
			name: "valid endpoint with API key should not return error",
			options: gofeatureflag.ProviderOptions{
				Endpoint: "http://localhost:1031",
				APIKey:   "test-api-key",
			},
			wantErr: false,
		},
		{
			name: "valid endpoint with evaluation type should not return error",
			options: gofeatureflag.ProviderOptions{
				Endpoint:       "https://example.com",
				EvaluationType: gofeatureflag.EvaluationTypeInProcess,
			},
			wantErr: false,
		},
		{
			name: "valid endpoint with path should not return error",
			options: gofeatureflag.ProviderOptions{
				Endpoint: "https://example.com/api/v1",
			},
			wantErr: false,
		},
		{
			name: "valid endpoint with custom context should not return error",
			ctx:  context.WithValue(context.Background(), contextKey("key"), "value"),
			options: gofeatureflag.ProviderOptions{
				Endpoint: "https://example.com",
			},
			wantErr: false,
		},
		{
			name: "empty endpoint with custom context should return error",
			ctx:  context.WithValue(context.Background(), contextKey("key"), "value"),
			options: gofeatureflag.ProviderOptions{
				Endpoint: "",
			},
			wantErr:     true,
			expectedErr: gofferror.NewInvalidOption("invalid option: endpoint is required"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Run(tt.name+"- without context", func(t *testing.T) {
				provider, err := gofeatureflag.NewProvider(tt.options)
				if tt.wantErr {
					require.Error(t, err, "Expected error but got nil")
					require.ErrorIs(t, err, tt.expectedErr)
					require.Nil(t, provider)
				} else {
					require.NoError(t, err)
					require.NotNil(t, provider)
				}
			})

			t.Run(tt.name+"- with context", func(t *testing.T) {
				if tt.ctx == nil {
					tt.ctx = context.Background()
				}
				provider, err := gofeatureflag.NewProviderWithContext(tt.ctx, tt.options)
				if tt.wantErr {
					require.Error(t, err, "Expected error but got nil")
					require.ErrorIs(t, err, tt.expectedErr)
					require.Nil(t, provider)
				} else {
					require.NoError(t, err)
					require.NotNil(t, provider)
				}
			})
		})
	}
}

func TestProvider_BooleanEvaluation(t *testing.T) {
	type args struct {
		flag           string
		defaultValue   bool
		evalCtx        openfeature.EvaluationContext
		evaluationType gofeatureflag.EvaluationType
	}
	tests := []struct {
		name string
		args args
		want openfeature.BooleanEvaluationDetails
	}{
		{
			name: "unauthorized flag",
			args: args{
				flag:           "unauthorized",
				defaultValue:   false,
				evalCtx:        testutils.DefaultEvaluationContext,
				evaluationType: gofeatureflag.EvaluationTypeRemote,
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
				flag:           "bool_targeting_match",
				defaultValue:   false,
				evalCtx:        testutils.DefaultEvaluationContext,
				evaluationType: gofeatureflag.EvaluationTypeRemote,
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
				flag:           "disabled_bool",
				defaultValue:   false,
				evalCtx:        testutils.DefaultEvaluationContext,
				evaluationType: gofeatureflag.EvaluationTypeRemote,
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
				flag:           "string_key",
				defaultValue:   false,
				evalCtx:        testutils.DefaultEvaluationContext,
				evaluationType: gofeatureflag.EvaluationTypeRemote,
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
				flag:           "does_not_exists",
				defaultValue:   false,
				evalCtx:        testutils.DefaultEvaluationContext,
				evaluationType: gofeatureflag.EvaluationTypeRemote,
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
				flag:           "unknown_reason",
				defaultValue:   false,
				evalCtx:        testutils.DefaultEvaluationContext,
				evaluationType: gofeatureflag.EvaluationTypeRemote,
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
			name: "should return error if no targeting key returned by relay proxy",
			args: args{
				flag:           "targetingkey_missing",
				defaultValue:   false,
				evalCtx:        openfeature.EvaluationContext{},
				evaluationType: gofeatureflag.EvaluationTypeRemote,
			},
			want: openfeature.BooleanEvaluationDetails{
				Value: false,
				EvaluationDetails: openfeature.EvaluationDetails{
					FlagKey:  "targetingkey_missing",
					FlagType: openfeature.Boolean,
					ResolutionDetail: openfeature.ResolutionDetail{
						Reason:       openfeature.ErrorReason,
						ErrorCode:    openfeature.TargetingKeyMissingCode,
						ErrorMessage: "",
						FlagMetadata: map[string]any{},
					},
				},
			},
		},
		{
			name: "should return an error if invalid json body",
			args: args{
				flag:           "invalid_json_body",
				defaultValue:   false,
				evalCtx:        testutils.DefaultEvaluationContext,
				evaluationType: gofeatureflag.EvaluationTypeRemote,
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
			options := gofeatureflag.ProviderOptions{
				Endpoint:       "https://gofeatureflag.org/",
				HTTPClient:     mock.NewDefaultMockClient(),
				EvaluationType: tt.args.evaluationType,
			}
			provider, err := gofeatureflag.NewProvider(options)
			assert.NoError(t, err)

			err = openfeature.SetNamedProviderAndWait(tt.name, provider)
			require.NoError(t, err)
			client := openfeature.NewClient(tt.name)
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
		flag           string
		defaultValue   string
		evalCtx        openfeature.EvaluationContext
		evaluationType gofeatureflag.EvaluationType
	}
	tests := []struct {
		name string
		args args
		want openfeature.StringEvaluationDetails
	}{
		{
			name: "should resolve a valid string flag with TARGETING_MATCH reason",
			args: args{
				flag:           "string_key",
				defaultValue:   "default",
				evalCtx:        testutils.DefaultEvaluationContext,
				evaluationType: gofeatureflag.EvaluationTypeRemote,
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
				flag:           "disabled_string",
				defaultValue:   "default",
				evalCtx:        testutils.DefaultEvaluationContext,
				evaluationType: gofeatureflag.EvaluationTypeRemote,
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
				flag:           "bool_targeting_match",
				defaultValue:   "default",
				evalCtx:        testutils.DefaultEvaluationContext,
				evaluationType: gofeatureflag.EvaluationTypeRemote,
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
				flag:           "does_not_exists",
				defaultValue:   "default",
				evalCtx:        testutils.DefaultEvaluationContext,
				evaluationType: gofeatureflag.EvaluationTypeRemote,
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
			options := gofeatureflag.ProviderOptions{
				Endpoint:       "https://gofeatureflag.org/",
				HTTPClient:     mock.NewDefaultMockClient(),
				EvaluationType: tt.args.evaluationType,
			}
			provider, err := gofeatureflag.NewProvider(options)
			assert.NoError(t, err)

			err = openfeature.SetProviderAndWait(provider)
			assert.NoError(t, err)
			client := openfeature.NewClient("test-app")
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
		flag           string
		defaultValue   float64
		evalCtx        openfeature.EvaluationContext
		evaluationType gofeatureflag.EvaluationType
	}
	tests := []struct {
		name string
		args args
		want openfeature.FloatEvaluationDetails
	}{
		{
			name: "should resolve a valid float flag with TARGETING_MATCH reason",
			args: args{
				flag:           "double_key",
				defaultValue:   123.45,
				evalCtx:        testutils.DefaultEvaluationContext,
				evaluationType: gofeatureflag.EvaluationTypeRemote,
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
				flag:           "disabled_float",
				defaultValue:   123.45,
				evalCtx:        testutils.DefaultEvaluationContext,
				evaluationType: gofeatureflag.EvaluationTypeRemote,
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
			name: "should error if we expect a string and got another type",
			args: args{
				flag:           "bool_targeting_match",
				defaultValue:   123.45,
				evalCtx:        testutils.DefaultEvaluationContext,
				evaluationType: gofeatureflag.EvaluationTypeRemote,
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
				flag:           "does_not_exists",
				defaultValue:   123.45,
				evalCtx:        testutils.DefaultEvaluationContext,
				evaluationType: gofeatureflag.EvaluationTypeRemote,
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
			options := gofeatureflag.ProviderOptions{
				Endpoint:       "https://gofeatureflag.org/",
				HTTPClient:     mock.NewDefaultMockClient(),
				EvaluationType: tt.args.evaluationType,
			}
			provider, err := gofeatureflag.NewProvider(options)
			assert.NoError(t, err)

			err = openfeature.SetProviderAndWait(provider)
			assert.NoError(t, err)
			client := openfeature.NewClient("test-app")
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
		flag           string
		defaultValue   int64
		evalCtx        openfeature.EvaluationContext
		evaluationType gofeatureflag.EvaluationType
	}
	tests := []struct {
		name string
		args args
		want openfeature.IntEvaluationDetails
	}{
		{
			name: "should resolve a valid float flag with TARGETING_MATCH reason",
			args: args{
				flag:           "integer_key",
				defaultValue:   123,
				evalCtx:        testutils.DefaultEvaluationContext,
				evaluationType: gofeatureflag.EvaluationTypeRemote,
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
						FlagMetadata: map[string]interface{}{},
					},
				},
			},
		},
		{
			name: "should use float default value if the flag is disabled",
			args: args{
				flag:           "disabled_int",
				defaultValue:   123,
				evalCtx:        testutils.DefaultEvaluationContext,
				evaluationType: gofeatureflag.EvaluationTypeRemote,
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
						FlagMetadata: map[string]interface{}{},
					},
				},
			},
		},
		{
			name: "should error if we expect a string and got another type",
			args: args{
				flag:           "bool_targeting_match",
				defaultValue:   123,
				evalCtx:        testutils.DefaultEvaluationContext,
				evaluationType: gofeatureflag.EvaluationTypeRemote,
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
						FlagMetadata: map[string]interface{}{},
					},
				},
			},
		},
		{
			name: "should error if flag does not exists",
			args: args{
				flag:           "does_not_exists",
				defaultValue:   123,
				evalCtx:        testutils.DefaultEvaluationContext,
				evaluationType: gofeatureflag.EvaluationTypeRemote,
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
						FlagMetadata: map[string]interface{}{},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := gofeatureflag.ProviderOptions{
				Endpoint:       "https://gofeatureflag.org/",
				HTTPClient:     mock.NewDefaultMockClient(),
				EvaluationType: tt.args.evaluationType,
			}
			provider, err := gofeatureflag.NewProvider(options)
			assert.NoError(t, err)

			err = openfeature.SetProviderAndWait(provider)
			assert.NoError(t, err)
			client := openfeature.NewClient("test-app")
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
		flag           string
		defaultValue   interface{}
		evalCtx        openfeature.EvaluationContext
		evaluationType gofeatureflag.EvaluationType
	}
	tests := []struct {
		name string
		args args
		want openfeature.InterfaceEvaluationDetails
	}{
		{
			name: "should resolve a valid interface flag with TARGETING_MATCH reason",
			args: args{
				flag:           "object_key",
				defaultValue:   nil,
				evalCtx:        testutils.DefaultEvaluationContext,
				evaluationType: gofeatureflag.EvaluationTypeRemote,
			},
			want: openfeature.InterfaceEvaluationDetails{
				Value: map[string]interface{}{
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
						FlagMetadata: map[string]interface{}{},
					},
				},
			},
		},
		{
			name: "should use interface default value if the flag is disabled",
			args: args{
				flag:           "disabled_int",
				defaultValue:   nil,
				evalCtx:        testutils.DefaultEvaluationContext,
				evaluationType: gofeatureflag.EvaluationTypeRemote,
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
						FlagMetadata: map[string]interface{}{},
					},
				},
			},
		},
		{
			name: "should error if flag does not exists",
			args: args{
				flag:           "does_not_exists",
				defaultValue:   nil,
				evalCtx:        testutils.DefaultEvaluationContext,
				evaluationType: gofeatureflag.EvaluationTypeRemote,
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
						FlagMetadata: map[string]interface{}{},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := gofeatureflag.ProviderOptions{
				Endpoint:       "https://gofeatureflag.org/",
				HTTPClient:     mock.NewDefaultMockClient(),
				EvaluationType: tt.args.evaluationType,
			}
			provider, err := gofeatureflag.NewProvider(options)
			assert.NoError(t, err)

			err = openfeature.SetProviderAndWait(provider)
			assert.NoError(t, err)
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

func TestProvider_DataCollectorHook(t *testing.T) {
	rt := mock.RoundTripper{}
	rt.RoundTripFunc = rt.DefaultRoundTripFunc
	client := mock.NewMockClient(rt.RoundTripFunc, nil)
	t.Run("Should not call the /data/collector endpoint if remote evaluation type", func(t *testing.T) {
		options := gofeatureflag.ProviderOptions{
			Endpoint:             "https://gofeatureflag.org/",
			HTTPClient:           client,
			DataFlushInterval:    100 * time.Millisecond,
			DisableDataCollector: false,
			ExporterMetadata:     map[string]interface{}{"toto": 123, "tata": "titi"},
			EvaluationType:       gofeatureflag.EvaluationTypeRemote,
		}
		provider, err := gofeatureflag.NewProvider(options)
		defer provider.Shutdown()
		assert.NoError(t, err)
		err = openfeature.SetProviderAndWait(provider)
		assert.NoError(t, err)
		client := openfeature.NewClient("test-app")

		_ = client.Boolean(context.TODO(), "bool_targeting_match", false, testutils.DefaultEvaluationContext)
		_ = client.Boolean(context.TODO(), "bool_targeting_match", false, testutils.DefaultEvaluationContext)
		_ = client.Boolean(context.TODO(), "bool_targeting_match", false, testutils.DefaultEvaluationContext)
		_ = client.Boolean(context.TODO(), "bool_targeting_match", false, testutils.DefaultEvaluationContext)
		_ = client.Boolean(context.TODO(), "bool_targeting_match", false, testutils.DefaultEvaluationContext)
		_ = client.Boolean(context.TODO(), "bool_targeting_match", false, testutils.DefaultEvaluationContext)
		_ = client.Boolean(context.TODO(), "bool_targeting_match", false, testutils.DefaultEvaluationContext)
		_ = client.Boolean(context.TODO(), "bool_targeting_match", false, testutils.DefaultEvaluationContext)
		_ = client.Boolean(context.TODO(), "bool_targeting_match", false, testutils.DefaultEvaluationContext)

		time.Sleep(500 * time.Millisecond)
		assert.Equal(t, 9, rt.CallCount)
		assert.Equal(t, 0, rt.CollectorCallCount)

		// convert cli.collectorRequests[0] to  DataCollectorRequest
		var dataCollectorRequest model.DataCollectorRequest
		err = json.Unmarshal([]byte(rt.CollectorRequests[0]), &dataCollectorRequest)
		assert.NoError(t, err)
		assert.Equal(t, map[string]interface{}{
			"openfeature": true,
			"provider":    "go",
			"tata":        "titi",
			"toto":        float64(123),
		}, dataCollectorRequest.Meta)
	})
}

func TestProvider_EvaluationEnrichmentHook(t *testing.T) {
	tests := []struct {
		name             string
		want             string
		evalCtx          openfeature.EvaluationContext
		exporterMetadata map[string]interface{}
		evaluationType   gofeatureflag.EvaluationType
	}{
		{
			name:             "should add the metadata to the evaluation context",
			exporterMetadata: map[string]interface{}{"toto": 123, "tata": "titi"},
			evalCtx:          testutils.DefaultEvaluationContext,
			want:             `{"context":{"admin":true,"age":30,"anonymous":false,"company_info":{"name":"my_company","size":120},"email":"john.doe@gofeatureflag.org","firstname":"john","gofeatureflag":{"exporterMetadata":{"openfeature":true,"provider":"go","tata":"titi","toto":123}},"labels":["pro","beta"],"lastname":"doe","professional":true,"rate":3.14,"targetingKey":"d45e303a-38c2-11ed-a261-0242ac120002"}}`,
			evaluationType:   gofeatureflag.EvaluationTypeRemote,
		},
		{
			name:             "should have the default metadata if not provided",
			exporterMetadata: nil,
			evalCtx:          testutils.DefaultEvaluationContext,
			want:             `{"context":{"admin":true,"age":30,"anonymous":false,"company_info":{"name":"my_company","size":120},"email":"john.doe@gofeatureflag.org","firstname":"john","gofeatureflag":{"exporterMetadata":{"openfeature":true,"provider":"go"}},"labels":["pro","beta"],"lastname":"doe","professional":true,"rate":3.14,"targetingKey":"d45e303a-38c2-11ed-a261-0242ac120002"}}`,
			evaluationType:   gofeatureflag.EvaluationTypeRemote,
		},
		{
			name:             "should not remove other gofeatureflag specific metadata",
			exporterMetadata: map[string]interface{}{"toto": 123, "tata": "titi"},
			evalCtx:          openfeature.NewEvaluationContext("d45e303a-38c2-11ed-a261-0242ac120002", map[string]interface{}{"age": 30, "gofeatureflag": map[string]interface{}{"flags": []string{"flag1", "flag2"}}}),
			want:             `{"context":{"age":30,"gofeatureflag":{"flags":["flag1","flag2"], "exporterMetadata":{"openfeature":true,"provider":"go","tata":"titi","toto":123}}, "targetingKey":"d45e303a-38c2-11ed-a261-0242ac120002"}}`,
			evaluationType:   gofeatureflag.EvaluationTypeRemote,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rt := mock.RoundTripper{}
			rt.RoundTripFunc = rt.DefaultRoundTripFunc
			mockClient := mock.NewMockClient(rt.RoundTripFunc, nil)
			options := gofeatureflag.ProviderOptions{
				Endpoint:         "https://gofeatureflag.org/",
				HTTPClient:       mockClient,
				ExporterMetadata: tt.exporterMetadata,
				EvaluationType:   tt.evaluationType,
			}
			provider, err := gofeatureflag.NewProvider(options)
			defer provider.Shutdown()
			assert.NoError(t, err)
			err = openfeature.SetNamedProviderAndWait(tt.name, provider)
			assert.NoError(t, err)
			client := openfeature.NewClient(tt.name)

			_, err = client.BooleanValueDetails(context.TODO(), "bool_targeting_match", false, tt.evalCtx)
			assert.NoError(t, err)

			want := tt.want
			got := rt.RequestBodies[0]
			assert.JSONEq(t, want, got)
		})
	}
}
