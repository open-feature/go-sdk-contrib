package hex

import (
	"errors"
	of "github.com/open-feature/golang-sdk/pkg/openfeature"
)

var errInvalidFormat = errors.New("invalid format")

type Validator struct{}

// IsValid returns an error if the flag evaluation details value isn't a hex color
func (v Validator) IsValid(flagEvaluationDetails of.EvaluationDetails) error {
	s, ok := flagEvaluationDetails.Value.(string)
	if !ok {
		return errors.New("flag value isn't of type string")
	}

	if len(s) != 7 && len(s) != 4 {
		return errInvalidFormat
	}

	if s[0] != '#' {
		return errInvalidFormat
	}

	for i := 1; i < len(s); i++ {
		if err := validateByte(s[i]); err != nil {
			return errInvalidFormat
		}
	}

	return nil
}

func validateByte(b byte) error {
	if (b >= '0' && b <= '9') || (b >= 'a' && b <= 'f') || (b >= 'A' && b <= 'F') {
		return nil
	}

	return errInvalidFormat
}
