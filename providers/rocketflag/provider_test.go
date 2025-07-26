package rocketflag_test

import (
	"context"
	"errors"
	"testing"

	rfProvider "github.com/open-feature/go-sdk-contrib/providers/rocketflag"
	"github.com/open-feature/go-sdk/openfeature"
	rfClient "github.com/rocketflag/go-sdk"
	"github.com/stretchr/testify/require"
)

// MockClient is a mock implementation of the RocketFlag client.
type MockClient struct {
	GetFlagFunc func(flag string, user rfClient.UserContext) (*rfClient.FlagStatus, error)
}

func (m *MockClient) GetFlag(flag string, user rfClient.UserContext) (*rfClient.FlagStatus, error) {
	if m.GetFlagFunc != nil {
		return m.GetFlagFunc(flag, user)
	}
	return &rfClient.FlagStatus{Enabled: true}, nil
}

func TestBoolean(t *testing.T) {
	client := &MockClient{}
	provider := rfProvider.NewProvider(client)
	err := openfeature.SetProviderAndWait(provider)
	if err != nil {
		t.Fatalf("error setting up provider %s", err)
	}

	ofClient := openfeature.NewClient("my-go-app")
	ctx := context.Background()

	t.Run("Successful evaluation with targeting context", func(t *testing.T) {
		evaluationContext := openfeature.NewEvaluationContext("user@example.com", nil)
		response, _ := ofClient.BooleanValueDetails(ctx, "test-flag", false, evaluationContext)

		require.True(t, response.Value)
		require.Equal(t, openfeature.TargetingMatchReason, response.Reason)
		require.Nil(t, nil, response.ErrorCode)
	})

	t.Run("Successful evaluation with unknown targeting context", func(t *testing.T) {
		evaluationContext := openfeature.NewEvaluationContext("", map[string]any{"unknownKey": "unknownVal"})
		response, _ := ofClient.BooleanValueDetails(ctx, "test-flag", false, evaluationContext)

		require.True(t, response.Value)
		require.Equal(t, openfeature.DefaultReason, response.Reason)
		require.Nil(t, nil, response.ErrorCode)
	})

	t.Run("Successful evaluation with empty targeting context", func(t *testing.T) {
		evaluationContext := openfeature.NewEvaluationContext("", nil)
		response, _ := ofClient.BooleanValueDetails(ctx, "flag-abc-123", false, evaluationContext)

		require.True(t, response.Value)                              // Mock returns true
		require.Equal(t, openfeature.DefaultReason, response.Reason) // Reason is _NOT_ "TARGETING_MATCH", and is the default.
		require.Nil(t, nil, response.ErrorCode)
	})

	t.Run("Successful evaluation without targeting context", func(t *testing.T) {
		var evaluationContext openfeature.EvaluationContext
		response, _ := ofClient.BooleanValueDetails(ctx, "test-flag", false, evaluationContext)

		require.True(t, response.Value)
		require.Equal(t, openfeature.DefaultReason, response.Reason)
		require.Nil(t, nil, response.ErrorCode)
	})

	t.Run("Flag not found", func(t *testing.T) {
		client := &MockClient{
			GetFlagFunc: func(flag string, user rfClient.UserContext) (*rfClient.FlagStatus, error) {
				return nil, errors.New("flag not found")
			},
		}
		provider := rfProvider.NewProvider(client)
		err := openfeature.SetProviderAndWait(provider)
		if err != nil {
			t.Fatalf("error setting up provider %s", err)
		}
		var evaluationContext openfeature.EvaluationContext

		response, _ := ofClient.BooleanValueDetails(ctx, "non-existent-flag", false, evaluationContext)
		require.False(t, response.Value)
		require.Equal(t, openfeature.ErrorReason, response.Reason)
		require.Equal(t, openfeature.ErrorCode("GENERAL"), response.ResolutionDetail.ErrorCode)
	})
}

func TestMetadata(t *testing.T) {
	client := &MockClient{}
	provider := rfProvider.NewProvider(client)
	require.Equal(t, openfeature.Metadata{Name: "RocketFlag"}, provider.Metadata())
}

func TestStringEvaluation(t *testing.T) {
	client := &MockClient{}
	provider := rfProvider.NewProvider(client)
	ctx := context.Background()

	resolution := provider.StringEvaluation(ctx, "flag", "default", nil)
	require.Equal(t, "default", resolution.Value)
	require.Equal(t, openfeature.ErrorReason, resolution.Reason)
	require.Equal(t, openfeature.NewTypeMismatchResolutionError("RocketFlag: String flags are not yet supported.").Error(), resolution.ResolutionError.Error())
}

func TestFloatEvaluation(t *testing.T) {
	client := &MockClient{}
	provider := rfProvider.NewProvider(client)
	ctx := context.Background()

	resolution := provider.FloatEvaluation(ctx, "flag", 1.23, nil)
	require.Equal(t, 1.23, resolution.Value)
	require.Equal(t, openfeature.ErrorReason, resolution.Reason)
	require.Equal(t, openfeature.NewTypeMismatchResolutionError("RocketFlag: Float flags are not yet supported.").Error(), resolution.ResolutionError.Error())
}

func TestIntEvaluation(t *testing.T) {
	client := &MockClient{}
	provider := rfProvider.NewProvider(client)
	ctx := context.Background()

	resolution := provider.IntEvaluation(ctx, "flag", 123, nil)
	require.Equal(t, int64(123), resolution.Value)
	require.Equal(t, openfeature.ErrorReason, resolution.Reason)
	require.Equal(t, openfeature.NewTypeMismatchResolutionError("RocketFlag: Int flags are not yet supported.").Error(), resolution.ResolutionError.Error())
}

func TestObjectEvaluation(t *testing.T) {
	client := &MockClient{}
	provider := rfProvider.NewProvider(client)
	ctx := context.Background()

	defaultValue := map[string]any{"key": "value"}
	resolution := provider.ObjectEvaluation(ctx, "flag", defaultValue, nil)
	require.Equal(t, defaultValue, resolution.Value)
	require.Equal(t, openfeature.ErrorReason, resolution.Reason)
	require.Equal(t, openfeature.NewTypeMismatchResolutionError("RocketFlag: Object flags are not yet supported.").Error(), resolution.ResolutionError.Error())
}
