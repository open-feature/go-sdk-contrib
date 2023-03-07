package posthog

import (
	"fmt"

	"github.com/open-feature/go-sdk/pkg/openfeature"
	"github.com/posthog/posthog-go"
)

const (
	GroupPropertiesKey       = "posthog.groupProperties"
	GroupsKey                = "posthog.groups"
	OnlyEvaluateLocallyKey   = "posthog.onlyEvaluateLocally"
	SendFeatureFlagEventsKey = "posthog.sendFeatureFlagEvents"
)

func evalContextToPayload(
	flag string,
	evalCtx openfeature.FlattenedContext,
) (posthog.FeatureFlagPayload, *openfeature.ResolutionError) {
	payload := posthog.FeatureFlagPayload{
		Key:              flag,
		PersonProperties: posthog.Properties(evalCtx),
	}

	if key, ok := evalCtx[openfeature.TargetingKey].(string); ok {
		payload.DistinctId = key
	} else {
		err := openfeature.NewTargetingKeyMissingResolutionError("no targeting key provided in the evaluation context")
		return payload, &err
	}

	var resErr *openfeature.ResolutionError

	payload.GroupProperties, resErr = getEvalContextByKey[map[string]posthog.Properties](evalCtx, GroupPropertiesKey)
	if resErr != nil {
		return payload, resErr
	}

	payload.Groups, resErr = getEvalContextByKey[posthog.Groups](evalCtx, GroupsKey)
	if resErr != nil {
		return payload, resErr
	}

	payload.OnlyEvaluateLocally, resErr = getEvalContextByKey[bool](evalCtx, OnlyEvaluateLocallyKey)
	if resErr != nil {
		return payload, resErr
	}

	payload.SendFeatureFlagEvents, resErr = getEvalContextByKey[*bool](evalCtx, SendFeatureFlagEventsKey)
	if resErr != nil {
		return payload, resErr
	}

	return payload, nil
}

func getEvalContextByKey[T any](
	evalCtx openfeature.FlattenedContext,
	key string,
) (T, *openfeature.ResolutionError) {
	if _, ok := evalCtx[key]; !ok {
		return *new(T), nil
	}

	if value, ok := evalCtx[key].(T); ok {
		return value, nil
	}

	msg := fmt.Sprintf("invalid type %T for key %s", evalCtx[key], key)
	err := openfeature.NewInvalidContextResolutionError(msg)
	return *new(T), &err
}
