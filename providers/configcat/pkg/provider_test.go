package configcat_test

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/configcat/go-sdk/v9/configcattest"

	sdk "github.com/configcat/go-sdk/v9"
	"github.com/stretchr/testify/require"
	configcat "go.openfeature.dev/contrib/providers/configcat/v2/pkg"
	"go.openfeature.dev/openfeature/v2"
)

func TestMetadata(t *testing.T) {
	client, _, _, _ := newTestServer(t)
	provider := configcat.NewProvider(client)
	require.Equal(t, openfeature.Metadata{
		Name: "ConfigCat",
	}, provider.Metadata())
}

func TestHooks(t *testing.T) {
	client, _, _, _ := newTestServer(t)
	provider := configcat.NewProvider(client)
	require.Len(t, provider.Hooks(), 0)
}

func TestBooleanEvaluation(t *testing.T) {
	ctx := context.Background()
	client, flagSrv, hooks, sdkKey := newTestServer(t)
	provider := configcat.NewProvider(client)
	defaultFlag := configcattest.Flag{Default: true}

	tests := []struct {
		name       string
		key        string
		defaultVal bool
		expVal     bool
		expVariant string
		errMsg     string
		errCode    openfeature.ErrorCode
		reason     openfeature.Reason
		evalCtx    map[string]any
		flag       configcattest.Flag
	}{
		{
			name:       "evalCtx empty",
			key:        "flag",
			defaultVal: false,
			expVal:     true,
			expVariant: "v_flag",
			errMsg:     "",
			errCode:    "",
			reason:     openfeature.DefaultReason,
			evalCtx:    nil,
			flag:       defaultFlag,
		},
		{
			name:       "key not found",
			key:        "non-existing",
			defaultVal: false,
			expVal:     false,
			expVariant: "",
			errMsg:     "failed to evaluate setting 'non-existing' (the key was not found in config JSON); available keys: ['flag']",
			errCode:    openfeature.FlagNotFoundCode,
			reason:     openfeature.ErrorReason,
			evalCtx:    nil,
			flag:       defaultFlag,
		},
		{
			name:       "type mismatch",
			key:        "flag",
			defaultVal: false,
			expVal:     false,
			expVariant: "",
			errMsg:     "the type of the setting 'flag' doesn't match with the expected type; setting's type was 'int' but the expected type was 'bool'",
			errCode:    openfeature.TypeMismatchCode,
			reason:     openfeature.ErrorReason,
			evalCtx:    nil,
			flag:       configcattest.Flag{Default: 5},
		},
		{
			name:       "unknown error",
			key:        "flag",
			defaultVal: false,
			expVal:     false,
			expVariant: "",
			errMsg:     "comparison value '<nil>' is invalid",
			errCode:    openfeature.GeneralCode,
			reason:     openfeature.ErrorReason,
			evalCtx:    openfeature.FlattenedContext{"attr": "val"},
			flag: configcattest.Flag{
				Default: true,
				Rules: []configcattest.Rule{{
					Value:               true,
					Comparator:          sdk.OpStartsWithAnyOfHashed,
					ComparisonAttribute: "attr",
					ComparisonValue:     "invalidnothashed",
				}},
			},
		},
		{
			name:       "matched evaluation rule",
			key:        "flag",
			defaultVal: false,
			expVal:     true,
			expVariant: "v0_flag",
			errMsg:     "",
			errCode:    "",
			reason:     openfeature.TargetingMatchReason,
			evalCtx:    openfeature.FlattenedContext{"attr": "val"},
			flag: configcattest.Flag{
				Default: false,
				Rules: []configcattest.Rule{{
					Value:               true,
					Comparator:          sdk.OpEq,
					ComparisonAttribute: "attr",
					ComparisonValue:     "val",
				}},
			},
		},
		{
			name:       "matched percentage rule",
			key:        "flag",
			defaultVal: false,
			expVal:     true,
			expVariant: "v0_flag",
			errMsg:     "",
			errCode:    "",
			reason:     openfeature.TargetingMatchReason,
			evalCtx:    openfeature.FlattenedContext{"attr": "val"},
			flag: configcattest.Flag{
				Default:                 false,
				PercentageEvalAttribute: "attr",
				Percentages: []configcattest.PercentageOption{{
					Percentage: 50,
					Value:      true,
				}, {
					Percentage: 50,
					Value:      false,
				}},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_ = flagSrv.SetFlags(sdkKey, map[string]*configcattest.Flag{
				"flag": &test.flag,
			})
			_ = client.Refresh(ctx)

			resolution := provider.BooleanEvaluation(ctx, test.key, test.defaultVal, test.evalCtx)
			require.Equal(t, test.expVal, resolution.Value)
			require.Equal(t, test.expVariant, resolution.Variant)
			require.Equal(t, test.reason, resolution.Reason)
			require.Equal(t, test.errCode, resolution.ResolutionDetail().ErrorCode)
			require.Equal(t, test.errMsg, resolution.ResolutionDetail().ErrorMessage)
		})
	}

	t.Run("evalCtx keys set", func(t *testing.T) {
		_ = flagSrv.SetFlags(sdkKey, map[string]*configcattest.Flag{
			"flag": {
				Default: true,
			},
		})
		_ = client.Refresh(ctx)

		testEvalCtxUserData(t, func(evalCtx openfeature.FlattenedContext) *sdk.EvaluationDetails {
			defer func() { hooks.OnFlagEvaluated = nil }()
			ch := make(chan *sdk.EvaluationDetails)
			hooks.OnFlagEvaluated = func(d *sdk.EvaluationDetails) {
				ch <- d
			}
			provider.BooleanEvaluation(ctx, "flag", false, evalCtx)
			return waitForChanMax(t, 1*time.Second, ch)
		})
	})
}

