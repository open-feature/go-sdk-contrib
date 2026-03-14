package hook_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/open-feature/go-sdk-contrib/providers/go-feature-flag/pkg/api"
	"github.com/open-feature/go-sdk-contrib/providers/go-feature-flag/pkg/hook"
	"github.com/open-feature/go-sdk-contrib/providers/go-feature-flag/pkg/manager"
	"github.com/open-feature/go-sdk-contrib/providers/go-feature-flag/pkg/model"
	"github.com/open-feature/go-sdk/openfeature"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type hookMockRoundTripper struct {
	lastBody   []byte
	status     int
	err        error
	numberCall int
}

func (m *hookMockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	m.numberCall++
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

type capturedCollectorRequest struct {
	Events []json.RawMessage `json:"events"`
	Meta   map[string]any    `json:"meta"`
}

func newDataCollectorHookForTest() (openfeature.Hook, *manager.DataCollectorManager, *hookMockRoundTripper) {
	mrt := &hookMockRoundTripper{status: http.StatusOK}
	client := &http.Client{Transport: mrt}
	goffAPI := *api.NewGoFeatureFlagAPI(api.GoFeatureFlagAPIOptions{
		Endpoint:   "http://localhost:1031",
		HTTPClient: client,
	})
	collector := manager.NewDataCollectorManager(goffAPI, 100, 0)
	return hook.NewDataCollectorHook(&collector), &collector, mrt
}

func Test_NewDataCollectorHook(t *testing.T) {
	h, _, _ := newDataCollectorHookForTest()
	require.NotNil(t, h)
}

func Test_DataCollectorHook_After_CollectsFeatureEvent(t *testing.T) {
	h, collector, mrt := newDataCollectorHookForTest()
	hookCtx := newHookContext("user-123", map[string]any{})
	evalDetails := openfeature.InterfaceEvaluationDetails{
		Value: "enabled-value",
		EvaluationDetails: openfeature.EvaluationDetails{
			FlagKey:  "test-flag",
			FlagType: openfeature.Object,
		},
	}
	evalDetails.Variant = "variant-A"
	evalDetails.Reason = openfeature.TargetingMatchReason

	err := h.After(context.Background(), hookCtx, evalDetails, openfeature.HookHints{})
	require.NoError(t, err)
	require.NoError(t, collector.SendData(context.Background()))
	assert.Equal(t, 1, mrt.numberCall)

	var payload capturedCollectorRequest
	require.NoError(t, json.Unmarshal(mrt.lastBody, &payload))
	require.Len(t, payload.Events, 1)

	var event model.FeatureEvent
	require.NoError(t, json.Unmarshal(payload.Events[0], &event))
	assert.Equal(t, "feature", event.Kind)
	assert.Equal(t, "user", event.ContextKind)
	assert.Equal(t, "user-123", event.UserKey)
	assert.Equal(t, "test-flag", event.Key)
	assert.Equal(t, "variant-A", event.Variation)
	assert.Equal(t, "enabled-value", event.Value)
	assert.False(t, event.Default)
	assert.Equal(t, "PROVIDER_CACHE", event.Source)
	assert.Greater(t, event.CreationDate, int64(0))
}

func Test_DataCollectorHook_Error_CollectsSdkDefaultEvent(t *testing.T) {
	h, collector, mrt := newDataCollectorHookForTest()
	hookCtx := newHookContext("user-789", map[string]any{})

	h.Error(context.Background(), hookCtx, errors.New("boom"), openfeature.HookHints{})
	require.NoError(t, collector.SendData(context.Background()))
	assert.Equal(t, 1, mrt.numberCall)

	var payload capturedCollectorRequest
	require.NoError(t, json.Unmarshal(mrt.lastBody, &payload))
	require.Len(t, payload.Events, 1)

	var event model.FeatureEvent
	require.NoError(t, json.Unmarshal(payload.Events[0], &event))
	assert.Equal(t, "feature", event.Kind)
	assert.Equal(t, "user", event.ContextKind)
	assert.Equal(t, "user-789", event.UserKey)
	assert.Equal(t, "test-flag", event.Key)
	assert.Equal(t, "SdkDefault", event.Variation)
	assert.Equal(t, false, event.Value)
	assert.True(t, event.Default)
	assert.Equal(t, "PROVIDER_CACHE", event.Source)
	assert.Greater(t, event.CreationDate, int64(0))
}

func Test_DataCollectorHook_After_AnonymousUser_SetsContextKind(t *testing.T) {
	h, collector, mrt := newDataCollectorHookForTest()
	hookCtx := newHookContext("anon-456", map[string]any{"anonymous": true})
	evalDetails := openfeature.InterfaceEvaluationDetails{
		Value: "some-value",
		EvaluationDetails: openfeature.EvaluationDetails{
			FlagKey:  "test-flag",
			FlagType: openfeature.String,
		},
	}
	evalDetails.Variant = "variant-B"

	err := h.After(context.Background(), hookCtx, evalDetails, openfeature.HookHints{})
	require.NoError(t, err)
	require.NoError(t, collector.SendData(context.Background()))

	var payload capturedCollectorRequest
	require.NoError(t, json.Unmarshal(mrt.lastBody, &payload))
	require.Len(t, payload.Events, 1)

	var event model.FeatureEvent
	require.NoError(t, json.Unmarshal(payload.Events[0], &event))
	assert.Equal(t, "anonymousUser", event.ContextKind)
}

func Test_DataCollectorHook_Error_AnonymousUser_SetsContextKind(t *testing.T) {
	h, collector, mrt := newDataCollectorHookForTest()
	hookCtx := newHookContext("anon-789", map[string]any{"anonymous": true})

	h.Error(context.Background(), hookCtx, errors.New("boom"), openfeature.HookHints{})
	require.NoError(t, collector.SendData(context.Background()))

	var payload capturedCollectorRequest
	require.NoError(t, json.Unmarshal(mrt.lastBody, &payload))
	require.Len(t, payload.Events, 1)

	var event model.FeatureEvent
	require.NoError(t, json.Unmarshal(payload.Events[0], &event))
	assert.Equal(t, "anonymousUser", event.ContextKind)
}
