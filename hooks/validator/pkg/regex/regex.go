package regex

import (
	"errors"
	of "github.com/open-feature/go-sdk/pkg/openfeature"
	"regexp"
)

// Validator implements the validator interface
type Validator struct {
	RegularExpression *regexp.Regexp
}

// NewValidator compiles the given regex and returns a Validator
func NewValidator(regularExpression string) (Validator, error) {
	r, err := regexp.Compile(regularExpression)
	if err != nil {
		return Validator{}, err
	}

	return Validator{RegularExpression: r}, nil
}

// IsValid returns an error if the flag evaluation details value isn't a hex color
func (v Validator) IsValid(flagEvaluationDetails of.InterfaceEvaluationDetails) error {
	s, ok := flagEvaluationDetails.Value.(string)
	if !ok {
		return errors.New("flag value isn't of type string")
	}

	if !v.RegularExpression.MatchString(s) {
		return errors.New("regex doesn't match on flag value")
	}

	return nil
}
