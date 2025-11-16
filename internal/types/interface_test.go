package types

import (
	"lunar/internal/lexer"
	"lunar/internal/parser"
	"testing"
)

func TestBasicInterface(t *testing.T) {
	input := `
interface Person
	name: string
	age: number
end

local p: Person = { name = "John", age = 30 }
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

func TestInterfaceWithMethods(t *testing.T) {
	input := `
interface Vehicle
	brand: string
	start(): void
	stop(): void
end

local v: Vehicle = {
	brand = "Toyota",
	start = function() end,
	stop = function() end
}
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

func TestClassImplementsInterface(t *testing.T) {
	input := `
interface Drawable
	draw(): void
	getArea(): number
end

class Circle implements Drawable
	radius: number

	constructor(r: number)
		self.radius = r
	end

	draw(): void
		local x: number = 1
	end

	getArea(): number
		return 3.14
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

func TestClassImplementsInterfaceIncompatible(t *testing.T) {
	input := `
interface Drawable
	draw(): void
	getArea(): number
end

class Circle implements Drawable
	radius: number

	constructor(r: number)
		self.radius = r
	end

	draw(): void
		local x: number = 1
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

	// Should have error: missing getArea method
	if len(errors) < 1 {
		t.Errorf("Expected at least 1 type error (missing method), got %d", len(errors))
	}
}

func TestInterfaceInheritance(t *testing.T) {
	input := `
interface Animal
	name: string
	makeSound(): void
end

interface Dog extends Animal
	breed: string
	bark(): void
end

local dog: Dog = {
	name = "Rex",
	breed = "Labrador",
	makeSound = function() end,
	bark = function() end
}
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

func TestInterfaceInFunction(t *testing.T) {
	input := `
interface Point
	x: number
	y: number
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

func TestInterfaceMissingProperty(t *testing.T) {
	input := `
interface Person
	name: string
	age: number
end

local p: Person = { name = "John" }
`

	l := lexer.New(input)
	p := parser.New(l)
	statements := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	checker := NewChecker()
	errors := checker.Check(statements)

	// Should have error: missing age property
	if len(errors) != 1 {
		t.Errorf("Expected 1 type error (missing property), got %d:", len(errors))
		for _, err := range errors {
			t.Errorf("  %s", err.Message)
		}
	}
}

func TestInterfaceWrongPropertyType(t *testing.T) {
	input := `
interface Person
	name: string
	age: number
end

local p: Person = { name = "John", age = "thirty" }
`

	l := lexer.New(input)
	p := parser.New(l)
	statements := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	checker := NewChecker()
	errors := checker.Check(statements)

	// Should have error: wrong type for age
	if len(errors) != 1 {
		t.Errorf("Expected 1 type error (wrong type), got %d:", len(errors))
		for _, err := range errors {
			t.Errorf("  %s", err.Message)
		}
	}
}
