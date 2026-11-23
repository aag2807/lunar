package parser

import (
	"strings"
	"testing"
)

// An arrow function type names its parameters as a function does: self,
// Lunar's own words, optional ones and varargs. (number?) stays a type.
func TestArrowTypeParameterNames(t *testing.T) {
	for _, src := range []string{
		"local f: (self: any) => void",
		"local f: (self: any, n: number) => number",
		"local f: (type: string, from: number) => void",
		"local f: (n?: number) => void",
		"local f: (...: number) => void",
		"local f: (number?) => void",
		"local f: (number, string) => void",
	} {
		if _, errs := parseWithErrors(src); len(errs) > 0 {
			t.Errorf("%s: %v", src, errs[0].Message)
		}
	}
}

// '[ ... ]' is not a type: one error that says what to write, then the
// statement goes on, with no cascade from the rest of it.
func TestBracketTypeIsOneError(t *testing.T) {
	_, errs := parseWithErrors("local a = function() end\nlocal t: [() => void] = { a }\nprint(t)\n")
	if len(errs) != 1 || !strings.Contains(errs[0].Message, "'[ ... ]' is not a type") {
		var msgs []string
		for _, e := range errs {
			msgs = append(msgs, e.Message)
		}
		t.Errorf("want one error about '[ ... ]', got %q", msgs)
	}
}