func TestStringEvaluation(t *testing.T) {
	ctx := context.Background()
	client, flagSrv, hooks, sdkKey := newTestServer(t)
	provider := configcat.NewProvider(client)
	defaultFlag := configcattest.Flag{Default: "hi"}

	tests := []struct {
		name       string
		key        string
		defaultVal string
		expVal     string
		expVariant string
		errMsg     string
		errCode    openfeature.ErrorCode
		reason     openfeature.Reason
		evalCtx    map[string]any
		flag       configcattest.Flag
	}{
		{
			name:       "evalCtx empty",
			key:        "flag",
			defaultVal: "hello",
			expVal:     "hi",
			expVariant: "v_flag",
			errMsg:     "",
			errCode:    "",
			reason:     openfeature.DefaultReason,
			evalCtx:    nil,
			flag:       defaultFlag,
		},
		{
			name:       "key not found",
			key:        "non-existing",
			defaultVal: "hello",
			expVal:     "hello",
			expVariant: "",
			errMsg:     "failed to evaluate setting 'non-existing' (the key was not found in config JSON); available keys: ['flag']",
			errCode:    openfeature.FlagNotFoundCode,
			reason:     openfeature.ErrorReason,
			evalCtx:    nil,
			flag:       defaultFlag,
		},
		{
			name:       "type mismatch",
			key:        "flag",
			defaultVal: "hello",
			expVal:     "hello",
			expVariant: "",
			errMsg:     "the type of the setting 'flag' doesn't match with the expected type; setting's type was 'int' but the expected type was 'string'",
			errCode:    openfeature.TypeMismatchCode,
			reason:     openfeature.ErrorReason,
			evalCtx:    nil,
			flag:       configcattest.Flag{Default: 5},
		},
		{
			name:       "unknown error",
			key:        "flag",
			defaultVal: "hello",
			expVal:     "hello",
			expVariant: "",
			errMsg:     "comparison value '<nil>' is invalid",
			errCode:    openfeature.GeneralCode,
			reason:     openfeature.ErrorReason,
			evalCtx:    openfeature.FlattenedContext{"attr": "val"},
			flag: configcattest.Flag{
				Default: "a",
				Rules: []configcattest.Rule{{
					Value:               "b",
					Comparator:          sdk.OpStartsWithAnyOfHashed,
					ComparisonAttribute: "attr",
					ComparisonValue:     "invalidnothashed",
				}},
			},
		},
		{
			name:       "matched evaluation rule",
			key:        "flag",
			defaultVal: "hello",
			expVal:     "hi",
			expVariant: "v0_flag",
			errMsg:     "",
			errCode:    "",
			reason:     openfeature.TargetingMatchReason,
			evalCtx:    openfeature.FlattenedContext{"attr": "val"},
			flag: configcattest.Flag{
				Default: "a",
				Rules: []configcattest.Rule{{
					Value:               "hi",
					Comparator:          sdk.OpEq,
					ComparisonAttribute: "attr",
					ComparisonValue:     "val",
				}},
			},
		},
		{
			name:       "matched percentage rule",
			key:        "flag",
			defaultVal: "hello",
			expVal:     "hi",
			expVariant: "v0_flag",
			errMsg:     "",
			errCode:    "",
			reason:     openfeature.TargetingMatchReason,
			evalCtx:    openfeature.FlattenedContext{"attr": "val"},
			flag: configcattest.Flag{
				Default:                 "a",
				PercentageEvalAttribute: "attr",
				Percentages: []configcattest.PercentageOption{{
					Percentage: 50,
					Value:      "hi",
				}, {
					Percentage: 50,
					Value:      "aloha",
				}},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_ = flagSrv.SetFlags(sdkKey, map[string]*configcattest.Flag{
				"flag": &test.flag,
			})
			_ = client.Refresh(ctx)

			resolution := provider.StringEvaluation(ctx, test.key, test.defaultVal, test.evalCtx)
			require.Equal(t, test.expVal, resolution.Value)
			require.Equal(t, test.expVariant, resolution.Variant)
			require.Equal(t, test.reason, resolution.Reason)
			require.Equal(t, test.errCode, resolution.ResolutionDetail().ErrorCode)
			require.Equal(t, test.errMsg, resolution.ResolutionDetail().ErrorMessage)
		})
	}

	t.Run("evalCtx keys set", func(t *testing.T) {
		_ = flagSrv.SetFlags(sdkKey, map[string]*configcattest.Flag{
			"flag": {
				Default: "hi",
			},
		})
		_ = client.Refresh(ctx)

		testEvalCtxUserData(t, func(evalCtx openfeature.FlattenedContext) *sdk.EvaluationDetails {
			defer func() { hooks.OnFlagEvaluated = nil }()
			ch := make(chan *sdk.EvaluationDetails)
			hooks.OnFlagEvaluated = func(d *sdk.EvaluationDetails) {
				ch <- d
			}
			provider.StringEvaluation(ctx, "flag", "hello", evalCtx)
			return waitForChanMax(t, 1*time.Second, ch)
		})
	})
}

