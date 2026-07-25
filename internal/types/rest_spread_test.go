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

// A rest parameter may be annotated with the element type it collects or with
// the array itself. Both spellings mean the same thing, and the stdlib
// declarations are written in the element form (`...: any`).
func TestRestParameterAcceptsElementOrArrayType(t *testing.T) {
	for _, input := range []string{
		"function f(...args: number): void\nend",
		"function f(...args: number[]): void\nend",
	} {
		l := lexer.New(input)
		p := parser.New(l)
		statements := p.Parse()

		if len(p.Errors()) > 0 {
			t.Fatalf("Parser errors: %v", p.Errors())
		}

		checker := NewChecker()
		for _, err := range checker.Check(statements) {
			t.Errorf("unexpected type error for %q: %s", input, err.Message)
		}

		typ, ok := checker.GetEnv().Get("f")
		if !ok {
			t.Fatalf("f not registered for %q", input)
		}

		fnType := typ.(*FunctionType)
		if !fnType.HasRestParameter {
			t.Errorf("expected a rest parameter for %q", input)
		}

		if _, isArray := fnType.Parameters[0].(*ArrayType); !isArray {
			t.Errorf("expected the rest parameter to collect an array for %q, got %s",
				input, fnType.Parameters[0].String())
		}
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
