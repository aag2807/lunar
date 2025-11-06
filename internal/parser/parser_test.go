package parser

import (
	"fmt"
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

func TestPrefixExpressions(t *testing.T) {
	prefixTests := []struct {
		input    string
		operator string
		value    interface{}
	}{
		{"-15", "-", 15},
		{"!true", "!", true},
		{"not value", "not", "value"},
	}

	for _, tt := range prefixTests {
		l := lexer.New(tt.input)
		p := New(l)
		exp := p.parseExpression(LOWEST)

		if exp == nil {
			t.Errorf("Parser errors: %v", p.Errors())
			continue
		}

		prefix, ok := exp.(*ast.PrefixExpression)
		if !ok {
			t.Fatalf("exp not *ast.PrefixExpression. got=%T", exp)
		}
		if prefix.Operator != tt.operator {
			t.Fatalf("exp.Operator is not '%s'. got=%s",
				tt.operator, prefix.Operator)
		}
	}
}

func TestGroupedExpressions(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			"(1 + 2) * 3",
			"((1 + 2) * 3)",
		},
		{
			"-(5 + 3)",
			"(-(5 + 3))",
		},
		{
			"(a + b) * (c + d)",
			"((a + b) * (c + d))",
		},
	}

	for _, tt := range tests {
		l := lexer.New(tt.input)
		p := New(l)
		exp := p.parseExpression(LOWEST)

		if exp == nil {
			t.Errorf("Parser errors: %v", p.Errors())
			continue
		}

		actual := exp.String()
		if actual != tt.expected {
			t.Errorf("expected=%q, got=%q", tt.expected, actual)
		}
	}
}

func TestCallExpressions(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			"add(1, 2)",
			"add(1, 2)",
		},
		{
			"print(\"hello\")",
			"print(\"hello\")",
		},
		{
			"math.max(1, 2, 3)",
			"math.max(1, 2, 3)",
		},
		{
			"myFunc()",
			"myFunc()",
		},
		{
			"callFunction(1 + 2, 3 * 4)",
			"callFunction((1 + 2), (3 * 4))",
		},
	}

	for _, tt := range tests {
		l := lexer.New(tt.input)
		p := New(l)
		exp := p.parseExpression(LOWEST)

		if exp == nil {
			t.Errorf("Parser errors: %v", p.Errors())
			continue
		}

		actual := exp.String()
		if actual != tt.expected {
			t.Errorf("input=%q: expected=%q, got=%q",
				tt.input, tt.expected, actual)
		}
	}
}

func TestDotExpressionCalls(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			"math.max(1, 2, 3)",
			"math.max(1, 2, 3)",
		},
	}

	for _, tt := range tests {
		l := lexer.New(tt.input)
		p := New(l)

		// Add debug output
		fmt.Printf("Current token: %+v\n", p.curToken)
		fmt.Printf("Peek token: %+v\n", p.peekToken)

		exp := p.parseExpression(LOWEST)

		if exp == nil {
			t.Errorf("Parser errors: %v", p.Errors())
			continue
		}

		actual := exp.String()
		fmt.Printf("Got expression: %T, String(): %s\n", exp, actual)

		if actual != tt.expected {
			t.Errorf("input=%q: expected=%q, got=%q",
				tt.input, tt.expected, actual)
		}
	}
}

func TestVariableDeclaration(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			"local x: number = 5",
			"local x: number = 5",
		},
		{
			"const MAX_SIZE: number = 100",
			"const MAX_SIZE: number = 100",
		},
		{
			"local name: string = \"luna\"",
			"local name: string = \"luna\"",
		},
		{
			"local data: string?",
			"local data: string?",
		},
	}

	for _, tt := range tests {
		l := lexer.New(tt.input)
		p := New(l)
		stmt := p.parseVariableDeclaration()

		if stmt == nil {
			t.Errorf("parseVariableDeclaration() returned nil. Parser errors: %v", p.Errors())
			continue
		}

		if stmt.String() != tt.expected {
			t.Errorf("expected=%q, got=%q", tt.expected, stmt.String())
		}
	}
}

