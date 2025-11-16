package types

import (
	"lunar/internal/lexer"
	"lunar/internal/parser"
	"testing"
)

func TestTypeGuardDeclaration(t *testing.T) {
	input := `
function isString(value: any): value is string
	return type(value) == "string"
end

function isNumber(value: any): value is number
	return type(value) == "number"
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

	if len(errors) > 0 {
		t.Errorf("Expected no type errors, got %d:", len(errors))
		for _, err := range errors {
			t.Errorf("  %s", err.Message)
		}
	}
}

func TestTypeGuardUsage(t *testing.T) {
	input := `
function isString(value: any): value is string
	return type(value) == "string"
end

function process(value: any): void
	if isString(value) then
		local str: string = value
	end
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

	if len(errors) > 0 {
		t.Errorf("Expected no type errors, got %d:", len(errors))
		for _, err := range errors {
			t.Errorf("  %s", err.Message)
		}
	}
}

func TestTypeGuardWithUnionType(t *testing.T) {
	input := `
function isString(value: any): value is string
	return type(value) == "string"
end

function handle(value: string | number): void
	if isString(value) then
		local str: string = value
	else
		local num: number = value
	end
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

	if len(errors) > 0 {
		t.Errorf("Expected no type errors, got %d:", len(errors))
		for _, err := range errors {
			t.Errorf("  %s", err.Message)
		}
	}
}

func TestTypeGuardForCustomType(t *testing.T) {
	input := `
class Animal
	name: string
end

class Dog extends Animal
	breed: string
end

function isDog(value: Animal): value is Dog
	return value.breed ~= nil
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

	if len(errors) > 0 {
		t.Errorf("Expected no type errors, got %d:", len(errors))
		for _, err := range errors {
			t.Errorf("  %s", err.Message)
		}
	}
}

func TestTypeGuardBooleanReturn(t *testing.T) {
	input := `
function isString(value: any): value is string
	return type(value) == "string"
end

local result: boolean = isString("hello")
local result2: boolean = isString(42)
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

func TestMultipleTypeGuards(t *testing.T) {
	input := `
function isString(value: any): value is string
	return type(value) == "string"
end

function isNumber(value: any): value is number
	return type(value) == "number"
end

function isBoolean(value: any): value is boolean
	return type(value) == "boolean"
end

function process(value: any): void
	if isString(value) then
		local s: string = value
	elseif isNumber(value) then
		local n: number = value
	elseif isBoolean(value) then
		local b: boolean = value
	end
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

	if len(errors) > 0 {
		t.Errorf("Expected no type errors, got %d:", len(errors))
		for _, err := range errors {
			t.Errorf("  %s", err.Message)
		}
	}
}
