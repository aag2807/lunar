package codegen

import (
	"lunar/internal/ast"
	"lunar/internal/lexer"
	"strings"
	"testing"
)

func TestGenerateVariableDeclaration(t *testing.T) {
	// local x = 5
	stmt := &ast.VariableDeclaration{
		Token: lexer.Token{Type: lexer.LOCAL, Literal: "local"},
		Name:  &ast.Identifier{Value: "x"},
		Value: &ast.NumberLiteral{
			Token: lexer.Token{Literal: "5"},
			Value: 5,
		},
	}

	g := New()
	result := g.generateStatement(stmt)
	expected := "local x = 5\n"

	if result != expected {
		t.Errorf("Expected:\n%s\nGot:\n%s", expected, result)
	}
}

func TestGenerateNumberExpression(t *testing.T) {
	expr := &ast.NumberLiteral{
		Token: lexer.Token{Literal: "42"},
		Value: 42,
	}

	g := New()
	result := g.generateExpression(expr)
	expected := "42"

	if result != expected {
		t.Errorf("Expected: %s, Got: %s", expected, result)
	}
}

func TestGenerateStringExpression(t *testing.T) {
	expr := &ast.StringLiteral{
		Token: lexer.Token{Literal: "hello"},
		Value: "hello",
	}

	g := New()
	result := g.generateExpression(expr)
	expected := "\"hello\""

	if result != expected {
		t.Errorf("Expected: %s, Got: %s", expected, result)
	}
}

func TestGenerateBooleanExpression(t *testing.T) {
	tests := []struct {
		value    bool
		expected string
	}{
		{true, "true"},
		{false, "false"},
	}

	for _, tt := range tests {
		expr := &ast.BooleanLiteral{
			Value: tt.value,
		}

		g := New()
		result := g.generateExpression(expr)

		if result != tt.expected {
			t.Errorf("Expected: %s, Got: %s", tt.expected, result)
		}
	}
}

func TestGenerateInfixExpression(t *testing.T) {
	// 2 + 3
	expr := &ast.InfixExpression{
		Left: &ast.NumberLiteral{
			Token: lexer.Token{Literal: "2"},
			Value: 2,
		},
		Operator: "+",
		Right: &ast.NumberLiteral{
			Token: lexer.Token{Literal: "3"},
			Value: 3,
		},
	}

	g := New()
	result := g.generateExpression(expr)
	expected := "(2 + 3)"

	if result != expected {
		t.Errorf("Expected: %s, Got: %s", expected, result)
	}
}

func TestGenerateCallExpression(t *testing.T) {
	// print("hello")
	expr := &ast.CallExpression{
		Function: &ast.Identifier{Value: "print"},
		Arguments: []ast.Expression{
			&ast.StringLiteral{Value: "hello"},
		},
	}

	g := New()
	result := g.generateExpression(expr)
	expected := "print(\"hello\")"

	if result != expected {
		t.Errorf("Expected: %s, Got: %s", expected, result)
	}
}

func TestGenerateReturnStatement(t *testing.T) {
	// return 42
	stmt := &ast.ReturnStatement{
		Token: lexer.Token{Type: lexer.RETURN, Literal: "return"},
		ReturnValue: &ast.NumberLiteral{
			Token: lexer.Token{Literal: "42"},
			Value: 42,
		},
	}

	g := New()
	result := g.generateStatement(stmt)
	expected := "return 42\n"

	if result != expected {
		t.Errorf("Expected:\n%s\nGot:\n%s", expected, result)
	}
}

func TestGenerateIfStatement(t *testing.T) {
	// if true then return 1 end
	stmt := &ast.IfStatement{
		Token: lexer.Token{Type: lexer.IF, Literal: "if"},
		Condition: &ast.BooleanLiteral{Value: true},
		Consequence: &ast.BlockStatement{
			Statements: []ast.Statement{
				&ast.ReturnStatement{
					Token: lexer.Token{Type: lexer.RETURN, Literal: "return"},
					ReturnValue: &ast.NumberLiteral{
						Token: lexer.Token{Literal: "1"},
						Value: 1,
					},
				},
			},
		},
	}

	g := New()
	result := g.generateStatement(stmt)
	expected := "if true then\n    return 1\nend\n"

	if result != expected {
		t.Errorf("Expected:\n%s\nGot:\n%s", expected, result)
	}
}

