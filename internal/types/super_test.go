package types

import (
	"lunar/internal/lexer"
	"lunar/internal/parser"
	"testing"
)

func TestSuperInConstructor(t *testing.T) {
	input := `
class Animal
	name: string

	constructor(name: string)
		self.name = name
	end
end

class Dog extends Animal
	breed: string

	constructor(name: string, breed: string)
		super(name)
		self.breed = breed
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

	// Should have no errors: valid super usage in constructor
	if len(errors) > 0 {
		t.Errorf("Expected no type errors, got %d:", len(errors))
		for _, err := range errors {
			t.Errorf("  %s", err.Message)
		}
	}
}

func TestSuperMethodCall(t *testing.T) {
	input := `
class Animal
	speak(): void
		local x: number = 1
	end
end

class Dog extends Animal
	speak(): void
		super.speak()
		local y: number = 2
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

	// Should have no errors: valid super method call
	if len(errors) > 0 {
		t.Errorf("Expected no type errors, got %d:", len(errors))
		for _, err := range errors {
			t.Errorf("  %s", err.Message)
		}
	}
}

func TestSuperOutsideClass(t *testing.T) {
	input := `
function test(): void
	super.something()
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

	// Should have 1 error: super used outside class
	if len(errors) != 1 {
		t.Errorf("Expected 1 type error, got %d:", len(errors))
		for _, err := range errors {
			t.Errorf("  %s", err.Message)
		}
		return
	}

	expectedMsg := "'super' can only be used inside a class"
	if errors[0].Message != expectedMsg {
		t.Errorf("Expected error: %s\nGot: %s", expectedMsg, errors[0].Message)
	}
}

func TestSuperWithoutParent(t *testing.T) {
	input := `
class Animal
	speak(): void
		super.speak()
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

	// Should have 1 error: class has no parent
	if len(errors) != 1 {
		t.Errorf("Expected 1 type error, got %d:", len(errors))
		for _, err := range errors {
			t.Errorf("  %s", err.Message)
		}
		return
	}

	expectedMsg := "Class 'Animal' has no parent class, cannot use 'super'"
	if errors[0].Message != expectedMsg {
		t.Errorf("Expected error: %s\nGot: %s", expectedMsg, errors[0].Message)
	}
}

func TestSuperPropertyAccess(t *testing.T) {
	input := `
class Animal
	protected name: string
end

class Dog extends Animal
	getName(): string
		return super.name
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

	// Should have no errors: accessing parent property via super
	if len(errors) > 0 {
		t.Errorf("Expected no type errors, got %d:", len(errors))
		for _, err := range errors {
			t.Errorf("  %s", err.Message)
		}
	}
}

func TestSuperMultipleLevels(t *testing.T) {
	input := `
class Animal
	speak(): void
		local x: number = 1
	end
end

class Mammal extends Animal
end

class Dog extends Mammal
	speak(): void
		super.speak()
		local y: number = 2
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

	// Should have no errors: super accesses immediate parent (Mammal),
	// which inherits from Animal
	if len(errors) > 0 {
		t.Errorf("Expected no type errors, got %d:", len(errors))
		for _, err := range errors {
			t.Errorf("  %s", err.Message)
		}
	}
}

func TestSuperWithMethodParameters(t *testing.T) {
	input := `
class Animal
	eat(food: string): void
		local x: number = 1
	end
end

class Dog extends Animal
	eat(food: string): void
		super.eat(food)
		local y: number = 2
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

	// Should have no errors: calling parent method with correct parameters
	if len(errors) > 0 {
		t.Errorf("Expected no type errors, got %d:", len(errors))
		for _, err := range errors {
			t.Errorf("  %s", err.Message)
		}
	}
}
