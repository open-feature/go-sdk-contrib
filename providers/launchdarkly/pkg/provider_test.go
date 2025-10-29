package launchdarkly

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/hooklift/assert"
	"github.com/launchdarkly/go-sdk-common/v3/ldlog"
	"github.com/launchdarkly/go-server-sdk/v7/ldcomponents"
	"github.com/launchdarkly/go-server-sdk/v7/ldfiledata"
	"github.com/open-feature/go-sdk/openfeature"

	ld "github.com/launchdarkly/go-server-sdk/v7"
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

func TestContextEvaluation(t *testing.T) {
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
				"anonymous":       true,
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
					"key":               "blah1234",
					"name":              "Redpanda",
					"tier":              "GOLD",
					"privateAttributes": []string{"name"},
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

	err := openfeature.SetProviderAndWait(NewProvider(
		makeLDClient(t, "testdata/flags.json"),
		WithLogger(newTestLogger(t)),
	))
	assert.Ok(t, err)

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

func TestStringEvaluation(t *testing.T) {
	err := openfeature.SetProviderAndWait(NewProvider(
		makeLDClient(t, "testdata/flags.json"),
		WithLogger(newTestLogger(t)),
	))
	assert.Ok(t, err)

	evalCtx := openfeature.NewEvaluationContext("blah1234", map[string]any{
		"kind":            "organization-id",
		"organization-id": "blah1234",
		"redpanda-id":     "redpanda-blah12342",
		"key":             "redpanda-blah12343",
		"cloud-provider":  "aws",
		"anonymous":       true,
	})
	client := openfeature.NewClient("stringEvalTests")
	generation, err := client.StringValue(context.Background(), "dataplane_generation", "k8s.v1", evalCtx)
	assert.Ok(t, err)
	assert.Equals(t, "metal.v1", generation)
}

func TestFloatEvaluation(t *testing.T) {
	err := openfeature.SetProviderAndWait(NewProvider(
		makeLDClient(t, "testdata/flags.json"),
		WithLogger(newTestLogger(t)),
	))
	assert.Ok(t, err)

	evalCtx := openfeature.NewEvaluationContext("blah1234", map[string]any{
		"kind":            "organization-id",
		"organization-id": "blah1234",
		"redpanda-id":     "redpanda-blah12342",
		"key":             "redpanda-blah12343",
		"cloud-provider":  "aws",
		"anonymous":       true,
	})
	client := openfeature.NewClient("floatEvalTests")
	discount, err := client.FloatValue(context.Background(), "global_discount_pct", 1.5, evalCtx)
	assert.Ok(t, err)
	assert.Equals(t, 5.5, discount)
}

func TestIntEvaluation(t *testing.T) {
	err := openfeature.SetProvider(NewProvider(
		makeLDClient(t, "testdata/flags.json"),
		WithLogger(newTestLogger(t)),
	))
	assert.Ok(t, err)

	evalCtx := openfeature.NewEvaluationContext("blah1234", map[string]any{
		"kind":            "organization-id",
		"organization-id": "blah1234",
		"redpanda-id":     "redpanda-blah12342",
		"key":             "redpanda-blah12343",
		"cloud-provider":  "aws",
		"anonymous":       true,
	})
	client := openfeature.NewClient("intEvalTests")
	abuseRiskWeight, err := client.IntValue(context.Background(), "abuse_risk_weight", 10, evalCtx)
	assert.Ok(t, err)
	assert.Equals(t, int64(50), abuseRiskWeight)
}

func TestObjectEvaluation(t *testing.T) {
	err := openfeature.SetProvider(NewProvider(
		makeLDClient(t, "testdata/flags.json"),
		WithLogger(newTestLogger(t)),
	))
	assert.Ok(t, err)

	evalCtx := openfeature.NewEvaluationContext("redpanda-blah12342", map[string]any{
		"kind":            "redpanda-id",
		"organization-id": "blah1234",
		"redpanda-id":     "redpanda-blah12342",
		"key":             "redpanda-blah12343",
		"cloud-provider":  "aws",
		"anonymous":       true,
	})
	client := openfeature.NewClient("objectEvalTests")
	rateLimits, err := client.ObjectValue(context.Background(), "rate_limit_config", nil, evalCtx)
	assert.Ok(t, err)
	assert.Equals(t, map[string]any{
		"target_quota_byte_rate":       float64(2147483648), // 2GB per second
		"target_fetch_quota_byte_rate": float64(1073741824), // 1GB
		"kafka_connection_rate_limit":  float64(100),        // per second
	}, rateLimits)
}

func TestContextCancellation(t *testing.T) {
	err := openfeature.SetProvider(NewProvider(
		makeLDClient(t, "testdata/flags.json"),
		WithLogger(newTestLogger(t)),
	))
	assert.Ok(t, err)

	evalCtx := openfeature.NewEvaluationContext("redpanda-blah12342", map[string]any{
		"kind":            "redpanda-id",
		"organization-id": "blah1234",
		"redpanda-id":     "redpanda-blah12342",
		"key":             "redpanda-blah12343",
		"cloud-provider":  "aws",
		"anonymous":       true,
	})
	client := openfeature.NewClient("objectEvalTests")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = client.ObjectValue(ctx, "rate_limit_config", nil, evalCtx)
	assert.Equals(t, errors.New("GENERAL: context canceled"), errors.Unwrap(err))
}

// mockLDClient can be a struct that implements the LDClient interface for testing.
type mockLDClient struct {
	ld.LDClient // Embedding the real client can be useful for mocking only specific methods
	closeCalled bool
	closeErr    error
}

func (c *mockLDClient) Close() error {
	c.closeCalled = true
	return c.closeErr
}

func TestShutdown(t *testing.T) {
	t.Run("should not call client close on shutdown", func(t *testing.T) {
		mockClient := &mockLDClient{}
		provider := NewProvider(mockClient)

		err := openfeature.SetProvider(provider)
		assert.Ok(t, err)

		openfeature.Shutdown()
		assert.Cond(t, !mockClient.closeCalled, "expected client.Close() not to be called")
	})

	t.Run("should call client close on shutdown", func(t *testing.T) {
		mockClient := &mockLDClient{}
		provider := NewProvider(mockClient, WithCloseOnShutdown(true))

		err := openfeature.SetProvider(provider)
		assert.Ok(t, err)

		openfeature.Shutdown()
		assert.Cond(t, mockClient.closeCalled, "expected client.Close() to be called")
	})
}
