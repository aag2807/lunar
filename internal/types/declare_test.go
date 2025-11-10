package types

import (
	"lunar/internal/lexer"
	"lunar/internal/parser"
	"testing"
)

func TestDeclareConst(t *testing.T) {
	input := `
declare const PI: number

local radius: number = 5
local circumference: number = PI
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

func TestDeclareFunction(t *testing.T) {
	input := `
declare function print(message: string): void

print("Hello, world!")
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

func TestDeclareInterface(t *testing.T) {
	input := `
declare interface Graphics {
	clear: function(): void
	setColor: function(r: number, g: number, b: number): void
}
end

declare const graphics: Graphics

local g: Graphics = graphics
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

func TestDeclareType(t *testing.T) {
	input := `
declare type Vector2 {
	x: number
	y: number
}
end

declare function distance(v1: Vector2, v2: Vector2): number

local v1: Vector2 = { x = 0, y = 0 }
local v2: Vector2 = { x = 3, y = 4 }
local d: number = distance(v1, v2)
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

func TestDeclareWithGeneric(t *testing.T) {
	input := `
declare type Optional<T> = T | nil

declare function getValue(): Optional<string>

local result: Optional<string> = getValue()
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
