package types

import (
	"lunar/internal/lexer"
	"lunar/internal/parser"
	"testing"
)

func TestEnumInFunctions(t *testing.T) {
	input := `
enum Color
    Red = 1
    Green = 2
    Blue = 3
end

function setColor(c: Color): void
    local x: number = 1
end

function getColor(): Color
    return Color.Red
end

setColor(Color.Red)
local col: Color = getColor()
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

func TestEnumInClass(t *testing.T) {
	input := `
enum Status
    Active
    Inactive
    Pending
end

class Widget
    private status: Status

    constructor(s: Status)
        self.status = s
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

func TestEnumTypeChecking(t *testing.T) {
	input := `
enum Color
    Red = 1
    Green = 2
end

enum Status
    Active
    Inactive
end

function setColor(c: Color): void
    local x: number = 1
end

-- This should fail - wrong enum type
setColor(Status.Active)
`

	l := lexer.New(input)
	p := parser.New(l)
	statements := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	checker := NewChecker()
	errors := checker.Check(statements)

	if len(errors) != 1 {
		t.Errorf("Expected 1 type error (wrong enum type), got %d", len(errors))
		for _, err := range errors {
			t.Errorf("  %s", err.Message)
		}
	}
}

func TestEnumVariableDeclaration(t *testing.T) {
	input := `
enum Color
    Red = 1
    Green = 2
    Blue = 3
end

local c: Color = Color.Red
local d: Color = Color.Green

-- Should fail - number is not Color
local e: Color = 5
`

	l := lexer.New(input)
	p := parser.New(l)
	statements := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	checker := NewChecker()
	errors := checker.Check(statements)

	// Should have 1 error: assigning number to Color
	if len(errors) != 1 {
		t.Errorf("Expected 1 type error, got %d:", len(errors))
		for _, err := range errors {
			t.Errorf("  %s", err.Message)
		}
	}
}

func TestEnumAutoIncrement(t *testing.T) {
	input := `
enum Priority
    Low
    Medium
    High
end

function setPriority(p: Priority): void
    local x: number = 1
end

setPriority(Priority.Low)
setPriority(Priority.High)
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
