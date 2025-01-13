package parser

import (
	"lunar/internal/ast"
	"lunar/internal/lexer"
	"testing"
)

func TestIdentifierExpression(t *testing.T) {
	input := "foobar;"

	l := lexer.New(input)
	p := New(l)

	identifier := p.parseIdentifier()
	if identifier == nil {
		t.Fatal("parseIdentifier returned nil")
	}

	ident, ok := identifier.(*ast.Identifier)
	if !ok {
		t.Fatalf("identifier is not *ast.Identifier. got=%T", identifier)
	}

	if ident.Value != "foobar" {
		t.Fatalf("identifier is not *ast.Identifier. got=%T", identifier)
	}

	if ident.TokenLiteral() != "foobar" {
		t.Fatalf("ident.TokenLiteral not %s. got=%s", "foobar", ident.TokenLiteral())
	}
}

func TestNumberLiteralExpression(t *testing.T) {
	input := "5;"

	l := lexer.New(input)
	p := New(l)

	literal := p.parseNumberLiteral()
	if literal == nil {
		t.Fatal("parseNumberLiteral() returned nil")
	}

	number, ok := literal.(*ast.NumberLiteral)
	if !ok {
		t.Fatalf("literal not *ast.NumberLiteral. got=%T", literal)
	}
	if number.Value != 5.0 {
		t.Errorf("literal.Value not %f. got=%f", 5.0, number.Value)
	}
	if number.TokenLiteral() != "5" {
		t.Errorf("literal.TokenLiteral not %s. got=%s", "5",
			number.TokenLiteral())
	}
}

func TestStringLiteralExpression(t *testing.T) {
	input := `"hello world";`

	l := lexer.New(input)
	p := New(l)

	literal := p.parseStringLiteral()
	if literal == nil {
		t.Fatal("parseStringLiteral() returned nil")
	}

	str, ok := literal.(*ast.StringLiteral)
	if !ok {
		t.Fatalf("literal not *ast.StringLiteral. got=%T", literal)
	}
	if str.Value != "hello world" {
		t.Errorf("literal.Value not %s. got=%s", "hello world", str.Value)
	}
}

func TestOperatorPrecedence(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			"1 + 2",
			"(1 + 2)",
		},
		{
			"1 + 2 * 3",
			"(1 + (2 * 3))",
		},
		{
			"1 + 2 + 3",
			"((1 + 2) + 3)",
		},
		{
			"a * b + c",
			"((a * b) + c)",
		},
		{
			"a + b * c",
			"(a + (b * c))",
		},
		{
			"5 > 4 == 3 < 4",
			"((5 > 4) == (3 < 4))",
		},
	}

	for i, tt := range tests {
		l := lexer.New(tt.input)
		p := New(l)
		expression := p.parseExpression(LOWEST)

		if expression == nil {
			t.Fatalf("tests[%d] - parseExpression() returned nil", i)
		}

		actual := expression.String()
		if actual != tt.expected {
			t.Errorf("tests[%d] - expected=%q, got=%q", i, tt.expected, actual)
		}
	}
}