func TestGenerateWhileStatement(t *testing.T) {
	// while true do break end
	stmt := &ast.WhileStatement{
		Token: lexer.Token{Type: lexer.WHILE, Literal: "while"},
		Condition: &ast.BooleanLiteral{Value: true},
		Body: &ast.BlockStatement{
			Statements: []ast.Statement{
				&ast.BreakStatement{
					Token: lexer.Token{Type: lexer.BREAK, Literal: "break"},
				},
			},
		},
	}

	g := New()
	result := g.generateStatement(stmt)
	expected := "while true do\n    break\nend\n"

	if result != expected {
		t.Errorf("Expected:\n%s\nGot:\n%s", expected, result)
	}
}

func TestGenerateForStatement(t *testing.T) {
	// for i = 1, 10 do break end
	stmt := &ast.ForStatement{
		Token:     lexer.Token{Type: lexer.FOR, Literal: "for"},
		Variable:  &ast.Identifier{Value: "i"},
		Start:     &ast.NumberLiteral{Token: lexer.Token{Literal: "1"}, Value: 1},
		End:       &ast.NumberLiteral{Token: lexer.Token{Literal: "10"}, Value: 10},
		IsGeneric: false,
		Body: &ast.BlockStatement{
			Statements: []ast.Statement{
				&ast.BreakStatement{
					Token: lexer.Token{Type: lexer.BREAK, Literal: "break"},
				},
			},
		},
	}

	g := New()
	result := g.generateStatement(stmt)
	expected := "for i = 1, 10 do\n    break\nend\n"

	if result != expected {
		t.Errorf("Expected:\n%s\nGot:\n%s", expected, result)
	}
}

func TestGenerateClass(t *testing.T) {
	// Simple class with constructor
	stmt := &ast.ClassDeclaration{
		Token: lexer.Token{Type: lexer.CLASS, Literal: "class"},
		Name:  &ast.Identifier{Value: "Car"},
		Constructor: &ast.ConstructorDeclaration{
			Token: lexer.Token{Type: lexer.CONSTRUCTOR, Literal: "constructor"},
			Parameters: []*ast.Parameter{
				{Name: &ast.Identifier{Value: "brand"}},
			},
			Body: &ast.BlockStatement{
				Statements: []ast.Statement{
					&ast.AssignmentStatement{
						Token: lexer.Token{Type: lexer.ASSIGN, Literal: "="},
						Name: &ast.DotExpression{
							Left:  &ast.Identifier{Value: "self"},
							Right: &ast.Identifier{Value: "brand"},
						},
						Value: &ast.Identifier{Value: "brand"},
					},
				},
			},
		},
		Methods: []*ast.FunctionDeclaration{},
	}

	g := New()
	result := g.generateStatement(stmt)

	// Check expected parts
	expectedParts := []string{
		"local Car = {}",
		"Car.__index = Car",
		"function Car.new(brand)",
		"local self = setmetatable({}, Car)",
		"self.brand = brand",
		"return self",
	}

	for _, part := range expectedParts {
		if !strings.Contains(result, part) {
			t.Errorf("Expected output to contain:\n%s\nGot:\n%s", part, result)
		}
	}
}

func TestGenerateEnum(t *testing.T) {
	// enum Color { Red = 1, Green = 2 }
	stmt := &ast.EnumDeclaration{
		Token: lexer.Token{Type: lexer.ENUM, Literal: "enum"},
		Name:  &ast.Identifier{Value: "Color"},
		Members: []*ast.EnumMember{
			{
				Name:  &ast.Identifier{Value: "Red"},
				Value: &ast.NumberLiteral{Token: lexer.Token{Literal: "1"}, Value: 1},
			},
			{
				Name:  &ast.Identifier{Value: "Green"},
				Value: &ast.NumberLiteral{Token: lexer.Token{Literal: "2"}, Value: 2},
			},
		},
	}

	g := New()
	result := g.generateStatement(stmt)

	expectedParts := []string{
		"local Color = {",
		"Red = 1,",
		"Green = 2,",
		"}",
	}

	for _, part := range expectedParts {
		if !strings.Contains(result, part) {
			t.Errorf("Expected output to contain:\n%s\nGot:\n%s", part, result)
		}
	}
}

