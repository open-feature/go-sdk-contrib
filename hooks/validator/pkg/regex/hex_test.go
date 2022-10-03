package regex_test

import (
	"github.com/open-feature/go-sdk-contrib/hooks/validator/pkg/regex"
	of "github.com/open-feature/go-sdk/pkg/openfeature"
	"testing"
)

func TestValidator_Hex(t *testing.T) {
	tests := map[string]struct {
		flagEvaluationDetails of.InterfaceEvaluationDetails
		expectedErr           bool
	}{
		"#112233": {
			flagEvaluationDetails: of.InterfaceEvaluationDetails{
				Value: "#112233",
			},
			expectedErr: false,
		},
		"#123": {
			flagEvaluationDetails: of.InterfaceEvaluationDetails{
				Value: "#123",
			},
			expectedErr: false,
		},
		"#000233": {
			flagEvaluationDetails: of.InterfaceEvaluationDetails{
				Value: "#000233",
			},
			expectedErr: false,
		},
		"#023": {
			flagEvaluationDetails: of.InterfaceEvaluationDetails{
				Value: "#023",
			},
			expectedErr: false,
		},
		"invalid": {
			flagEvaluationDetails: of.InterfaceEvaluationDetails{
				Value: "invalid",
			},
			expectedErr: true,
		},
		"#abcd": {
			flagEvaluationDetails: of.InterfaceEvaluationDetails{
				Value: "#abcd",
			},
			expectedErr: true,
		},
		"#-12": {
			flagEvaluationDetails: of.InterfaceEvaluationDetails{
				Value: "#-12",
			},
			expectedErr: true,
		},
		"non string": {
			flagEvaluationDetails: of.InterfaceEvaluationDetails{
				Value: 3,
			},
			expectedErr: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			validator, err := regex.Hex()
			if err != nil {
				t.Fatal(err)
			}

			err = validator.IsValid(tt.flagEvaluationDetails)
			if err != nil {
				if !tt.expectedErr {
					t.Errorf("didn't expect error, got: %v", err)
				}
			} else {
				if tt.expectedErr {
					t.Error("expected error, got nil")
				}
			}
		})
	}
}
