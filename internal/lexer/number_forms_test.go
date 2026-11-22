package lexer

import "testing"

// Lunar is a superset of Lua, so every Lua numeral must lex as one NUMBER
// token: exponents, hexadecimal (with optional fraction and binary
// exponent) and a leading decimal point.
func TestLuaNumberForms(t *testing.T) {
	for _, src := range []string{
		"42", "3.14", "1e-9", "3E+2", "2.5e10", "6e2",
		"0xFF", "0X1f", "0xA.8p1", "0x1P-2", "0x.8",
		".5", ".25e2",
	} {
		l := New(src)
		tok := l.NextToken()
		if tok.Type != NUMBER || tok.Literal != src {
			t.Errorf("%q lexed as %s %q, want one NUMBER token", src, tok.Type, tok.Literal)
			continue
		}
		if next := l.NextToken(); next.Type != EOF {
			t.Errorf("%q left %s %q after the number", src, next.Type, next.Literal)
		}
	}
}

// The new number forms must not swallow the operators that share their
// characters.
func TestNumberFormsLeaveDotOperatorsAlone(t *testing.T) {
	cases := map[string][]TokenType{
		"a..b":   {IDENT, CONCAT, IDENT},
		"1 .. 2": {NUMBER, CONCAT, NUMBER},
		"t.x":    {IDENT, DOT, IDENT},
		"f(...)": {IDENT, LPAREN, ELLIPSIS, RPAREN},
		"x[1].e": {IDENT, LBRACKET, NUMBER, RBRACKET, DOT, IDENT},
		"n-1":    {IDENT, MINUS, NUMBER},
		"2e":     {NUMBER, IDENT},
	}
	for src, want := range cases {
		l := New(src)
		for i, typ := range want {
			tok := l.NextToken()
			if tok.Type != typ {
				t.Errorf("%q token %d: got %s %q, want %s", src, i, tok.Type, tok.Literal, typ)
				break
			}
		}
	}
}
