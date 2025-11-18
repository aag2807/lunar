package types

import (
	"lunar/internal/lexer"
	"lunar/internal/parser"
	"testing"
)

// Test that type checker reports multiple errors instead of stopping at first error
func TestErrorRecoveryMultipleErrors(t *testing.T) {
	input := `
local x: number = "string"  -- Type mismatch error
local y: string = 42         -- Type mismatch error
local z: boolean = nil       -- Type mismatch error
`

	l := lexer.New(input)
	p := parser.New(l)
	statements := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	checker := NewChecker()
	errors := checker.Check(statements)

	// Should report all 3 errors, not just the first one
	if len(errors) < 3 {
		t.Errorf("Expected at least 3 errors (one for each mismatch), got %d", len(errors))
		for _, err := range errors {
			t.Logf("  Error: %s (line %d)", err.Message, err.Line)
		}
	}
}

// Test error recovery in function calls
func TestErrorRecoveryFunctionCalls(t *testing.T) {
	input := `
function takes_number(x: number): void
end

function takes_string(x: string): void
end

takes_number("wrong")  -- Error: string not assignable to number
takes_string(123)       -- Error: number not assignable to string
takes_number(true)      -- Error: boolean not assignable to number
`

	l := lexer.New(input)
	p := parser.New(l)
	statements := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	checker := NewChecker()
	errors := checker.Check(statements)

	// Should report all 3 call errors
	if len(errors) < 3 {
		t.Errorf("Expected at least 3 errors, got %d", len(errors))
		for _, err := range errors {
			t.Logf("  Error: %s (line %d)", err.Message, err.Line)
		}
	}
}

// Test error recovery in class declarations
func TestErrorRecoveryClassErrors(t *testing.T) {
	input := `
class MyClass
	x: number
	y: string

	function method1(): number
		return "wrong"  -- Error: string not assignable to number
	end

	function method2(): string
		return 42  -- Error: number not assignable to string
	end

	function method3(): boolean
		return nil  -- Error: nil not assignable to boolean
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

	// Should report all 3 return type errors
	if len(errors) < 3 {
		t.Errorf("Expected at least 3 errors, got %d", len(errors))
		for _, err := range errors {
			t.Logf("  Error: %s (line %d)", err.Message, err.Line)
		}
	}
}

// Test error recovery with undefined variables
func TestErrorRecoveryUndefinedVariables(t *testing.T) {
	input := `
local a = undefinedVar1  -- Error: undefined
local b = undefinedVar2  -- Error: undefined
local c = undefinedVar3  -- Error: undefined
`

	l := lexer.New(input)
	p := parser.New(l)
	statements := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	checker := NewChecker()
	errors := checker.Check(statements)

	// Should report all 3 undefined errors
	if len(errors) < 3 {
		t.Errorf("Expected at least 3 errors, got %d", len(errors))
		for _, err := range errors {
			t.Logf("  Error: %s (line %d)", err.Message, err.Line)
		}
	}
}

// Test error recovery in complex nested structures
func TestErrorRecoveryNestedStructures(t *testing.T) {
	input := `
interface Person
	name: string
	age: number
end

class Worker
	salary: number

	function getSalary(): number
		return "not a number"  -- Error: wrong return type
	end

	function processAge(age: number): void
		local bad: string = age  -- Error: number not assignable to string
	end

	function formatName(name: string): number
		return name  -- Error: string not assignable to number
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

	// Should report multiple errors across different structures
	if len(errors) < 3 {
		t.Errorf("Expected at least 3 errors, got %d", len(errors))
		for _, err := range errors {
			t.Logf("  Error: %s (line %d)", err.Message, err.Line)
		}
	}
}

// Test that error recovery allows later correct code to still be checked
func TestErrorRecoveryMixedCorrectAndIncorrect(t *testing.T) {
	input := `
local bad: number = "wrong"  -- Error

-- This should still be checked and NOT error
local good: number = 42

function myFunc(): string
	return "correct"  -- No error
end

local bad2: string = 123  -- Error

-- This should still work
local result: string = myFunc()
`

	l := lexer.New(input)
	p := parser.New(l)
	statements := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	checker := NewChecker()
	errors := checker.Check(statements)

	// Should report exactly 2 errors (bad and bad2), not more
	if len(errors) != 2 {
		t.Errorf("Expected exactly 2 errors, got %d", len(errors))
		for _, err := range errors {
			t.Logf("  Error: %s (line %d)", err.Message, err.Line)
		}
	}
}
