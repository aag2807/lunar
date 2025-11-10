package parser

import (
	"fmt"
	"lunar/internal/ast"
	"lunar/internal/lexer"
	"strings"
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

func TestBooleanLiteral(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"true", "true"},
		{"false", "false"},
	}

	for _, tt := range tests {
		l := lexer.New(tt.input)
		p := New(l)
		exp := p.parseBooleanLiteral()

		if exp == nil {
			t.Errorf("parseBooleanLiteral() returned nil for input %q", tt.input)
			continue
		}

		if exp.String() != tt.expected {
			t.Errorf("expected=%q, got=%q", tt.expected, exp.String())
		}
	}
}

func TestNilLiteral(t *testing.T) {
	input := "nil"

	l := lexer.New(input)
	p := New(l)
	exp := p.parseNilLiteral()

	if exp == nil {
		t.Fatal("parseNilLiteral() returned nil")
	}

	if exp.String() != "nil" {
		t.Errorf("expected=%q, got=%q", "nil", exp.String())
	}
}

func TestTableLiteral(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"{}", "{}"},
		{"{1, 2, 3}", "{1, 2, 3}"},
		{"{x = 10, y = 20}", "{x = 10, y = 20}"},
	}

	for _, tt := range tests {
		l := lexer.New(tt.input)
		p := New(l)
		exp := p.parseTableLiteral()

		if exp == nil {
			t.Errorf("parseTableLiteral() returned nil for input %q. Errors: %v", tt.input, p.Errors())
			continue
		}

		result := exp.String()
		// For key-value pairs, the order might vary due to map iteration
		// So we'll just check it's not empty and contains the right structure
		if tt.input == "{x = 10, y = 20}" {
			if !strings.Contains(result, "x = 10") || !strings.Contains(result, "y = 20") {
				t.Errorf("expected result to contain key-value pairs, got=%q", result)
			}
		} else {
			if result != tt.expected {
				t.Errorf("expected=%q, got=%q", tt.expected, result)
			}
		}
	}
}

func TestIndexExpression(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"array[1]", "array[1]"},
		{"data[\"key\"]", "data[\"key\"]"},
		{"matrix[i][j]", "matrix[i][j]"},
	}

	for _, tt := range tests {
		l := lexer.New(tt.input)
		p := New(l)
		exp := p.parseExpression(LOWEST)

		if exp == nil {
			t.Errorf("parseExpression() returned nil for input %q. Errors: %v", tt.input, p.Errors())
			continue
		}

		if exp.String() != tt.expected {
			t.Errorf("input=%q: expected=%q, got=%q", tt.input, tt.expected, exp.String())
		}
	}
}

func TestLogicalOperators(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"true and false", "(true and false)"},
		{"true or false", "(true or false)"},
		{"x > 0 and x < 10", "((x > 0) and (x < 10))"},
		{"a or b and c", "(a or (b and c))"}, // and has higher precedence
	}

	for _, tt := range tests {
		l := lexer.New(tt.input)
		p := New(l)
		exp := p.parseExpression(LOWEST)

		if exp == nil {
			t.Errorf("parseExpression() returned nil for input %q. Errors: %v", tt.input, p.Errors())
			continue
		}

		if exp.String() != tt.expected {
			t.Errorf("input=%q: expected=%q, got=%q", tt.input, tt.expected, exp.String())
		}
	}
}

func TestModuloOperator(t *testing.T) {
	input := "10 % 3"

	l := lexer.New(input)
	p := New(l)
	exp := p.parseExpression(LOWEST)

	if exp == nil {
		t.Fatalf("parseExpression() returned nil. Errors: %v", p.Errors())
	}

	expected := "(10 % 3)"
	if exp.String() != expected {
		t.Errorf("expected=%q, got=%q", expected, exp.String())
	}
}

func TestComplexTypes(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Array types
		{"local numbers: number[]", "local numbers: number[]"},
		{"local users: User[]", "local users: User[]"},

		// Table types
		{"local cache: table<string, any>", "local cache: table<string, any>"},
		{"local map: table<number, User>", "local map: table<number, User>"},

		// Union types
		{"local status: string | number", "local status: string | number"},
		{"local data: User | nil", "local data: User | nil"},

		// Tuple types
		{"local coords: (number, number)", "local coords: (number, number)"},
		{"local point: (number, number, number)", "local point: (number, number, number)"},

		// Generic types
		{"local stack: Stack<number>", "local stack: Stack<number>"},
		{"local map: Map<string, User>", "local map: Map<string, User>"},

		// Optional types (existing feature, but testing with complex types)
		{"local users: User[]?", "local users: User[]?"},
		{"local cache: table<string, number>?", "local cache: table<string, number>?"},

		// Nested complex types
		{"local matrix: number[][]", "local matrix: number[][]"},
		{"local users: Array<User[]>", "local users: Array<User[]>"},
	}

	for _, tt := range tests {
		l := lexer.New(tt.input)
		p := New(l)
		stmt := p.parseVariableDeclaration()

		if stmt == nil {
			t.Errorf("parseVariableDeclaration() returned nil for input %q. Errors: %v", tt.input, p.Errors())
			continue
		}

		if stmt.String() != tt.expected {
			t.Errorf("input=%q: expected=%q, got=%q", tt.input, tt.expected, stmt.String())
		}
	}
}

