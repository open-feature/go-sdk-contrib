package gofferror_test

import (
	"testing"

	"github.com/open-feature/go-sdk-contrib/providers/go-feature-flag/pkg/gofferror"
)

func TestInvalidOption_Error(t *testing.T) {
	tests := []struct {
		name    string
		message string
		want    string
	}{
		{
			name:    "simple error message",
			message: "invalid configuration",
			want:    "invalid configuration",
		},
		{
			name: "empty message",
			want: "",
		},
		{
			name:    "complex error message",
			message: "invalid option: timeout must be greater than 0",
			want:    "invalid option: timeout must be greater than 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := gofferror.InvalidOption{Message: tt.message}
			if got := i.Error(); got != tt.want {
				t.Errorf("InvalidOption.Error() = %v, want %v", got, tt.want)
			}
		})
	}
}
