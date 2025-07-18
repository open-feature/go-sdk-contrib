package rocketflag_test

import (
	"context"
	"errors"
	"testing"

	rocketflag_provider "github.com/open-feature/go-sdk-contrib/providers/rocketflag/pkg"
	"github.com/open-feature/go-sdk/openfeature"
	rocketflag_client "github.com/rocketflag/go-sdk"
	"github.com/stretchr/testify/require"
)

// MockClient is a mock implementation of the RocketFlag client.
type MockClient struct {
	GetFlagFunc func(flag string, user rocketflag_client.UserContext) (*rocketflag_client.FlagStatus, error)
}

func (m *MockClient) GetFlag(flag string, user rocketflag_client.UserContext) (*rocketflag_client.FlagStatus, error) {
	if m.GetFlagFunc != nil {
		return m.GetFlagFunc(flag, user)
	}
	return &rocketflag_client.FlagStatus{Enabled: true}, nil
}

func TestMetadata(t *testing.T) {
	client := &MockClient{}
	provider := rocketflag_provider.NewProvider(client)
	require.Equal(t, openfeature.Metadata{Name: "RocketFlag"}, provider.Metadata())
}

func TestBooleanEvaluation(t *testing.T) {
	ctx := context.Background()

	t.Run("Successful evaluation with targeting context", func(t *testing.T) {
		client := &MockClient{}
		provider := rocketflag_provider.NewProvider(client)
		evalCtx := openfeature.FlattenedContext{"targetingKey": "user@example.com"}
		resolution := provider.BooleanEvaluation(ctx, "test-flag", false, evalCtx)
		require.True(t, resolution.Value)
		require.Equal(t, openfeature.TargetingMatchReason, resolution.Reason)
		require.NoError(t, nil, resolution.Error())
	})

	t.Run("Successful evaluation with invalid targeting context", func(t *testing.T) {
		client := &MockClient{}
		provider := rocketflag_provider.NewProvider(client)
		evalCtx := openfeature.FlattenedContext{"targetingKey": 1}
		resolution := provider.BooleanEvaluation(ctx, "test-flag", false, evalCtx)
		require.True(t, resolution.Value)
		require.Equal(t, openfeature.DefaultReason, resolution.Reason)
		require.NoError(t, nil, resolution.Error())
	})

	t.Run("Successful evaluation with empty targeting context", func(t *testing.T) {
		client := &MockClient{}
		provider := rocketflag_provider.NewProvider(client)
		evalCtx := openfeature.FlattenedContext{"targetingKey": ""}
		resolution := provider.BooleanEvaluation(ctx, "test-flag", false, evalCtx)
		require.True(t, resolution.Value)
		require.Equal(t, openfeature.DefaultReason, resolution.Reason)
		require.NoError(t, nil, resolution.Error())
	})

	t.Run("Successful evaluation with unknown targeting context", func(t *testing.T) {
		client := &MockClient{}
		provider := rocketflag_provider.NewProvider(client)
		evalCtx := openfeature.FlattenedContext{"unknownKey": "unknownVal"}
		resolution := provider.BooleanEvaluation(ctx, "test-flag", false, evalCtx)
		require.True(t, resolution.Value)
		require.Equal(t, openfeature.DefaultReason, resolution.Reason)
		require.NoError(t, nil, resolution.Error())
	})

	t.Run("Successful evaluation without targeting context", func(t *testing.T) {
		client := &MockClient{}
		provider := rocketflag_provider.NewProvider(client)
		resolution := provider.BooleanEvaluation(ctx, "test-flag", false, nil)
		require.True(t, resolution.Value)
		require.Equal(t, openfeature.DefaultReason, resolution.Reason)
		require.NoError(t, nil, resolution.Error())
	})

	t.Run("Flag not found", func(t *testing.T) {
		client := &MockClient{
			GetFlagFunc: func(flag string, user rocketflag_client.UserContext) (*rocketflag_client.FlagStatus, error) {
				return nil, errors.New("flag not found")
			},
		}
		provider := rocketflag_provider.NewProvider(client)
		resolution := provider.BooleanEvaluation(ctx, "non-existent-flag", false, nil)
		require.False(t, resolution.Value)
		require.Equal(t, openfeature.ErrorReason, resolution.Reason)
		require.Equal(t, openfeature.NewGeneralResolutionError("flag not found").Error(), resolution.ResolutionError.Error())
	})
}

func TestStringEvaluation(t *testing.T) {
	client := &MockClient{}
	provider := rocketflag_provider.NewProvider(client)
	ctx := context.Background()

	resolution := provider.StringEvaluation(ctx, "flag", "default", nil)
	require.Equal(t, "default", resolution.Value)
	require.Equal(t, openfeature.ErrorReason, resolution.Reason)
	require.Equal(t, openfeature.NewTypeMismatchResolutionError("RocketFlag: String flags are not yet supported.").Error(), resolution.ResolutionError.Error())
}

func TestFloatEvaluation(t *testing.T) {
	client := &MockClient{}
	provider := rocketflag_provider.NewProvider(client)
	ctx := context.Background()

	resolution := provider.FloatEvaluation(ctx, "flag", 1.23, nil)
	require.Equal(t, 1.23, resolution.Value)
	require.Equal(t, openfeature.ErrorReason, resolution.Reason)
	require.Equal(t, openfeature.NewTypeMismatchResolutionError("RocketFlag: Float flags are not yet supported.").Error(), resolution.ResolutionError.Error())
}

func TestIntEvaluation(t *testing.T) {
	client := &MockClient{}
	provider := rocketflag_provider.NewProvider(client)
	ctx := context.Background()

	resolution := provider.IntEvaluation(ctx, "flag", 123, nil)
	require.Equal(t, int64(123), resolution.Value)
	require.Equal(t, openfeature.ErrorReason, resolution.Reason)
	require.Equal(t, openfeature.NewTypeMismatchResolutionError("RocketFlag: Int flags are not yet supported.").Error(), resolution.ResolutionError.Error())
}

func TestObjectEvaluation(t *testing.T) {
	client := &MockClient{}
	provider := rocketflag_provider.NewProvider(client)
	ctx := context.Background()

	defaultValue := map[string]any{"key": "value"}
	resolution := provider.ObjectEvaluation(ctx, "flag", defaultValue, nil)
	require.Equal(t, defaultValue, resolution.Value)
	require.Equal(t, openfeature.ErrorReason, resolution.Reason)
	require.Equal(t, openfeature.NewTypeMismatchResolutionError("RocketFlag: Object flags are not yet supported.").Error(), resolution.ResolutionError.Error())
}
