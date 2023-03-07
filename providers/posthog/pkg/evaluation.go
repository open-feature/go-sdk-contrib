package posthog

import (
	"fmt"

	"github.com/open-feature/go-sdk/pkg/openfeature"
	"github.com/posthog/posthog-go"
)

// evaluate is a generic function for evaluating feature flags.
// It returns a value of the same type as defaultValue.
// In case of error, it returns defaultValue and an error.
// It forms request payload from evalCtx and sends it to Posthog.
// All ints are converted to int64, all floats are converted to float64.
// For multivariate feature flags, it returns a string value, the Variant will contain the same string.
func evaluate[T any](
	client posthog.Client,
	flag string,
	defaultValue T,
	evalCtx openfeature.FlattenedContext,
) (T, openfeature.ProviderResolutionDetail) {
	payload, resolutionErr := evalContextToPayload(flag, evalCtx)
	if resolutionErr != nil {
		return defaultValue, openfeature.ProviderResolutionDetail{
			Reason:          openfeature.ErrorReason,
			ResolutionError: *resolutionErr,
		}
	}

	res, err := client.GetFeatureFlag(payload)
	if err != nil {
		return defaultValue, errToResolutionDetail(err)
	}

	// res will be nil when using OnlyEvaluateLocally or
	// when an error occurs during the evaluation. Error will be nil too.
	// We don't need additional check, because type assertion will fail on nil
	// and TypeMismatchResolutionError will be returned.

	if res, ok := normalizeResult(res).(T); ok {
		return res, openfeature.ProviderResolutionDetail{
			Reason:  openfeature.UnknownReason, // Posthog doesn't expose the reason
			Variant: getFlagVariant(res),
		}
	}

	return defaultValue, openfeature.ProviderResolutionDetail{
		Reason:          openfeature.ErrorReason,
		ResolutionError: openfeature.NewTypeMismatchResolutionError(fmt.Sprintf("expected %T, received %T", defaultValue, res)),
	}
}

func errToResolutionDetail(err error) openfeature.ProviderResolutionDetail {
	// Currently, PostHog lib doesn't expose errors, so we can only return a generic error
	return openfeature.ProviderResolutionDetail{
		Reason:          openfeature.ErrorReason,
		ResolutionError: openfeature.NewGeneralResolutionError(err.Error()),
	}
}

func normalizeResult(n any) any {
	switch n := n.(type) {
	case int:
		return int64(n)
	case int8:
		return int64(n)
	case int16:
		return int64(n)
	case int32:
		return int64(n)
	case int64:
		return int64(n)
	case float32:
		return float64(n)
	default:
		return n
	}
}

func getFlagVariant(res any) string {
	// In Posthog Multivariate feature flags are just list of strings,
	// each variant is a string representing both variant and value.
	// https://posthog.com/manual/feature-flags#creating-a-feature-flag-with-multiple-variants

	if str, ok := res.(string); ok {
		return str
	}

	return ""
}