func TestFloatEvaluation(t *testing.T) {
	ctx := context.Background()
	client, flagSrv, hooks, sdkKey := newTestServer(t)
	provider := configcat.NewProvider(client)
	defaultFlag := configcattest.Flag{Default: 1.1}

	tests := []struct {
		name       string
		key        string
		defaultVal float64
		expVal     float64
		expVariant string
		errMsg     string
		errCode    openfeature.ErrorCode
		reason     openfeature.Reason
		evalCtx    map[string]any
		flag       configcattest.Flag
	}{
		{
			name:       "evalCtx empty",
			key:        "flag",
			defaultVal: 2.2,
			expVal:     1.1,
			expVariant: "v_flag",
			errMsg:     "",
			errCode:    "",
			reason:     openfeature.DefaultReason,
			evalCtx:    nil,
			flag:       defaultFlag,
		},
		{
			name:       "key not found",
			key:        "non-existing",
			defaultVal: 2.2,
			expVal:     2.2,
			expVariant: "",
			errMsg:     "failed to evaluate setting 'non-existing' (the key was not found in config JSON); available keys: ['flag']",
			errCode:    openfeature.FlagNotFoundCode,
			reason:     openfeature.ErrorReason,
			evalCtx:    nil,
			flag:       defaultFlag,
		},
		{
			name:       "type mismatch",
			key:        "flag",
			defaultVal: 2.2,
			expVal:     2.2,
			expVariant: "",
			errMsg:     "the type of the setting 'flag' doesn't match with the expected type; setting's type was 'string' but the expected type was 'float'",
			errCode:    openfeature.TypeMismatchCode,
			reason:     openfeature.ErrorReason,
			evalCtx:    nil,
			flag:       configcattest.Flag{Default: "a"},
		},
		{
			name:       "unknown error",
			key:        "flag",
			defaultVal: 2.2,
			expVal:     2.2,
			expVariant: "",
			errMsg:     "comparison value '<nil>' is invalid",
			errCode:    openfeature.GeneralCode,
			reason:     openfeature.ErrorReason,
			evalCtx:    openfeature.FlattenedContext{"attr": "val"},
			flag: configcattest.Flag{
				Default: 3.3,
				Rules: []configcattest.Rule{{
					Value:               4.4,
					Comparator:          sdk.OpStartsWithAnyOfHashed,
					ComparisonAttribute: "attr",
					ComparisonValue:     "invalidnothashed",
				}},
			},
		},
		{
			name:       "matched evaluation rule",
			key:        "flag",
			defaultVal: 2.2,
			expVal:     4.4,
			expVariant: "v0_flag",
			errMsg:     "",
			errCode:    "",
			reason:     openfeature.TargetingMatchReason,
			evalCtx:    openfeature.FlattenedContext{"attr": "val"},
			flag: configcattest.Flag{
				Default: 3.3,
				Rules: []configcattest.Rule{{
					Value:               4.4,
					Comparator:          sdk.OpEq,
					ComparisonAttribute: "attr",
					ComparisonValue:     "val",
				}},
			},
		},
		{
			name:       "matched percentage rule",
			key:        "flag",
			defaultVal: 2.2,
			expVal:     4.4,
			expVariant: "v0_flag",
			errMsg:     "",
			errCode:    "",
			reason:     openfeature.TargetingMatchReason,
			evalCtx:    openfeature.FlattenedContext{"attr": "val"},
			flag: configcattest.Flag{
				Default:                 3.3,
				PercentageEvalAttribute: "attr",
				Percentages: []configcattest.PercentageOption{{
					Percentage: 50,
					Value:      4.4,
				}, {
					Percentage: 50,
					Value:      5.5,
				}},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_ = flagSrv.SetFlags(sdkKey, map[string]*configcattest.Flag{
				"flag": &test.flag,
			})
			_ = client.Refresh(ctx)

			resolution := provider.FloatEvaluation(ctx, test.key, test.defaultVal, test.evalCtx)
			require.Equal(t, test.expVal, resolution.Value)
			require.Equal(t, test.expVariant, resolution.Variant)
			require.Equal(t, test.reason, resolution.Reason)
			require.Equal(t, test.errCode, resolution.ResolutionDetail().ErrorCode)
			require.Equal(t, test.errMsg, resolution.ResolutionDetail().ErrorMessage)
		})
	}

	t.Run("evalCtx keys set", func(t *testing.T) {
		_ = flagSrv.SetFlags(sdkKey, map[string]*configcattest.Flag{
			"flag": {
				Default: 1.1,
			},
		})
		_ = client.Refresh(ctx)

		testEvalCtxUserData(t, func(evalCtx openfeature.FlattenedContext) *sdk.EvaluationDetails {
			defer func() { hooks.OnFlagEvaluated = nil }()
			ch := make(chan *sdk.EvaluationDetails)
			hooks.OnFlagEvaluated = func(d *sdk.EvaluationDetails) {
				ch <- d
			}
			provider.FloatEvaluation(ctx, "flag", 1.7, evalCtx)
			return waitForChanMax(t, 1*time.Second, ch)
		})
	})
}

