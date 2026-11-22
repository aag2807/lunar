package lexer

import (
	"strings"
	"testing"
)

// firstString lexes src and returns its first STRING token.
func firstString(t *testing.T, src string) Token {
	t.Helper()
	l := New(src)
	for i := 0; i < 50; i++ {
		tok := l.NextToken()
		if tok.Type == STRING || tok.Type == ILLEGAL || tok.Type == EOF {
			return tok
		}
	}
	t.Fatalf("%q: no string", src)
	return Token{}
}

// String contents are bytes, decoded as Lua decodes them.
func TestStringEscapesAreLuas(t *testing.T) {
	cases := map[string]string{
		`"\65\066\0671"`:    "ABC1",
		`"\0001"`:           "\x001",
		`"\a\b\f\n\r\t\v"`:  "\a\b\f\n\r\t\v",
		`"\xff\x00"`:        "\xff\x00",
		"\"a\\z  \n\t b\"":  "ab",
		"\"a\\\nb\"":        "a\nb",
		"\"a\\\r\nb\"":      "a\nb",
		"\"a\\\n\rb\"":      "a\nb",
		`"\u{41}\u{7FF}"`:   "A߿",
		`"\u{D800}"`:        "\xed\xa0\x80",
		`"\u{110000}"`:      "\xf4\x90\x80\x80",
		`"\u{7FFFFFFF}"`:    "\xfd\xbf\xbf\xbf\xbf\xbf",
		"\"caf\xe9\"":       "caf\xe9",
		"\"a\x00b\"":        "a\x00b",
		"[[\nx]]":           "x",
		"[[\r\nx\r\ny\rz]]": "x\ny\nz",
		"[[\n\nx]]":         "\nx",
		"[==[a]]b]=]c]==]":  "a]]b]=]c",
	}
	for src, want := range cases {
		tok := firstString(t, src)
		if tok.Type != STRING || tok.Literal != want {
			t.Errorf("%q: got %s %q, want %q", src, tok.Type, tok.Literal, want)
		}
	}
}

// What Lua refuses, Lunar refuses with Lua's reason.
func TestInvalidEscapesAreErrors(t *testing.T) {
	cases := map[string]string{
		`"\q"`:           "invalid escape sequence '\\q'",
		`"\256"`:         "decimal escape too large",
		`"\xg0"`:         "hexadecimal digit expected",
		`"\u{}"`:         "hexadecimal digit expected",
		`"\u{80000000}"`: "UTF-8 value too large",
		"\"a\nb\"":       "unterminated string",
	}
	for src, want := range cases {
		tok := firstString(t, src)
		if tok.Type != ILLEGAL || tok.Literal != want {
			t.Errorf("%q: got %s %q, want ILLEGAL %q", src, tok.Type, tok.Literal, want)
		}
	}
}

func types(src string) []Token {
	var out []Token
	l := New(src)
	for i := 0; i < 100; i++ {
		tok := l.NextToken()
		out = append(out, tok)
		if tok.Type == EOF {
			break
		}
	}
	return out
}

// A NUL byte is not the end of the file; Lua refuses it outside a string.
func TestNulByteIsNotEndOfFile(t *testing.T) {
	toks := types("print(1)\x00print(2)")
	sawIllegal := false
	for _, tok := range toks {
		if tok.Type == ILLEGAL {
			sawIllegal = true
		}
	}
	if !sawIllegal {
		t.Errorf("NUL byte lexed as %v", toks)
	}
}

// Long comments of every level, and numerals with a trailing point.
func TestLongCommentsAndNumerals(t *testing.T) {
	toks := types("--[==[ x ]] ]=] ]==] a --[=[\n]]\n]=] 4. 2.E-1 0xA. 1..2")
	var lits []string
	for _, tok := range toks {
		if tok.Type != EOF {
			lits = append(lits, tok.Literal)
		}
	}
	want := []string{"a", "4.", "2.E-1", "0xA.", "1", "..", "2"}
	if len(lits) != len(want) {
		t.Fatalf("got %q, want %q", lits, want)
	}
	for i := range want {
		if lits[i] != want[i] {
			t.Fatalf("got %q, want %q", lits, want)
		}
	}
}

// A leading UTF-8 byte-order mark and a #! line are skipped, as lua does;
// form feed and vertical tab are whitespace.
func TestBomShebangAndWhitespace(t *testing.T) {
	for _, src := range []string{"\xef\xbb\xbfprint(1)", "#!/usr/bin/env lua\nprint(1)", "\xef\xbb\xbf#!lua\nprint(1)", "\f\vprint(1)"} {
		toks := types(src)
		if toks[0].Literal != "print" {
			t.Errorf("%q: first token %s %q", src, toks[0].Type, toks[0].Literal)
		}
	}
}

// A carriage return alone ends a line, as in Lua: a comment on the first
// line of a CR-only file does not swallow the rest.
func TestCarriageReturnEndsALine(t *testing.T) {
	toks := types("-- comment\rprint(1)\rprint(2)\r")
	if toks[0].Literal != "print" || toks[0].Line != 2 {
		t.Errorf("first token %q on line %d", toks[0].Literal, toks[0].Line)
	}
}

// A numeral runs into the letters after it: 5print is one malformed
// number, as Lua reads it, not 5 then print.
func TestMalformedNumber(t *testing.T) {
	for _, src := range []string{"5print()", "0x1g", "3e"} {
		toks := types(src)
		if toks[0].Type != ILLEGAL || toks[0].Literal != "malformed number near '"+src[:len(src)-len(trailing(src))]+"'" {
			t.Errorf("%q: %s %q", src, toks[0].Type, toks[0].Literal)
		}
	}
}

func trailing(src string) string {
	if i := strings.IndexByte(src, '('); i >= 0 {
		return src[i:]
	}
	return ""
}
