package configcat_test

import (
	"context"
	"errors"
	"testing"

	sdk "github.com/configcat/go-sdk/v7"
	"github.com/open-feature/go-sdk-contrib/providers/configcat/internal/clienttest"
	configcat "github.com/open-feature/go-sdk-contrib/providers/configcat/pkg"
	"github.com/open-feature/go-sdk/pkg/openfeature"
	"github.com/stretchr/testify/require"
)

func TestBooleanEvaluation(t *testing.T) {
	ctx := context.Background()
	client := clienttest.NewClient()
	provider := configcat.NewProvider(client)

	t.Run("evalCtx empty", func(t *testing.T) {
		defer client.Reset()
		expectedVariant := "ksljf"
		client.WithBoolEvaluation(func(req clienttest.Request) sdk.BoolEvaluationDetails {
			return sdk.BoolEvaluationDetails{
				Value: true,
				Data: sdk.EvaluationDetailsData{
					VariationID: expectedVariant,
				},
			}
		})

		resolution := provider.BooleanEvaluation(ctx, "flag", false, nil)
		require.Equal(t, true, resolution.Value)
		require.Equal(t, expectedVariant, resolution.Variant)
		require.Equal(t, openfeature.DefaultReason, resolution.Reason)
	})

	t.Run("evalCtx non-stringer value", func(t *testing.T) {
		defer client.Reset()
		client.WithBoolEvaluation(func(req clienttest.Request) sdk.BoolEvaluationDetails {
			return sdk.BoolEvaluationDetails{
				Value: true,
			}
		})

		inputs := []struct {
			name    string
			evalCtx map[string]any
		}{
			{
				name: "targeting",
				evalCtx: map[string]any{
					openfeature.TargetingKey: client,
				},
			},
			{
				name: "identifier",
				evalCtx: map[string]any{
					configcat.IdentifierKey: client,
				},
			},
			{
				name: "email",
				evalCtx: map[string]any{
					configcat.EmailKey: client,
				},
			},
			{
				name: "country",
				evalCtx: map[string]any{
					configcat.CountryKey: client,
				},
			},
			{
				name: "custom",
				evalCtx: map[string]any{
					"some-key": client,
				},
			},
		}

		for _, input := range inputs {
			t.Run(input.name, func(t *testing.T) {
				resolution := provider.BooleanEvaluation(ctx, "flag", false, input.evalCtx)
				require.False(t, resolution.Value)
				require.Equal(t, openfeature.ErrorReason, resolution.Reason)
				require.Contains(t, resolution.ResolutionError.Error(), openfeature.InvalidContextCode)
			})
		}
	})

	t.Run("evalCtx keys set", func(t *testing.T) {
		defer client.Reset()
		client.WithBoolEvaluation(func(req clienttest.Request) sdk.BoolEvaluationDetails {
			return sdk.BoolEvaluationDetails{
				Value: true,
			}
		})

		expectedIdentifier := "123"
		expectedEmail := "example@example.com"
		expectedCountry := "AQ"
		expectedSomeKey := "some-value"

		resolution := provider.BooleanEvaluation(ctx, "flag", false, map[string]any{
			openfeature.TargetingKey: expectedIdentifier,
			configcat.EmailKey:       expectedEmail,
			configcat.CountryKey:     expectedCountry,
			"some-key":               expectedSomeKey,
		})
		require.Equal(t, true, resolution.Value)
		require.Len(t, client.GetRequests(), 1)

		request := client.GetRequests()[0]
		require.Equal(t, expectedIdentifier, request.UserData().Identifier)
		require.Equal(t, expectedEmail, request.UserData().Email)
		require.Equal(t, expectedCountry, request.UserData().Country)
		require.Len(t, request.UserData().Custom, 1)
		require.Equal(t, expectedSomeKey, request.UserData().Custom["some-key"])
	})

	t.Run("evalCtx stringer", func(t *testing.T) {
		defer client.Reset()

		inputs := []struct {
			name     string
			val      any
			expected string
		}{
			{name: "string", val: "some-string", expected: "some-string"},
			{name: "int", val: int(1), expected: "1"},
			{name: "int8", val: int8(1), expected: "1"},
			{name: "int16", val: int16(1), expected: "1"},
			{name: "int32", val: int32(1), expected: "1"},
			{name: "int64", val: int64(1), expected: "1"},
			{name: "float32", val: float32(1), expected: "1.000000"},
			{name: "float64", val: float64(1), expected: "1.000000"},
			{name: "true", val: true, expected: "true"},
			{name: "false", val: false, expected: "false"},
		}

		for _, input := range inputs {
			t.Run(input.name, func(t *testing.T) {
				client.Reset()

				resolution := provider.BooleanEvaluation(ctx, "flag", false, map[string]any{
					openfeature.TargetingKey: input.val,
				})
				require.False(t, resolution.Value)
				require.Len(t, client.GetRequests(), 1)

				request := client.GetRequests()[0]
				require.Equal(t, input.expected, request.UserData().Identifier)
			})
		}
	})

	t.Run("key not found", func(t *testing.T) {
		defer client.Reset()

		resolution := provider.BooleanEvaluation(ctx, "flag", false, nil)
		require.False(t, resolution.Value)
		require.Equal(t, openfeature.ErrorReason, resolution.Reason)
		require.Contains(t, resolution.ResolutionError.Error(), openfeature.FlagNotFoundCode)
	})

	t.Run("unknown error", func(t *testing.T) {
		defer client.Reset()
		client.WithBoolEvaluation(func(req clienttest.Request) sdk.BoolEvaluationDetails {
			return sdk.BoolEvaluationDetails{
				Value: true,
				Data: sdk.EvaluationDetailsData{
					Error: errors.New("something went wrong"),
				},
			}
		})

		resolution := provider.BooleanEvaluation(ctx, "flag", false, nil)
		require.True(t, resolution.Value)
		require.Equal(t, openfeature.ErrorReason, resolution.Reason)
		require.Contains(t, resolution.ResolutionError.Error(), openfeature.GeneralCode)
	})

	t.Run("matched evaluation rule", func(t *testing.T) {
		defer client.Reset()
		client.WithBoolEvaluation(func(req clienttest.Request) sdk.BoolEvaluationDetails {
			return sdk.BoolEvaluationDetails{
				Value: true,
				Data: sdk.EvaluationDetailsData{
					MatchedEvaluationRule: &sdk.RolloutRule{
						ComparisonAttribute: "attr",
						ComparisonValue:     "val",
						Comparator:          1,
					},
				},
			}
		})

		resolution := provider.BooleanEvaluation(ctx, "flag", false, nil)
		require.True(t, resolution.Value)
		require.Equal(t, openfeature.TargetingMatchReason, resolution.Reason)
	})

	t.Run("matched percentage rule", func(t *testing.T) {
		defer client.Reset()
		client.WithBoolEvaluation(func(req clienttest.Request) sdk.BoolEvaluationDetails {
			return sdk.BoolEvaluationDetails{
				Value: true,
				Data: sdk.EvaluationDetailsData{
					MatchedEvaluationPercentageRule: &sdk.PercentageRule{
						Percentage: 50,
					},
				},
			}
		})

		resolution := provider.BooleanEvaluation(ctx, "flag", false, nil)
		require.True(t, resolution.Value)
		require.Equal(t, openfeature.TargetingMatchReason, resolution.Reason)
	})
}
