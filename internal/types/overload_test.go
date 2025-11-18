package types

import (
	"lunar/internal/lexer"
	"lunar/internal/parser"
	"testing"
)

func TestBasicFunctionOverloading(t *testing.T) {
	input := `
function add(a: number, b: number): number
	return a + b
end

function add(a: string, b: string): string
	return a .. b
end

local numResult: number = add(1, 2)
local strResult: string = add("hello", "world")
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

func TestOverloadWithDifferentArityNumber(t *testing.T) {
	input := `
function format(value: number): string
	return tostring(value)
end

function format(value: number, decimals: number): string
	return tostring(value)
end

local result1: string = format(42)
local result2: string = format(3.14, 2)
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

func TestOverloadNoMatchError(t *testing.T) {
	input := `
function process(x: number): number
	return x
end

function process(x: string): string
	return x
end

local result = process(true)
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
		t.Error("Expected type error for no matching overload, got none")
	}

	// Should have error about no overload matching
	foundError := false
	for _, err := range errors {
		if err.Message == "No overload matches this call with 1 arguments" ||
			err.Message == "Argument 1: cannot pass type 'boolean' to parameter of type 'number'" {
			foundError = true
			break
		}
	}

	if !foundError {
		t.Errorf("Expected 'no overload matches' error, got: %v", errors)
	}
}

func TestOverloadWithReturnTypeSelection(t *testing.T) {
	input := `
function getValue(key: number): number
	return key
end

function getValue(key: string): string
	return key
end

local numVal: number = getValue(42)
local strVal: string = getValue("test")
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

func TestOverloadWrongReturnType(t *testing.T) {
	input := `
function transform(x: number): number
	return x
end

function transform(x: string): string
	return x
end

-- This should error: number overload returns number, not string
local result: string = transform(42)
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
		t.Error("Expected type error for wrong return type assignment, got none")
	}
}

func TestOverloadWithThreeSignatures(t *testing.T) {
	input := `
function parse(value: number): number
	return value
end

function parse(value: string): string
	return value
end

function parse(value: boolean): boolean
	return value
end

local num: number = parse(42)
local str: string = parse("hello")
local bool: boolean = parse(true)
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

func TestOverloadWithOptionalParameter(t *testing.T) {
	input := `
function greet(name: string): string
	return "Hello, " .. name
end

function greet(name: string, formal: boolean): string
	return "Greetings, " .. name
end

local casual: string = greet("World")
local formal: string = greet("World", true)
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

func TestOverloadSpecificitySelection(t *testing.T) {
	input := `
function handle(x: any): any
	return x
end

function handle(x: number): number
	return x
end

-- Should select the number overload (more specific)
local result: number = handle(42)
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

func TestOverloadWithUnionTypes(t *testing.T) {
	input := `
function stringify(value: number | string): string
	return tostring(value)
end

function stringify(value: boolean): string
	return tostring(value)
end

local num: string = stringify(42)
local str: string = stringify("hello")
local bool: string = stringify(true)
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

func TestDeclaredFunctionOverloading(t *testing.T) {
	input := `
declare function print(value: string): void end
declare function print(value: number): void end
declare function print(value: boolean): void end

print("hello")
print(42)
print(true)
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

func TestOverloadArgumentCountError(t *testing.T) {
	input := `
function calc(a: number): number
	return a
end

function calc(a: number, b: number): number
	return a + b
end

-- Should error: no overload takes 3 arguments
local result = calc(1, 2, 3)
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
		t.Error("Expected type error for wrong argument count, got none")
	}
}
