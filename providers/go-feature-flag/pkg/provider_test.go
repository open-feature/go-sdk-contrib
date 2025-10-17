package gofeatureflag_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	gofeatureflag "github.com/open-feature/go-sdk-contrib/providers/go-feature-flag/pkg"
	"github.com/open-feature/go-sdk-contrib/providers/go-feature-flag/pkg/gofferror"
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
						FlagMetadata: map[string]interface{}{},
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
						FlagMetadata: map[string]interface{}{},
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
						FlagMetadata: map[string]interface{}{},
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
						FlagMetadata: map[string]interface{}{},
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
						FlagMetadata: map[string]interface{}{},
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
						FlagMetadata: map[string]interface{}{},
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
				EvaluationType: gofeatureflag.EvaluationTypeRemote,
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
