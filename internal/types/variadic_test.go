package types

import (
	"lunar/internal/lexer"
	"lunar/internal/parser"
	"testing"
)

func TestVariadicFunctionWithNoArgs(t *testing.T) {
	input := `
declare function print(...args: any): void end

print()
`

	l := lexer.New(input)
	p := parser.New(l)
	statements := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	checker := NewChecker()
	errors := checker.Check(statements)

	if len(errors) > 0 {
		t.Errorf("Expected no type errors, got %d:", len(errors))
		for _, err := range errors {
			t.Errorf("  %s", err.Message)
		}
	}
}

func TestVariadicFunctionWithMultipleArgs(t *testing.T) {
	input := `
declare function print(...args: any): void end

print("hello", "world", 123, true, nil)
`

	l := lexer.New(input)
	p := parser.New(l)
	statements := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	checker := NewChecker()
	errors := checker.Check(statements)

	if len(errors) > 0 {
		t.Errorf("Expected no type errors, got %d:", len(errors))
		for _, err := range errors {
			t.Errorf("  %s", err.Message)
		}
	}
}

func TestVariadicFunctionWithRequiredAndRestParams(t *testing.T) {
	input := `
declare function pcall(func: any, ...args: any): any end

pcall(print)
pcall(print, "hello")
pcall(print, "hello", "world", 123)
`

	l := lexer.New(input)
	p := parser.New(l)
	statements := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	checker := NewChecker()
	errors := checker.Check(statements)

	if len(errors) > 0 {
		t.Errorf("Expected no type errors, got %d:", len(errors))
		for _, err := range errors {
			t.Errorf("  %s", err.Message)
		}
	}
}

func TestVariadicFunctionMissingRequiredParams(t *testing.T) {
	input := `
declare function xpcall(func: any, errorHandler: any, ...args: any): any end

xpcall(print)
`

	l := lexer.New(input)
	p := parser.New(l)
	statements := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	checker := NewChecker()
	errors := checker.Check(statements)

	// Should have errors: missing required errorHandler argument
	if len(errors) == 0 {
		t.Error("Expected type errors for missing required argument, got none")
	}
}

func TestVariadicFunctionWithTwoRequiredParams(t *testing.T) {
	input := `
declare function xpcall(func: any, errorHandler: any, ...args: any): any end

xpcall(print, error)
xpcall(print, error, "hello", 123)
`

	l := lexer.New(input)
	p := parser.New(l)
	statements := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	checker := NewChecker()
	errors := checker.Check(statements)

	if len(errors) > 0 {
		t.Errorf("Expected no type errors, got %d:", len(errors))
		for _, err := range errors {
			t.Errorf("  %s", err.Message)
		}
	}
}

func TestVariadicSelectFunction(t *testing.T) {
	input := `
declare function select(index: any, ...args: any): any end

local x: any = select(1, "a", "b", "c")
local y: any = select("#", 1, 2, 3, 4, 5)
`

	l := lexer.New(input)
	p := parser.New(l)
	statements := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	checker := NewChecker()
	errors := checker.Check(statements)

	if len(errors) > 0 {
		t.Errorf("Expected no type errors, got %d:", len(errors))
		for _, err := range errors {
			t.Errorf("  %s", err.Message)
		}
	}
}
