package launchdarkly

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/hooklift/assert"
	"github.com/launchdarkly/go-sdk-common/v3/ldlog"
	"github.com/launchdarkly/go-server-sdk/v6/ldcomponents"
	"github.com/launchdarkly/go-server-sdk/v6/ldfiledata"
	"github.com/open-feature/go-sdk/pkg/openfeature"

	ld "github.com/launchdarkly/go-server-sdk/v6"
)

type testLogger struct {
	t *testing.T
}

func newTestLogger(t *testing.T) Logger {
	return &testLogger{
		t: t,
	}
}

func (l *testLogger) Debug(msg string, args ...any) {
	l.t.Logf(msg, args...)
}

func (l *testLogger) Error(msg string, args ...any) {
	l.t.Logf(msg, args...)
}

func (l *testLogger) Warn(msg string, args ...any) {
	l.t.Logf(msg, args...)
}

func makeLDClient(t *testing.T, flagsFilePath string) *ld.LDClient {
	var config ld.Config
	config.DataSource = ldfiledata.DataSource().FilePaths(flagsFilePath)
	config.Logging = ldcomponents.Logging().MinLevel(ldlog.Debug)
	config.Events = ldcomponents.NoEvents()
	config.Offline = false
	client, err := ld.MakeCustomClient("no sdk key", config, 5*time.Second)
	assert.Ok(t, err)

	return client
}

func TestProvider(t *testing.T) {
	tests := []struct {
		desc         string
		flagKey      string
		targetKey    string
		evalCtx      map[string]any
		expErr       error
		expFlagValue any
	}{
		{
			desc:      "happy path",
			flagKey:   "mtls_enabled",
			targetKey: "redpanda-blah12342",
			evalCtx: map[string]any{
				"kind":            "redpanda-id",
				"organization-id": "blah1234",
				"redpanda-id":     "redpanda-blah12342",
				"key":             "redpanda-blah12343",
				"cloud-provider":  "aws",
			},
			expErr:       nil,
			expFlagValue: true,
		},
		{
			desc:      "it complains when key nor targeting key are specified in the evaluation context",
			flagKey:   "mtls_enabled",
			targetKey: "",
			evalCtx: map[string]any{
				"kind":            "redpanda-id",
				"organization-id": "blah1234",
				"redpanda-id":     "redpanda-blah12342",
				"cloud-provider":  "aws",
			},
			expErr:       errors.New("TARGETING_KEY_MISSING: key and targetingKey attributes are missing, at least 1 required"),
			expFlagValue: false,
		},
		{
			desc:      "it fails when no kind attribute is found",
			flagKey:   "mtls_enabled",
			targetKey: "redpanda-blah12343",
			evalCtx: map[string]any{
				"organization-id": "blah1234",
				"redpanda-id":     "redpanda-blah12342",
				"key":             "redpanda-blah12343",
				"cloud-provider":  "aws",
			},
			expErr:       errors.New("PARSE_ERROR: LaunchDarkly returned ERROR(MALFORMED_FLAG)"),
			expFlagValue: false,
		},
		{
			desc:      "it fails if the feature flag is not found in LaunchDarkly",
			flagKey:   "not_found",
			targetKey: "redpanda-blah12343",
			evalCtx: map[string]any{
				"kind":            "redpanda-id",
				"organization-id": "blah1234",
				"redpanda-id":     "redpanda-blah12342",
				"key":             "redpanda-blah12343",
				"cloud-provider":  "aws",
			},
			expErr:       errors.New("FLAG_NOT_FOUND: LaunchDarkly returned ERROR(FLAG_NOT_FOUND)"),
			expFlagValue: false,
		},
		{
			desc:      "it succeeds with a well formed multi context",
			flagKey:   "mtls_enabled",
			targetKey: "redpanda-blah12342",
			evalCtx: map[string]any{
				"kind": "multi",
				"organization": map[string]any{
					"key":  "blah1234",
					"name": "Redpanda",
					"tier": "GOLD",
				},
				"redpanda-id": map[string]any{
					"key":            "redpanda-blah12342",
					"cloud-provider": "aws",
				},
			},
			expErr:       nil,
			expFlagValue: true,
		},
		{
			desc:      "it fails if the context kind is not found in LaunchDarkly",
			flagKey:   "mtls_enabled",
			targetKey: "redpanda-blah12342",
			evalCtx: map[string]any{
				"kind": "redpanda-blah",
				"organization": map[string]any{
					"key":  "blah1234",
					"name": "Redpanda",
					"tier": "GOLD",
				},
				"redpanda-id": map[string]any{
					"key":            "redpanda-blah12342",
					"cloud-provider": "aws",
				},
			},
			expErr:       errors.New("PARSE_ERROR: LaunchDarkly returned ERROR(MALFORMED_FLAG)"),
			expFlagValue: false,
		},
	}

	openfeature.SetProvider(NewProvider(
		makeLDClient(t, "testdata/flags.json"),
		WithLogger(newTestLogger(t)),
	))

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			evalCtx := openfeature.NewEvaluationContext(test.targetKey, test.evalCtx)
			client := openfeature.NewClient("tests")
			mtlsEnabled, err := client.BooleanValue(context.Background(), test.flagKey, false, evalCtx)
			assert.Equals(t, test.expErr, errors.Unwrap(err))
			assert.Equals(t, test.expFlagValue, mtlsEnabled)
		})
	}
}
