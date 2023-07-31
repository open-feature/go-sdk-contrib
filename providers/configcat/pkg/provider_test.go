package configcat_test

import (
	"context"
	"errors"
	"testing"

	sdk "github.com/configcat/go-sdk/v8"
	"github.com/open-feature/go-sdk-contrib/providers/configcat/internal/clienttest"
	configcat "github.com/open-feature/go-sdk-contrib/providers/configcat/pkg"
	"github.com/open-feature/go-sdk/pkg/openfeature"
	"github.com/stretchr/testify/require"
)

func TestMetadata(t *testing.T) {
	provider := configcat.NewProvider(clienttest.NewClient())
	require.Equal(t, openfeature.Metadata{
		Name: "ConfigCat",
	}, provider.Metadata())
}

func TestHooks(t *testing.T) {
	provider := configcat.NewProvider(clienttest.NewClient())
	require.Len(t, provider.Hooks(), 0)
}

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
		testEvalCtxNotString(t, func(evalCtx openfeature.FlattenedContext) openfeature.ProviderResolutionDetail {
			resolution := provider.BooleanEvaluation(ctx, "flag", false, evalCtx)
			return resolution.ProviderResolutionDetail
		})
	})

	t.Run("evalCtx stringer", func(t *testing.T) {
		testEvalCtxStringer(t, func(evalCtx openfeature.FlattenedContext) []clienttest.Request {
			defer client.Reset()

			provider.BooleanEvaluation(ctx, "flag", true, evalCtx)
			return client.GetRequests()
		})
	})

	t.Run("evalCtx keys set", func(t *testing.T) {
		testEvalCtxUserData(t, func(evalCtx openfeature.FlattenedContext) []clienttest.Request {
			defer client.Reset()

			provider.BooleanEvaluation(ctx, "flag", false, evalCtx)
			return client.GetRequests()
		})
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

func TestStringEvaluation(t *testing.T) {
	ctx := context.Background()
	client := clienttest.NewClient()
	provider := configcat.NewProvider(client)

	t.Run("evalCtx empty", func(t *testing.T) {
		defer client.Reset()
		expectedVariant := "ksljf"
		client.WithStringEvaluation(func(req clienttest.Request) sdk.StringEvaluationDetails {
			return sdk.StringEvaluationDetails{
				Value: "hi",
				Data: sdk.EvaluationDetailsData{
					VariationID: expectedVariant,
				},
			}
		})

		resolution := provider.StringEvaluation(ctx, "flag", "hello", nil)
		require.Equal(t, "hi", resolution.Value)
		require.Equal(t, expectedVariant, resolution.Variant)
		require.Equal(t, openfeature.DefaultReason, resolution.Reason)
	})

	t.Run("evalCtx non-stringer value", func(t *testing.T) {
		testEvalCtxNotString(t, func(evalCtx openfeature.FlattenedContext) openfeature.ProviderResolutionDetail {
			resolution := provider.StringEvaluation(ctx, "flag", "hello", evalCtx)
			return resolution.ProviderResolutionDetail
		})
	})

	t.Run("evalCtx stringer", func(t *testing.T) {
		testEvalCtxStringer(t, func(evalCtx openfeature.FlattenedContext) []clienttest.Request {
			defer client.Reset()

			provider.StringEvaluation(ctx, "flag", "hello", evalCtx)
			return client.GetRequests()
		})
	})

	t.Run("evalCtx keys set", func(t *testing.T) {
		testEvalCtxUserData(t, func(evalCtx openfeature.FlattenedContext) []clienttest.Request {
			defer client.Reset()

			provider.StringEvaluation(ctx, "flag", "hello", evalCtx)
			return client.GetRequests()
		})
	})

	t.Run("key not found", func(t *testing.T) {
		defer client.Reset()

		resolution := provider.StringEvaluation(ctx, "flag", "hello", nil)
		require.Equal(t, "hello", resolution.Value)
		require.Equal(t, openfeature.ErrorReason, resolution.Reason)
		require.Contains(t, resolution.ResolutionError.Error(), openfeature.FlagNotFoundCode)
	})

	t.Run("unknown error", func(t *testing.T) {
		defer client.Reset()
		client.WithStringEvaluation(func(req clienttest.Request) sdk.StringEvaluationDetails {
			return sdk.StringEvaluationDetails{
				Value: "hello",
				Data: sdk.EvaluationDetailsData{
					Error: errors.New("something went wrong"),
				},
			}
		})

		resolution := provider.StringEvaluation(ctx, "flag", "hello", nil)
		require.Equal(t, "hello", resolution.Value)
		require.Equal(t, openfeature.ErrorReason, resolution.Reason)
		require.Contains(t, resolution.ResolutionError.Error(), openfeature.GeneralCode)
	})

	t.Run("matched evaluation rule", func(t *testing.T) {
		defer client.Reset()
		client.WithStringEvaluation(func(req clienttest.Request) sdk.StringEvaluationDetails {
			return sdk.StringEvaluationDetails{
				Value: "hello",
				Data: sdk.EvaluationDetailsData{
					MatchedEvaluationRule: &sdk.RolloutRule{
						ComparisonAttribute: "attr",
						ComparisonValue:     "val",
						Comparator:          1,
					},
				},
			}
		})

		resolution := provider.StringEvaluation(ctx, "flag", "hello", nil)
		require.Equal(t, "hello", resolution.Value)
		require.Equal(t, openfeature.TargetingMatchReason, resolution.Reason)
	})

	t.Run("matched percentage rule", func(t *testing.T) {
		defer client.Reset()
		client.WithStringEvaluation(func(req clienttest.Request) sdk.StringEvaluationDetails {
			return sdk.StringEvaluationDetails{
				Value: "hello",
				Data: sdk.EvaluationDetailsData{
					MatchedEvaluationPercentageRule: &sdk.PercentageRule{
						Percentage: 50,
					},
				},
			}
		})

		resolution := provider.StringEvaluation(ctx, "flag", "hello", nil)
		require.Equal(t, "hello", resolution.Value)
		require.Equal(t, openfeature.TargetingMatchReason, resolution.Reason)
	})
}

func TestFloatEvaluation(t *testing.T) {
	ctx := context.Background()
	client := clienttest.NewClient()
	provider := configcat.NewProvider(client)

	t.Run("evalCtx empty", func(t *testing.T) {
		defer client.Reset()
		expectedVariant := "ksljf"
		client.WithFloatEvaluation(func(req clienttest.Request) sdk.FloatEvaluationDetails {
			return sdk.FloatEvaluationDetails{
				Value: 1.1,
				Data: sdk.EvaluationDetailsData{
					VariationID: expectedVariant,
				},
			}
		})

		resolution := provider.FloatEvaluation(ctx, "flag", 2.2, nil)
		require.Equal(t, 1.1, resolution.Value)
		require.Equal(t, expectedVariant, resolution.Variant)
		require.Equal(t, openfeature.DefaultReason, resolution.Reason)
	})

	t.Run("evalCtx non-stringer value", func(t *testing.T) {
		testEvalCtxNotString(t, func(evalCtx openfeature.FlattenedContext) openfeature.ProviderResolutionDetail {
			resolution := provider.FloatEvaluation(ctx, "flag", 1.7, evalCtx)
			return resolution.ProviderResolutionDetail
		})
	})

	t.Run("evalCtx stringer", func(t *testing.T) {
		testEvalCtxStringer(t, func(evalCtx openfeature.FlattenedContext) []clienttest.Request {
			defer client.Reset()

			provider.FloatEvaluation(ctx, "flag", 1.7, evalCtx)
			return client.GetRequests()
		})
	})

	t.Run("evalCtx keys set", func(t *testing.T) {
		testEvalCtxUserData(t, func(evalCtx openfeature.FlattenedContext) []clienttest.Request {
			defer client.Reset()

			provider.FloatEvaluation(ctx, "flag", 1.7, evalCtx)
			return client.GetRequests()
		})
	})

	t.Run("key not found", func(t *testing.T) {
		defer client.Reset()

		resolution := provider.FloatEvaluation(ctx, "flag", 1.7, nil)
		require.Equal(t, 1.7, resolution.Value)
		require.Equal(t, openfeature.ErrorReason, resolution.Reason)
		require.Contains(t, resolution.ResolutionError.Error(), openfeature.FlagNotFoundCode)
	})

	t.Run("unknown error", func(t *testing.T) {
		defer client.Reset()
		client.WithFloatEvaluation(func(req clienttest.Request) sdk.FloatEvaluationDetails {
			return sdk.FloatEvaluationDetails{
				Value: 3.4,
				Data: sdk.EvaluationDetailsData{
					Error: errors.New("something went wrong"),
				},
			}
		})

		resolution := provider.FloatEvaluation(ctx, "flag", 3.4, nil)
		require.Equal(t, 3.4, resolution.Value)
		require.Equal(t, openfeature.ErrorReason, resolution.Reason)
		require.Contains(t, resolution.ResolutionError.Error(), openfeature.GeneralCode)
	})

	t.Run("matched evaluation rule", func(t *testing.T) {
		defer client.Reset()
		client.WithFloatEvaluation(func(req clienttest.Request) sdk.FloatEvaluationDetails {
			return sdk.FloatEvaluationDetails{
				Value: 3.9,
				Data: sdk.EvaluationDetailsData{
					MatchedEvaluationRule: &sdk.RolloutRule{
						ComparisonAttribute: "attr",
						ComparisonValue:     "val",
						Comparator:          1,
					},
				},
			}
		})

		resolution := provider.FloatEvaluation(ctx, "flag", 3.9, nil)
		require.Equal(t, 3.9, resolution.Value)
		require.Equal(t, openfeature.TargetingMatchReason, resolution.Reason)
	})

	t.Run("matched percentage rule", func(t *testing.T) {
		defer client.Reset()
		client.WithFloatEvaluation(func(req clienttest.Request) sdk.FloatEvaluationDetails {
			return sdk.FloatEvaluationDetails{
				Value: 3.9,
				Data: sdk.EvaluationDetailsData{
					MatchedEvaluationPercentageRule: &sdk.PercentageRule{
						Percentage: 50,
					},
				},
			}
		})

		resolution := provider.FloatEvaluation(ctx, "flag", 3.9, nil)
		require.Equal(t, 3.9, resolution.Value)
		require.Equal(t, openfeature.TargetingMatchReason, resolution.Reason)
	})
}

func TestIntEvaluation(t *testing.T) {
	ctx := context.Background()
	client := clienttest.NewClient()
	provider := configcat.NewProvider(client)

	t.Run("evalCtx empty", func(t *testing.T) {
		defer client.Reset()
		expectedVariant := "ksljf"
		client.WithIntEvaluation(func(req clienttest.Request) sdk.IntEvaluationDetails {
			return sdk.IntEvaluationDetails{
				Value: 1,
				Data: sdk.EvaluationDetailsData{
					VariationID: expectedVariant,
				},
			}
		})

		resolution := provider.IntEvaluation(ctx, "flag", 2, nil)
		require.Equal(t, int64(1), resolution.Value)
		require.Equal(t, expectedVariant, resolution.Variant)
		require.Equal(t, openfeature.DefaultReason, resolution.Reason)
	})

	t.Run("evalCtx non-stringer value", func(t *testing.T) {
		testEvalCtxNotString(t, func(evalCtx openfeature.FlattenedContext) openfeature.ProviderResolutionDetail {
			resolution := provider.IntEvaluation(ctx, "flag", 1, evalCtx)
			return resolution.ProviderResolutionDetail
		})
	})

	t.Run("evalCtx stringer", func(t *testing.T) {
		testEvalCtxStringer(t, func(evalCtx openfeature.FlattenedContext) []clienttest.Request {
			defer client.Reset()

			provider.IntEvaluation(ctx, "flag", 1, evalCtx)
			return client.GetRequests()
		})
	})

	t.Run("evalCtx keys set", func(t *testing.T) {
		testEvalCtxUserData(t, func(evalCtx openfeature.FlattenedContext) []clienttest.Request {
			defer client.Reset()

			provider.IntEvaluation(ctx, "flag", 1, evalCtx)
			return client.GetRequests()
		})
	})

	t.Run("key not found", func(t *testing.T) {
		defer client.Reset()

		resolution := provider.IntEvaluation(ctx, "flag", 1, nil)
		require.Equal(t, int64(1), resolution.Value)
		require.Equal(t, openfeature.ErrorReason, resolution.Reason)
		require.Contains(t, resolution.ResolutionError.Error(), openfeature.FlagNotFoundCode)
	})

	t.Run("unknown error", func(t *testing.T) {
		defer client.Reset()
		client.WithIntEvaluation(func(req clienttest.Request) sdk.IntEvaluationDetails {
			return sdk.IntEvaluationDetails{
				Value: 3,
				Data: sdk.EvaluationDetailsData{
					Error: errors.New("something went wrong"),
				},
			}
		})

		resolution := provider.IntEvaluation(ctx, "flag", 3, nil)
		require.Equal(t, int64(3), resolution.Value)
		require.Equal(t, openfeature.ErrorReason, resolution.Reason)
		require.Contains(t, resolution.ResolutionError.Error(), openfeature.GeneralCode)
	})

	t.Run("matched evaluation rule", func(t *testing.T) {
		defer client.Reset()
		client.WithIntEvaluation(func(req clienttest.Request) sdk.IntEvaluationDetails {
			return sdk.IntEvaluationDetails{
				Value: 3,
				Data: sdk.EvaluationDetailsData{
					MatchedEvaluationRule: &sdk.RolloutRule{
						ComparisonAttribute: "attr",
						ComparisonValue:     "val",
						Comparator:          1,
					},
				},
			}
		})

		resolution := provider.IntEvaluation(ctx, "flag", 3, nil)
		require.Equal(t, int64(3), resolution.Value)
		require.Equal(t, openfeature.TargetingMatchReason, resolution.Reason)
	})

	t.Run("matched percentage rule", func(t *testing.T) {
		defer client.Reset()
		client.WithIntEvaluation(func(req clienttest.Request) sdk.IntEvaluationDetails {
			return sdk.IntEvaluationDetails{
				Value: 3,
				Data: sdk.EvaluationDetailsData{
					MatchedEvaluationPercentageRule: &sdk.PercentageRule{
						Percentage: 50,
					},
				},
			}
		})

		resolution := provider.IntEvaluation(ctx, "flag", 3, nil)
		require.Equal(t, int64(3), resolution.Value)
		require.Equal(t, openfeature.TargetingMatchReason, resolution.Reason)
	})
}

func TestObjectEvaluation(t *testing.T) {
	ctx := context.Background()
	client := clienttest.NewClient()
	provider := configcat.NewProvider(client)

	t.Run("evalCtx empty", func(t *testing.T) {
		defer client.Reset()
		expectedVariant := "ksljf"
		client.WithStringEvaluation(func(req clienttest.Request) sdk.StringEvaluationDetails {
			return sdk.StringEvaluationDetails{
				Value: `{"name":"test"}`,
				Data: sdk.EvaluationDetailsData{
					VariationID: expectedVariant,
				},
			}
		})

		resolution := provider.ObjectEvaluation(ctx, "flag", map[string]string{"name": "test"}, nil)
		require.Equal(t, map[string]interface{}{"name": "test"}, resolution.Value)
		require.Equal(t, expectedVariant, resolution.Variant)
		require.Equal(t, openfeature.DefaultReason, resolution.Reason)
	})

	t.Run("evalCtx non-stringer value", func(t *testing.T) {
		testEvalCtxNotString(t, func(evalCtx openfeature.FlattenedContext) openfeature.ProviderResolutionDetail {
			resolution := provider.ObjectEvaluation(ctx, "flag", 1, evalCtx)
			return resolution.ProviderResolutionDetail
		})
	})

	t.Run("evalCtx stringer", func(t *testing.T) {
		testEvalCtxStringer(t, func(evalCtx openfeature.FlattenedContext) []clienttest.Request {
			defer client.Reset()

			provider.ObjectEvaluation(ctx, "flag", 1, evalCtx)
			return client.GetRequests()
		})
	})

	t.Run("evalCtx keys set", func(t *testing.T) {
		testEvalCtxUserData(t, func(evalCtx openfeature.FlattenedContext) []clienttest.Request {
			defer client.Reset()

			provider.ObjectEvaluation(ctx, "flag", 1, evalCtx)
			return client.GetRequests()
		})
	})

	t.Run("key not found", func(t *testing.T) {
		defer client.Reset()

		expected := map[string]interface{}{"some": "default"}

		resolution := provider.ObjectEvaluation(ctx, "flag", expected, nil)
		require.Equal(t, expected, resolution.Value)
		require.Equal(t, openfeature.ErrorReason, resolution.Reason)
		require.Contains(t, resolution.ResolutionError.Error(), openfeature.FlagNotFoundCode)
	})

	t.Run("unknown error", func(t *testing.T) {
		defer client.Reset()
		client.WithStringEvaluation(func(req clienttest.Request) sdk.StringEvaluationDetails {
			return sdk.StringEvaluationDetails{
				Value: "",
				Data: sdk.EvaluationDetailsData{
					Error: errors.New("something went wrong"),
				},
			}
		})

		resolution := provider.ObjectEvaluation(ctx, "flag", nil, nil)
		require.Equal(t, nil, resolution.Value)
		require.Equal(t, openfeature.ErrorReason, resolution.Reason)
		require.Contains(t, resolution.ResolutionError.Error(), openfeature.GeneralCode)
	})

	t.Run("invalid json", func(t *testing.T) {
		defer client.Reset()
		client.WithStringEvaluation(func(req clienttest.Request) sdk.StringEvaluationDetails {
			return sdk.StringEvaluationDetails{
				Value: `{"invalid"}`,
			}
		})

		resolution := provider.ObjectEvaluation(ctx, "flag", nil, nil)
		require.Equal(t, nil, resolution.Value)
		require.Equal(t, openfeature.ErrorReason, resolution.Reason)
		require.Contains(t, resolution.ResolutionError.Error(), openfeature.TypeMismatchCode)
		require.Contains(t, resolution.ResolutionError.Error(), "failed to unmarshal")
	})

	t.Run("matched evaluation rule", func(t *testing.T) {
		defer client.Reset()
		client.WithStringEvaluation(func(req clienttest.Request) sdk.StringEvaluationDetails {
			return sdk.StringEvaluationDetails{
				Value: `{"domain":"example.org"}`,
				Data: sdk.EvaluationDetailsData{
					MatchedEvaluationRule: &sdk.RolloutRule{
						ComparisonAttribute: "attr",
						ComparisonValue:     "val",
						Comparator:          1,
					},
				},
			}
		})

		resolution := provider.ObjectEvaluation(ctx, "flag", nil, nil)
		require.Equal(t, map[string]interface{}{"domain": "example.org"}, resolution.Value)
		require.Equal(t, openfeature.TargetingMatchReason, resolution.Reason)
	})

	t.Run("matched percentage rule", func(t *testing.T) {
		defer client.Reset()
		client.WithStringEvaluation(func(req clienttest.Request) sdk.StringEvaluationDetails {
			return sdk.StringEvaluationDetails{
				Value: `{"domain":"example.org"}`,
				Data: sdk.EvaluationDetailsData{
					MatchedEvaluationPercentageRule: &sdk.PercentageRule{
						Percentage: 50,
					},
				},
			}
		})

		resolution := provider.ObjectEvaluation(ctx, "flag", nil, nil)
		require.Equal(t, map[string]interface{}{"domain": "example.org"}, resolution.Value)
		require.Equal(t, openfeature.TargetingMatchReason, resolution.Reason)
	})
}

func testEvalCtxNotString(t *testing.T, cb func(evalCtx openfeature.FlattenedContext) openfeature.ProviderResolutionDetail) {
	t.Helper()

	inputs := []struct {
		name    string
		evalCtx map[string]interface{}
	}{
		{
			name: "targeting",
			evalCtx: map[string]interface{}{
				openfeature.TargetingKey: new(configcat.Provider),
			},
		},
		{
			name: "identifier",
			evalCtx: map[string]interface{}{
				configcat.IdentifierKey: new(configcat.Provider),
			},
		},
		{
			name: "email",
			evalCtx: map[string]interface{}{
				configcat.EmailKey: new(configcat.Provider),
			},
		},
		{
			name: "country",
			evalCtx: map[string]interface{}{
				configcat.CountryKey: new(configcat.Provider),
			},
		},
		{
			name: "custom",
			evalCtx: map[string]interface{}{
				"some-key": new(configcat.Provider),
			},
		},
	}

	for _, input := range inputs {
		t.Run(input.name, func(t *testing.T) {
			detail := cb(input.evalCtx)
			require.Equal(t, openfeature.ErrorReason, detail.Reason)
			require.Contains(t, detail.ResolutionError.Error(), openfeature.InvalidContextCode)
		})
	}
}

func testEvalCtxStringer(t *testing.T, cb func(evalCtx openfeature.FlattenedContext) []clienttest.Request) {
	t.Helper()

	inputs := []struct {
		name     string
		val      interface{}
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
			requests := cb(map[string]interface{}{
				openfeature.TargetingKey: input.val,
			})
			require.Len(t, requests, 1)

			request := requests[0]
			require.Equal(t, input.expected, request.UserData().Identifier)
		})
	}
}

func testEvalCtxUserData(t *testing.T, cb func(evalCtx openfeature.FlattenedContext) []clienttest.Request) {
	t.Helper()

	expectedIdentifier := "123"
	expectedEmail := "example@example.com"
	expectedCountry := "AQ"
	expectedSomeKey := "some-value"

	requests := cb(map[string]interface{}{
		openfeature.TargetingKey: expectedIdentifier,
		configcat.EmailKey:       expectedEmail,
		configcat.CountryKey:     expectedCountry,
		"some-key":               expectedSomeKey,
	})
	require.Len(t, requests, 1)

	request := requests[0]
	require.Equal(t, expectedIdentifier, request.UserData().Identifier)
	require.Equal(t, expectedEmail, request.UserData().Email)
	require.Equal(t, expectedCountry, request.UserData().Country)
	require.Len(t, request.UserData().Custom, 1)
	require.Equal(t, expectedSomeKey, request.UserData().Custom["some-key"])
}