func TestFunctionDeclaration(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			`function add(x: number, y: number): number
    return x + y
end`,
			`function add(x: number, y: number): number
    return (x + y)
end`,
		},
		{
			`function greet(name: string)
    return "Hello, " .. name
end`,
			`function greet(name: string)
    return ("Hello, " .. name)
end`,
		},
	}

	for _, tt := range tests {
		l := lexer.New(tt.input)
		p := New(l)
		stmt := p.parseFunctionDeclaration()

		if stmt == nil {
			t.Errorf("parseFunctionDeclaration() returned nil. Parser errors: %v", p.Errors())
			continue
		}

		if stmt.String() != tt.expected {
			t.Errorf("expected=%q, got=%q", tt.expected, stmt.String())
		}
	}
}

func TestIfStatement(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			`if x > 0 then
    return x
end`,
			`if (x > 0) then
    return x
end`,
		},
		{
			`if x > 0 then
    return x
else
    return 0
end`,
			`if (x > 0) then
    return x
else
    return 0
end`,
		},
	}

	for _, tt := range tests {
		l := lexer.New(tt.input)
		p := New(l)
		stmt := p.parseIfStatement()

		if stmt == nil {
			t.Errorf("parseIfStatement() returned nil. Parser errors: %v", p.Errors())
			continue
		}

		if stmt.String() != tt.expected {
			t.Errorf("expected=%q, got=%q", tt.expected, stmt.String())
		}
	}
}

func TestWhileStatement(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			`while x > 0 do
    x = x - 1
end`,
			`while (x > 0) do
    x = (x - 1)
end`,
		},
	}

	for _, tt := range tests {
		l := lexer.New(tt.input)
		p := New(l)
		stmt := p.parseWhileStatement()

		if stmt == nil {
			t.Errorf("parseWhileStatement() returned nil. Parser errors: %v", p.Errors())
			continue
		}

		if stmt.String() != tt.expected {
			t.Errorf("expected=%q, got=%q", tt.expected, stmt.String())
		}
	}
}

func TestForStatement(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			`for i = 1, 10 do
    print(i)
end`,
			`for i = 1, 10 do
    print(i)
end`,
		},
		{
			`for i = 1, 10, 2 do
    print(i)
end`,
			`for i = 1, 10, 2 do
    print(i)
end`,
		},
		{
			`for item in items do
    print(item)
end`,
			`for item in items do
    print(item)
end`,
		},
	}

	for _, tt := range tests {
		l := lexer.New(tt.input)
		p := New(l)
		stmt := p.parseForStatement()

		if stmt == nil {
			t.Errorf("parseForStatement() returned nil. Parser errors: %v", p.Errors())
			continue
		}

		if stmt.String() != tt.expected {
			t.Errorf("expected=%q, got=%q", tt.expected, stmt.String())
		}
	}
}

func TestDoStatement(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			`do
    local x = 5
end`,
			`do
    local x = 5
end`,
		},
	}

	for _, tt := range tests {
		l := lexer.New(tt.input)
		p := New(l)
		stmt := p.parseDoStatement()

		if stmt == nil {
			t.Errorf("parseDoStatement() returned nil. Parser errors: %v", p.Errors())
			continue
		}

		if stmt.String() != tt.expected {
			t.Errorf("expected=%q, got=%q", tt.expected, stmt.String())
		}
	}
}

func TestBreakStatement(t *testing.T) {
	input := "break"

	l := lexer.New(input)
	p := New(l)
	stmt := p.parseBreakStatement()

	if stmt == nil {
		t.Fatal("parseBreakStatement() returned nil")
	}

	if stmt.String() != "break" {
		t.Errorf("expected=%q, got=%q", "break", stmt.String())
	}
}
