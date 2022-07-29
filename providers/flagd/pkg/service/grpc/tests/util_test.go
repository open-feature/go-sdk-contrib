package tests

import (
	"errors"
	"testing"
)

type TestCheckErrorArgs struct {
	name string
	err1 error
	err2 error
	out  bool
}

func TestCheckError(t *testing.T) {
	tests := []TestCheckErrorArgs{
		{
			name: "both nil",
			err1: nil,
			err2: nil,
			out:  true,
		},
		{
			name: "err1 nil",
			err1: nil,
			err2: errors.New("error"),
			out:  false,
		},
		{
			name: "err2 nil",
			err1: errors.New("error"),
			err2: nil,
			out:  false,
		},
		{
			name: "unmatched",
			err1: errors.New("error!"),
			err2: errors.New("error"),
			out:  false,
		},

		{
			name: "match",
			err1: errors.New("error"),
			err2: errors.New("error"),
			out:  true,
		},
	}

	for _, test := range tests {
		if out := errorCompare(test.err1, test.err2); out != test.out {
			t.Errorf("%s: unexpected result %t", test.name, out)
		}
	}
}
