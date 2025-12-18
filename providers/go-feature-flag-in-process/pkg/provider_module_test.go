package gofeatureflaginprocess_test

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	ffclient "github.com/thomaspoignant/go-feature-flag"
	"github.com/thomaspoignant/go-feature-flag/retriever/fileretriever"
	gofeatureflaginprocess "go.openfeature.dev/contrib/providers/go-feature-flag-in-process/v2/pkg"
	of "go.openfeature.dev/openfeature/v2"
)

func defaultEvaluationCtx() of.EvaluationContext {
	return of.NewEvaluationContext(
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

func TestProvider_module_BooleanEvaluation(t *testing.T) {
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
			name: "should resolve a valid boolean flag with TARGETING_MATCH reason",
			args: args{
				flag:         "bool_targeting_match",
				defaultValue: false,
				evalCtx:      defaultEvaluationCtx(),
			},
			want: of.BooleanEvaluationDetails{
				Value:    true,
				FlagKey:  "bool_targeting_match",
				FlagType: of.Boolean,
				ResolutionDetail: of.ResolutionDetail{
					Variant:      "True",
					Reason:       of.TargetingMatchReason,
					ErrorCode:    "",
					ErrorMessage: "",
					FlagMetadata: map[string]any{},
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
				Value:    false,
				FlagKey:  "disabled_bool",
				FlagType: of.Boolean,
				ResolutionDetail: of.ResolutionDetail{
					Variant:      "SdkDefault",
					Reason:       of.DisabledReason,
					ErrorCode:    "",
					ErrorMessage: "",
					FlagMetadata: map[string]any{},
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
				Value:    false,
				FlagKey:  "string_key",
				FlagType: of.Boolean,
				ResolutionDetail: of.ResolutionDetail{
					Variant:      "",
					Reason:       of.ErrorReason,
					ErrorCode:    of.TypeMismatchCode,
					ErrorMessage: "unexpected type for flag string_key",
					FlagMetadata: map[string]any{},
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
				Value:    false,
				FlagKey:  "does_not_exists",
				FlagType: of.Boolean,
				ResolutionDetail: of.ResolutionDetail{
					Variant:      "",
					Reason:       of.ErrorReason,
					ErrorCode:    of.FlagNotFoundCode,
					ErrorMessage: "flag does_not_exists was not found in GO Feature Flag",
					FlagMetadata: map[string]any{},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := gofeatureflaginprocess.NewProvider(gofeatureflaginprocess.ProviderOptions{
				GOFeatureFlagConfig: &ffclient.Config{
					PollingInterval: 10 * time.Second,
					Logger:          log.New(os.Stdout, "", 0),
					Context:         context.Background(),
					Retriever: &fileretriever.Retriever{
						Path: "../testutils/module/flags.yaml",
					},
				},
			})
			assert.NoError(t, err)

			err = of.SetProviderAndWait(t.Context(), provider)
			assert.NoError(t, err)
			client := of.NewClient("test-app")
			value, err := client.BooleanValueDetails(context.TODO(), tt.args.flag, tt.args.defaultValue, tt.args.evalCtx)

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

func TestProvider_module_StringEvaluation(t *testing.T) {
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
				Value:    "CC0000",
				FlagKey:  "string_key",
				FlagType: of.String,
				ResolutionDetail: of.ResolutionDetail{
					Variant:      "True",
					Reason:       of.TargetingMatchReason,
					ErrorCode:    "",
					ErrorMessage: "",
					FlagMetadata: map[string]any{},
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
				Value:    "default",
				FlagKey:  "disabled_string",
				FlagType: of.String,
				ResolutionDetail: of.ResolutionDetail{
					Variant:      "SdkDefault",
					Reason:       of.DisabledReason,
					ErrorCode:    "",
					ErrorMessage: "",
					FlagMetadata: map[string]any{},
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
				Value:    "default",
				FlagKey:  "bool_targeting_match",
				FlagType: of.String,
				ResolutionDetail: of.ResolutionDetail{
					Variant:      "",
					Reason:       of.ErrorReason,
					ErrorCode:    of.TypeMismatchCode,
					ErrorMessage: "unexpected type for flag bool_targeting_match",
					FlagMetadata: map[string]any{},
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
				Value:    "default",
				FlagKey:  "does_not_exists",
				FlagType: of.String,
				ResolutionDetail: of.ResolutionDetail{
					Variant:      "",
					Reason:       of.ErrorReason,
					ErrorCode:    of.FlagNotFoundCode,
					ErrorMessage: "flag does_not_exists was not found in GO Feature Flag",
					FlagMetadata: map[string]any{},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := gofeatureflaginprocess.NewProvider(gofeatureflaginprocess.ProviderOptions{
				GOFeatureFlagConfig: &ffclient.Config{
					PollingInterval: 10 * time.Second,
					Logger:          log.New(os.Stdout, "", 0),
					Context:         context.Background(),
					Retriever: &fileretriever.Retriever{
						Path: "../testutils/module/flags.yaml",
					},
				},
			})
			assert.NoError(t, err)

			err = of.SetProviderAndWait(t.Context(), provider)
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

func TestProvider_module_FloatEvaluation(t *testing.T) {
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
				Value:    100.25,
				FlagKey:  "double_key",
				FlagType: of.Float,
				ResolutionDetail: of.ResolutionDetail{
					Variant:      "True",
					Reason:       of.TargetingMatchReason,
					ErrorCode:    "",
					ErrorMessage: "",
					FlagMetadata: map[string]any{},
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
				Value:    123.45,
				FlagKey:  "disabled_float",
				FlagType: of.Float,
				ResolutionDetail: of.ResolutionDetail{
					Variant:      "SdkDefault",
					Reason:       of.DisabledReason,
					ErrorCode:    "",
					ErrorMessage: "",
					FlagMetadata: map[string]any{},
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
				Value:    123.45,
				FlagKey:  "bool_targeting_match",
				FlagType: of.Float,
				ResolutionDetail: of.ResolutionDetail{
					Variant:      "",
					Reason:       of.ErrorReason,
					ErrorCode:    of.TypeMismatchCode,
					ErrorMessage: "unexpected type for flag bool_targeting_match",
					FlagMetadata: map[string]any{},
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
				Value:    123.45,
				FlagKey:  "does_not_exists",
				FlagType: of.Float,
				ResolutionDetail: of.ResolutionDetail{
					Variant:      "",
					Reason:       of.ErrorReason,
					ErrorCode:    of.FlagNotFoundCode,
					ErrorMessage: "flag does_not_exists was not found in GO Feature Flag",
					FlagMetadata: map[string]any{},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := gofeatureflaginprocess.NewProvider(gofeatureflaginprocess.ProviderOptions{
				GOFeatureFlagConfig: &ffclient.Config{
					PollingInterval: 10 * time.Second,
					Logger:          log.New(os.Stdout, "", 0),
					Context:         context.Background(),
					Retriever: &fileretriever.Retriever{
						Path: "../testutils/module/flags.yaml",
					},
				},
			})
			assert.NoError(t, err)

			err = of.SetProviderAndWait(t.Context(), provider)
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

func TestProvider_module_IntEvaluation(t *testing.T) {
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
				Value:    100,
				FlagKey:  "integer_key",
				FlagType: of.Int,
				ResolutionDetail: of.ResolutionDetail{
					Variant:      "True",
					Reason:       of.TargetingMatchReason,
					ErrorCode:    "",
					ErrorMessage: "",
					FlagMetadata: map[string]any{},
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
				Value:    123,
				FlagKey:  "disabled_int",
				FlagType: of.Int,
				ResolutionDetail: of.ResolutionDetail{
					Variant:      "SdkDefault",
					Reason:       of.DisabledReason,
					ErrorCode:    "",
					ErrorMessage: "",
					FlagMetadata: map[string]any{},
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
				Value:    123,
				FlagKey:  "bool_targeting_match",
				FlagType: of.Int,
				ResolutionDetail: of.ResolutionDetail{
					Variant:      "",
					Reason:       of.ErrorReason,
					ErrorCode:    of.TypeMismatchCode,
					ErrorMessage: "unexpected type for flag bool_targeting_match",
					FlagMetadata: map[string]any{},
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
				Value:    123,
				FlagKey:  "does_not_exists",
				FlagType: of.Int,
				ResolutionDetail: of.ResolutionDetail{
					Variant:      "",
					Reason:       of.ErrorReason,
					ErrorCode:    of.FlagNotFoundCode,
					ErrorMessage: "flag does_not_exists was not found in GO Feature Flag",
					FlagMetadata: map[string]any{},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := gofeatureflaginprocess.NewProvider(gofeatureflaginprocess.ProviderOptions{
				GOFeatureFlagConfig: &ffclient.Config{
					PollingInterval: 10 * time.Second,
					Logger:          log.New(os.Stdout, "", 0),
					Context:         context.Background(),
					Retriever: &fileretriever.Retriever{
						Path: "../testutils/module/flags.yaml",
					},
				},
			})
			assert.NoError(t, err)

			err = of.SetProviderAndWait(t.Context(), provider)
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

func TestProvider_module_ObjectEvaluation(t *testing.T) {
	type args struct {
		flag         string
		defaultValue any
		evalCtx      of.EvaluationContext
	}
	tests := []struct {
		name string
		args args
		want of.ObjectEvaluationDetails
	}{
		{
			name: "should resolve a valid interface flag with TARGETING_MATCH reason",
			args: args{
				flag:         "object_key",
				defaultValue: nil,
				evalCtx:      defaultEvaluationCtx(),
			},
			want: of.ObjectEvaluationDetails{
				Value: map[string]any{
					"test":  "test1",
					"test2": false,
					"test3": 123.3,
					"test4": 1,
				},
				FlagKey:  "object_key",
				FlagType: of.Object,
				ResolutionDetail: of.ResolutionDetail{
					Variant:      "True",
					Reason:       of.TargetingMatchReason,
					ErrorCode:    "",
					ErrorMessage: "",
					FlagMetadata: map[string]any{},
				},
			},
		},
		{
			name: "should use interface default value if the flag is disabled",
			args: args{
				flag:         "disabled_interface",
				defaultValue: nil,
				evalCtx:      defaultEvaluationCtx(),
			},
			want: of.ObjectEvaluationDetails{
				Value:    nil,
				FlagKey:  "disabled_interface",
				FlagType: of.Object,
				ResolutionDetail: of.ResolutionDetail{
					Variant:      "SdkDefault",
					Reason:       of.DisabledReason,
					ErrorCode:    "",
					ErrorMessage: "",
					FlagMetadata: map[string]any{},
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
			want: of.ObjectEvaluationDetails{
				Value:    nil,
				FlagKey:  "does_not_exists",
				FlagType: of.Object,
				ResolutionDetail: of.ResolutionDetail{
					Variant:      "",
					Reason:       of.ErrorReason,
					ErrorCode:    of.FlagNotFoundCode,
					ErrorMessage: "flag does_not_exists was not found in GO Feature Flag",
					FlagMetadata: map[string]any{},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := gofeatureflaginprocess.NewProvider(gofeatureflaginprocess.ProviderOptions{
				GOFeatureFlagConfig: &ffclient.Config{
					PollingInterval: 10 * time.Second,
					Logger:          log.New(os.Stdout, "", 0),
					Context:         context.Background(),
					Retriever: &fileretriever.Retriever{
						Path: "../testutils/module/flags.yaml",
					},
				},
			})
			assert.NoError(t, err)

			err = of.SetProviderAndWait(t.Context(), provider)
			assert.NoError(t, err)
			client := of.NewClient("test-app")
			value, err := client.ObjectValueDetails(context.TODO(), tt.args.flag, tt.args.defaultValue, tt.args.evalCtx)

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
