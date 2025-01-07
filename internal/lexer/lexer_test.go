package lexer

import (
	"testing"
)

func TestReadChar(t *testing.T) {
	input := "class\npoint"
	l := New(input)

	if l.ch != 'c' {
		t.Errorf("first char wrong. expected='c', got='%c'", l.ch)
	}

	if l.line != 1 {
		t.Errorf("line number wrong. expected=1 got=%d", l.line)
	}

	for i := 0; i < 5; i++ {
		l.readChar()
	}

	if l.line != 2 {
		t.Errorf("line number after newline wrong. expected=2 got=%d", l.line)
	}

	if l.column != 0 {
		t.Errorf("column after newline wrong. expected=0, got=%d", l.column)
	}
}

func TestNextToken(t *testing.T) {
	input := `class Point
	private x: number`

	tests := []struct {
		expectedType    TokenType
		expectedLiteral string
	}{
		{TokenType(CLASS), "class"},
		{TokenType(IDENT), "Point"},
		{TokenType(PRIVATE), "private"},
		{TokenType(IDENT), "x"},
		{TokenType(COLON), ":"},
		{TokenType(IDENT), "number"},
		{TokenType(EOF), ""},
	}

	l := New(input)

	for i, tt := range tests {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype is wrong, expected=%q, got=%q",
				i, tt.expectedType, tok.Type)
		}

		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal is wrong, expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}

}
