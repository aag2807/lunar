package types

import (
	"lunar/internal/lexer"
	"lunar/internal/parser"
	"testing"
)

func TestClassInheritance(t *testing.T) {
	input := `
class Animal
	name: string
end

class Dog extends Animal
	breed: string
end

local dog: Dog = Dog()
local animal: Animal = dog
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

func TestAbstractClassInheritance(t *testing.T) {
	input := `
abstract class Animal
	abstract speak(): void
end

class Dog extends Animal
	speak(): void
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

	if len(errors) > 0 {
		t.Errorf("Expected no type errors, got %d:", len(errors))
		for _, err := range errors {
			t.Errorf("  %s", err.Message)
		}
	}
}

func TestAbstractMethodNotImplemented(t *testing.T) {
	input := `
abstract class Animal
	abstract speak(): void
end

class Dog extends Animal
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

	// Should have 1 error: abstract method not implemented
	if len(errors) != 1 {
		t.Errorf("Expected 1 type error, got %d:", len(errors))
		for _, err := range errors {
			t.Errorf("  %s", err.Message)
		}
		return
	}

	expectedMsg := "Class 'Dog' must implement abstract method 'speak' from parent class 'Animal'"
	if errors[0].Message != expectedMsg {
		t.Errorf("Expected error: %s\nGot: %s", expectedMsg, errors[0].Message)
	}
}

func TestParentPropertyAccess(t *testing.T) {
	input := `
class Animal
	name: string
end

class Dog extends Animal
	breed: string
end

local dog: Dog = Dog()
local n: string = dog.name
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

func TestMultiLevelInheritance(t *testing.T) {
	input := `
class Animal
	name: string
end

class Mammal extends Animal
	furColor: string
end

class Dog extends Mammal
	breed: string
end

local dog: Dog = Dog()
local n: string = dog.name
local fur: string = dog.furColor
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