func TestIntEvaluation(t *testing.T) {
	ctx := context.Background()
	client, flagSrv, hooks, sdkKey := newTestServer(t)
	provider := configcat.NewProvider(client)
	defaultFlag := configcattest.Flag{Default: 1}

	tests := []struct {
		name       string
		key        string
		defaultVal int64
		expVal     int64
		expVariant string
		errMsg     string
		errCode    openfeature.ErrorCode
		reason     openfeature.Reason
		evalCtx    map[string]any
		flag       configcattest.Flag
	}{
		{
			name:       "evalCtx empty",
			key:        "flag",
			defaultVal: int64(0),
			expVal:     int64(1),
			expVariant: "v_flag",
			errMsg:     "",
			errCode:    "",
			reason:     openfeature.DefaultReason,
			evalCtx:    nil,
			flag:       defaultFlag,
		},
		{
			name:       "key not found",
			key:        "non-existing",
			defaultVal: int64(0),
			expVal:     int64(0),
			expVariant: "",
			errMsg:     "failed to evaluate setting 'non-existing' (the key was not found in config JSON); available keys: ['flag']",
			errCode:    openfeature.FlagNotFoundCode,
			reason:     openfeature.ErrorReason,
			evalCtx:    nil,
			flag:       defaultFlag,
		},
		{
			name:       "type mismatch",
			key:        "flag",
			defaultVal: int64(0),
			expVal:     int64(0),
			expVariant: "",
			errMsg:     "the type of the setting 'flag' doesn't match with the expected type; setting's type was 'string' but the expected type was 'int'",
			errCode:    openfeature.TypeMismatchCode,
			reason:     openfeature.ErrorReason,
			evalCtx:    nil,
			flag:       configcattest.Flag{Default: "a"},
		},
		{
			name:       "unknown error",
			key:        "flag",
			defaultVal: int64(0),
			expVal:     int64(0),
			expVariant: "",
			errMsg:     "comparison value '<nil>' is invalid",
			errCode:    openfeature.GeneralCode,
			reason:     openfeature.ErrorReason,
			evalCtx:    openfeature.FlattenedContext{"attr": "val"},
			flag: configcattest.Flag{
				Default: 2,
				Rules: []configcattest.Rule{{
					Value:               3,
					Comparator:          sdk.OpStartsWithAnyOfHashed,
					ComparisonAttribute: "attr",
					ComparisonValue:     "invalidnothashed",
				}},
			},
		},
		{
			name:       "matched evaluation rule",
			key:        "flag",
			defaultVal: int64(0),
			expVal:     int64(3),
			expVariant: "v0_flag",
			errMsg:     "",
			errCode:    "",
			reason:     openfeature.TargetingMatchReason,
			evalCtx:    openfeature.FlattenedContext{"attr": "val"},
			flag: configcattest.Flag{
				Default: 2,
				Rules: []configcattest.Rule{{
					Value:               3,
					Comparator:          sdk.OpEq,
					ComparisonAttribute: "attr",
					ComparisonValue:     "val",
				}},
			},
		},
		{
			name:       "matched percentage rule",
			key:        "flag",
			defaultVal: int64(0),
			expVal:     int64(3),
			expVariant: "v0_flag",
			errMsg:     "",
			errCode:    "",
			reason:     openfeature.TargetingMatchReason,
			evalCtx:    openfeature.FlattenedContext{"attr": "val"},
			flag: configcattest.Flag{
				Default:                 2,
				PercentageEvalAttribute: "attr",
				Percentages: []configcattest.PercentageOption{{
					Percentage: 50,
					Value:      3,
				}, {
					Percentage: 50,
					Value:      4,
				}},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_ = flagSrv.SetFlags(sdkKey, map[string]*configcattest.Flag{
				"flag": &test.flag,
			})
			_ = client.Refresh(ctx)

			resolution := provider.IntEvaluation(ctx, test.key, test.defaultVal, test.evalCtx)
			require.Equal(t, test.expVal, resolution.Value)
			require.Equal(t, test.expVariant, resolution.Variant)
			require.Equal(t, test.reason, resolution.Reason)
			require.Equal(t, test.errCode, resolution.ResolutionDetail().ErrorCode)
			require.Equal(t, test.errMsg, resolution.ResolutionDetail().ErrorMessage)
		})
	}

	t.Run("evalCtx keys set", func(t *testing.T) {
		_ = flagSrv.SetFlags(sdkKey, map[string]*configcattest.Flag{
			"flag": {
				Default: 1,
			},
		})
		_ = client.Refresh(ctx)

		testEvalCtxUserData(t, func(evalCtx openfeature.FlattenedContext) *sdk.EvaluationDetails {
			defer func() { hooks.OnFlagEvaluated = nil }()
			ch := make(chan *sdk.EvaluationDetails)
			hooks.OnFlagEvaluated = func(d *sdk.EvaluationDetails) {
				ch <- d
			}
			provider.IntEvaluation(ctx, "flag", 1, evalCtx)
			return waitForChanMax(t, 1*time.Second, ch)
		})
	})
}

