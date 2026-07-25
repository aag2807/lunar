package lexer

import "testing"

// Lua's hexadecimal literals have to lex as one number; without this 0xFF read
// as the number 0 followed by an identifier.
func TestHexadecimalLiterals(t *testing.T) {
	for _, input := range []string{"0x0F", "0xFF", "0Xff", "0x7fffffff"} {
		l := New(input)
		tok := l.NextToken()

		if tok.Type != NUMBER {
			t.Errorf("%q: expected NUMBER, got %s", input, tok.Type)
		}
		if tok.Literal != input {
			t.Errorf("%q: expected the whole literal, got %q", input, tok.Literal)
		}
		if next := l.NextToken(); next.Type != EOF {
			t.Errorf("%q: expected nothing after the number, got %s %q", input, next.Type, next.Literal)
		}
	}
}

func TestScientificNotation(t *testing.T) {
	for _, input := range []string{"1e3", "2.5e-3", "1E6", "1.5e+2"} {
		l := New(input)
		tok := l.NextToken()

		if tok.Type != NUMBER || tok.Literal != input {
			t.Errorf("%q: got %s %q", input, tok.Type, tok.Literal)
		}
	}
}

// A plain decimal must not swallow a following identifier.
func TestDecimalFollowedByName(t *testing.T) {
	l := New("1 end")

	if tok := l.NextToken(); tok.Literal != "1" {
		t.Errorf("expected 1, got %q", tok.Literal)
	}
	if tok := l.NextToken(); tok.Type != END {
		t.Errorf("expected END, got %s", tok.Type)
	}
}
