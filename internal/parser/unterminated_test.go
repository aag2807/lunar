package parser

import (
	"strings"
	"testing"
)

// The parser reports what the lexer could not close, in plain words.
func TestUnterminatedLiteralIsASyntaxError(t *testing.T) {
	for src, want := range map[string]string{
		`local s = "abc`:            "unterminated string",
		"local s = `abc":            "unterminated template string",
		"local s = [[abc":           "unterminated long string",
		"local x = 1\n--[[ comment": "unterminated long comment",
	} {
		_, errs := parseWithErrors(src)
		if len(errs) == 0 || !strings.Contains(errs[0].Message, want) {
			t.Errorf("%q: got %v, want %q", src, errs, want)
		}
	}
}