func TestFunctionTypes(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Simple function types
		{"local callback: (x: number) => void", "local callback: (x: number) => void"},
		{"local mapper: (item: User) => string", "local mapper: (item: User) => string"},

		// Multiple parameters
		{"local add: (a: number, b: number) => number", "local add: (a: number, b: number) => number"},

		// Complex parameter/return types
		{"local transform: (users: User[]) => string[]", "local transform: (users: User[]) => string[]"},
	}

	for _, tt := range tests {
		l := lexer.New(tt.input)
		p := New(l)
		stmt := p.parseVariableDeclaration()

		if stmt == nil {
			t.Errorf("parseVariableDeclaration() returned nil for input %q. Errors: %v", tt.input, p.Errors())
			continue
		}

		if stmt.String() != tt.expected {
			t.Errorf("input=%q: expected=%q, got=%q", tt.input, tt.expected, stmt.String())
		}
	}
}

func TestFunctionWithComplexTypes(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			`function map(array: number[], fn: (x: number) => string): string[]
end`,
			`function map(array: number[], fn: (x: number) => string): string[]

end`,
		},
		{
			`function getUser(id: number): User?
end`,
			`function getUser(id: number): User?

end`,
		},
	}

	for _, tt := range tests {
		l := lexer.New(tt.input)
		p := New(l)
		stmt := p.parseFunctionDeclaration()

		if stmt == nil {
			t.Errorf("parseFunctionDeclaration() returned nil for input %q. Errors: %v", tt.input, p.Errors())
			continue
		}

		if stmt.String() != tt.expected {
			t.Errorf("input=%q: expected=%q, got=%q", tt.input, tt.expected, stmt.String())
		}
	}
}

func TestClassDeclaration(t *testing.T) {
	input := `class Car implements Vehicle
    private brand: string
    private year: number

    constructor(brand: string, year: number)
        self.brand = brand
    end

    public start(): void
        self.running = true
    end
end`

	l := lexer.New(input)
	p := New(l)
	stmt := p.parseClassDeclaration()

	if stmt == nil {
		t.Fatalf("parseClassDeclaration() returned nil. Errors: %v", p.Errors())
	}

	if stmt.Name.Value != "Car" {
		t.Errorf("class name wrong. expected=Car, got=%s", stmt.Name.Value)
	}

	if len(stmt.Properties) != 2 {
		t.Errorf("expected 2 properties, got=%d", len(stmt.Properties))
	}

	if len(stmt.Implements) != 1 {
		t.Errorf("expected 1 implement, got=%d", len(stmt.Implements))
	}

	if stmt.Constructor == nil {
		t.Error("constructor is nil")
	}

	if len(stmt.Methods) != 1 {
		t.Errorf("expected 1 method, got=%d", len(stmt.Methods))
	}
}

func TestInterfaceDeclaration(t *testing.T) {
	input := `interface Vehicle
    brand: string
    year: number
    start(): void
    stop(): void
end`

	l := lexer.New(input)
	p := New(l)
	stmt := p.parseInterfaceDeclaration()

	if stmt == nil {
		t.Fatalf("parseInterfaceDeclaration() returned nil. Errors: %v", p.Errors())
	}

	if stmt.Name.Value != "Vehicle" {
		t.Errorf("interface name wrong. expected=Vehicle, got=%s", stmt.Name.Value)
	}

	if len(stmt.Properties) != 2 {
		t.Errorf("expected 2 properties, got=%d", len(stmt.Properties))
	}

	if len(stmt.Methods) != 2 {
		t.Errorf("expected 2 methods, got=%d", len(stmt.Methods))
	}
}

func TestInterfaceWithExtends(t *testing.T) {
	input := `interface ElectricVehicle extends Vehicle
    batteryLevel: number
    charge(duration: number): void
end`

	l := lexer.New(input)
	p := New(l)
	stmt := p.parseInterfaceDeclaration()

	if stmt == nil {
		t.Fatalf("parseInterfaceDeclaration() returned nil. Errors: %v", p.Errors())
	}

	if stmt.Name.Value != "ElectricVehicle" {
		t.Errorf("interface name wrong. expected=ElectricVehicle, got=%s", stmt.Name.Value)
	}

	if len(stmt.Extends) != 1 {
		t.Errorf("expected 1 parent, got=%d", len(stmt.Extends))
	}
}

func TestEnumDeclaration(t *testing.T) {
	tests := []struct {
		input           string
		expectedName    string
		expectedMembers int
	}{
		{
			`enum Direction
    North
    South
    East
    West
end`,
			"Direction",
			4,
		},
		{
			`enum HttpStatus
    OK = 200
    NotFound = 404
    ServerError = 500
end`,
			"HttpStatus",
			3,
		},
	}

	for _, tt := range tests {
		l := lexer.New(tt.input)
		p := New(l)
		stmt := p.parseEnumDeclaration()

		if stmt == nil {
			t.Errorf("parseEnumDeclaration() returned nil for input %q. Errors: %v", tt.input, p.Errors())
			continue
		}

		if stmt.Name.Value != tt.expectedName {
			t.Errorf("enum name wrong. expected=%s, got=%s", tt.expectedName, stmt.Name.Value)
		}

		if len(stmt.Members) != tt.expectedMembers {
			t.Errorf("expected %d members, got=%d", tt.expectedMembers, len(stmt.Members))
		}
	}
}

func TestTypeDeclaration(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"type UserId = number", "type UserId = number"},
		{"type Email = string", "type Email = string"},
		{"type Status = string | number", "type Status = string | number"},
		{"type UserCallback = (user: User) => void", "type UserCallback = (user: User) => void"},
	}

	for _, tt := range tests {
		l := lexer.New(tt.input)
		p := New(l)
		stmt := p.parseTypeDeclaration()

		if stmt == nil {
			t.Errorf("parseTypeDeclaration() returned nil for input %q. Errors: %v", tt.input, p.Errors())
			continue
		}

		if stmt.String() != tt.expected {
			t.Errorf("input=%q: expected=%q, got=%q", tt.input, tt.expected, stmt.String())
		}
	}
}
