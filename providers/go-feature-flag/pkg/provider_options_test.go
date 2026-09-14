package gofeatureflag

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProviderOptions_Validation(t *testing.T) {
	tests := []struct {
		name       string
		options    ProviderOptions
		wantErr    bool
		wantErrMsg string
	}{
		{
			name:       "valid endpoint",
			options:    ProviderOptions{Endpoint: "http://localhost:1031"},
			wantErr:    false,
			wantErrMsg: "",
		},
		{
			name:       "missing endpoint",
			options:    ProviderOptions{Endpoint: ""},
			wantErr:    true,
			wantErrMsg: "invalid option: ",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.options.Validation()
			if tc.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErrMsg)
				return
			}
			require.NoError(t, err)
		})
	}
}