func TestObjectEvaluation(t *testing.T) {
	ctx := context.Background()
	client, flagSrv, hooks, sdkKey := newTestServer(t)
	provider := configcat.NewProvider(client)
	defaultFlag := configcattest.Flag{Default: `{"name":"test"}`}

	tests := []struct {
		name       string
		key        string
		defaultVal any
		expVal     any
		expVariant string
		errMsg     string
		errCode    openfeature.ErrorCode
		reason     openfeature.Reason
		evalCtx    map[string]any
		flag       configcattest.Flag
	}{
		{
			name:       "evalCtx empty",
			key:        "flag",
			defaultVal: nil,
			expVal:     map[string]any{"name": "test"},
			expVariant: "v_flag",
			errMsg:     "",
			errCode:    "",
			reason:     openfeature.DefaultReason,
			evalCtx:    nil,
			flag:       defaultFlag,
		},
		{
			name:       "key not found",
			key:        "non-existing",
			defaultVal: nil,
			expVal:     nil,
			expVariant: "",
			errMsg:     "failed to evaluate setting 'non-existing' (the key was not found in config JSON); available keys: ['flag']",
			errCode:    openfeature.FlagNotFoundCode,
			reason:     openfeature.ErrorReason,
			evalCtx:    nil,
			flag:       defaultFlag,
		},
		{
			name:       "type mismatch",
			key:        "flag",
			defaultVal: nil,
			expVal:     nil,
			expVariant: "",
			errMsg:     "the type of the setting 'flag' doesn't match with the expected type; setting's type was 'int' but the expected type was 'string'",
			errCode:    openfeature.TypeMismatchCode,
			reason:     openfeature.ErrorReason,
			evalCtx:    nil,
			flag:       configcattest.Flag{Default: 5},
		},
		{
			name:       "unknown error",
			key:        "flag",
			defaultVal: nil,
			expVal:     nil,
			expVariant: "",
			errMsg:     "comparison value '<nil>' is invalid",
			errCode:    openfeature.GeneralCode,
			reason:     openfeature.ErrorReason,
			evalCtx:    openfeature.FlattenedContext{"attr": "val"},
			flag: configcattest.Flag{
				Default: `{"name":"test1"}`,
				Rules: []configcattest.Rule{{
					Value:               `{"name":"test2"}`,
					Comparator:          sdk.OpStartsWithAnyOfHashed,
					ComparisonAttribute: "attr",
					ComparisonValue:     "invalidnothashed",
				}},
			},
		},
		{
			name:       "invalid json",
			key:        "flag",
			defaultVal: nil,
			expVal:     nil,
			expVariant: "",
			errMsg:     "failed to unmarshal string flag as json: invalid character '}' after object key",
			errCode:    openfeature.TypeMismatchCode,
			reason:     openfeature.ErrorReason,
			evalCtx:    openfeature.FlattenedContext{"attr": "val"},
			flag:       configcattest.Flag{Default: `{"invalid"}`},
		},
		{
			name:       "matched evaluation rule",
			key:        "flag",
			defaultVal: nil,
			expVal:     map[string]any{"domain": "example.org"},
			expVariant: "v0_flag",
			errMsg:     "",
			errCode:    "",
			reason:     openfeature.TargetingMatchReason,
			evalCtx:    openfeature.FlattenedContext{"attr": "val"},
			flag: configcattest.Flag{
				Default: `{"some":"default"}`,
				Rules: []configcattest.Rule{{
					Value:               `{"domain":"example.org"}`,
					Comparator:          sdk.OpEq,
					ComparisonAttribute: "attr",
					ComparisonValue:     "val",
				}},
			},
		},
		{
			name:       "matched percentage rule",
			key:        "flag",
			defaultVal: nil,
			expVal:     map[string]any{"domain": "example.org"},
			expVariant: "v0_flag",
			errMsg:     "",
			errCode:    "",
			reason:     openfeature.TargetingMatchReason,
			evalCtx:    openfeature.FlattenedContext{"attr": "val"},
			flag: configcattest.Flag{
				Default:                 `{"some":"default"}`,
				PercentageEvalAttribute: "attr",
				Percentages: []configcattest.PercentageOption{{
					Percentage: 50,
					Value:      `{"domain":"example.org"}`,
				}, {
					Percentage: 50,
					Value:      `{"domain":"example2.org"}`,
				}},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_ = flagSrv.SetFlags(sdkKey, map[string]*configcattest.Flag{
				"flag": &test.flag,
			})
			_ = client.Refresh(ctx)

			resolution := provider.ObjectEvaluation(ctx, test.key, test.defaultVal, test.evalCtx)
			require.Equal(t, test.expVal, resolution.Value)
			require.Equal(t, test.expVariant, resolution.Variant)
			require.Equal(t, test.reason, resolution.Reason)
			require.Equal(t, test.errCode, resolution.ResolutionDetail().ErrorCode)
			require.Equal(t, test.errMsg, resolution.ResolutionDetail().ErrorMessage)
		})
	}

	t.Run("evalCtx keys set", func(t *testing.T) {
		_ = flagSrv.SetFlags(sdkKey, map[string]*configcattest.Flag{
			"flag": {
				Default: `{"name":"test"}`,
			},
		})
		_ = client.Refresh(ctx)

		testEvalCtxUserData(t, func(evalCtx openfeature.FlattenedContext) *sdk.EvaluationDetails {
			defer func() { hooks.OnFlagEvaluated = nil }()
			ch := make(chan *sdk.EvaluationDetails)
			hooks.OnFlagEvaluated = func(d *sdk.EvaluationDetails) {
				ch <- d
			}
			provider.ObjectEvaluation(ctx, "flag", 1, evalCtx)
			return waitForChanMax(t, 1*time.Second, ch)
		})
	})
}

