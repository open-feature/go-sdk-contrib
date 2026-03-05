package gofeatureflag

import (
	of "github.com/open-feature/go-sdk/openfeature"
)

const targetingKey = "targetingKey"

func NewEvalFlagRequest[T JsonType](flatCtx of.FlattenedContext, defaultValue T) (EvalFlagRequest, *of.ResolutionError) {
	if _, ok := flatCtx[targetingKey]; !ok {
		err := of.NewTargetingKeyMissingResolutionError("no targetingKey provided in the evaluation context")
		return EvalFlagRequest{}, &err
	}
	targetingKeyVal, ok := flatCtx[targetingKey].(string)
	if !ok {
		err := of.NewTargetingKeyMissingResolutionError("targetingKey field MUST be a string")
		return EvalFlagRequest{}, &err
	}

	anonymous := true
	if val, ok := flatCtx["anonymous"].(bool); ok {
		anonymous = val
	}

	return EvalFlagRequest{
		User: &UserRequest{
			Key:       targetingKeyVal,
			Anonymous: anonymous,
			Custom:    flatCtx,
		},
		EvaluationContext: &EvaluationContextRequest{
			Key:    targetingKeyVal,
			Custom: flatCtx,
		},
		DefaultValue: defaultValue,
	}, nil
}

type EvalFlagRequest struct {
	User              *UserRequest              `json:"user" xml:"user" form:"user" query:"user"`
	EvaluationContext *EvaluationContextRequest `json:"evaluationContext,omitempty" xml:"evaluationContext,omitempty" form:"evaluationContext,omitempty" query:"evaluationContext,omitempty"`
	DefaultValue      any                       `json:"defaultValue" xml:"defaultValue" form:"defaultValue" query:"defaultValue"`
}

type UserRequest struct {
	Key       string         `json:"key" xml:"key" form:"key" query:"key" example:"08b5ffb7-7109-42f4-a6f2-b85560fbd20f"`
	Anonymous bool           `json:"anonymous" xml:"anonymous" form:"anonymous" query:"anonymous" example:"false"`
	Custom    map[string]any `json:"custom" xml:"custom" form:"custom" query:"custom" swaggertype:"object,string" example:"email:contact@gofeatureflag.org,firstname:John,lastname:Doe,company:GO Feature Flag"`
}

type EvaluationContextRequest struct {
	Key    string         `json:"key" xml:"key" form:"key" query:"key" example:"08b5ffb7-7109-42f4-a6f2-b85560fbd20f"`
	Custom map[string]any `json:"custom" xml:"custom" form:"custom" query:"custom" swaggertype:"object,string" example:"email:contact@gofeatureflag.org,firstname:John,lastname:Doe,company:GO Feature Flag"`
}
