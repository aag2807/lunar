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

	if l.line != 1 {
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
		expectedLine    int
	}{
		{TokenType(CLASS), "class", 1},
		{TokenType(IDENT), "Point", 1},
		{TokenType(PRIVATE), "private", 2},
		{TokenType(IDENT), "x", 2},
		{TokenType(COLON), ":", 2},
		{TokenType(IDENT), "number", 2},
		{TokenType(EOF), "", 2},
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

		if tok.Line != tt.expectedLine {
			t.Fatalf("tests[%d] - line is wrong, expected=%q, got=%q",
				i, tt.expectedLine, tok.Line)
		}
	}
}

func TestNumberTokens(t *testing.T) {
	input := `42
	3.14
	100
	0.123`

	tests := []struct {
		expectedType    TokenType
		expectedLiteral string
		expectedLine    int
	}{
		{TokenType(NUMBER), "42", 1},
		{TokenType(NUMBER), "3.14", 2},
		{TokenType(NUMBER), "100", 3},
		{TokenType(NUMBER), "0.123", 4},
		{TokenType(EOF), "", 4},
	}

	l := New(input)

	for i, tt := range tests {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q",
				i, tt.expectedType, tok.Type)
		}

		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}

		if tok.Line != tt.expectedLine {
			t.Fatalf("tests[%d] - line is wrong, expected=%q, got=%q",
				i, tt.expectedLine, tok.Line)
		}
	}
}

func TestStringTokens(t *testing.T) {
	input := `"simple string"
    "string with \"quotes\""
    "string with \n newline"
    "string with \t tab"
    "multiple
    lines"
    "escaped \\backslash"`

	tests := []struct {
		expectedType    TokenType
		expectedLiteral string
		expectedLine    int
	}{
		{TokenType(STRING), "simple string", 1},
		{TokenType(STRING), "string with \"quotes\"", 2},
		{TokenType(STRING), "string with \n newline", 3},
		{TokenType(STRING), "string with \t tab", 4},
		{TokenType(STRING), "multiple\n    lines", 5},
		{TokenType(STRING), "escaped \\backslash", 6},
		{TokenType(EOF), "", 6},
	}

	l := New(input)
	for i, tt := range tests {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q",
				i, tt.expectedType, tok.Type)
		}

		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}

		if tok.Line != tt.expectedLine {
			t.Errorf("tests[%d] - line number wrong. expected=%d, got=%d",
				i, tt.expectedLine, tok.Line)
		}
	}
}

func TestComments(t *testing.T) {
	input := `-- Single line comment
local x = 5 -- Inline comment
--[[ Multi
line
comment ]]
local y = 10`

	tests := []struct {
		expectedType    TokenType
		expectedLiteral string
		expectedLine    int
	}{
		{TokenType(LOCAL), "local", 2},
		{TokenType(IDENT), "x", 2},
		{TokenType(ASSIGN), "=", 2},
		{TokenType(NUMBER), "5", 2},
		{TokenType(LOCAL), "local", 6},
		{TokenType(IDENT), "y", 6},
		{TokenType(ASSIGN), "=", 6},
		{TokenType(NUMBER), "10", 6},
		{TokenType(EOF), "", 6},
	}

	l := New(input)

	for i, tt := range tests {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Errorf("tests[%d] - tokentype wrong. expected=%q, got=%q",
				i, tt.expectedType, tok.Type)
		}

		if tok.Literal != tt.expectedLiteral {
			t.Errorf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}

		if tok.Line != tt.expectedLine {
			t.Errorf("tests[%d] - line number wrong. expected=%d, got=%d",
				i, tt.expectedLine, tok.Line)
		}
	}
}