func TestGenerateEnumAutoIncrement(t *testing.T) {
	// enum Status { Pending, Active }
	stmt := &ast.EnumDeclaration{
		Token: lexer.Token{Type: lexer.ENUM, Literal: "enum"},
		Name:  &ast.Identifier{Value: "Status"},
		Members: []*ast.EnumMember{
			{Name: &ast.Identifier{Value: "Pending"}, Value: nil},
			{Name: &ast.Identifier{Value: "Active"}, Value: nil},
		},
	}

	g := New()
	result := g.generateStatement(stmt)

	expectedParts := []string{
		"local Status = {",
		"Pending = 0,",
		"Active = 1,",
		"}",
	}

	for _, part := range expectedParts {
		if !strings.Contains(result, part) {
			t.Errorf("Expected output to contain:\n%s\nGot:\n%s", part, result)
		}
	}
}

func TestGenerateDotExpression(t *testing.T) {
	// math.max
	expr := &ast.DotExpression{
		Left:  &ast.Identifier{Value: "math"},
		Right: &ast.Identifier{Value: "max"},
	}

	g := New()
	result := g.generateExpression(expr)
	expected := "math.max"

	if result != expected {
		t.Errorf("Expected: %s, Got: %s", expected, result)
	}
}

func TestGenerateIndexExpression(t *testing.T) {
	// arr[1]
	expr := &ast.IndexExpression{
		Left:  &ast.Identifier{Value: "arr"},
		Index: &ast.NumberLiteral{Token: lexer.Token{Literal: "1"}, Value: 1},
	}

	g := New()
	result := g.generateExpression(expr)
	expected := "arr[1]"

	if result != expected {
		t.Errorf("Expected: %s, Got: %s", expected, result)
	}
}

func TestInterfaceGeneratesNoCode(t *testing.T) {
	stmt := &ast.InterfaceDeclaration{
		Token: lexer.Token{Type: lexer.INTERFACE, Literal: "interface"},
		Name:  &ast.Identifier{Value: "Vehicle"},
	}

	g := New()
	result := g.generateStatement(stmt)

	if result != "" {
		t.Errorf("Expected empty string for interface, got: %s", result)
	}
}

func TestTypeDeclarationGeneratesNoCode(t *testing.T) {
	stmt := &ast.TypeDeclaration{
		Token: lexer.Token{Type: lexer.TYPE, Literal: "type"},
		Name:  &ast.Identifier{Value: "UserID"},
		Type:  &ast.Identifier{Value: "string"},
	}

	g := New()
	result := g.generateStatement(stmt)

	if result != "" {
		t.Errorf("Expected empty string for type declaration, got: %s", result)
	}
}

func TestGenerateMultipleStatements(t *testing.T) {
	statements := []ast.Statement{
		&ast.VariableDeclaration{
			Token: lexer.Token{Type: lexer.LOCAL, Literal: "local"},
			Name:  &ast.Identifier{Value: "x"},
			Value: &ast.NumberLiteral{Token: lexer.Token{Literal: "10"}, Value: 10},
		},
		&ast.VariableDeclaration{
			Token: lexer.Token{Type: lexer.LOCAL, Literal: "local"},
			Name:  &ast.Identifier{Value: "y"},
			Value: &ast.NumberLiteral{Token: lexer.Token{Literal: "20"}, Value: 20},
		},
	}

	g := New()
	result := g.Generate(statements)

	expectedParts := []string{
		"local x = 10",
		"local y = 20",
	}

	for _, part := range expectedParts {
		if !strings.Contains(result, part) {
			t.Errorf("Expected output to contain:\n%s\nGot:\n%s", part, result)
		}
	}
}
