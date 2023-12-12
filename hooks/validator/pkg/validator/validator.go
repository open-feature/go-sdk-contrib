package validator

import (
	"context"
	of "github.com/open-feature/go-sdk/openfeature"
)

type validator interface {
	IsValid(flagEvaluationDetails of.InterfaceEvaluationDetails) error
}

// Hook validates the flag evaluation details After flag resolution
type Hook struct {
	of.UnimplementedHook
	Validator validator
}

func (h Hook) After(ctx context.Context, hookContext of.HookContext, flagEvaluationDetails of.InterfaceEvaluationDetails, hookHints of.HookHints) error {
	err := h.Validator.IsValid(flagEvaluationDetails)
	if err != nil {
		return err
	}

	return nil
}
