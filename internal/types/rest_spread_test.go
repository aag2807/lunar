package types

import (
	"lunar/internal/lexer"
	"lunar/internal/parser"
	"testing"
)

func TestRestParameter(t *testing.T) {
	input := `
function sum(...numbers: number[]): number
	local total: number = 0
	for _, n in ipairs(numbers) do
		total = total + n
	end
	return total
end

local result: number = sum(1, 2, 3, 4, 5)
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

func TestRestParameterMustBeLast(t *testing.T) {
	input := `
function bad(...args: number[], x: number): void
end
`

	l := lexer.New(input)
	p := parser.New(l)
	statements := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	checker := NewChecker()
	errors := checker.Check(statements)

	if len(errors) == 0 {
		t.Error("Expected error for rest parameter not being last")
	}

	foundError := false
	for _, err := range errors {
		if err.Message == "Rest parameter must be the last parameter" {
			foundError = true
			break
		}
	}

	if !foundError {
		t.Error("Expected 'Rest parameter must be the last parameter' error")
	}
}

func TestRestParameterMustBeArray(t *testing.T) {
	input := `
function bad(...args: number): void
end
`

	l := lexer.New(input)
	p := parser.New(l)
	statements := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	checker := NewChecker()
	errors := checker.Check(statements)

	if len(errors) == 0 {
		t.Error("Expected error for rest parameter not being array type")
	}
}

func TestSpreadInFunctionCall(t *testing.T) {
	input := `
function add(a: number, b: number, c: number): number
	return a + b + c
end

local nums: number[] = {1, 2, 3}
local result: number = add(...nums)
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

func TestSpreadInArray(t *testing.T) {
	input := `
local arr1: number[] = {1, 2, 3}
local arr2: number[] = {4, 5, 6}
local combined: number[] = {...arr1, ...arr2}
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

func TestSpreadInArrayWithMixedElements(t *testing.T) {
	input := `
local arr: number[] = {2, 3}
local result: number[] = {1, ...arr, 4, 5}
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

func TestRestParameterWithMixedParams(t *testing.T) {
	input := `
function greet(prefix: string, ...names: string[]): string
	local result: string = prefix
	for _, name in ipairs(names) do
		result = result .. " " .. name
	end
	return result
end

local message: string = greet("Hello", "Alice", "Bob", "Charlie")
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

func TestSpreadNonArrayError(t *testing.T) {
	input := `
local x: number = 42
local arr: number[] = {...x}
`

	l := lexer.New(input)
	p := parser.New(l)
	statements := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	checker := NewChecker()
	errors := checker.Check(statements)

	if len(errors) == 0 {
		t.Error("Expected error for spreading non-array type")
	}
}
