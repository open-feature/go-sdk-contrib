package validator

import (
	of "github.com/open-feature/go-sdk/pkg/openfeature"
)

type validator interface {
	IsValid(flagEvaluationDetails of.InterfaceEvaluationDetails) error
}

// Hook validates the flag evaluation details After flag resolution.
type Hook struct {
	Validator validator
}

func (h Hook) Before(hookContext of.HookContext, hookHints of.HookHints) (*of.EvaluationContext, error) {
	return nil, nil
}

func (h Hook) After(hookContext of.HookContext, flagEvaluationDetails of.InterfaceEvaluationDetails, hookHints of.HookHints) error {
	err := h.Validator.IsValid(flagEvaluationDetails)
	if err != nil {
		return err
	}

	return nil
}

func (h Hook) Error(hookContext of.HookContext, err error, hookHints of.HookHints) {}

func (h Hook) Finally(hookContext of.HookContext, hookHints of.HookHints) {}
