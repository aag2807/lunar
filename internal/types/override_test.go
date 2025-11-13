package types

import (
	"lunar/internal/lexer"
	"lunar/internal/parser"
	"testing"
)

func TestValidMethodOverride(t *testing.T) {
	input := `
class Animal
	speak(): void
		local x: number = 1
	end
end

class Dog extends Animal
	speak(): void
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

	// Should have no errors: valid override with same signature
	if len(errors) > 0 {
		t.Errorf("Expected no type errors, got %d:", len(errors))
		for _, err := range errors {
			t.Errorf("  %s", err.Message)
		}
	}
}

func TestOverrideWithDifferentParameterCount(t *testing.T) {
	input := `
class Animal
	speak(): void
		local x: number = 1
	end
end

class Dog extends Animal
	speak(message: string): void
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

	// Should have 1 error: parameter count mismatch
	if len(errors) != 1 {
		t.Errorf("Expected 1 type error, got %d:", len(errors))
		for _, err := range errors {
			t.Errorf("  %s", err.Message)
		}
		return
	}

	expectedMsg := "Method 'speak' override has 1 parameters, but parent method has 0 parameters"
	if errors[0].Message != expectedMsg {
		t.Errorf("Expected error: %s\nGot: %s", expectedMsg, errors[0].Message)
	}
}

func TestOverrideWithDifferentParameterType(t *testing.T) {
	input := `
class Animal
	eat(food: string): void
		local x: number = 1
	end
end

class Dog extends Animal
	eat(food: number): void
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

	// Should have 1 error: parameter type mismatch
	if len(errors) != 1 {
		t.Errorf("Expected 1 type error, got %d:", len(errors))
		for _, err := range errors {
			t.Errorf("  %s", err.Message)
		}
		return
	}

	expectedMsg := "Method 'eat' override parameter 1 has type 'number', but parent method expects 'string'"
	if errors[0].Message != expectedMsg {
		t.Errorf("Expected error: %s\nGot: %s", expectedMsg, errors[0].Message)
	}
}

func TestOverrideWithDifferentReturnType(t *testing.T) {
	input := `
class Animal
	getAge(): number
		return 5
	end
end

class Dog extends Animal
	getAge(): string
		return "five"
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

	// Should have 1 error: return type mismatch
	if len(errors) != 1 {
		t.Errorf("Expected 1 type error, got %d:", len(errors))
		for _, err := range errors {
			t.Errorf("  %s", err.Message)
		}
		return
	}

	expectedMsg := "Method 'getAge' override has return type 'string', but parent method returns 'number'"
	if errors[0].Message != expectedMsg {
		t.Errorf("Expected error: %s\nGot: %s", expectedMsg, errors[0].Message)
	}
}

func TestOverrideVisibilityReduction(t *testing.T) {
	input := `
class Animal
	public speak(): void
		local x: number = 1
	end
end

class Dog extends Animal
	private speak(): void
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

	// Should have 1 error: visibility reduction
	if len(errors) != 1 {
		t.Errorf("Expected 1 type error, got %d:", len(errors))
		for _, err := range errors {
			t.Errorf("  %s", err.Message)
		}
		return
	}

	expectedMsg := "Method 'speak' override cannot reduce visibility from public to private"
	if errors[0].Message != expectedMsg {
		t.Errorf("Expected error: %s\nGot: %s", expectedMsg, errors[0].Message)
	}
}

func TestOverrideVisibilityExpansion(t *testing.T) {
	input := `
class Animal
	protected speak(): void
		local x: number = 1
	end
end

class Dog extends Animal
	public speak(): void
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

	// Should have no errors: expanding visibility is allowed
	if len(errors) > 0 {
		t.Errorf("Expected no type errors, got %d:", len(errors))
		for _, err := range errors {
			t.Errorf("  %s", err.Message)
		}
	}
}

func TestOverrideCovariantReturnType(t *testing.T) {
	input := `
class Animal
end

class Dog extends Animal
end

class AnimalFactory
	create(): Animal
		return Animal()
	end
end

class DogFactory extends AnimalFactory
	create(): Dog
		return Dog()
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

	// Should have no errors: covariant return type (Dog is subtype of Animal)
	if len(errors) > 0 {
		t.Errorf("Expected no type errors, got %d:", len(errors))
		for _, err := range errors {
			t.Errorf("  %s", err.Message)
		}
	}
}

func TestOverrideMultipleLevels(t *testing.T) {
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

	// Should have no errors: overriding grandparent method
	if len(errors) > 0 {
		t.Errorf("Expected no type errors, got %d:", len(errors))
		for _, err := range errors {
			t.Errorf("  %s", err.Message)
		}
	}
}
