package util

import "go.openfeature.dev/openfeature/v2"

const targetingKey = "targetingKey"

func ValidateTargetingKey(evalCtx openfeature.FlattenedContext) *openfeature.ResolutionError {
	if _, ok := evalCtx[targetingKey]; !ok {
		err := openfeature.NewTargetingKeyMissingResolutionError("no targetingKey provided in the evaluation context")
		return &err
	}

	if _, ok := evalCtx[targetingKey].(string); !ok {
		err := openfeature.NewTargetingKeyMissingResolutionError("targetingKey field MUST be a string")
		return &err
	}

	return nil
}