func testEvalCtxUserData(t *testing.T, cb func(evalCtx openfeature.FlattenedContext) *sdk.EvaluationDetails) {
	t.Helper()

	expectedIdentifier := "123"
	expectedEmail := "example@example.com"
	expectedCountry := "AQ"
	expectedSomeKey := "some-value"

	details := cb(map[string]any{
		openfeature.TargetingKey: expectedIdentifier,
		configcat.EmailKey:       expectedEmail,
		configcat.CountryKey:     expectedCountry,
		"some-key":               expectedSomeKey,
	})

	user, ok := details.Data.User.(sdk.UserAttributes)
	require.True(t, ok)

	require.Equal(t, expectedIdentifier, user.GetAttribute("Identifier"))
	require.Equal(t, expectedEmail, user.GetAttribute("Email"))
	require.Equal(t, expectedCountry, user.GetAttribute("Country"))
	require.Equal(t, expectedSomeKey, user.GetAttribute("some-key"))
}

func newTestServer(t *testing.T) (*sdk.Client, *configcattest.Handler, *sdk.Hooks, string) {
	key := configcattest.RandomSDKKey()
	var handler configcattest.Handler
	hooks := sdk.Hooks{}
	srv := httptest.NewServer(&handler)
	cfg := sdk.Config{
		LogLevel:    sdk.LogLevelDebug,
		BaseURL:     srv.URL,
		SDKKey:      key,
		PollingMode: sdk.Manual,
		Hooks:       &hooks,
	}
	client := sdk.NewCustomClient(cfg)
	t.Cleanup(func() {
		srv.Close()
		client.Close()
	})
	return client, &handler, &hooks, key
}

func waitForChanMax[T any](t *testing.T, timeout time.Duration, ch <-chan *T) *T {
	ti := time.After(timeout)
	done := make(chan *T)
	go func() {
		select {
		case <-ti:
			t.Errorf("timed out waiting for %v", timeout)
			done <- nil
		case res := <-ch:
			done <- res
		}
	}()
	return <-done
}
