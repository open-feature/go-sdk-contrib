package tck

import (
	"reflect"
	"strings"
	"testing"
)

func TestAContextValueCarriesTheTypeItsRowDeclares(t *testing.T) {
	for _, c := range []struct {
		rawType string
		raw     string
		want    any
	}{
		{"Boolean", "true", true},
		{"String", "eu", "eu"},
		{"Integer", "7", int64(7)},
		{"Float", "0.5", 0.5},
		{"Long", "4294967295", int64(4294967295)},
		{"StringList", "one, two", []string{"one", "two"}},
		// Not [""]: a blank cell says the key carries no entries, and an
		// operator asked whether "" is among them would otherwise answer yes.
		{"StringList", "", []string{}},
	} {
		t.Run(c.rawType+"/"+c.raw, func(t *testing.T) {
			got, err := parseContextValue(c.raw, c.rawType)
			if err != nil {
				t.Fatalf("parseContextValue(%q, %q): %v", c.raw, c.rawType, err)
			}
			if !reflect.DeepEqual(got, c.want) {
				t.Errorf("got %s, want %s", describeValue(got), describeValue(c.want))
			}
		})
	}
}

func TestAnIntegerContextValueIsNotBoundedToThirtyTwoBits(t *testing.T) {
	// This is where Java's TCK needs a widening rule and this one does not: its
	// flag conversion refuses a value getIntegerDetails cannot be asked for,
	// and its context conversion has to let the same value through because a
	// targeting operator may hash it. IntValueDetails takes an int64, so both
	// conversions are 64 bits wide here and the value passes either way.
	got, err := parseContextValue("4294967295", "Integer")
	if err != nil {
		t.Fatalf("an Integer context value a targeting operator may hash was refused: %v", err)
	}
	if got != int64(4294967295) {
		t.Errorf("got %s, want int64(4294967295)", describeValue(got))
	}

	viaFlagType, err := typeInteger.parseValue("4294967295")
	if err != nil {
		t.Fatalf("the flag conversion refused it, which would make the two rules differ: %v", err)
	}
	if viaFlagType != got {
		t.Errorf("the flag and context conversions disagree: %s against %s",
			describeValue(viaFlagType), describeValue(got))
	}
}

func TestAnUnknownContextValueTypeNamesEveryTypeItCouldHaveBeen(t *testing.T) {
	_, err := parseContextValue("PT1S", "Duration")
	if err == nil {
		t.Fatal("parseContextValue accepted a type no evaluation context entry can carry")
	}

	// The whole vocabulary, because the usual cause is a table written against
	// a name one of the other languages' suites happens to use.
	for _, want := range []string{"Duration", "Boolean", "String", "Integer", "Float", "Object", "Long", "StringList"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the error does not mention %q: %v", want, err)
		}
	}
}
