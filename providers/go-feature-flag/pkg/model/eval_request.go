package model

import (
	of "github.com/open-feature/go-sdk/pkg/openfeature"
)

const targetingKey = "targetingKey"

func NewEvalFlagRequest[T JsonType](flatCtx of.FlattenedContext, defaultValue T) (EvalFlagRequest, *of.ResolutionError) {
	if _, ok := flatCtx[targetingKey]; !ok {
		err := of.NewTargetingKeyMissingResolutionError("no targetingKey provided in the evaluation context")
		return EvalFlagRequest{}, &err
	}
	targetingKey, ok := flatCtx[targetingKey].(string)
	if !ok {
		err := of.NewTargetingKeyMissingResolutionError("targetingKey field MUST be a string")
		return EvalFlagRequest{}, &err
	}

	anonymous := true
	if val, ok := flatCtx["anonymous"].(bool); ok {
		anonymous = val
	}

	return EvalFlagRequest{
		// We keep user to be compatible with old version of GO Feature Flag proxy.
		User: &UserRequest{
			Key:       targetingKey,
			Anonymous: anonymous,
			Custom:    flatCtx,
		},
		EvaluationContext: &EvaluationContextRequest{
			Key:    targetingKey,
			Custom: flatCtx,
		},
		DefaultValue: defaultValue,
	}, nil
}

type EvalFlagRequest struct {
	// User The representation of a user for your feature flag system.
	User *UserRequest `json:"user" xml:"user" form:"user" query:"user"`
	// EvaluationContext the context to evaluate the flag.
	EvaluationContext *EvaluationContextRequest `json:"evaluationContext,omitempty" xml:"evaluationContext,omitempty" form:"evaluationContext,omitempty" query:"evaluationContext,omitempty"`
	// The value will we use if we are not able to get the variation of the flag.
	DefaultValue interface{} `json:"defaultValue" xml:"defaultValue" form:"defaultValue" query:"defaultValue"`
}

// UserRequest The representation of a user for your feature flag system.
type UserRequest struct {
	// Key is the identifier of the UserRequest.
	Key string `json:"key" xml:"key" form:"key" query:"key" example:"08b5ffb7-7109-42f4-a6f2-b85560fbd20f"`

	// Anonymous set if this is a logged-in user or not.
	Anonymous bool `json:"anonymous" xml:"anonymous" form:"anonymous" query:"anonymous" example:"false"`

	// Custom is a map containing all extra information for this user.
	Custom map[string]interface{} `json:"custom" xml:"custom" form:"custom" query:"custom"  swaggertype:"object,string" example:"email:contact@gofeatureflag.org,firstname:John,lastname:Doe,company:GO Feature Flag"` // nolint: lll
}

// EvaluationContextRequest The representation of the evaluation context.
type EvaluationContextRequest struct {
	// Key is the identifier of the UserRequest.
	Key string `json:"key" xml:"key" form:"key" query:"key" example:"08b5ffb7-7109-42f4-a6f2-b85560fbd20f"`

	// Custom is a map containing all extra information for this user.
	Custom map[string]interface{} `json:"custom" xml:"custom" form:"custom" query:"custom"  swaggertype:"object,string" example:"email:contact@gofeatureflag.org,firstname:John,lastname:Doe,company:GO Feature Flag"` // nolint: lll
}
