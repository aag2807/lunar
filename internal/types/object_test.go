package types

import (
	"lunar/internal/lexer"
	"lunar/internal/parser"
	"testing"
)

func TestObjectTypeDeclaration(t *testing.T) {
	input := `
type Point {
	x: number
	y: number
}
end

local p: Point = { x = 10, y = 20 }
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

func TestObjectTypeWithMissingProperty(t *testing.T) {
	input := `
type Point {
	x: number
	y: number
}
end

local p: Point = { x = 10 }
`

	l := lexer.New(input)
	p := parser.New(l)
	statements := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	checker := NewChecker()
	errors := checker.Check(statements)

	// Should have 1 error: missing property 'y'
	if len(errors) != 1 {
		t.Errorf("Expected 1 type error, got %d:", len(errors))
		for _, err := range errors {
			t.Errorf("  %s", err.Message)
		}
	}
}

func TestObjectTypeWithWrongPropertyType(t *testing.T) {
	input := `
type Point {
	x: number
	y: number
}
end

local p: Point = { x = "not a number", y = 20 }
`

	l := lexer.New(input)
	p := parser.New(l)
	statements := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	checker := NewChecker()
	errors := checker.Check(statements)

	// Should have 1 error: wrong type for 'x'
	if len(errors) != 1 {
		t.Errorf("Expected 1 type error, got %d:", len(errors))
		for _, err := range errors {
			t.Errorf("  %s", err.Message)
		}
	}
}

func TestObjectTypeWithExtraProperties(t *testing.T) {
	input := `
type Point {
	x: number
	y: number
}
end

local p: Point = { x = 10, y = 20, z = 30 }
`

	l := lexer.New(input)
	p := parser.New(l)
	statements := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	checker := NewChecker()
	errors := checker.Check(statements)

	// Extra properties are allowed (structural subtyping)
	if len(errors) > 0 {
		t.Errorf("Expected no type errors (extra properties allowed), got %d:", len(errors))
		for _, err := range errors {
			t.Errorf("  %s", err.Message)
		}
	}
}

func TestNestedObjectTypes(t *testing.T) {
	input := `
type Address {
	street: string
	city: string
}
end

type Person {
	name: string
	age: number
	address: Address
}
end

local addr: Address = { street = "123 Main St", city = "Springfield" }
local person: Person = { name = "John", age = 30, address = addr }
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

func TestObjectTypeInFunction(t *testing.T) {
	input := `
type Point {
	x: number
	y: number
}
end

function distance(p1: Point, p2: Point): number
	return 0
end

local p1: Point = { x = 0, y = 0 }
local p2: Point = { x = 3, y = 4 }
local d: number = distance(p1, p2)
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
